use std::{
    collections::{BTreeMap, HashMap, HashSet},
    env,
    sync::{Arc, OnceLock},
    time::Duration,
};

use anyhow::{Context, Result, anyhow, bail};
use askama::Template;
use axum::{
    Router,
    extract::{Form, Path, Query, State, rejection::FormRejection},
    http::{HeaderName, HeaderValue, StatusCode, header},
    response::{IntoResponse, Redirect, Response},
    routing::{any, get, post},
};
use firebase_sdk_rust::firebase_firestore::{Document, FirebaseFirestoreClient, RunQueryRequest};
use gcp_auth::{CustomServiceAccount, TokenProvider, provider};
use serde::{Deserialize, Serialize, de::DeserializeOwned};
use serde_json::{Value as JsonValue, json};
use tower_http::services::ServeDir;
use tower_http::set_header::SetResponseHeaderLayer;
use uuid::Uuid;

const DATASTORE_SCOPE: &str = "https://www.googleapis.com/auth/datastore";
const DEFAULT_KANJI_CANDIDATE_COUNT: usize = 6;
const ADMIN_PROXY_MAX_BODY_BYTES: usize = 16 * 1024 * 1024;
const HX_REDIRECT_HEADER: &str = "hx-redirect";
const WEB_STATIC_DIR: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/static");
const WEB_BLOG_CONTENT_DIR: &str = concat!(env!("CARGO_MANIFEST_DIR"), "/content/blog");
const LANGUAGE_REGISTRY_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../config/languages.json"
));
const EXTERNAL_LEGAL_BASE_URL: &str = "https://finitefield.org";
const DEFAULT_LOCALE: &str = "en";
static WEB_LANGUAGE_REGISTRY: OnceLock<WebLanguageRegistry> = OnceLock::new();
static WEB_COPY_DOCUMENT: OnceLock<WebCopyDocument> = OnceLock::new();

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum RunMode {
    Mock,
    Dev,
    Prod,
}

impl RunMode {
    fn as_str(self) -> &'static str {
        match self {
            Self::Mock => "mock",
            Self::Dev => "dev",
            Self::Prod => "prod",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct WebLanguageRegistry {
    languages: Vec<WebLanguage>,
    default_route_code: String,
}

impl WebLanguageRegistry {
    fn from_json(source: &str) -> Result<Self> {
        let entries: Vec<LanguageRegistryEntry> =
            serde_json::from_str(source).context("failed to parse language registry JSON")?;
        Self::from_entries(entries)
    }

    fn from_entries(entries: Vec<LanguageRegistryEntry>) -> Result<Self> {
        let mut route_codes = HashSet::new();
        let mut enabled_prefixes = HashSet::new();
        let mut languages = Vec::new();

        for entry in entries {
            let route_code = normalize_registry_code(&entry.route_code);
            if route_code.is_empty() {
                bail!("language registry route_code must not be empty");
            }
            if !route_codes.insert(route_code.clone()) {
                bail!("duplicate language registry route_code: {route_code}");
            }
            if entry.web.indexed && !entry.web.enabled {
                bail!("indexed web language must also be enabled: {route_code}");
            }
            if !entry.web.enabled {
                continue;
            }

            let url_prefix = entry.web.url_prefix.trim().trim_matches('/').to_lowercase();
            if route_code != DEFAULT_LOCALE && url_prefix.is_empty() {
                bail!("non-default enabled web language must define url_prefix: {route_code}");
            }
            if !url_prefix.is_empty() && !enabled_prefixes.insert(url_prefix.clone()) {
                bail!("duplicate enabled web url_prefix: {url_prefix}");
            }

            languages.push(WebLanguage {
                route_code,
                bcp47: entry.bcp47.trim().to_owned(),
                native_name: entry.native_name.trim().to_owned(),
                english_name: entry.english_name.trim().to_owned(),
                text_direction: entry.text_direction,
                url_prefix,
                indexed: entry.web.indexed,
            });
        }

        if languages.is_empty() {
            bail!("language registry must include at least one enabled web language");
        }

        let Some(default_language) = languages
            .iter()
            .find(|language| language.route_code == DEFAULT_LOCALE)
        else {
            bail!("language registry must include enabled default locale {DEFAULT_LOCALE}");
        };
        if !default_language.url_prefix.is_empty() {
            bail!("default web language must use an empty url_prefix");
        }

        Ok(Self {
            languages,
            default_route_code: DEFAULT_LOCALE.to_owned(),
        })
    }

    fn enabled_languages(&self) -> &[WebLanguage] {
        &self.languages
    }

    fn indexed_languages(&self) -> impl Iterator<Item = &WebLanguage> {
        self.languages.iter().filter(|language| language.indexed)
    }

    fn default_language(&self) -> &WebLanguage {
        self.enabled_language_exact(&self.default_route_code)
            .expect("default web language must exist")
    }

    fn enabled_language_exact(&self, route_code: &str) -> Option<&WebLanguage> {
        let normalized = normalize_registry_code(route_code);
        self.languages
            .iter()
            .find(|language| language.route_code == normalized)
    }

    fn enabled_language_for_input(&self, raw: &str) -> Option<&WebLanguage> {
        let normalized = normalize_registry_code(raw);
        if normalized.is_empty() {
            return None;
        }
        if normalized == "jp" {
            return self.enabled_language_exact("ja");
        }
        if let Some(language) = self.enabled_language_exact(&normalized) {
            return Some(language);
        }
        if let Some(language) = self
            .languages
            .iter()
            .find(|language| normalize_registry_code(&language.bcp47) == normalized)
        {
            return Some(language);
        }
        let language = normalized
            .split(['-', '_'])
            .next()
            .unwrap_or(normalized.as_str());
        self.enabled_language_exact(language)
    }

    fn enabled_language_for_path_segment(&self, raw: &str) -> Option<&WebLanguage> {
        let normalized = normalize_registry_code(raw);
        if normalized.is_empty() {
            return None;
        }
        if normalized == "jp" {
            return self.enabled_language_exact("ja");
        }
        self.languages.iter().find(|language| {
            language.route_code == normalized
                || (!language.url_prefix.is_empty() && language.url_prefix == normalized)
        })
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct WebLanguage {
    route_code: String,
    bcp47: String,
    native_name: String,
    english_name: String,
    text_direction: RegistryTextDirection,
    url_prefix: String,
    indexed: bool,
}

#[derive(Debug, Clone, Copy, Deserialize, PartialEq, Eq)]
#[serde(rename_all = "lowercase")]
enum RegistryTextDirection {
    Ltr,
    Rtl,
}

#[derive(Debug, Deserialize)]
struct LanguageRegistryEntry {
    route_code: String,
    bcp47: String,
    native_name: String,
    english_name: String,
    text_direction: RegistryTextDirection,
    web: LanguageRegistryWebConfig,
}

#[derive(Debug, Deserialize)]
struct LanguageRegistryWebConfig {
    enabled: bool,
    indexed: bool,
    url_prefix: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct LanguageLink {
    route_code: String,
    bcp47: String,
    label: String,
    url: String,
    is_default: bool,
    is_indexed: bool,
}

#[derive(Debug)]
struct WebCopyDocument {
    common: LocalizedCopySection,
    about: LocalizedCopySection,
    blog_article: LocalizedCopySection,
    blog_index: LocalizedCopySection,
    commercial_transactions: LocalizedCopySection,
    design: LocalizedCopySection,
    kanji_suggestions: LocalizedCopySection,
    payment_failure: LocalizedCopySection,
    payment_success: LocalizedCopySection,
    purchase_result: LocalizedCopySection,
    terms: LocalizedCopySection,
    top: LocalizedCopySection,
}

macro_rules! web_copy_section {
    ($section:literal) => {
        LocalizedCopySection::from_sources(
            $section,
            include_str!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/content/i18n/",
                $section,
                "/en.json"
            )),
            include_str!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/content/i18n/",
                $section,
                "/ja.json"
            )),
            include_str!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/content/i18n/",
                $section,
                "/zh.json"
            )),
            include_str!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/content/i18n/",
                $section,
                "/zhtw.json"
            )),
            include_str!(concat!(
                env!("CARGO_MANIFEST_DIR"),
                "/content/i18n/",
                $section,
                "/ar.json"
            )),
        )
    };
}

impl WebCopyDocument {
    fn load() -> Self {
        Self {
            common: web_copy_section!("common"),
            about: web_copy_section!("about"),
            blog_article: web_copy_section!("blog_article"),
            blog_index: web_copy_section!("blog_index"),
            commercial_transactions: web_copy_section!("commercial_transactions"),
            design: web_copy_section!("design"),
            kanji_suggestions: web_copy_section!("kanji_suggestions"),
            payment_failure: web_copy_section!("payment_failure"),
            payment_success: web_copy_section!("payment_success"),
            purchase_result: web_copy_section!("purchase_result"),
            terms: web_copy_section!("terms"),
            top: web_copy_section!("top"),
        }
    }

    fn section(&self, section: &str) -> &LocalizedCopySection {
        match section {
            "common" => &self.common,
            "about" => &self.about,
            "blog_article" => &self.blog_article,
            "blog_index" => &self.blog_index,
            "commercial_transactions" => &self.commercial_transactions,
            "design" => &self.design,
            "kanji_suggestions" => &self.kanji_suggestions,
            "payment_failure" => &self.payment_failure,
            "payment_success" => &self.payment_success,
            "purchase_result" => &self.purchase_result,
            "terms" => &self.terms,
            "top" => &self.top,
            _ => panic!("unknown web copy section: {section}"),
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct LocalizedCopySection {
    en: HashMap<String, String>,
    ja: HashMap<String, String>,
    zh: HashMap<String, String>,
    zhtw: HashMap<String, String>,
    ar: HashMap<String, String>,
}

impl LocalizedCopySection {
    fn from_sources(section: &str, en: &str, ja: &str, zh: &str, zhtw: &str, ar: &str) -> Self {
        let en = parse_web_copy_locale(section, "en", en);
        let ja = parse_web_copy_locale(section, "ja", ja);
        let zh = parse_web_copy_locale(section, "zh", zh);
        let zhtw = parse_web_copy_locale(section, "zhtw", zhtw);
        let ar = parse_web_copy_locale(section, "ar", ar);
        assert_web_copy_key_parity(
            section,
            &en,
            &[("ja", &ja), ("zh", &zh), ("zhtw", &zhtw), ("ar", &ar)],
        );
        Self {
            en,
            ja,
            zh,
            zhtw,
            ar,
        }
    }

    fn for_locale(&self, locale: &str) -> &HashMap<String, String> {
        match web_copy_locale_key(locale) {
            "ja" => &self.ja,
            "zh" => &self.zh,
            "zhtw" => &self.zhtw,
            "ar" => &self.ar,
            _ => &self.en,
        }
    }
}

fn parse_web_copy_locale(section: &str, locale: &str, source: &str) -> HashMap<String, String> {
    serde_json::from_str(source).unwrap_or_else(|error| {
        panic!("checked-in web copy JSON must be valid: {section}/{locale}: {error}")
    })
}

fn assert_web_copy_key_parity(
    section: &str,
    en: &HashMap<String, String>,
    localized_maps: &[(&str, &HashMap<String, String>)],
) {
    let en_keys = en.keys().collect::<HashSet<_>>();
    for (locale, localized) in localized_maps {
        let localized_keys = localized.keys().collect::<HashSet<_>>();
        if localized_keys != en_keys {
            let missing = en_keys
                .difference(&localized_keys)
                .map(|key| key.as_str())
                .collect::<Vec<_>>();
            let extra = localized_keys
                .difference(&en_keys)
                .map(|key| key.as_str())
                .collect::<Vec<_>>();
            panic!(
                "web copy key mismatch in {section}/{locale}: missing={missing:?} extra={extra:?}"
            );
        }
    }
}

fn web_language_registry() -> &'static WebLanguageRegistry {
    WEB_LANGUAGE_REGISTRY.get_or_init(|| {
        WebLanguageRegistry::from_json(LANGUAGE_REGISTRY_JSON)
            .expect("checked-in language registry must be valid for web")
    })
}

fn web_copy_document() -> &'static WebCopyDocument {
    WEB_COPY_DOCUMENT.get_or_init(WebCopyDocument::load)
}

fn web_copy_locale_key(locale: &str) -> &'static str {
    let normalized = normalize_registry_code(locale).replace('_', "-");
    if normalized == "ja" || normalized == "jp" || normalized.starts_with("ja-") {
        return "ja";
    }
    if normalized == "zhtw"
        || normalized == "zh-hant"
        || normalized == "zh-tw"
        || normalized == "zh-hk"
        || normalized == "zh-mo"
        || normalized.starts_with("zh-hant-")
    {
        return "zhtw";
    }
    if normalized == "zh" || normalized == "zh-hans" || normalized.starts_with("zh-") {
        return "zh";
    }
    if normalized == "ar" || normalized.starts_with("ar-") {
        return "ar";
    }
    "en"
}

fn web_copy_text(section: &str, locale: &str, key: &str) -> &'static str {
    let section_name = section;
    let section = web_copy_document().section(section);
    let localized = section.for_locale(locale);
    localized
        .get(key)
        .or_else(|| section.en.get(key))
        .map(String::as_str)
        .unwrap_or_else(|| panic!("missing web copy key: {section_name}.{key}"))
}

fn normalize_registry_code(raw: &str) -> String {
    raw.trim().to_lowercase()
}

#[derive(Debug, Clone)]
struct AppConfig {
    port: String,
    mode: RunMode,
    locale: String,
    default_locale: String,
    site_base_url: String,
    api_base_url: String,
    admin_base_url: String,
    firestore_project_id: Option<String>,
    credentials_file: Option<String>,
    storage_assets_bucket: Option<String>,
}

#[derive(Debug, Clone)]
struct FontOption {
    key: String,
    label: String,
    family: String,
    stylesheet_url: String,
    kanji_style: String,
}

#[derive(Debug, Clone)]
struct MaterialPhoto {
    asset_id: String,
    storage_path: String,
    alt_i18n: HashMap<String, String>,
    sort_order: i64,
    is_primary: bool,
}

#[derive(Debug, Clone)]
struct MaterialOption {
    key: String,
    label: String,
    description: String,
    story: String,
    has_description: bool,
    has_story: bool,
    price_by_currency: HashMap<String, i64>,
    shape: String,
    shape_label: String,
    color_family: String,
    pattern_primary: String,
    color_tag_labels: Vec<String>,
    pattern_tag_labels: Vec<String>,
    has_color_tag_labels: bool,
    has_pattern_tag_labels: bool,
    price: i64,
    price_display: String,
    photo_url: String,
    photo_alt: String,
    has_photo: bool,
}

#[derive(Debug, Clone)]
struct MaterialCategory {
    label: String,
}

#[derive(Debug, Clone)]
struct MaterialFilterOption {
    value: String,
    label: String,
}

#[derive(Debug, Clone, Default)]
struct MaterialFilters {
    color_options: Vec<MaterialFilterOption>,
    pattern_options: Vec<MaterialFilterOption>,
}

#[derive(Debug, Clone, Default)]
struct MaterialFilterState {
    color_family: String,
    pattern_primary: String,
}

#[derive(Debug, Clone)]
struct BlogPostCard {
    title: String,
    excerpt: String,
    image_url: String,
    image_alt: String,
    post_url: String,
}

#[derive(Debug, Clone)]
struct BlogPost {
    slug: String,
    published_date: String,
    last_modified_date: String,
    date_display: String,
    date_display_ja: String,
    title: String,
    title_ja: String,
    excerpt: String,
    excerpt_ja: String,
    meta_description: String,
    meta_description_ja: String,
    image_url: String,
    image_alt: String,
    image_alt_ja: String,
}

#[derive(Debug, Deserialize)]
struct BlogPostMetadata {
    slug: String,
    published_date: String,
    last_modified_date: String,
    image_url: String,
    locales: HashMap<String, BlogPostLocaleMetadata>,
}

#[derive(Debug, Deserialize)]
struct BlogPostLocaleMetadata {
    date_display: String,
    title: String,
    excerpt: String,
    meta_description: String,
    image_alt: String,
}

#[derive(Debug, Clone)]
struct BlogPostView {
    published_date: String,
    date_display: String,
    title: String,
    excerpt: String,
    meta_description: String,
    image_url: String,
    image_alt: String,
}

#[derive(Debug, Clone, Copy)]
struct SitemapPage {
    path: &'static str,
    lastmod: &'static str,
}

#[derive(Debug, Clone, Default)]
struct FacetTagLabels {
    labels_by_type: HashMap<String, HashMap<String, String>>,
}

impl FacetTagLabels {
    fn insert(&mut self, facet_type: &str, key: &str, label: &str, aliases: &[String]) {
        let facet_type = normalize_facet_tag_value(facet_type);
        let key = normalize_facet_tag_value(key);
        let label = label.trim().to_owned();
        if facet_type.is_empty() || key.is_empty() || label.is_empty() {
            return;
        }

        let labels = self.labels_by_type.entry(facet_type).or_default();
        labels.insert(key, label.clone());
        for alias in aliases {
            let alias = normalize_facet_tag_value(alias);
            if !alias.is_empty() {
                labels.insert(alias, label.clone());
            }
        }
    }

    fn resolve_or_raw(&self, facet_type: &str, value: &str) -> String {
        let facet_type = normalize_facet_tag_value(facet_type);
        let value = value.trim();
        if facet_type.is_empty() || value.is_empty() {
            return String::new();
        }

        self.labels_by_type
            .get(&facet_type)
            .and_then(|labels| labels.get(&normalize_facet_tag_value(value)).cloned())
            .unwrap_or_else(|| value.to_owned())
    }

    fn resolve_list(&self, facet_type: &str, values: &[String]) -> Vec<String> {
        let mut labels = Vec::new();
        let mut seen = HashSet::new();

        for value in values {
            let label = self.resolve_or_raw(facet_type, value);
            if label.is_empty() {
                continue;
            }

            if seen.insert(normalize_facet_tag_value(&label)) {
                labels.push(label);
            }
        }

        labels
    }
}

#[derive(Debug, Clone)]
struct StoneListingRecord {
    key: String,
    material_key: String,
    title: String,
    description: String,
    story: String,
    price_by_currency: HashMap<String, i64>,
    color_family: String,
    pattern_primary: String,
    color_tags: Vec<String>,
    pattern_tags: Vec<String>,
    stone_shape: String,
    photos: Vec<MaterialPhoto>,
}

#[derive(Debug, Clone)]
struct CountryOption {
    code: String,
    label: String,
    shipping_fee_by_currency: HashMap<String, i64>,
    shipping: i64,
}

#[derive(Debug, Clone)]
struct CatalogData {
    fonts: Vec<FontOption>,
    materials: Vec<MaterialOption>,
    countries: Vec<CountryOption>,
    material_filters: MaterialFilters,
}

#[derive(Debug, Clone)]
struct KanjiCandidate {
    kanji: String,
    line1: String,
    line2: String,
    reading: String,
    reason: String,
}

#[derive(Debug, Clone, Default)]
struct PurchaseResultData {
    error: String,
    selected_locale: String,
    seal_line1: String,
    seal_line2: String,
    font_label: String,
    shape_label: String,
    listing_label: String,
    stripe_name: String,
    stripe_phone: String,
    country_label: String,
    postal_code: String,
    state: String,
    city: String,
    address_line1: String,
    address_line2: String,
    subtotal: i64,
    shipping: i64,
    total: i64,
    email: String,
    source_label: String,
    is_mock: bool,
}

#[derive(Template)]
#[template(path = "top.html")]
struct TopPageTemplate {
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    design_url: String,
    blog_index_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
    blog_posts: Vec<BlogPostCard>,
}

#[derive(Template)]
#[template(path = "about.html")]
struct AboutTemplate {
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    design_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "index.html")]
struct PageTemplate {
    fonts: Vec<FontOption>,
    font_stylesheet_urls: Vec<String>,
    materials: Vec<MaterialOption>,
    countries: Vec<CountryOption>,
    material_filters: MaterialFilters,
    selected_color_family: String,
    selected_pattern_primary: String,
    default_font_key: String,
    default_font_label: String,
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    purchase_action_url: String,
    purchase_note: String,
    top_url: String,
    about_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "kanji_suggestions.html")]
struct KanjiSuggestionsTemplate {
    real_name: String,
    kanji_style: String,
    selected_locale: String,
    suggestions: Vec<KanjiCandidate>,
    has_suggestions: bool,
    error: String,
}

#[derive(Debug, Deserialize)]
struct KanjiCandidatesApiResponse {
    candidates: Vec<KanjiCandidatesApiItem>,
}

#[derive(Debug, Deserialize)]
struct KanjiCandidatesApiItem {
    kanji: String,
    #[serde(alias = "reading_romaji", alias = "romaji")]
    reading: String,
    reason: String,
}

#[derive(Debug, Serialize)]
struct CreateOrderApiRequest {
    channel: String,
    locale: String,
    idempotency_key: String,
    terms_agreed: bool,
    seal: CreateOrderSealApiRequest,
    listing_id: String,
    shipping: CreateOrderShippingApiRequest,
    contact: CreateOrderContactApiRequest,
}

#[derive(Debug, Serialize)]
struct CreateOrderSealApiRequest {
    line1: String,
    line2: String,
    shape: String,
    font_key: String,
}

#[derive(Debug, Serialize)]
struct CreateOrderShippingApiRequest {
    country_code: String,
    recipient_name: String,
    phone: String,
    postal_code: String,
    state: String,
    city: String,
    address_line1: String,
    address_line2: String,
}

#[derive(Debug, Serialize)]
struct CreateOrderContactApiRequest {
    email: String,
    preferred_locale: String,
}

#[derive(Debug, Deserialize)]
struct CreateOrderApiResponse {
    order_id: String,
}

#[derive(Debug, Serialize)]
struct CreateStripeCheckoutSessionApiRequest {
    order_id: String,
    customer_email: String,
}

#[derive(Debug, Deserialize)]
struct CreateStripeCheckoutSessionApiResponse {
    checkout_url: String,
}

#[derive(Debug, Deserialize)]
struct ApiErrorEnvelope {
    error: ApiErrorBody,
}

#[derive(Debug, Deserialize)]
struct ApiErrorBody {
    code: String,
    message: String,
}

#[derive(Template)]
#[template(path = "purchase_result.html")]
struct PurchaseResultTemplate {
    has_error: bool,
    error: String,
    selected_locale: String,
    seal_line1: String,
    seal_line2: String,
    has_seal_line2: bool,
    font_label: String,
    shape_label: String,
    listing_label: String,
    stripe_name: String,
    stripe_phone: String,
    country_label: String,
    postal_code: String,
    state: String,
    city: String,
    address_line1: String,
    address_line2: String,
    has_address_line2: bool,
    subtotal_display: String,
    shipping_display: String,
    total_display: String,
    email: String,
    source_label: String,
    is_mock: bool,
}

#[derive(Template)]
#[template(path = "payment_success.html")]
struct PaymentSuccessTemplate {
    has_session_id: bool,
    session_id: String,
    has_order_id: bool,
    order_id: String,
    has_app_redirect_url: bool,
    app_redirect_url: String,
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    contact_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "payment_failure.html")]
struct PaymentFailureTemplate {
    has_order_id: bool,
    order_id: String,
    has_app_redirect_url: bool,
    app_redirect_url: String,
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    design_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    contact_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "commercial_transactions.html")]
struct CommercialTransactionsTemplate {
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    contact_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "terms.html")]
struct TermsTemplate {
    contact_url: String,
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    about_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
}

#[derive(Template)]
#[template(path = "blog_index.html")]
struct BlogIndexTemplate {
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    company_url: String,
    top_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
    blog_posts: Vec<BlogPostCard>,
}

#[derive(Template)]
#[template(path = "blog_article.html")]
struct BlogArticleTemplate {
    selected_locale: String,
    page_title: String,
    meta_description: String,
    robots_meta: String,
    canonical_url: String,
    x_default_url: String,
    seo_language_links: Vec<LanguageLink>,
    language_links: Vec<LanguageLink>,
    og_image_url: String,
    company_url: String,
    top_url: String,
    blog_index_url: String,
    terms_url: String,
    commercial_transactions_url: String,
    privacy_policy_url: String,
    post: BlogPostView,
    body_html: String,
}

macro_rules! impl_template_copy_methods {
    ($type:ty, $section:literal) => {
        impl $type {
            #[allow(dead_code)]
            fn copy_text(&self, key: &str) -> &str {
                web_copy_text($section, &self.selected_locale, key)
            }

            #[allow(dead_code)]
            fn copy_html(&self, key: &str) -> &str {
                web_copy_text($section, &self.selected_locale, key)
            }

            #[allow(dead_code)]
            fn language_active_class(&self, route_code: &str) -> &str {
                if self.selected_locale == route_code {
                    " is-active"
                } else {
                    ""
                }
            }

            #[allow(dead_code)]
            fn html_dir(&self) -> &str {
                html_dir_for_locale(&self.selected_locale)
            }
        }
    };
}

impl_template_copy_methods!(TopPageTemplate, "top");
impl_template_copy_methods!(AboutTemplate, "about");
impl_template_copy_methods!(PageTemplate, "design");
impl_template_copy_methods!(KanjiSuggestionsTemplate, "kanji_suggestions");
impl_template_copy_methods!(PurchaseResultTemplate, "purchase_result");
impl_template_copy_methods!(PaymentSuccessTemplate, "payment_success");
impl_template_copy_methods!(PaymentFailureTemplate, "payment_failure");
impl_template_copy_methods!(CommercialTransactionsTemplate, "commercial_transactions");
impl_template_copy_methods!(TermsTemplate, "terms");
impl_template_copy_methods!(BlogIndexTemplate, "blog_index");
impl_template_copy_methods!(BlogArticleTemplate, "blog_article");

fn html_dir_for_locale(locale: &str) -> &'static str {
    match locale.trim().to_ascii_lowercase().as_str() {
        "ar" | "fa" | "he" | "ps" | "ur" => "rtl",
        _ => "ltr",
    }
}

#[derive(Debug, Deserialize, Default)]
struct PaymentRedirectQuery {
    checkout: Option<String>,
    session_id: Option<String>,
    order_id: Option<String>,
    lang: Option<String>,
    return_to: Option<String>,
    color_family: Option<String>,
    pattern_primary: Option<String>,
}

#[derive(Debug, Deserialize, Default)]
struct LocaleQuery {
    lang: Option<String>,
}

#[derive(Clone)]
enum CatalogSource {
    Mock(MockCatalogSource),
    Firestore(FirestoreCatalogSource),
}

impl CatalogSource {
    fn label(&self) -> &str {
        match self {
            Self::Mock(_) => "Mock",
            Self::Firestore(source) => &source.label,
        }
    }

    async fn load_catalog(&self, locale: &str) -> Result<CatalogData> {
        match self {
            Self::Mock(source) => {
                let _ = &source.catalog;
                let mut catalog = new_mock_catalog_source(locale).catalog;
                catalog.material_filters =
                    build_material_filters(&catalog.materials, &mock_facet_tag_labels(locale));
                Ok(catalog)
            }
            Self::Firestore(source) => source.load_catalog(locale).await,
        }
    }
}

#[derive(Debug, Clone)]
struct MockCatalogSource {
    catalog: CatalogData,
}

#[derive(Clone)]
struct FirestoreCatalogSource {
    project_id: String,
    default_locale: String,
    label: String,
    storage_assets_bucket: String,
    allow_mock_fallback: bool,
    token_provider: Arc<dyn TokenProvider>,
}

impl FirestoreCatalogSource {
    async fn load_catalog(&self, locale: &str) -> Result<CatalogData> {
        let access_token = self
            .token_provider
            .token(&[DATASTORE_SCOPE])
            .await
            .context("failed to acquire firestore access token")?;

        let client = Self::firestore_client_from_access_token(access_token.as_str())?;
        let parent = format!("projects/{}/databases/(default)/documents", self.project_id);
        let fallback_catalog = self
            .allow_mock_fallback
            .then(|| new_mock_catalog_source(locale).catalog);
        let facet_tag_labels = match self.load_facet_tag_labels(&client, &parent, locale).await {
            Ok(labels) => labels,
            Err(error) => match fallback_catalog.as_ref() {
                Some(_) => {
                    eprintln!(
                        "warning: failed to load facet_tags from firestore: {error}; using empty facet labels for dev"
                    );
                    FacetTagLabels::default()
                }
                None => return Err(error),
            },
        };

        let fonts = match self.load_fonts(&client, &parent, locale).await {
            Ok(fonts) => fonts,
            Err(error) => match fallback_catalog.as_ref() {
                Some(fallback_catalog) => {
                    eprintln!(
                        "warning: failed to load fonts from firestore: {error}; using mock fonts for dev"
                    );
                    fallback_catalog.fonts.clone()
                }
                None => return Err(error),
            },
        };
        let materials = match self
            .load_materials(&client, &parent, locale, &facet_tag_labels)
            .await
        {
            Ok(materials) => materials,
            Err(error) => match fallback_catalog.as_ref() {
                Some(fallback_catalog) => {
                    eprintln!(
                        "warning: failed to load materials from firestore: {error}; using mock materials for dev"
                    );
                    fallback_catalog.materials.clone()
                }
                None => return Err(error),
            },
        };
        let countries = match self.load_countries(&client, &parent, locale).await {
            Ok(countries) => countries,
            Err(error) => match fallback_catalog.as_ref() {
                Some(fallback_catalog) => {
                    eprintln!(
                        "warning: failed to load countries from firestore: {error}; using mock countries for dev"
                    );
                    fallback_catalog.countries.clone()
                }
                None => return Err(error),
            },
        };
        let material_filters = build_material_filters(&materials, &facet_tag_labels);

        Ok(CatalogData {
            fonts,
            materials,
            countries,
            material_filters,
        })
    }

    fn firestore_client_from_access_token(access_token: &str) -> Result<FirebaseFirestoreClient> {
        Ok(FirebaseFirestoreClient::new(access_token.to_owned()))
    }

    async fn load_fonts(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
    ) -> Result<Vec<FontOption>> {
        let documents = self.query_active_documents(client, parent, "fonts").await?;

        let mut fonts = Vec::with_capacity(documents.len());
        for document in documents {
            let doc_id =
                document_id(&document).ok_or_else(|| anyhow!("fonts document is missing name"))?;
            let family = read_string_field(&document.fields, "font_family");
            if family.is_empty() {
                bail!("fonts/{doc_id} is missing font_family");
            }
            let mut stylesheet_url = read_string_field(&document.fields, "font_stylesheet_url");
            if stylesheet_url.is_empty() {
                stylesheet_url = build_google_fonts_stylesheet_url(&family).map_err(|error| {
                    anyhow!(
                        "fonts/{doc_id} is missing font_stylesheet_url and URL generation failed: {error}"
                    )
                })?;
            }

            let label =
                resolve_font_label_field(&document.fields, locale, &self.default_locale, &doc_id);
            let kanji_style = read_string_field(&document.fields, "kanji_style");
            let kanji_style = normalize_kanji_style(&kanji_style).to_owned();

            fonts.push(FontOption {
                key: doc_id,
                label,
                family,
                stylesheet_url,
                kanji_style,
            });
        }

        if fonts.is_empty() {
            bail!("no active fonts found in firestore");
        }

        Ok(fonts)
    }

    async fn load_materials(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
        facet_tag_labels: &FacetTagLabels,
    ) -> Result<Vec<MaterialOption>> {
        let categories = self
            .load_material_categories(client, parent, locale)
            .await?;
        let listings = self.load_stone_listings(client, parent, locale).await?;

        let mut materials = Vec::with_capacity(listings.len());
        for listing in listings {
            let Some(category) = categories.get(&listing.material_key) else {
                eprintln!(
                    "warning: skipping stone_listings/{}: missing materials/{} category",
                    listing.key, listing.material_key
                );
                continue;
            };

            materials.push(build_material_option_from_listing(
                category,
                &listing,
                &facet_tag_labels,
                locale,
                &self.default_locale,
                &self.storage_assets_bucket,
            ));
        }

        if materials.is_empty() {
            bail!("no active materials found in firestore");
        }

        Ok(materials)
    }

    async fn load_material_categories(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
    ) -> Result<HashMap<String, MaterialCategory>> {
        let documents = self
            .query_active_documents(client, parent, "materials")
            .await?;

        let mut categories = HashMap::with_capacity(documents.len());
        for document in documents {
            let doc_id = document_id(&document)
                .ok_or_else(|| anyhow!("materials document is missing name"))?;

            let label = resolve_localized_field(
                &document.fields,
                "label_i18n",
                locale,
                &self.default_locale,
                &doc_id,
            );

            categories.insert(doc_id.clone(), MaterialCategory { label });
        }

        if categories.is_empty() {
            bail!("no active materials found in firestore");
        }

        Ok(categories)
    }

    async fn load_facet_tag_labels(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
    ) -> Result<FacetTagLabels> {
        let documents = match self
            .run_documents_query(client, parent, "facet_tags", false, false)
            .await
        {
            Ok(documents) => documents,
            Err(error) => {
                eprintln!("warning: failed to load facet_tags from firestore: {error:#}");
                return Ok(FacetTagLabels::default());
            }
        };

        let mut labels = FacetTagLabels::default();
        for document in documents {
            if matches!(read_bool_field(&document.fields, "is_active"), Some(false)) {
                continue;
            }

            let Some(doc_id) = document_id(&document) else {
                eprintln!("warning: skipping facet_tags document with missing name");
                continue;
            };
            let facet_type = normalize_facet_tag_value(
                &first_non_empty(&[
                    Some(read_string_field(&document.fields, "facet_type")),
                    doc_id.split_once(':').map(|(prefix, _)| prefix.to_owned()),
                ])
                .unwrap_or_default(),
            );
            let key = first_non_empty(&[
                Some(read_string_field(&document.fields, "key")),
                doc_id.split_once(':').map(|(_, key)| key.to_owned()),
                Some(doc_id.clone()),
            ])
            .unwrap_or_default();
            if facet_type.is_empty() || key.is_empty() {
                continue;
            }

            let label = resolve_localized_field(
                &document.fields,
                "label_i18n",
                locale,
                &self.default_locale,
                &key,
            );
            let aliases = read_string_array_field(&document.fields, "aliases");
            labels.insert(&facet_type, &key, &label, &aliases);
        }

        Ok(labels)
    }

    async fn load_stone_listings(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
    ) -> Result<Vec<StoneListingRecord>> {
        let documents = self
            .run_documents_query(client, parent, "stone_listings", false, true)
            .await?;

        let mut listings = Vec::with_capacity(documents.len());
        for document in documents {
            let doc_id = document_id(&document)
                .ok_or_else(|| anyhow!("stone_listings document is missing name"))?;
            let is_active = read_bool_field(&document.fields, "is_active").unwrap_or(true);
            let status = read_string_field(&document.fields, "status");
            if !stone_listing_is_catalog_visible(is_active, &status) {
                continue;
            }

            let price_by_currency = read_int_map_field(&document.fields, "price_by_currency");
            let price_currency = locale_currency_code(locale);
            if resolve_amount_for_currency(&price_by_currency, price_currency).is_none() {
                eprintln!(
                    "warning: skipping stone_listings/{doc_id}: missing or empty price_by_currency"
                );
                continue;
            }

            let title = resolve_localized_field(
                &document.fields,
                "title_i18n",
                locale,
                &self.default_locale,
                &doc_id,
            );
            let description = resolve_localized_field(
                &document.fields,
                "description_i18n",
                locale,
                &self.default_locale,
                "",
            );
            let story = resolve_localized_field(
                &document.fields,
                "story_i18n",
                locale,
                &self.default_locale,
                "",
            );
            let facets = read_map_field(&document.fields, "facets");
            let color_family = stone_listing_color_family_from_facets(&facets);
            let pattern_primary = read_string_field(&facets, "pattern_primary");
            let stone_shape = normalize_stone_shape(&read_string_field(&facets, "stone_shape"));
            let color_tags = read_string_array_field(&facets, "color_tags");
            let pattern_tags = read_string_array_field(&facets, "pattern_tags");
            let photos = read_material_photos(&document.fields);

            listings.push(StoneListingRecord {
                key: doc_id.clone(),
                material_key: read_string_field(&document.fields, "material_key"),
                title,
                description,
                story,
                price_by_currency,
                color_family,
                pattern_primary,
                color_tags,
                pattern_tags,
                stone_shape,
                photos,
            });
        }

        if listings.is_empty() {
            bail!("no active stone listings found in firestore");
        }

        Ok(listings)
    }

    async fn load_countries(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        locale: &str,
    ) -> Result<Vec<CountryOption>> {
        let documents = self
            .query_active_documents(client, parent, "countries")
            .await?;

        let mut countries = Vec::with_capacity(documents.len());
        for document in documents {
            let doc_id = document_id(&document)
                .ok_or_else(|| anyhow!("countries document is missing name"))?;

            let shipping_fee_by_currency =
                read_int_map_field(&document.fields, "shipping_fee_by_currency");

            let shipping_currency = locale_currency_code(locale);
            let Some(shipping) =
                resolve_amount_for_currency(&shipping_fee_by_currency, shipping_currency)
            else {
                eprintln!(
                    "warning: skipping countries/{doc_id}: missing or empty shipping_fee_by_currency"
                );
                continue;
            };

            let label = resolve_localized_field(
                &document.fields,
                "label_i18n",
                locale,
                &self.default_locale,
                &doc_id,
            );

            countries.push(CountryOption {
                code: doc_id,
                label,
                shipping_fee_by_currency,
                shipping,
            });
        }

        if countries.is_empty() {
            bail!("no active countries found in firestore");
        }

        Ok(countries)
    }

    async fn query_active_documents(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        collection: &str,
    ) -> Result<Vec<Document>> {
        let documents = self
            .run_documents_query(client, parent, collection, true, false)
            .await?;
        if documents.is_empty() {
            bail!("no active {collection} found in firestore");
        }

        Ok(documents)
    }

    async fn run_documents_query(
        &self,
        client: &FirebaseFirestoreClient,
        parent: &str,
        collection: &str,
        active_only: bool,
        sort_by_published_at: bool,
    ) -> Result<Vec<Document>> {
        let query = RunQueryRequest {
            structured_query: Some({
                let mut query = json!({
                    "from": [
                        { "collectionId": collection }
                    ],
                });
                if active_only {
                    query["where"] = json!({
                        "fieldFilter": {
                            "field": { "fieldPath": "is_active" },
                            "op": "EQUAL",
                            "value": { "booleanValue": true }
                        }
                    });
                }
                query
            }),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(parent, &query)
            .await
            .with_context(|| format!("failed to load {collection}"))?;

        let mut documents = rows
            .into_iter()
            .filter_map(|row| row.document)
            .collect::<Vec<_>>();
        documents.sort_by(|left, right| {
            let left_sort_order = read_int_field(&left.fields, "sort_order").unwrap_or_default();
            let right_sort_order = read_int_field(&right.fields, "sort_order").unwrap_or_default();
            left_sort_order.cmp(&right_sort_order).then_with(|| {
                if sort_by_published_at {
                    let left_published_at =
                        read_timestamp_field(&left.fields, "published_at").unwrap_or_default();
                    let right_published_at =
                        read_timestamp_field(&right.fields, "published_at").unwrap_or_default();
                    right_published_at
                        .cmp(&left_published_at)
                        .then_with(|| document_id(left).cmp(&document_id(right)))
                } else {
                    document_id(left).cmp(&document_id(right))
                }
            })
        });
        Ok(documents)
    }
}

#[derive(Clone)]
struct KanjiApiClient {
    base_url: String,
    http_client: reqwest::Client,
}

#[derive(Clone)]
struct AdminProxyClient {
    base_url: String,
    http_client: reqwest::Client,
}

impl KanjiApiClient {
    async fn generate_candidates(
        &self,
        real_name: &str,
        reason_language: &str,
        gender: &str,
        kanji_style: &str,
    ) -> Result<Vec<KanjiCandidate>> {
        let endpoint = format!(
            "{}/v1/kanji-candidates",
            self.base_url.trim_end_matches('/')
        );

        let response = self
            .http_client
            .post(endpoint)
            .json(&json!({
                "real_name": real_name,
                "reason_language": reason_language,
                "gender": gender,
                "kanji_style": kanji_style,
                "count": DEFAULT_KANJI_CANDIDATE_COUNT,
            }))
            .send()
            .await
            .context("failed to request kanji candidates")?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response
                .text()
                .await
                .unwrap_or_else(|_| "<unable to read response body>".to_owned());
            bail!("kanji candidate API failed status={} body={}", status, body);
        }

        let payload = response
            .json::<KanjiCandidatesApiResponse>()
            .await
            .context("failed to decode kanji candidates response")?;

        let suggestions = payload
            .candidates
            .into_iter()
            .filter_map(|item| {
                let kanji = item.kanji.trim().to_owned();
                if kanji.is_empty()
                    || kanji.chars().count() > 2
                    || kanji.chars().any(char::is_whitespace)
                {
                    return None;
                }

                let reading = item.reading.trim().to_owned();
                let reason = item.reason.trim().to_owned();
                if reading.is_empty() || reason.is_empty() {
                    return None;
                }

                Some(KanjiCandidate {
                    kanji: kanji.clone(),
                    line1: kanji,
                    line2: String::new(),
                    reading,
                    reason,
                })
            })
            .collect::<Vec<_>>();

        Ok(suggestions)
    }

    async fn create_order(
        &self,
        request: &CreateOrderApiRequest,
    ) -> Result<CreateOrderApiResponse> {
        let endpoint = format!("{}/v1/orders", self.base_url.trim_end_matches('/'));
        let response = self
            .http_client
            .post(endpoint)
            .json(request)
            .send()
            .await
            .context("failed to request order creation")?;

        decode_api_response(response, "create order")
            .await
            .context("failed to decode create order response")
    }

    async fn create_stripe_checkout_session(
        &self,
        request: &CreateStripeCheckoutSessionApiRequest,
    ) -> Result<CreateStripeCheckoutSessionApiResponse> {
        let endpoint = format!(
            "{}/v1/payments/stripe/checkout-session",
            self.base_url.trim_end_matches('/')
        );
        let response = self
            .http_client
            .post(endpoint)
            .json(request)
            .send()
            .await
            .context("failed to request stripe checkout session")?;

        decode_api_response(response, "create stripe checkout session")
            .await
            .context("failed to decode stripe checkout session response")
    }
}

async fn decode_api_response<T: DeserializeOwned>(
    response: reqwest::Response,
    op: &str,
) -> Result<T> {
    let status = response.status();
    let body = response
        .bytes()
        .await
        .with_context(|| format!("failed to read response body for {op}"))?;

    if !status.is_success() {
        if let Ok(err) = serde_json::from_slice::<ApiErrorEnvelope>(&body) {
            bail!(
                "{} failed status={} code={} message={}",
                op,
                status,
                err.error.code,
                err.error.message
            );
        }

        let text = String::from_utf8_lossy(&body);
        bail!("{op} failed status={} body={}", status, text);
    }

    serde_json::from_slice::<T>(&body).with_context(|| format!("invalid JSON for {op}"))
}

#[derive(Clone)]
struct AppState {
    source: Arc<CatalogSource>,
    kanji_api: Arc<KanjiApiClient>,
    admin_proxy: Arc<AdminProxyClient>,
    mode: RunMode,
    locale: String,
    default_locale: String,
    site_base_url: String,
}

#[tokio::main]
async fn main() {
    if let Err(error) = run().await {
        eprintln!("failed to start web server: {error:#}");
        std::process::exit(1);
    }
}

async fn run() -> Result<()> {
    let cfg = load_config().context("failed to load config")?;
    let state = build_state(&cfg).await?;

    let app = build_router(state.clone());

    let addr = format!("0.0.0.0:{}", cfg.port);
    if let Some(project_id) = cfg.firestore_project_id.as_deref() {
        println!(
            "hanko web listening on http://localhost:{} mode={} source={} project={} locale={} kanji_api={}",
            cfg.port,
            cfg.mode.as_str(),
            state.source.label(),
            project_id,
            cfg.locale,
            cfg.api_base_url
        );
    } else {
        println!(
            "hanko web listening on http://localhost:{} mode={} source={} locale={} kanji_api={}",
            cfg.port,
            cfg.mode.as_str(),
            state.source.label(),
            cfg.locale,
            cfg.api_base_url
        );
    }

    let listener = tokio::net::TcpListener::bind(&addr)
        .await
        .with_context(|| format!("failed to bind {addr}"))?;

    axum::serve(listener, app)
        .await
        .context("web server terminated unexpectedly")
}

fn build_router(state: AppState) -> Router {
    Router::new()
        .route("/", get(handle_top))
        .route("/robots.txt", get(handle_robots_txt))
        .route("/sitemap.xml", get(handle_sitemap_xml))
        .route("/design", get(handle_design))
        .route("/blog", get(handle_blog_index))
        .route("/blog/{slug}", get(handle_blog_article))
        .route("/about", get(handle_about))
        .route("/terms", get(handle_terms))
        .route(
            "/commercial-transactions",
            get(handle_commercial_transactions),
        )
        .route("/payment/success", get(handle_payment_success))
        .route("/payment/failure", get(handle_payment_failure))
        .route("/{locale}", get(handle_localized_top))
        .route("/{locale}/", get(handle_localized_top))
        .route("/{locale}/about", get(handle_localized_about))
        .route("/{locale}/design", get(handle_localized_design))
        .route("/{locale}/blog", get(handle_localized_blog_index))
        .route("/{locale}/blog/{slug}", get(handle_localized_blog_article))
        .route("/{locale}/terms", get(handle_localized_terms))
        .route(
            "/{locale}/commercial-transactions",
            get(handle_localized_commercial_transactions),
        )
        .route(
            "/{locale}/payment/success",
            get(handle_localized_payment_success),
        )
        .route(
            "/{locale}/payment/failure",
            get(handle_localized_payment_failure),
        )
        .route("/kanji", post(handle_kanji_suggestions))
        .route("/purchase", post(handle_purchase))
        .route("/mock/kanji", post(handle_kanji_suggestions))
        .route("/mock/purchase", post(handle_mock_purchase))
        .route("/admin-login", any(handle_admin_proxy))
        .route("/admin", any(handle_admin_proxy))
        .route("/admin/{*path}", any(handle_admin_proxy))
        .nest_service("/static", ServeDir::new(WEB_STATIC_DIR))
        .layer(SetResponseHeaderLayer::if_not_present(
            header::CACHE_CONTROL,
            HeaderValue::from_static("no-cache, no-store, must-revalidate"),
        ))
        .with_state(state)
}

async fn build_state(cfg: &AppConfig) -> Result<AppState> {
    let source = Arc::new(new_catalog_source(cfg).await?);
    let kanji_http_client = reqwest::Client::builder()
        .timeout(Duration::from_secs(20))
        .build()
        .context("failed to initialize kanji API client")?;
    let kanji_api = Arc::new(KanjiApiClient {
        base_url: cfg.api_base_url.clone(),
        http_client: kanji_http_client,
    });
    let admin_proxy_http_client = reqwest::Client::builder()
        .redirect(reqwest::redirect::Policy::none())
        .build()
        .context("failed to initialize admin proxy client")?;
    let admin_proxy = Arc::new(AdminProxyClient {
        base_url: cfg.admin_base_url.clone(),
        http_client: admin_proxy_http_client,
    });

    let _catalog = load_catalog_with_timeout(source.as_ref(), &cfg.locale).await?;

    Ok(AppState {
        source,
        kanji_api,
        admin_proxy,
        mode: cfg.mode,
        locale: cfg.locale.clone(),
        default_locale: cfg.default_locale.clone(),
        site_base_url: cfg.site_base_url.clone(),
    })
}

fn load_config() -> Result<AppConfig> {
    let mut cfg = AppConfig {
        port: env_first(&["HANKO_WEB_PORT", "PORT"]),
        mode: RunMode::Mock,
        locale: env::var("HANKO_WEB_LOCALE")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        default_locale: env::var("HANKO_WEB_DEFAULT_LOCALE")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        site_base_url: env::var("HANKO_WEB_SITE_BASE_URL")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        api_base_url: env::var("HANKO_WEB_API_BASE_URL")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        admin_base_url: env_first(&["HANKO_WEB_ADMIN_BASE_URL_PROD", "HANKO_WEB_ADMIN_BASE_URL"]),
        firestore_project_id: None,
        credentials_file: None,
        storage_assets_bucket: None,
    };

    if cfg.port.is_empty() {
        cfg.port = "3052".to_owned();
    }

    if cfg.locale.is_empty() {
        cfg.locale = "en".to_owned();
    }

    if cfg.default_locale.is_empty() {
        cfg.default_locale = "en".to_owned();
    }

    if cfg.site_base_url.is_empty() {
        if matches!(cfg.mode, RunMode::Prod) {
            bail!("prod web requires HANKO_WEB_SITE_BASE_URL");
        }
        cfg.site_base_url = "http://127.0.0.1:3052".to_owned();
    }
    cfg.site_base_url = normalize_site_base_url(&cfg.site_base_url)?;

    if cfg.api_base_url.is_empty() {
        cfg.api_base_url = "http://127.0.0.1:3050".to_owned();
    }

    let mut mode_value = env_first(&["HANKO_WEB_MODE", "HANKO_WEB_ENV"]).to_lowercase();
    if mode_value.is_empty() {
        mode_value = "mock".to_owned();
    }

    match mode_value.as_str() {
        "mock" => {
            cfg.mode = RunMode::Mock;
            return Ok(cfg);
        }
        "dev" => cfg.mode = RunMode::Dev,
        "prod" => cfg.mode = RunMode::Prod,
        _ => bail!("invalid HANKO_WEB_MODE {mode_value:?}: use mock, dev, or prod"),
    }

    if cfg.admin_base_url.is_empty() {
        if matches!(cfg.mode, RunMode::Prod) {
            bail!("prod web requires HANKO_WEB_ADMIN_BASE_URL[_PROD]");
        }
        cfg.admin_base_url = "http://localhost:3051".to_owned();
    }

    let (project_id_keys, credentials_keys, storage_bucket_keys): (&[&str], &[&str], &[&str]) =
        match cfg.mode {
            RunMode::Dev => (
                &[
                    "HANKO_WEB_FIREBASE_PROJECT_ID_DEV",
                    "HANKO_WEB_FIREBASE_PROJECT_ID",
                    "FIREBASE_PROJECT_ID",
                    "GOOGLE_CLOUD_PROJECT",
                ],
                &[
                    "HANKO_WEB_FIREBASE_CREDENTIALS_FILE_DEV",
                    "HANKO_WEB_FIREBASE_CREDENTIALS_FILE",
                    "GOOGLE_APPLICATION_CREDENTIALS",
                ],
                &[
                    "HANKO_WEB_STORAGE_ASSETS_BUCKET_DEV",
                    "HANKO_WEB_STORAGE_ASSETS_BUCKET",
                    "API_STORAGE_ASSETS_BUCKET",
                ],
            ),
            RunMode::Prod => (
                &[
                    "HANKO_WEB_FIREBASE_PROJECT_ID_PROD",
                    "HANKO_WEB_FIREBASE_PROJECT_ID",
                    "FIREBASE_PROJECT_ID",
                    "GOOGLE_CLOUD_PROJECT",
                ],
                &[
                    "HANKO_WEB_FIREBASE_CREDENTIALS_FILE_PROD",
                    "HANKO_WEB_FIREBASE_CREDENTIALS_FILE",
                    "GOOGLE_APPLICATION_CREDENTIALS",
                ],
                &[
                    "HANKO_WEB_STORAGE_ASSETS_BUCKET_PROD",
                    "HANKO_WEB_STORAGE_ASSETS_BUCKET",
                    "API_STORAGE_ASSETS_BUCKET",
                ],
            ),
            RunMode::Mock => (&[], &[], &[]),
        };

    let project_id = env_first(project_id_keys);
    if project_id.is_empty() {
        bail!(
            "firebase mode ({}) requires project id env var: {}",
            cfg.mode.as_str(),
            project_id_keys.join(", ")
        );
    }
    cfg.firestore_project_id = Some(project_id);

    let credentials_file = env_first(credentials_keys);
    if !credentials_file.is_empty() {
        cfg.credentials_file = Some(credentials_file);
    }
    let storage_assets_bucket = env_first(storage_bucket_keys);
    if !storage_assets_bucket.is_empty() {
        cfg.storage_assets_bucket = Some(storage_assets_bucket);
    } else if matches!(cfg.mode, RunMode::Dev) {
        cfg.storage_assets_bucket = Some("hanko-field-dev".to_owned());
    }

    Ok(cfg)
}

fn env_first(keys: &[&str]) -> String {
    for key in keys {
        if let Ok(value) = env::var(key) {
            let trimmed = value.trim();
            if !trimmed.is_empty() {
                return trimmed.to_owned();
            }
        }
    }
    String::new()
}

async fn new_catalog_source(cfg: &AppConfig) -> Result<CatalogSource> {
    match cfg.mode {
        RunMode::Mock => Ok(CatalogSource::Mock(new_mock_catalog_source(&cfg.locale))),
        RunMode::Dev | RunMode::Prod => {
            let label = if cfg.mode == RunMode::Prod {
                "Firebase Prod"
            } else {
                "Firebase Dev"
            }
            .to_owned();

            let token_provider: Arc<dyn TokenProvider> =
                if let Some(credentials_file) = cfg.credentials_file.as_deref() {
                    Arc::new(
                        CustomServiceAccount::from_file(credentials_file).with_context(|| {
                            format!("failed to read credentials file: {credentials_file}")
                        })?,
                    )
                } else {
                    provider()
                        .await
                        .context("failed to initialize default GCP auth provider")?
                };

            Ok(CatalogSource::Firestore(FirestoreCatalogSource {
                project_id: cfg
                    .firestore_project_id
                    .clone()
                    .context("firestore project id is empty")?,
                default_locale: cfg.default_locale.clone(),
                label,
                storage_assets_bucket: cfg.storage_assets_bucket.clone().unwrap_or_default(),
                allow_mock_fallback: matches!(cfg.mode, RunMode::Dev),
                token_provider,
            }))
        }
    }
}

impl AdminProxyClient {
    async fn proxy(&self, request: axum::extract::Request) -> Response {
        let (parts, body) = request.into_parts();

        let path_and_query = parts
            .uri
            .path_and_query()
            .map(|value| value.as_str())
            .unwrap_or("/");
        let target = match reqwest::Url::parse(self.base_url.trim_end_matches('/'))
            .and_then(|base| base.join(path_and_query))
        {
            Ok(url) => url,
            Err(error) => {
                return plain_error(
                    StatusCode::BAD_GATEWAY,
                    format!("failed to build admin proxy URL: {error}"),
                );
            }
        };

        let body = match axum::body::to_bytes(body, ADMIN_PROXY_MAX_BODY_BYTES).await {
            Ok(body) => body,
            Err(error) => {
                return plain_error(
                    StatusCode::PAYLOAD_TOO_LARGE,
                    format!("failed to read admin proxy request body: {error}"),
                );
            }
        };

        let mut request_builder = self
            .http_client
            .request(parts.method.clone(), target)
            .body(body.to_vec());

        for (name, value) in &parts.headers {
            if should_forward_admin_request_header(name) {
                request_builder = request_builder.header(name, value);
            }
        }

        let upstream = match request_builder.send().await {
            Ok(response) => response,
            Err(error) => {
                return plain_error(
                    StatusCode::BAD_GATEWAY,
                    format!("failed to proxy admin request: {error}"),
                );
            }
        };

        let status = upstream.status();
        let upstream_headers = upstream.headers().clone();
        let body = match upstream.bytes().await {
            Ok(body) => body,
            Err(error) => {
                return plain_error(
                    StatusCode::BAD_GATEWAY,
                    format!("failed to read admin proxy response: {error}"),
                );
            }
        };

        let mut response_builder = Response::builder().status(status);
        if let Some(content_type) = upstream_headers.get(header::CONTENT_TYPE) {
            response_builder = response_builder.header(header::CONTENT_TYPE, content_type);
        }
        for (name, value) in &upstream_headers {
            if should_forward_admin_response_header(name) {
                response_builder = response_builder.header(name, value);
            }
        }

        match response_builder.body(axum::body::Body::from(body.to_vec())) {
            Ok(response) => response,
            Err(error) => plain_error(
                StatusCode::BAD_GATEWAY,
                format!("failed to build admin proxy response: {error}"),
            ),
        }
    }
}

async fn handle_admin_proxy(
    State(state): State<AppState>,
    request: axum::extract::Request,
) -> Response {
    state.admin_proxy.proxy(request).await
}

fn new_mock_catalog_source(locale: &str) -> MockCatalogSource {
    let english = !is_japanese_locale(locale);

    MockCatalogSource {
        catalog: CatalogData {
            fonts: vec![
                FontOption {
                    key: "zen_maru_gothic".to_owned(),
                    label: "Zen Maru Gothic".to_owned(),
                    family: "'Zen Maru Gothic', sans-serif".to_owned(),
                    stylesheet_url:
                        "https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic:wght@400;700&display=swap"
                            .to_owned(),
                    kanji_style: "japanese".to_owned(),
                },
                FontOption {
                    key: "kosugi_maru".to_owned(),
                    label: "Kosugi Maru".to_owned(),
                    family: "'Kosugi Maru', sans-serif".to_owned(),
                    stylesheet_url:
                        "https://fonts.googleapis.com/css2?family=Kosugi+Maru&display=swap"
                            .to_owned(),
                    kanji_style: "chinese".to_owned(),
                },
                FontOption {
                    key: "potta_one".to_owned(),
                    label: "Potta One".to_owned(),
                    family: "'Potta One', sans-serif".to_owned(),
                    stylesheet_url:
                        "https://fonts.googleapis.com/css2?family=Potta+One&display=swap"
                            .to_owned(),
                    kanji_style: "taiwanese".to_owned(),
                },
                FontOption {
                    key: "kiwi_maru".to_owned(),
                    label: "Kiwi Maru".to_owned(),
                    family: "'Kiwi Maru', sans-serif".to_owned(),
                    stylesheet_url:
                        "https://fonts.googleapis.com/css2?family=Kiwi+Maru:wght@400;700&display=swap"
                            .to_owned(),
                    kanji_style: "japanese".to_owned(),
                },
                FontOption {
                    key: "wdxl_lubrifont_jp_n".to_owned(),
                    label: "WDXL Lubrifont JP N".to_owned(),
                    family: "'WDXL Lubrifont JP N', sans-serif".to_owned(),
                    stylesheet_url:
                        "https://fonts.googleapis.com/css2?family=WDXL+Lubrifont+JP+N&display=swap"
                            .to_owned(),
                    kanji_style: "chinese".to_owned(),
                },
            ],
            materials: vec![
                MaterialOption {
                    key: "rose_quartz".to_owned(),
                    label: if english {
                        "Rose Quartz"
                    } else {
                        "ローズクオーツ"
                    }
                    .to_owned(),
                    description: if english {
                        "A soft-toned stone with a warm, approachable presence"
                    } else {
                        "やわらかな色合いで、親しみやすい印象の石材"
                    }
                    .to_owned(),
                    story: if english {
                        "A gentle rose quartz listing selected for its soft translucent tone and easy everyday presence."
                    } else {
                        "やわらかな透明感と日常に馴染む穏やかな雰囲気を持つローズクオーツの一点物です。"
                    }
                    .to_owned(),
                    has_description: true,
                    has_story: true,
                    price_by_currency: HashMap::from([
                        ("USD".to_owned(), 16500),
                        ("JPY".to_owned(), 28000),
                    ]),
                    shape: "square".to_owned(),
                    shape_label: if english { "Square seal" } else { "角印" }.to_owned(),
                    color_family: "pink".to_owned(),
                    pattern_primary: "cloud".to_owned(),
                    color_tag_labels: if english {
                        vec!["Soft Pink".to_owned()]
                    } else {
                        vec!["淡桃".to_owned()]
                    },
                    pattern_tag_labels: if english {
                        vec!["Cloud".to_owned()]
                    } else {
                        vec!["雲状".to_owned()]
                    },
                    has_color_tag_labels: true,
                    has_pattern_tag_labels: true,
                    price: if english { 16500 } else { 28000 },
                    price_display: if english {
                        format_usd(16500)
                    } else {
                        format_jpy(28000)
                    },
                    photo_url: "https://picsum.photos/seed/hf-rose-quartz/640/420".to_owned(),
                    photo_alt: if english {
                        "Rose quartz photo"
                    } else {
                        "ローズクオーツ材の写真"
                    }
                    .to_owned(),
                    has_photo: true,
                },
                MaterialOption {
                    key: "lapis_lazuli".to_owned(),
                    label: if english {
                        "Lapis Lazuli"
                    } else {
                        "ラピスラビリ"
                    }
                    .to_owned(),
                    description: if english {
                        "A deep-blue stone with a strong, distinctive presence"
                    } else {
                        "深い青が印象的な、存在感のある石材"
                    }
                    .to_owned(),
                    story: if english {
                        "This lapis lazuli listing has a vivid blue field with small bright flecks that make each seal feel distinct."
                    } else {
                        "深い青に小さなきらめきが入り、一点物らしい個性が際立つラピスラビリです。"
                    }
                    .to_owned(),
                    has_description: true,
                    has_story: true,
                    price_by_currency: HashMap::from([
                        ("USD".to_owned(), 32500),
                        ("JPY".to_owned(), 55000),
                    ]),
                    shape: "round".to_owned(),
                    shape_label: if english { "Round seal" } else { "丸印" }.to_owned(),
                    color_family: "blue".to_owned(),
                    pattern_primary: "speckled".to_owned(),
                    color_tag_labels: if english {
                        vec!["Deep Blue".to_owned()]
                    } else {
                        vec!["深青".to_owned()]
                    },
                    pattern_tag_labels: if english {
                        vec!["Speckled".to_owned()]
                    } else {
                        vec!["点状".to_owned()]
                    },
                    has_color_tag_labels: true,
                    has_pattern_tag_labels: true,
                    price: if english { 32500 } else { 55000 },
                    price_display: if english {
                        format_usd(32500)
                    } else {
                        format_jpy(55000)
                    },
                    photo_url: "https://picsum.photos/seed/hf-lapis-lazuli/640/420".to_owned(),
                    photo_alt: if english {
                        "Lapis lazuli photo"
                    } else {
                        "ラピスラビリ材の写真"
                    }
                    .to_owned(),
                    has_photo: true,
                },
                MaterialOption {
                    key: "jade".to_owned(),
                    label: if english { "Jade" } else { "翡翠" }.to_owned(),
                    description: if english {
                        "A dignified stone with a calm green sheen"
                    } else {
                        "落ち着いた緑の艶が映える、格調ある石材"
                    }
                    .to_owned(),
                    story: if english {
                        "A jade listing with a quiet green sheen and a composed appearance suited to a refined seal."
                    } else {
                        "静かな緑の艶と端正な表情を備え、上品な印影に合わせやすい翡翠の一点物です。"
                    }
                    .to_owned(),
                    has_description: true,
                    has_story: true,
                    price_by_currency: HashMap::from([
                        ("USD".to_owned(), 88500),
                        ("JPY".to_owned(), 150000),
                    ]),
                    shape: "square".to_owned(),
                    shape_label: if english { "Square seal" } else { "角印" }.to_owned(),
                    color_family: "green".to_owned(),
                    pattern_primary: "marble".to_owned(),
                    color_tag_labels: if english {
                        vec!["Deep Green".to_owned()]
                    } else {
                        vec!["濃緑".to_owned()]
                    },
                    pattern_tag_labels: if english {
                        vec!["Banded".to_owned()]
                    } else {
                        vec!["縞".to_owned()]
                    },
                    has_color_tag_labels: true,
                    has_pattern_tag_labels: true,
                    price: if english { 88500 } else { 150000 },
                    price_display: if english {
                        format_usd(88500)
                    } else {
                        format_jpy(150000)
                    },
                    photo_url: "https://picsum.photos/seed/hf-jade/640/420".to_owned(),
                    photo_alt: if english {
                        "Jade photo"
                    } else {
                        "翡翠材の写真"
                    }
                    .to_owned(),
                    has_photo: true,
                },
            ],
            countries: vec![
                CountryOption {
                    code: "JP".to_owned(),
                    label: if english { "Japan" } else { "日本" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 600),
                        ("JPY".to_owned(), 600),
                    ]),
                    shipping: 600,
                },
                CountryOption {
                    code: "US".to_owned(),
                    label: if english { "United States" } else { "アメリカ" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 1800),
                        ("JPY".to_owned(), 1800),
                    ]),
                    shipping: 1800,
                },
                CountryOption {
                    code: "CA".to_owned(),
                    label: if english { "Canada" } else { "カナダ" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 1900),
                        ("JPY".to_owned(), 1900),
                    ]),
                    shipping: 1900,
                },
                CountryOption {
                    code: "GB".to_owned(),
                    label: if english { "United Kingdom" } else { "イギリス" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 2000),
                        ("JPY".to_owned(), 2000),
                    ]),
                    shipping: 2000,
                },
                CountryOption {
                    code: "AU".to_owned(),
                    label: if english { "Australia" } else { "オーストラリア" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 2100),
                        ("JPY".to_owned(), 2100),
                    ]),
                    shipping: 2100,
                },
                CountryOption {
                    code: "SG".to_owned(),
                    label: if english { "Singapore" } else { "シンガポール" }.to_owned(),
                    shipping_fee_by_currency: HashMap::from([
                        ("USD".to_owned(), 1300),
                        ("JPY".to_owned(), 1300),
                    ]),
                    shipping: 1300,
                },
            ],
            material_filters: MaterialFilters::default(),
        },
    }
}

async fn load_catalog_with_timeout(source: &CatalogSource, locale: &str) -> Result<CatalogData> {
    let catalog = tokio::time::timeout(Duration::from_secs(7), source.load_catalog(locale))
        .await
        .context("catalog load timed out after 7s")??;

    validate_catalog(&catalog)?;
    Ok(catalog)
}

fn validate_catalog(catalog: &CatalogData) -> Result<()> {
    if catalog.fonts.is_empty() {
        bail!("catalog validation failed: fonts is empty");
    }
    if catalog.materials.is_empty() {
        bail!("catalog validation failed: materials is empty");
    }
    if catalog.countries.is_empty() {
        bail!("catalog validation failed: countries is empty");
    }
    Ok(())
}

fn build_google_fonts_stylesheet_url(font_family: &str) -> Result<String> {
    let first_font_name = extract_primary_font_name(font_family)
        .ok_or_else(|| anyhow!("font-family does not contain a primary font name"))?;
    if is_generic_css_font_family(&first_font_name) {
        bail!("font-family primary value must be a concrete Google Fonts family name");
    }

    let mut url = reqwest::Url::parse("https://fonts.googleapis.com/css2")
        .context("failed to parse Google Fonts css2 endpoint")?;
    {
        let mut query = url.query_pairs_mut();
        query.append_pair("family", &first_font_name);
        query.append_pair("display", "swap");
    }
    Ok(url.to_string())
}

fn extract_primary_font_name(font_family: &str) -> Option<String> {
    let first = font_family.split(',').next()?.trim();
    if first.is_empty() {
        return None;
    }

    let unquoted = first
        .strip_prefix('\'')
        .and_then(|value| value.strip_suffix('\''))
        .or_else(|| {
            first
                .strip_prefix('"')
                .and_then(|value| value.strip_suffix('"'))
        })
        .unwrap_or(first)
        .trim();
    if unquoted.is_empty() {
        return None;
    }

    let normalized = unquoted.split_whitespace().collect::<Vec<_>>().join(" ");
    if normalized.is_empty() {
        None
    } else {
        Some(normalized)
    }
}

fn is_generic_css_font_family(font_name: &str) -> bool {
    let normalized = font_name.trim().to_ascii_lowercase();
    matches!(
        normalized.as_str(),
        "serif"
            | "sans-serif"
            | "monospace"
            | "cursive"
            | "fantasy"
            | "system-ui"
            | "ui-serif"
            | "ui-sans-serif"
            | "ui-monospace"
            | "ui-rounded"
            | "emoji"
            | "math"
            | "fangsong"
            | "inherit"
            | "initial"
            | "unset"
            | "revert"
            | "revert-layer"
    )
}

fn collect_font_stylesheet_urls(fonts: &[FontOption]) -> Vec<String> {
    let mut seen = HashSet::new();
    let mut urls = Vec::new();

    for font in fonts {
        let url = font.stylesheet_url.trim();
        if url.is_empty() {
            continue;
        }
        if seen.insert(url.to_owned()) {
            urls.push(url.to_owned());
        }
    }

    urls
}

fn load_blog_posts() -> Result<Vec<BlogPost>> {
    let mut posts = Vec::new();
    for entry in std::fs::read_dir(WEB_BLOG_CONTENT_DIR)
        .with_context(|| format!("failed to read blog content directory {WEB_BLOG_CONTENT_DIR}"))?
    {
        let entry = entry.with_context(|| {
            format!("failed to read blog content entry in {WEB_BLOG_CONTENT_DIR}")
        })?;
        let path = entry.path();
        if !path.is_dir() {
            continue;
        }

        posts.push(load_blog_post_from_content_dir(&path)?);
    }

    posts.sort_by(|left, right| {
        right
            .published_date
            .cmp(&left.published_date)
            .then_with(|| left.slug.cmp(&right.slug))
    });
    Ok(posts)
}

fn load_blog_post_from_content_dir(path: &std::path::Path) -> Result<BlogPost> {
    let metadata_path = path.join("metadata.json");
    let source = std::fs::read_to_string(&metadata_path)
        .with_context(|| format!("failed to read blog metadata {}", metadata_path.display()))?;
    let metadata = serde_json::from_str::<BlogPostMetadata>(&source)
        .with_context(|| format!("failed to parse blog metadata {}", metadata_path.display()))?;
    let slug = metadata.slug.trim().to_owned();
    if !is_safe_slug(&slug) {
        bail!("invalid blog slug in {}: {slug}", metadata_path.display());
    }
    let dir_slug = path
        .file_name()
        .and_then(|file_name| file_name.to_str())
        .context("blog content directory name should be valid utf-8")?;
    if dir_slug != slug {
        bail!(
            "blog metadata slug must match directory name: {} has slug {slug}",
            path.display()
        );
    }
    for locale in ["en", "ja"] {
        let article_path = path.join(format!("{locale}.html"));
        if !article_path.is_file() {
            bail!("missing blog article body {}", article_path.display());
        }
    }

    let published_date = metadata.published_date;
    ensure_valid_sitemap_lastmod(&published_date)
        .with_context(|| format!("invalid blog published_date in {}", metadata_path.display()))?;
    let last_modified_date = metadata.last_modified_date;
    ensure_valid_sitemap_lastmod(&last_modified_date).with_context(|| {
        format!(
            "invalid blog last_modified_date in {}",
            metadata_path.display()
        )
    })?;
    let en = required_blog_locale_metadata(&metadata.locales, "en", &metadata_path)?;
    let ja = required_blog_locale_metadata(&metadata.locales, "ja", &metadata_path)?;
    Ok(BlogPost {
        slug,
        published_date,
        last_modified_date,
        date_display: required_blog_locale_value(en, "date_display", &metadata_path)?,
        date_display_ja: required_blog_locale_value(ja, "date_display", &metadata_path)?,
        title: required_blog_locale_value(en, "title", &metadata_path)?,
        title_ja: required_blog_locale_value(ja, "title", &metadata_path)?,
        excerpt: required_blog_locale_value(en, "excerpt", &metadata_path)?,
        excerpt_ja: required_blog_locale_value(ja, "excerpt", &metadata_path)?,
        meta_description: required_blog_locale_value(en, "meta_description", &metadata_path)?,
        meta_description_ja: required_blog_locale_value(ja, "meta_description", &metadata_path)?,
        image_url: required_blog_metadata_value(&metadata.image_url, "image_url", &metadata_path)?,
        image_alt: required_blog_locale_value(en, "image_alt", &metadata_path)?,
        image_alt_ja: required_blog_locale_value(ja, "image_alt", &metadata_path)?,
    })
}

fn required_blog_locale_metadata<'a>(
    locales: &'a HashMap<String, BlogPostLocaleMetadata>,
    locale: &str,
    path: &std::path::Path,
) -> Result<&'a BlogPostLocaleMetadata> {
    locales.get(locale).with_context(|| {
        format!(
            "missing blog metadata locale `{locale}` in {}",
            path.display()
        )
    })
}

fn required_blog_locale_value(
    metadata: &BlogPostLocaleMetadata,
    key: &str,
    path: &std::path::Path,
) -> Result<String> {
    let value = match key {
        "date_display" => &metadata.date_display,
        "title" => &metadata.title,
        "excerpt" => &metadata.excerpt,
        "meta_description" => &metadata.meta_description,
        "image_alt" => &metadata.image_alt,
        _ => bail!("unknown blog metadata key `{key}`"),
    };
    required_blog_metadata_value(value, key, path)
}

fn required_blog_metadata_value(value: &str, key: &str, path: &std::path::Path) -> Result<String> {
    let value = value.trim();
    if value.is_empty() {
        bail!(
            "missing required blog metadata key `{key}` in {}",
            path.display()
        );
    }
    Ok(value.to_owned())
}

fn blog_post_cards(posts: &[BlogPost], base_url: &str, locale: &str) -> Vec<BlogPostCard> {
    posts
        .iter()
        .map(|post| {
            let view = localized_blog_post(post, locale);
            BlogPostCard {
                title: view.title,
                excerpt: view.excerpt,
                image_url: view.image_url,
                image_alt: view.image_alt,
                post_url: blog_article_url(base_url, &post.slug, locale),
            }
        })
        .collect()
}

fn find_blog_post(posts: &[BlogPost], slug: &str) -> Option<BlogPost> {
    posts.iter().find(|post| post.slug == slug).cloned()
}

fn localized_blog_post(post: &BlogPost, locale: &str) -> BlogPostView {
    if is_japanese_locale(locale) {
        return BlogPostView {
            published_date: post.published_date.clone(),
            date_display: post.date_display_ja.clone(),
            title: post.title_ja.clone(),
            excerpt: post.excerpt_ja.clone(),
            meta_description: post.meta_description_ja.clone(),
            image_url: post.image_url.clone(),
            image_alt: post.image_alt_ja.clone(),
        };
    }

    BlogPostView {
        published_date: post.published_date.clone(),
        date_display: post.date_display.clone(),
        title: post.title.clone(),
        excerpt: post.excerpt.clone(),
        meta_description: post.meta_description.clone(),
        image_url: post.image_url.clone(),
        image_alt: post.image_alt.clone(),
    }
}

fn read_blog_article_body(slug: &str, locale: &str) -> Result<String> {
    if !is_safe_slug(slug) {
        bail!("invalid blog slug");
    }
    let route_code = if is_japanese_locale(locale) {
        "ja"
    } else {
        "en"
    };
    let path = format!("{WEB_BLOG_CONTENT_DIR}/{slug}/{route_code}.html");
    let source = std::fs::read_to_string(&path)
        .with_context(|| format!("failed to read blog article {path}"))?;
    Ok(source)
}

fn is_safe_slug(slug: &str) -> bool {
    !slug.is_empty()
        && slug
            .bytes()
            .all(|byte| byte.is_ascii_lowercase() || byte.is_ascii_digit() || byte == b'-')
}

async fn handle_top(
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_top_page(state, query, None).await
}

async fn handle_localized_top(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_top_page(state, query, Some(locale)).await
}

async fn render_top_page(
    state: AppState,
    query: PaymentRedirectQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();

    if let Some(path) = checkout_redirect_path(site_base_url, &query, &selected_locale) {
        return Redirect::to(&path).into_response();
    }

    let blog_posts = match load_blog_posts() {
        Ok(posts) => posts,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load blog posts: {error:#}"),
            );
        }
    };

    let template = TopPageTemplate {
        page_title: web_copy_text("top", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("top", &selected_locale, "seo_description").to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(site_base_url, "/", &selected_locale),
        x_default_url: x_default_url_for_path(site_base_url, "/"),
        seo_language_links: indexed_hreflang_links_for_path(site_base_url, "/"),
        language_links: language_links_for_path(site_base_url, "/"),
        company_url: company_url(site_base_url),
        selected_locale: selected_locale.clone(),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        design_url: localized_navigation_page_url(site_base_url, "/design", &selected_locale),
        blog_index_url: blog_index_url(site_base_url, &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        blog_posts: blog_post_cards(&blog_posts, site_base_url, &selected_locale),
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render page: {error}"),
        ),
    }
}

async fn handle_about(State(state): State<AppState>, Query(query): Query<LocaleQuery>) -> Response {
    render_about_page(state, query, None).await
}

async fn handle_localized_about(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_about_page(state, query, Some(locale)).await
}

async fn render_about_page(
    state: AppState,
    query: LocaleQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let template = AboutTemplate {
        page_title: web_copy_text("about", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("about", &selected_locale, "seo_description").to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(site_base_url, "/about", &selected_locale),
        x_default_url: x_default_url_for_path(site_base_url, "/about"),
        seo_language_links: indexed_hreflang_links_for_path(site_base_url, "/about"),
        language_links: language_links_for_path(site_base_url, "/about"),
        company_url: company_url(site_base_url),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        design_url: localized_navigation_page_url(site_base_url, "/design", &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        selected_locale,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render about page: {error}"),
        ),
    }
}

async fn handle_design(
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_design_page(state, query, None).await
}

async fn handle_localized_design(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_design_page(state, query, Some(locale)).await
}

async fn render_design_page(
    state: AppState,
    query: PaymentRedirectQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();

    if let Some(path) = checkout_redirect_path(site_base_url, &query, &selected_locale) {
        return Redirect::to(&path).into_response();
    }

    let catalog = match load_catalog_with_timeout(state.source.as_ref(), &selected_locale).await {
        Ok(catalog) => catalog,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load catalog: {error}"),
            );
        }
    };
    let catalog = localize_catalog_prices(catalog, &selected_locale);
    let material_filter_state = material_filter_state_from_query(&query);

    let Some(default_font) = catalog
        .fonts
        .iter()
        .find(|font| font.kanji_style == "japanese")
        .or_else(|| catalog.fonts.first())
    else {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            "catalog validation failed: fonts is empty".to_owned(),
        );
    };
    let materials = catalog.materials;

    let template = PageTemplate {
        font_stylesheet_urls: collect_font_stylesheet_urls(&catalog.fonts),
        default_font_key: default_font.key.clone(),
        default_font_label: default_font.label.clone(),
        fonts: catalog.fonts,
        materials,
        countries: catalog.countries,
        material_filters: catalog.material_filters,
        selected_color_family: material_filter_state.color_family.clone(),
        selected_pattern_primary: material_filter_state.pattern_primary.clone(),
        page_title: web_copy_text("design", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("design", &selected_locale, "seo_description").to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        purchase_action_url: if state.mode == RunMode::Mock {
            site_url(site_base_url, "/mock/purchase")
        } else {
            site_url(site_base_url, "/purchase")
        },
        purchase_note: web_copy_text(
            "design",
            &selected_locale,
            if state.mode == RunMode::Mock {
                "purchase_note_mock"
            } else {
                "purchase_note_live"
            },
        )
        .to_owned(),
        canonical_url: canonical_url_for_path(site_base_url, "/design", &selected_locale),
        x_default_url: x_default_url_for_path(site_base_url, "/design"),
        seo_language_links: indexed_hreflang_links_for_path(site_base_url, "/design"),
        language_links: language_links_with_urls(|language| {
            design_url_with_filters(site_base_url, &language.route_code, &material_filter_state)
        }),
        company_url: company_url(site_base_url),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        selected_locale,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render page: {error}"),
        ),
    }
}

async fn handle_blog_index(
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_blog_index_page(state, query, None).await
}

async fn handle_localized_blog_index(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_blog_index_page(state, query, Some(locale)).await
}

async fn render_blog_index_page(
    state: AppState,
    query: LocaleQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let blog_posts = match load_blog_posts() {
        Ok(posts) => posts,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load blog posts: {error:#}"),
            );
        }
    };
    let template = BlogIndexTemplate {
        page_title: web_copy_text("blog_index", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("blog_index", &selected_locale, "seo_description")
            .to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(site_base_url, "/blog", &selected_locale),
        x_default_url: x_default_url_for_path(site_base_url, "/blog"),
        seo_language_links: indexed_hreflang_links_for_path(site_base_url, "/blog"),
        language_links: language_links_for_path(site_base_url, "/blog"),
        company_url: company_url(site_base_url),
        selected_locale: selected_locale.clone(),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        blog_posts: blog_post_cards(&blog_posts, site_base_url, &selected_locale),
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render blog index page: {error}"),
        ),
    }
}

async fn handle_blog_article(
    State(state): State<AppState>,
    Path(slug): Path<String>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_blog_article_page(state, slug, query, None).await
}

async fn handle_localized_blog_article(
    Path((locale, slug)): Path<(String, String)>,
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_blog_article_page(state, slug, query, Some(locale)).await
}

async fn render_blog_article_page(
    state: AppState,
    slug: String,
    query: LocaleQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let blog_posts = match load_blog_posts() {
        Ok(posts) => posts,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load blog posts: {error:#}"),
            );
        }
    };
    let Some(post) = find_blog_post(&blog_posts, &slug) else {
        return plain_error(StatusCode::NOT_FOUND, "blog article not found".to_owned());
    };
    let localized_post = localized_blog_post(&post, &selected_locale);
    let body_html = match read_blog_article_body(&post.slug, &selected_locale) {
        Ok(body_html) => body_html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load blog article: {error:#}"),
            );
        }
    };

    let template = BlogArticleTemplate {
        page_title: format!("{} | STONE SIGNATURE", &localized_post.title),
        meta_description: localized_post.meta_description.clone(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(
            site_base_url,
            &format!("/blog/{}", post.slug),
            &selected_locale,
        ),
        x_default_url: blog_article_url(site_base_url, &post.slug, "en"),
        seo_language_links: seo_language_links_with_urls(|language| {
            blog_article_url(site_base_url, &post.slug, &language.route_code)
        }),
        language_links: language_links_with_urls(|language| {
            blog_article_url(site_base_url, &post.slug, &language.route_code)
        }),
        og_image_url: absolute_content_url(site_base_url, &localized_post.image_url),
        company_url: company_url(site_base_url),
        selected_locale: selected_locale.clone(),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        blog_index_url: blog_index_url(site_base_url, &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        post: localized_post,
        body_html,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render blog article page: {error}"),
        ),
    }
}

fn material_filter_state_from_query(query: &PaymentRedirectQuery) -> MaterialFilterState {
    MaterialFilterState {
        color_family: normalize_facet_tag_value(query.color_family.as_deref().unwrap_or_default()),
        pattern_primary: normalize_facet_tag_value(
            query.pattern_primary.as_deref().unwrap_or_default(),
        ),
    }
}

fn checkout_redirect_path(
    base_url: &str,
    query: &PaymentRedirectQuery,
    locale: &str,
) -> Option<String> {
    let checkout = query.checkout.as_deref()?.trim().to_lowercase();
    let base_path = match checkout.as_str() {
        "success" => "/payment/success",
        "cancel" => "/payment/failure",
        _ => return None,
    };

    let mut params = locale_query_params(locale);
    if let Some(session_id) = query
        .session_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("session_id={session_id}"));
    }
    if let Some(order_id) = query
        .order_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("order_id={order_id}"));
    }
    if let Some(return_to) = query
        .return_to
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("return_to={return_to}"));
    }

    let query = if params.is_empty() {
        String::new()
    } else {
        format!("?{}", params.join("&"))
    };

    let path = localized_page_path(base_path, locale);
    Some(site_url(base_url, &format!("{path}{query}")))
}

fn payment_result_locale_url(
    base_url: &str,
    base_path: &str,
    query: &PaymentRedirectQuery,
    locale: &str,
) -> String {
    let normalized = parse_supported_locale(locale).unwrap_or("en");
    let mut params = locale_query_params(normalized);

    if let Some(checkout) = query
        .checkout
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("checkout={checkout}"));
    }
    if let Some(session_id) = query
        .session_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("session_id={session_id}"));
    }
    if let Some(order_id) = query
        .order_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("order_id={order_id}"));
    }
    if let Some(return_to) = query
        .return_to
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("return_to={return_to}"));
    }

    let query = if params.is_empty() {
        String::new()
    } else {
        format!("?{}", params.join("&"))
    };

    let path = localized_page_path(base_path, normalized);
    site_url(base_url, &format!("{path}{query}"))
}

fn app_checkout_return_url(
    outcome: &str,
    query: &PaymentRedirectQuery,
    locale: &str,
) -> Option<String> {
    let return_to = query.return_to.as_deref()?.trim().to_ascii_lowercase();
    if return_to != "app" {
        return None;
    }

    let outcome = match outcome.trim().to_ascii_lowercase().as_str() {
        "success" | "succeeded" | "paid" => "success",
        "cancel" | "canceled" | "cancelled" => "cancel",
        "failed" | "failure" | "error" => "failed",
        _ => return None,
    };

    let mut url = reqwest::Url::parse(&format!("hankofield://checkout/{outcome}")).ok()?;
    {
        let mut query_pairs = url.query_pairs_mut();
        query_pairs.append_pair("checkout", outcome);
        if let Some(order_id) = query
            .order_id
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
        {
            query_pairs.append_pair("order_id", order_id);
        }
        if let Some(session_id) = query
            .session_id
            .as_deref()
            .map(str::trim)
            .filter(|value| !value.is_empty())
        {
            query_pairs.append_pair("session_id", session_id);
        }
        query_pairs.append_pair(
            "lang",
            parse_supported_locale(locale).unwrap_or(DEFAULT_LOCALE),
        );
    }

    Some(url.to_string())
}

fn checkout_failure_app_outcome(query: &PaymentRedirectQuery) -> &'static str {
    match query
        .checkout
        .as_deref()
        .map(str::trim)
        .map(str::to_lowercase)
    {
        Some(value) if value == "cancel" || value == "canceled" || value == "cancelled" => "cancel",
        _ => "failed",
    }
}

#[cfg(test)]
fn payment_result_navigation_url(
    base_url: &str,
    base_path: &str,
    query: &PaymentRedirectQuery,
    locale: &str,
) -> String {
    let normalized = parse_supported_locale(locale).unwrap_or(DEFAULT_LOCALE);
    let mut params = localized_navigation_query_params(normalized);

    if let Some(checkout) = query
        .checkout
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("checkout={checkout}"));
    }
    if let Some(session_id) = query
        .session_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("session_id={session_id}"));
    }
    if let Some(order_id) = query
        .order_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("order_id={order_id}"));
    }
    if let Some(return_to) = query
        .return_to
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        params.push(format!("return_to={return_to}"));
    }

    let query = if params.is_empty() {
        String::new()
    } else {
        format!("?{}", params.join("&"))
    };

    let path = localized_page_path(base_path, normalized);
    site_url(base_url, &format!("{path}{query}"))
}

async fn handle_robots_txt(State(state): State<AppState>) -> Response {
    (
        [(header::CONTENT_TYPE, "text/plain; charset=utf-8")],
        build_robots_txt(&state.site_base_url),
    )
        .into_response()
}

async fn handle_sitemap_xml(State(state): State<AppState>) -> Response {
    match build_sitemap_xml(&state.site_base_url) {
        Ok(sitemap) => (
            [(header::CONTENT_TYPE, "application/xml; charset=utf-8")],
            sitemap,
        )
            .into_response(),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to build sitemap: {error:#}"),
        ),
    }
}

async fn handle_payment_success(
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_payment_success_page(state, query, None).await
}

async fn handle_localized_payment_success(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_payment_success_page(state, query, Some(locale)).await
}

async fn render_payment_success_page(
    state: AppState,
    query: PaymentRedirectQuery,
    path_locale: Option<String>,
) -> Response {
    let session_id = query
        .session_id
        .as_deref()
        .unwrap_or_default()
        .trim()
        .to_owned();
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let order_id = query
        .order_id
        .as_deref()
        .unwrap_or_default()
        .trim()
        .to_owned();
    let app_redirect_url =
        app_checkout_return_url("success", &query, &selected_locale).unwrap_or_default();
    let has_app_redirect_url = !app_redirect_url.is_empty();
    let template = PaymentSuccessTemplate {
        contact_url: inquiry_url(site_base_url, &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        page_title: web_copy_text("payment_success", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("payment_success", &selected_locale, "seo_description")
            .to_owned(),
        robots_meta: "noindex,follow".to_owned(),
        canonical_url: payment_result_locale_url(site_base_url, "/payment/success", &query, "en"),
        has_order_id: !order_id.is_empty(),
        order_id,
        has_session_id: !session_id.is_empty(),
        session_id,
        has_app_redirect_url,
        app_redirect_url,
        x_default_url: payment_result_locale_url(site_base_url, "/payment/success", &query, "en"),
        seo_language_links: seo_language_links_with_urls(|language| {
            payment_result_locale_url(
                site_base_url,
                "/payment/success",
                &query,
                &language.route_code,
            )
        }),
        language_links: language_links_with_urls(|language| {
            payment_result_locale_url(
                site_base_url,
                "/payment/success",
                &query,
                &language.route_code,
            )
        }),
        company_url: company_url(site_base_url),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        selected_locale,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render payment success page: {error}"),
        ),
    }
}

async fn handle_payment_failure(
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_payment_failure_page(state, query, None).await
}

async fn handle_localized_payment_failure(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<PaymentRedirectQuery>,
) -> Response {
    render_payment_failure_page(state, query, Some(locale)).await
}

async fn render_payment_failure_page(
    state: AppState,
    query: PaymentRedirectQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let order_id = query
        .order_id
        .as_deref()
        .unwrap_or_default()
        .trim()
        .to_owned();
    let app_redirect_url = app_checkout_return_url(
        checkout_failure_app_outcome(&query),
        &query,
        &selected_locale,
    )
    .unwrap_or_default();
    let has_app_redirect_url = !app_redirect_url.is_empty();
    let template = PaymentFailureTemplate {
        contact_url: inquiry_url(site_base_url, &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        page_title: web_copy_text("payment_failure", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("payment_failure", &selected_locale, "seo_description")
            .to_owned(),
        robots_meta: "noindex,follow".to_owned(),
        canonical_url: payment_result_locale_url(site_base_url, "/payment/failure", &query, "en"),
        has_order_id: !order_id.is_empty(),
        order_id,
        has_app_redirect_url,
        app_redirect_url,
        x_default_url: payment_result_locale_url(site_base_url, "/payment/failure", &query, "en"),
        seo_language_links: seo_language_links_with_urls(|language| {
            payment_result_locale_url(
                site_base_url,
                "/payment/failure",
                &query,
                &language.route_code,
            )
        }),
        language_links: language_links_with_urls(|language| {
            payment_result_locale_url(
                site_base_url,
                "/payment/failure",
                &query,
                &language.route_code,
            )
        }),
        company_url: company_url(site_base_url),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        design_url: localized_navigation_page_url(site_base_url, "/design", &selected_locale),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        selected_locale,
    };
    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render payment failure page: {error}"),
        ),
    }
}

async fn handle_commercial_transactions(
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_commercial_transactions_page(state, query, None).await
}

async fn handle_localized_commercial_transactions(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_commercial_transactions_page(state, query, Some(locale)).await
}

async fn render_commercial_transactions_page(
    state: AppState,
    query: LocaleQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let template = CommercialTransactionsTemplate {
        contact_url: inquiry_url(site_base_url, &selected_locale),
        page_title: web_copy_text("commercial_transactions", &selected_locale, "seo_title")
            .to_owned(),
        meta_description: web_copy_text(
            "commercial_transactions",
            &selected_locale,
            "seo_description",
        )
        .to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        x_default_url: x_default_url_for_path(site_base_url, "/commercial-transactions"),
        seo_language_links: indexed_hreflang_links_for_path(
            site_base_url,
            "/commercial-transactions",
        ),
        language_links: language_links_for_path(site_base_url, "/commercial-transactions"),
        company_url: company_url(site_base_url),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        selected_locale,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render commercial transactions page: {error}"),
        ),
    }
}

async fn handle_terms(State(state): State<AppState>, Query(query): Query<LocaleQuery>) -> Response {
    render_terms_page(state, query, None).await
}

async fn handle_localized_terms(
    Path(locale): Path<String>,
    State(state): State<AppState>,
    Query(query): Query<LocaleQuery>,
) -> Response {
    render_terms_page(state, query, Some(locale)).await
}

async fn render_terms_page(
    state: AppState,
    query: LocaleQuery,
    path_locale: Option<String>,
) -> Response {
    let selected_locale = match resolve_page_locale(
        path_locale.as_deref(),
        query.lang.as_deref(),
        &state.locale,
        &state.default_locale,
    ) {
        Ok(locale) => locale,
        Err(response) => return response,
    };
    let site_base_url = state.site_base_url.as_str();
    let template = TermsTemplate {
        contact_url: inquiry_url(site_base_url, &selected_locale),
        page_title: web_copy_text("terms", &selected_locale, "seo_title").to_owned(),
        meta_description: web_copy_text("terms", &selected_locale, "seo_description").to_owned(),
        robots_meta: robots_meta_for_locale(&selected_locale),
        canonical_url: canonical_url_for_path(site_base_url, "/terms", &selected_locale),
        x_default_url: x_default_url_for_path(site_base_url, "/terms"),
        seo_language_links: indexed_hreflang_links_for_path(site_base_url, "/terms"),
        language_links: language_links_for_path(site_base_url, "/terms"),
        company_url: company_url(site_base_url),
        terms_url: localized_navigation_page_url(site_base_url, "/terms", &selected_locale),
        about_url: localized_navigation_page_url(site_base_url, "/about", &selected_locale),
        commercial_transactions_url: localized_navigation_page_url(
            site_base_url,
            "/commercial-transactions",
            &selected_locale,
        ),
        privacy_policy_url: privacy_policy_url(site_base_url, &selected_locale),
        top_url: localized_navigation_page_url(site_base_url, "/", &selected_locale),
        selected_locale,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render terms page: {error}"),
        ),
    }
}

async fn handle_kanji_suggestions(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    let real_name = form_value(&form, "real_name");
    let requested_locale = form_value(&form, "locale");
    let selected_locale = resolve_request_locale(
        Some(&requested_locale),
        &state.locale,
        &state.default_locale,
    );
    let reason_language = selected_locale.clone();
    let candidate_gender = normalize_candidate_gender(&form_value(&form, "candidate_gender"));
    let kanji_style = normalize_kanji_style(&form_value(&form, "kanji_style"));

    let (suggestions, error) = if real_name.is_empty() {
        (Vec::new(), String::new())
    } else {
        match state
            .kanji_api
            .generate_candidates(&real_name, &reason_language, candidate_gender, kanji_style)
            .await
        {
            Ok(suggestions) => (suggestions, String::new()),
            Err(error) => {
                eprintln!("failed to load kanji candidates: {error:#}");
                (
                    Vec::new(),
                    localized_text(
                        &selected_locale,
                        "候補を生成できませんでした。時間をおいて再度お試しください。",
                        "Could not generate suggestions. Please try again later.",
                    ),
                )
            }
        }
    };

    let template = KanjiSuggestionsTemplate {
        real_name,
        kanji_style: kanji_style.to_owned(),
        selected_locale,
        has_suggestions: !suggestions.is_empty(),
        suggestions,
        error,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render suggestions: {error}"),
        ),
    }
}

async fn handle_purchase(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    handle_purchase_impl(state, form, false).await
}

async fn handle_mock_purchase(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    handle_purchase_impl(state, form, true).await
}

async fn handle_purchase_impl(
    state: AppState,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
    show_mock_confirmation: bool,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    let requested_locale = form_value(&form, "locale");
    let order_locale = resolve_request_locale(
        Some(&requested_locale),
        &state.locale,
        &state.default_locale,
    );

    let catalog = match load_catalog_with_timeout(state.source.as_ref(), &order_locale).await {
        Ok(catalog) => catalog,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to load catalog: {error}"),
            );
        }
    };
    let catalog = localize_catalog_prices(catalog, &order_locale);

    let font_by_key = catalog
        .fonts
        .iter()
        .cloned()
        .map(|font| (font.key.clone(), font))
        .collect::<HashMap<_, _>>();
    let listing_by_key = catalog
        .materials
        .iter()
        .cloned()
        .map(|listing| (listing.key.clone(), listing))
        .collect::<HashMap<_, _>>();
    let country_by_code = catalog
        .countries
        .iter()
        .cloned()
        .map(|country| (country.code.clone(), country))
        .collect::<HashMap<_, _>>();

    let seal_line1 = form_value(&form, "seal_line1");
    let seal_line2 = form_value(&form, "seal_line2");
    let font_key = form_value(&form, "font");
    let shape_key = form_value(&form, "shape");
    let listing_id = form_value(&form, "listing_id");
    let recipient_name = form_value(&form, "recipient_name");
    let phone = form_value(&form, "phone");
    let country_code = form_value(&form, "country");
    let postal_code = form_value(&form, "postal_code");
    let state_name = form_value(&form, "state");
    let city = form_value(&form, "city");
    let address_line1 = form_value(&form, "address_line1");
    let address_line2 = form_value(&form, "address_line2");
    let email = form_value(&form, "email");
    let terms_value = form_value(&form, "terms_agreed");
    let terms_agreed =
        terms_value == "on" || terms_value == "1" || terms_value.eq_ignore_ascii_case("true");

    let is_mock_confirmation = show_mock_confirmation || state.mode == RunMode::Mock;

    let mut result = PurchaseResultData {
        source_label: state.source.label().to_owned(),
        is_mock: is_mock_confirmation,
        selected_locale: order_locale.clone(),
        ..PurchaseResultData::default()
    };

    if let Err(message) = validate_seal_lines(&order_locale, &seal_line1, &seal_line2) {
        result.error = message;
        return render_purchase_result(&result);
    }

    let Some(font) = font_by_key.get(&font_key) else {
        result.error = localized_text(
            &order_locale,
            "フォントを選択してください。",
            "Please choose a font.",
        );
        return render_purchase_result(&result);
    };

    let Some(selected_shape_label) = shape_label_for_locale(&shape_key, &order_locale) else {
        result.error = localized_text(
            &order_locale,
            "印鑑の形状を選択してください。",
            "Please choose a seal shape.",
        );
        return render_purchase_result(&result);
    };

    let Some(listing) = listing_by_key.get(&listing_id) else {
        result.error = localized_text(
            &order_locale,
            "出品個体を選択してください。",
            "Please choose a listing.",
        );
        return render_purchase_result(&result);
    };
    if !listing.shape.eq_ignore_ascii_case(&shape_key) {
        result.error = localized_text(
            &order_locale,
            "選択した形状に対応する出品個体を選択してください。",
            "Please choose a listing that matches the selected shape.",
        );
        return render_purchase_result(&result);
    }

    let Some(country) = country_by_code.get(&country_code) else {
        result.error = localized_text(
            &order_locale,
            "配送先の国を選択してください。",
            "Please choose a shipping country.",
        );
        return render_purchase_result(&result);
    };

    if recipient_name.is_empty() {
        result.error = localized_text(
            &order_locale,
            "購入者名を入力してください。",
            "Enter the recipient name.",
        );
        return render_purchase_result(&result);
    }

    if phone.is_empty() {
        result.error = localized_text(
            &order_locale,
            "電話番号を入力してください。",
            "Enter a phone number.",
        );
        return render_purchase_result(&result);
    }

    if postal_code.is_empty() {
        result.error = localized_text(
            &order_locale,
            "郵便番号を入力してください。",
            "Enter a postal code.",
        );
        return render_purchase_result(&result);
    }

    if state_name.is_empty() {
        result.error = localized_text(
            &order_locale,
            "都道府県 / 州を入力してください。",
            "Enter a state or prefecture.",
        );
        return render_purchase_result(&result);
    }

    if city.is_empty() {
        result.error = localized_text(
            &order_locale,
            "市区町村 / City を入力してください。",
            "Enter a city.",
        );
        return render_purchase_result(&result);
    }

    if address_line1.is_empty() {
        result.error = localized_text(
            &order_locale,
            "住所1を入力してください。",
            "Enter address line 1.",
        );
        return render_purchase_result(&result);
    }

    if email.is_empty() {
        result.error = localized_text(
            &order_locale,
            "購入確認用のメールアドレスを入力してください。",
            "Enter the confirmation email address.",
        );
        return render_purchase_result(&result);
    }

    if !terms_agreed {
        result.error = localized_text(
            &order_locale,
            "利用規約への同意が必要です。",
            "Please agree to the terms of service.",
        );
        return render_purchase_result(&result);
    }

    let subtotal = listing.price;
    let shipping = country.shipping;
    let total = subtotal + shipping;
    let create_order_request = CreateOrderApiRequest {
        channel: "web".to_owned(),
        locale: order_locale.clone(),
        idempotency_key: generate_idempotency_key(),
        terms_agreed: true,
        seal: CreateOrderSealApiRequest {
            line1: seal_line1.clone(),
            line2: seal_line2.clone(),
            shape: shape_key.clone(),
            font_key: font_key.clone(),
        },
        listing_id: listing.key.clone(),
        shipping: CreateOrderShippingApiRequest {
            country_code: country_code.clone(),
            recipient_name: recipient_name.clone(),
            phone: phone.clone(),
            postal_code: postal_code.clone(),
            state: state_name.clone(),
            city: city.clone(),
            address_line1: address_line1.clone(),
            address_line2: address_line2.clone(),
        },
        contact: CreateOrderContactApiRequest {
            email: email.clone(),
            preferred_locale: order_locale.clone(),
        },
    };

    result.seal_line1 = seal_line1;
    result.seal_line2 = seal_line2;
    result.font_label = font.label.clone();
    result.shape_label = selected_shape_label.to_owned();
    result.listing_label = listing.label.clone();
    result.stripe_name = recipient_name;
    result.stripe_phone = phone;
    result.country_label = country.label.clone();
    result.postal_code = postal_code;
    result.state = state_name;
    result.city = city;
    result.address_line1 = address_line1;
    result.address_line2 = address_line2;
    result.subtotal = subtotal;
    result.shipping = shipping;
    result.total = total;
    result.email = email;

    if is_mock_confirmation {
        return render_purchase_result(&result);
    }

    let order = match state.kanji_api.create_order(&create_order_request).await {
        Ok(order) => order,
        Err(error) => {
            eprintln!("failed to create order for stripe checkout: {error:#}");
            result.error = localized_text(
                &order_locale,
                "注文の作成に失敗しました。時間をおいて再度お試しください。",
                "Could not create the order. Please try again later.",
            );
            return render_purchase_result(&result);
        }
    };

    let checkout_request = CreateStripeCheckoutSessionApiRequest {
        order_id: order.order_id,
        customer_email: result.email.clone(),
    };
    let checkout_session = match state
        .kanji_api
        .create_stripe_checkout_session(&checkout_request)
        .await
    {
        Ok(session) => session,
        Err(error) => {
            eprintln!("failed to create stripe checkout session: {error:#}");
            result.error = localized_text(
                &order_locale,
                "決済画面の作成に失敗しました。時間をおいて再度お試しください。",
                "Could not create the checkout session. Please try again later.",
            );
            return render_purchase_result(&result);
        }
    };

    hx_redirect_response(&checkout_session.checkout_url)
}

fn render_purchase_result(data: &PurchaseResultData) -> Response {
    let template = PurchaseResultTemplate {
        has_error: !data.error.is_empty(),
        error: data.error.clone(),
        selected_locale: data.selected_locale.clone(),
        seal_line1: data.seal_line1.clone(),
        seal_line2: data.seal_line2.clone(),
        has_seal_line2: !data.seal_line2.is_empty(),
        font_label: data.font_label.clone(),
        shape_label: data.shape_label.clone(),
        listing_label: data.listing_label.clone(),
        stripe_name: data.stripe_name.clone(),
        stripe_phone: data.stripe_phone.clone(),
        country_label: data.country_label.clone(),
        postal_code: data.postal_code.clone(),
        state: data.state.clone(),
        city: data.city.clone(),
        address_line1: data.address_line1.clone(),
        address_line2: data.address_line2.clone(),
        has_address_line2: !data.address_line2.is_empty(),
        subtotal_display: format_locale_amount(data.subtotal, &data.selected_locale),
        shipping_display: format_locale_amount(data.shipping, &data.selected_locale),
        total_display: format_locale_amount(data.total, &data.selected_locale),
        email: data.email.clone(),
        source_label: data.source_label.clone(),
        is_mock: data.is_mock,
    };

    match render_html(&template) {
        Ok(html) => html_response(html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render purchase result: {error}"),
        ),
    }
}

fn hx_redirect_response(url: &str) -> Response {
    let redirect = url.trim();
    if redirect.is_empty() {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            "failed to redirect to checkout".to_owned(),
        );
    }

    let mut response = StatusCode::OK.into_response();
    if let Ok(value) = HeaderValue::from_str(redirect) {
        response
            .headers_mut()
            .insert(HeaderName::from_static(HX_REDIRECT_HEADER), value);
        return response;
    }

    plain_error(
        StatusCode::INTERNAL_SERVER_ERROR,
        "failed to redirect to checkout".to_owned(),
    )
}

fn generate_idempotency_key() -> String {
    format!("web_{}", Uuid::new_v4().as_simple())
}

fn render_html<T: Template>(template: &T) -> Result<String> {
    template
        .render()
        .map_err(|error| anyhow!(error.to_string()))
}

fn should_forward_admin_request_header(name: &HeaderName) -> bool {
    let name = name.as_str().to_ascii_lowercase();
    !matches!(
        name.as_str(),
        "accept-encoding"
            | "connection"
            | "content-length"
            | "host"
            | "keep-alive"
            | "proxy-authenticate"
            | "proxy-authorization"
            | "te"
            | "trailer"
            | "transfer-encoding"
            | "upgrade"
    )
}

fn should_forward_admin_response_header(name: &HeaderName) -> bool {
    let name = name.as_str().to_ascii_lowercase();
    !matches!(
        name.as_str(),
        "connection"
            | "content-type"
            | "content-length"
            | "keep-alive"
            | "proxy-authenticate"
            | "proxy-authorization"
            | "te"
            | "trailer"
            | "transfer-encoding"
            | "upgrade"
    )
}

fn html_response(body: String) -> Response {
    ([(header::CONTENT_TYPE, "text/html; charset=utf-8")], body).into_response()
}

fn plain_error(status: StatusCode, message: String) -> Response {
    (status, message).into_response()
}

fn form_value(form: &HashMap<String, String>, key: &str) -> String {
    form.get(key)
        .map(|value| value.trim().to_owned())
        .unwrap_or_default()
}

fn resolve_request_locale(requested: Option<&str>, locale: &str, default_locale: &str) -> String {
    if let Some(value) = requested.and_then(parse_supported_locale) {
        return value.to_owned();
    }
    if let Some(value) = parse_supported_locale(locale) {
        return value.to_owned();
    }
    if let Some(value) = parse_supported_locale(default_locale) {
        return value.to_owned();
    }
    DEFAULT_LOCALE.to_owned()
}

fn parse_supported_locale(raw: &str) -> Option<&'static str> {
    web_language_registry()
        .enabled_language_for_input(raw)
        .map(|language| language.route_code.as_str())
}

fn parse_path_locale(raw: &str) -> Option<&'static str> {
    web_language_registry()
        .enabled_language_for_path_segment(raw)
        .map(|language| language.route_code.as_str())
}

fn is_japanese_locale(locale: &str) -> bool {
    parse_supported_locale(locale) == Some("ja")
}

fn resolve_page_locale(
    path_locale: Option<&str>,
    requested: Option<&str>,
    _locale: &str,
    _default_locale: &str,
) -> std::result::Result<String, Response> {
    if let Some(path_locale) = path_locale {
        if let Some(locale) = parse_path_locale(path_locale) {
            return Ok(locale.to_owned());
        }
        return Err(plain_error(
            StatusCode::NOT_FOUND,
            "localized page not found".to_owned(),
        ));
    }
    if let Some(locale) = requested.and_then(parse_supported_locale) {
        return Ok(locale.to_owned());
    }
    Ok(DEFAULT_LOCALE.to_owned())
}

fn site_url(base_url: &str, path: &str) -> String {
    let base = reqwest::Url::parse(base_url.trim_end_matches('/'))
        .expect("site base URL must be a valid absolute URL");
    base.join(path)
        .expect("failed to join site base URL with path")
        .to_string()
}

fn absolute_content_url(base_url: &str, path_or_url: &str) -> String {
    if path_or_url.starts_with("https://") || path_or_url.starts_with("http://") {
        return path_or_url.to_owned();
    }
    site_url(base_url, path_or_url)
}

fn normalize_site_base_url(raw: &str) -> Result<String> {
    let trimmed = raw.trim().trim_end_matches('/');
    let url = reqwest::Url::parse(trimmed)
        .with_context(|| format!("invalid HANKO_WEB_SITE_BASE_URL {trimmed:?}"))?;
    if url.scheme() != "http" && url.scheme() != "https" {
        bail!("HANKO_WEB_SITE_BASE_URL must use http or https");
    }
    Ok(url.as_str().trim_end_matches('/').to_owned())
}

fn locale_query_params(locale: &str) -> Vec<String> {
    let _ = locale;
    Vec::new()
}

fn localized_page_path(path: &str, locale: &str) -> String {
    let registry = web_language_registry();
    let language = registry
        .enabled_language_for_input(locale)
        .unwrap_or_else(|| registry.default_language());
    localized_page_path_for_language(path, language)
}

fn localized_page_path_for_language(path: &str, language: &WebLanguage) -> String {
    if language.route_code == DEFAULT_LOCALE {
        return path.to_owned();
    }

    let path = if path.is_empty() { "/" } else { path };
    if path == "/" {
        return format!("/{}/", language.url_prefix);
    }

    if let Some(path) = path.strip_prefix('/') {
        format!("/{}/{path}", language.url_prefix)
    } else {
        format!("/{}/{path}", language.url_prefix)
    }
}

fn localized_page_url(base_url: &str, path: &str, locale: &str) -> String {
    site_url(base_url, &localized_page_path(path, locale))
}

fn language_links_for_path(base_url: &str, path: &str) -> Vec<LanguageLink> {
    language_links_for_path_with_registry(web_language_registry(), base_url, path)
}

fn language_links_with_urls<F>(url_for_language: F) -> Vec<LanguageLink>
where
    F: Fn(&WebLanguage) -> String,
{
    web_language_registry()
        .enabled_languages()
        .iter()
        .map(|language| language_link_with_url(language, url_for_language(language)))
        .collect()
}

fn seo_language_links_with_urls<F>(url_for_language: F) -> Vec<LanguageLink>
where
    F: Fn(&WebLanguage) -> String,
{
    web_language_registry()
        .indexed_languages()
        .map(|language| language_link_with_url(language, url_for_language(language)))
        .collect()
}

fn language_links_for_path_with_registry(
    registry: &WebLanguageRegistry,
    base_url: &str,
    path: &str,
) -> Vec<LanguageLink> {
    registry
        .enabled_languages()
        .iter()
        .map(|language| language_link_for_path(base_url, path, language))
        .collect()
}

fn canonical_url_for_path(base_url: &str, path: &str, locale: &str) -> String {
    canonical_url_for_path_with_registry(web_language_registry(), base_url, path, locale)
}

fn robots_meta_for_locale(locale: &str) -> String {
    web_language_registry()
        .enabled_language_exact(locale)
        .filter(|language| language.indexed)
        .map(|_| "index,follow")
        .unwrap_or("noindex,follow")
        .to_owned()
}

fn canonical_url_for_path_with_registry(
    registry: &WebLanguageRegistry,
    base_url: &str,
    path: &str,
    locale: &str,
) -> String {
    let language = registry
        .enabled_language_exact(locale)
        .filter(|language| language.indexed)
        .unwrap_or_else(|| registry.default_language());
    site_url(base_url, &localized_page_path_for_language(path, language))
}

fn x_default_url_for_path(base_url: &str, path: &str) -> String {
    site_url(
        base_url,
        &localized_page_path_for_language(path, web_language_registry().default_language()),
    )
}

fn indexed_hreflang_links_for_path(base_url: &str, path: &str) -> Vec<LanguageLink> {
    indexed_hreflang_links_for_path_with_registry(web_language_registry(), base_url, path)
}

fn indexed_hreflang_links_for_path_with_registry(
    registry: &WebLanguageRegistry,
    base_url: &str,
    path: &str,
) -> Vec<LanguageLink> {
    registry
        .indexed_languages()
        .map(|language| language_link_for_path(base_url, path, language))
        .collect()
}

fn language_link_for_path(base_url: &str, path: &str, language: &WebLanguage) -> LanguageLink {
    language_link_with_url(
        language,
        site_url(base_url, &localized_page_path_for_language(path, language)),
    )
}

fn language_link_with_url(language: &WebLanguage, url: String) -> LanguageLink {
    LanguageLink {
        route_code: language.route_code.clone(),
        bcp47: language.bcp47.clone(),
        label: language.native_name.clone(),
        url,
        is_default: language.route_code == DEFAULT_LOCALE,
        is_indexed: language.indexed,
    }
}

#[cfg(test)]
fn localized_navigation_query_params(locale: &str) -> Vec<String> {
    let _ = locale;
    Vec::new()
}

fn localized_navigation_page_path(path: &str, locale: &str) -> String {
    localized_page_path(path, locale)
}

fn localized_navigation_page_url(base_url: &str, path: &str, locale: &str) -> String {
    site_url(base_url, &localized_navigation_page_path(path, locale))
}

fn privacy_policy_url(_base_url: &str, locale: &str) -> String {
    let normalized = parse_supported_locale(locale).unwrap_or("en");
    if normalized == "ja" {
        return site_url(EXTERNAL_LEGAL_BASE_URL, "/privacy/");
    }
    site_url(EXTERNAL_LEGAL_BASE_URL, &format!("/{normalized}/privacy/"))
}

fn commercial_transactions_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/commercial-transactions", locale)
}

fn top_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/", locale)
}

fn about_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/about", locale)
}

fn design_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/design", locale)
}

fn blog_index_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/blog", locale)
}

fn blog_article_url(base_url: &str, slug: &str, locale: &str) -> String {
    localized_page_url(base_url, &format!("/blog/{slug}"), locale)
}

fn design_url_with_filters(base_url: &str, locale: &str, filters: &MaterialFilterState) -> String {
    let base = reqwest::Url::parse(base_url.trim_end_matches('/'))
        .expect("site base URL must be a valid absolute URL");
    let mut url = base
        .join(&localized_page_path("/design", locale))
        .expect("failed to join site base URL with path");

    {
        let mut query_pairs = url.query_pairs_mut();
        if !filters.color_family.is_empty() {
            query_pairs.append_pair("color_family", &filters.color_family);
        }
        if !filters.pattern_primary.is_empty() {
            query_pairs.append_pair("pattern_primary", &filters.pattern_primary);
        }
    }

    url.to_string()
}

fn terms_url(base_url: &str, locale: &str) -> String {
    localized_page_url(base_url, "/terms", locale)
}

fn inquiry_url(_base_url: &str, locale: &str) -> String {
    let normalized = parse_supported_locale(locale).unwrap_or("en");
    if normalized == "ja" {
        return site_url(EXTERNAL_LEGAL_BASE_URL, "/contact/");
    }
    site_url(EXTERNAL_LEGAL_BASE_URL, &format!("/{normalized}/contact/"))
}

fn company_url(_base_url: &str) -> String {
    site_url(EXTERNAL_LEGAL_BASE_URL, "/company/")
}

const SITEMAP_STATIC_PAGES: &[SitemapPage] = &[
    SitemapPage {
        path: "/",
        lastmod: "2026-05-11",
    },
    SitemapPage {
        path: "/about",
        lastmod: "2026-05-11",
    },
    SitemapPage {
        path: "/design",
        lastmod: "2026-05-11",
    },
    SitemapPage {
        path: "/terms",
        lastmod: "2026-05-11",
    },
    SitemapPage {
        path: "/commercial-transactions",
        lastmod: "2026-05-11",
    },
];

const SITEMAP_BLOG_INDEX_FALLBACK_LASTMOD: &str = "2026-05-11";

fn build_robots_txt(base_url: &str) -> String {
    let sitemap_url = site_url(base_url, "/sitemap.xml");
    format!(
        "User-agent: *\nDisallow: /admin\nDisallow: /mock\nDisallow: /kanji\nDisallow: /purchase\nDisallow: /payment/\nSitemap: {sitemap_url}\n"
    )
}

fn sitemap_url_entry(base_url: &str, path: &str, lastmod: &str) -> Result<String> {
    sitemap_url_entry_with_registry(web_language_registry(), base_url, path, lastmod)
}

fn sitemap_url_entry_with_registry(
    registry: &WebLanguageRegistry,
    base_url: &str,
    path: &str,
    lastmod: &str,
) -> Result<String> {
    ensure_canonical_sitemap_path(path)?;
    ensure_valid_sitemap_lastmod(lastmod)?;

    let links = indexed_hreflang_links_for_path_with_registry(registry, base_url, path);
    let Some(default_link) = links.iter().find(|link| link.is_default) else {
        bail!("sitemap path requires an indexed default language: {path}");
    };
    let mut entry = String::new();
    for loc_link in &links {
        entry.push_str("  <url>\n");
        entry.push_str(&format!("    <loc>{}</loc>\n", loc_link.url));
        entry.push_str(&format!("    <lastmod>{lastmod}</lastmod>\n"));
        for alternate in &links {
            entry.push_str(&format!(
                "    <xhtml:link rel=\"alternate\" hreflang=\"{}\" href=\"{}\" />\n",
                alternate.bcp47, alternate.url
            ));
        }
        entry.push_str(&format!(
            "    <xhtml:link rel=\"alternate\" hreflang=\"x-default\" href=\"{}\" />\n",
            default_link.url
        ));
        entry.push_str("  </url>\n");
    }
    Ok(entry)
}

fn build_sitemap_xml(base_url: &str) -> Result<String> {
    let mut sitemap = String::from(
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<urlset xmlns=\"http://www.sitemaps.org/schemas/sitemap/0.9\" xmlns:xhtml=\"http://www.w3.org/1999/xhtml\">\n",
    );

    let posts = load_blog_posts()?;

    for page in SITEMAP_STATIC_PAGES {
        sitemap.push_str(&sitemap_url_entry(base_url, page.path, page.lastmod)?);
    }
    let blog_index_lastmod = posts
        .iter()
        .map(|post| post.last_modified_date.as_str())
        .max()
        .unwrap_or(SITEMAP_BLOG_INDEX_FALLBACK_LASTMOD);
    sitemap.push_str(&sitemap_url_entry(base_url, "/blog", blog_index_lastmod)?);
    for post in posts {
        sitemap.push_str(&sitemap_url_entry(
            base_url,
            &format!("/blog/{}", post.slug),
            &post.last_modified_date,
        )?);
    }
    sitemap.push_str("</urlset>\n");
    Ok(sitemap)
}

fn ensure_canonical_sitemap_path(path: &str) -> Result<()> {
    if !path.starts_with('/') {
        bail!("sitemap path must start with /: {path}");
    }
    if path.contains('?') || path.contains('#') {
        bail!("sitemap path must not include query or fragment: {path}");
    }
    if path != "/" && path.ends_with('/') {
        bail!("sitemap non-root paths must not end with /: {path}");
    }
    Ok(())
}

fn ensure_valid_sitemap_lastmod(lastmod: &str) -> Result<()> {
    let bytes = lastmod.as_bytes();
    if bytes.len() != 10
        || !bytes[0..4].iter().all(u8::is_ascii_digit)
        || bytes[4] != b'-'
        || !bytes[5..7].iter().all(u8::is_ascii_digit)
        || bytes[7] != b'-'
        || !bytes[8..10].iter().all(u8::is_ascii_digit)
    {
        bail!("sitemap lastmod must use YYYY-MM-DD: {lastmod}");
    }
    Ok(())
}

fn localized_text(locale: &str, ja: &str, en: &str) -> String {
    if is_japanese_locale(locale) {
        ja.to_owned()
    } else {
        en.to_owned()
    }
}

fn first_non_empty(values: &[Option<String>]) -> Option<String> {
    values
        .iter()
        .filter_map(|value| value.as_deref())
        .map(str::trim)
        .find(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

fn normalize_candidate_gender(raw: &str) -> &'static str {
    match raw.trim().to_lowercase().as_str() {
        "male" => "male",
        "female" => "female",
        _ => "unspecified",
    }
}

fn normalize_kanji_style(raw: &str) -> &'static str {
    match raw.trim().to_lowercase().as_str() {
        "chinese" => "chinese",
        "taiwanese" => "taiwanese",
        _ => "japanese",
    }
}

fn normalize_stone_shape(raw: &str) -> String {
    // Preserve unknown values so catalog data can round-trip without being coerced.
    match raw.trim().to_ascii_lowercase().as_str() {
        "round" => "round".to_owned(),
        "square" => "square".to_owned(),
        _ => raw.trim().to_owned(),
    }
}

fn stone_listing_is_published(status: &str) -> bool {
    status.trim().eq_ignore_ascii_case("published")
}

fn stone_listing_is_catalog_visible(is_active: bool, status: &str) -> bool {
    is_active && stone_listing_is_published(status)
}

fn normalize_facet_tag_value(raw: &str) -> String {
    raw.trim().to_ascii_lowercase()
}

fn mock_facet_tag_labels(locale: &str) -> FacetTagLabels {
    let japanese = is_japanese_locale(locale);
    let mut labels = FacetTagLabels::default();

    if japanese {
        labels.insert("color", "pink", "淡桃", &[]);
        labels.insert("color", "blue", "深青", &[]);
        labels.insert("color", "green", "濃緑", &[]);
        labels.insert("pattern", "cloud", "雲状", &[]);
        labels.insert("pattern", "speckled", "点状", &[]);
        labels.insert("pattern", "marble", "縞", &[]);
    } else {
        labels.insert("color", "pink", "Soft Pink", &[]);
        labels.insert("color", "blue", "Deep Blue", &[]);
        labels.insert("color", "green", "Deep Green", &[]);
        labels.insert("pattern", "cloud", "Cloud", &[]);
        labels.insert("pattern", "speckled", "Speckled", &[]);
        labels.insert("pattern", "marble", "Banded", &[]);
    }

    labels
}

fn material_shape_label(shape_key: &str, locale: &str) -> &'static str {
    let japanese = is_japanese_locale(locale);
    match shape_key {
        "round" => {
            if japanese {
                "丸印"
            } else {
                "Round seal"
            }
        }
        _ => {
            if japanese {
                "角印"
            } else {
                "Square seal"
            }
        }
    }
}

fn build_material_option_from_listing(
    category: &MaterialCategory,
    listing: &StoneListingRecord,
    facet_tag_labels: &FacetTagLabels,
    locale: &str,
    default_locale: &str,
    storage_assets_bucket: &str,
) -> MaterialOption {
    let price_currency = locale_currency_code(locale);
    let price =
        resolve_amount_for_currency(&listing.price_by_currency, price_currency).unwrap_or_default();
    let title = if listing.title.is_empty() {
        category.label.clone()
    } else {
        listing.title.clone()
    };
    let description = listing.description.clone();
    let story = listing.story.clone();
    let has_description = !description.trim().is_empty();
    let has_story = !story.trim().is_empty();
    let shape = normalize_stone_shape(&listing.stone_shape);
    let shape_label = match shape.as_str() {
        "square" => material_shape_label("square", locale).to_owned(),
        "round" => material_shape_label("round", locale).to_owned(),
        _ => shape.clone(),
    };

    let (photo_url, photo_alt, has_photo) = resolve_listing_photo(
        &listing.photos,
        storage_assets_bucket,
        locale,
        default_locale,
    );
    let photo_alt = if has_photo && photo_alt.is_empty() {
        localized_text(locale, &format!("{title}の写真"), &format!("{title} photo"))
    } else {
        photo_alt
    };
    let color_tag_labels = facet_tag_labels.resolve_list("color", &listing.color_tags);
    let pattern_tag_labels = facet_tag_labels.resolve_list("pattern", &listing.pattern_tags);
    let has_color_tag_labels = !color_tag_labels.is_empty();
    let has_pattern_tag_labels = !pattern_tag_labels.is_empty();
    MaterialOption {
        key: listing.key.clone(),
        label: title,
        description,
        story,
        has_description,
        has_story,
        price_by_currency: listing.price_by_currency.clone(),
        shape,
        shape_label,
        color_family: listing.color_family.clone(),
        pattern_primary: listing.pattern_primary.clone(),
        color_tag_labels,
        pattern_tag_labels,
        has_color_tag_labels,
        has_pattern_tag_labels,
        price,
        price_display: format_currency_amount(price, price_currency),
        photo_url,
        photo_alt,
        has_photo,
    }
}

fn build_material_filters(
    materials: &[MaterialOption],
    facet_tag_labels: &FacetTagLabels,
) -> MaterialFilters {
    let color_options = collect_canonical_filter_options(
        materials,
        "color",
        |material| material.color_family.as_str(),
        facet_tag_labels,
    );
    let pattern_options = collect_canonical_filter_options(
        materials,
        "pattern",
        |material| material.pattern_primary.as_str(),
        facet_tag_labels,
    );

    MaterialFilters {
        color_options,
        pattern_options,
    }
}

fn collect_canonical_filter_options(
    materials: &[MaterialOption],
    facet_type: &str,
    value_fn: impl Fn(&MaterialOption) -> &str,
    facet_tag_labels: &FacetTagLabels,
) -> Vec<MaterialFilterOption> {
    let mut seen = HashSet::new();
    let mut options = Vec::new();

    for material in materials {
        let value = normalize_facet_tag_value(value_fn(material));
        if value.is_empty() || !seen.insert(value.clone()) {
            continue;
        }

        options.push(MaterialFilterOption {
            value: value.clone(),
            label: facet_tag_labels.resolve_or_raw(facet_type, &value),
        });
    }

    options
}

#[allow(dead_code)]
fn filter_materials_by_facets(
    materials: &[MaterialOption],
    filter_state: &MaterialFilterState,
) -> Vec<MaterialOption> {
    materials
        .iter()
        .filter(|material| {
            let matches_color_family = filter_state.color_family.is_empty()
                || normalize_facet_tag_value(&material.color_family) == filter_state.color_family;
            let matches_pattern_primary = filter_state.pattern_primary.is_empty()
                || normalize_facet_tag_value(&material.pattern_primary)
                    == filter_state.pattern_primary;

            matches_color_family && matches_pattern_primary
        })
        .cloned()
        .collect()
}

fn shape_label_for_locale(shape_key: &str, locale: &str) -> Option<&'static str> {
    let japanese = is_japanese_locale(locale);
    match shape_key {
        "square" => Some(if japanese { "角印" } else { "Square seal" }),
        "round" => Some(if japanese { "丸印" } else { "Round seal" }),
        _ => None,
    }
}

fn validate_seal_lines(locale: &str, line1: &str, line2: &str) -> std::result::Result<(), String> {
    let first = line1.trim();
    let second = line2.trim();

    if first.is_empty() {
        return Err(localized_text(
            locale,
            "お名前を入力してください。",
            "Enter your name.",
        ));
    }

    if contains_whitespace(first) {
        return Err(localized_text(
            locale,
            "1行目に空白は使えません。",
            "Line 1 cannot contain spaces.",
        ));
    }

    if !second.is_empty() && contains_whitespace(second) {
        return Err(localized_text(
            locale,
            "2行目に空白は使えません。",
            "Line 2 cannot contain spaces.",
        ));
    }

    if first.chars().count() + second.chars().count() > 2 {
        return Err(localized_text(
            locale,
            "印影テキストは1行目と2行目の合計で2文字以内で入力してください。",
            "Use up to 2 characters total across lines 1 and 2.",
        ));
    }

    Ok(())
}

fn contains_whitespace(value: &str) -> bool {
    value.chars().any(char::is_whitespace)
}

fn format_usd(amount_cents: i64) -> String {
    let sign = if amount_cents < 0 { "-" } else { "" };
    let cents = amount_cents.abs();
    let whole = cents / 100;
    let fraction = cents % 100;
    let whole_display = format_with_grouping(whole);
    format!("{sign}USD {whole_display}.{fraction:02}")
}

fn format_jpy(amount_yen: i64) -> String {
    let sign = if amount_yen < 0 { "-" } else { "" };
    let yen = amount_yen.abs();
    format!("{sign}JPY {}", format_with_grouping(yen))
}

fn format_currency_amount(amount: i64, currency: &str) -> String {
    let normalized = currency.trim().to_ascii_uppercase();
    match normalized.as_str() {
        "JPY" => format_jpy(amount),
        _ => format_usd(amount),
    }
}

fn format_locale_amount(amount: i64, locale: &str) -> String {
    format_currency_amount(amount, locale_currency_code(locale))
}

fn format_with_grouping(value: i64) -> String {
    if value == 0 {
        return "0".to_owned();
    }

    let digits = value.to_string();
    let mut out = String::with_capacity(digits.len() + digits.len() / 3);
    for (index, ch) in digits.chars().enumerate() {
        if index > 0 && (digits.len() - index) % 3 == 0 {
            out.push(',');
        }
        out.push(ch);
    }
    out
}

fn document_id(document: &Document) -> Option<String> {
    document
        .name
        .as_deref()
        .and_then(|name| name.rsplit('/').next())
        .map(ToOwned::to_owned)
}

fn resolve_localized_field(
    data: &BTreeMap<String, JsonValue>,
    i18n_field: &str,
    locale: &str,
    default_locale: &str,
    fallback: &str,
) -> String {
    let values = read_string_map_field(data, i18n_field);
    let localized = resolve_localized_text(&values, locale, default_locale);
    if !localized.is_empty() {
        return localized;
    }

    fallback.to_owned()
}

fn stone_listing_color_family_from_facets(facets: &BTreeMap<String, JsonValue>) -> String {
    normalize_facet_tag_value(&read_string_field(facets, "color_family"))
}

fn resolve_font_label_field(
    data: &BTreeMap<String, JsonValue>,
    locale: &str,
    default_locale: &str,
    fallback: &str,
) -> String {
    let label = read_string_field(data, "label");
    if !label.is_empty() {
        return label;
    }

    let localized = resolve_localized_text(
        &read_string_map_field(data, "label_i18n"),
        locale,
        default_locale,
    );
    if !localized.is_empty() {
        return localized;
    }

    fallback.to_owned()
}

fn resolve_localized_text(
    values: &HashMap<String, String>,
    locale: &str,
    default_locale: &str,
) -> String {
    if values.is_empty() {
        return String::new();
    }

    if let Some(value) = lookup_locale(values, locale) {
        return value;
    }
    if let Some(value) = lookup_locale(values, default_locale) {
        return value;
    }
    if let Some(value) = lookup_locale(values, "ja") {
        return value;
    }

    let mut keys = values
        .iter()
        .filter_map(|(key, value)| {
            if value.trim().is_empty() {
                None
            } else {
                Some(key.clone())
            }
        })
        .collect::<Vec<_>>();
    keys.sort();

    if let Some(key) = keys.first() {
        return values
            .get(key)
            .map(|value| value.trim().to_owned())
            .unwrap_or_default();
    }

    String::new()
}

fn lookup_locale(values: &HashMap<String, String>, target: &str) -> Option<String> {
    let target = target.trim().to_lowercase();
    if target.is_empty() {
        return None;
    }

    for (key, value) in values {
        if key.trim().to_lowercase() == target {
            let trimmed = value.trim();
            if !trimmed.is_empty() {
                return Some(trimmed.to_owned());
            }
        }
    }

    if let Some((base, _)) = target.split_once('-') {
        for (key, value) in values {
            if key.trim().to_lowercase() == base {
                let trimmed = value.trim();
                if !trimmed.is_empty() {
                    return Some(trimmed.to_owned());
                }
            }
        }
    }

    None
}

fn read_string_field(data: &BTreeMap<String, JsonValue>, key: &str) -> String {
    data.get(key)
        .and_then(|value| {
            value
                .get("stringValue")
                .and_then(JsonValue::as_str)
                .or_else(|| value.as_str())
        })
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .unwrap_or_default()
}

fn read_int_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Option<i64> {
    let value = data.get(key)?;

    if let Some(integer_value) = value.get("integerValue") {
        if let Some(text) = integer_value.as_str() {
            if let Ok(parsed) = text.trim().parse::<i64>() {
                return Some(parsed);
            }
        }
        if let Some(parsed) = integer_value.as_i64() {
            return Some(parsed);
        }
        if let Some(parsed) = integer_value.as_u64().and_then(|v| i64::try_from(v).ok()) {
            return Some(parsed);
        }
    }

    if let Some(double_value) = value.get("doubleValue").and_then(JsonValue::as_f64) {
        return Some(double_value as i64);
    }

    value
        .as_i64()
        .or_else(|| value.as_u64().and_then(|v| i64::try_from(v).ok()))
}

fn read_timestamp_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Option<String> {
    let value = data.get(key)?;
    let raw = value
        .get("timestampValue")
        .and_then(JsonValue::as_str)
        .or_else(|| value.as_str())?;

    let trimmed = raw.trim();
    if trimmed.is_empty() {
        None
    } else {
        Some(trimmed.to_owned())
    }
}

fn read_int_map_field(data: &BTreeMap<String, JsonValue>, key: &str) -> HashMap<String, i64> {
    let Some(value) = data.get(key) else {
        return HashMap::new();
    };

    let Some(fields) = value
        .get("mapValue")
        .and_then(|map_value| map_value.get("fields"))
        .and_then(JsonValue::as_object)
        .or_else(|| value.as_object())
    else {
        return HashMap::new();
    };

    let mut result = HashMap::new();
    for (map_key, map_value) in fields {
        let mut field = BTreeMap::new();
        field.insert("amount".to_owned(), map_value.clone());
        if let Some(amount) = read_int_field(&field, "amount") {
            result.insert(map_key.trim().to_ascii_uppercase(), amount);
        }
    }

    result
}

fn locale_currency_code(locale: &str) -> &'static str {
    if is_japanese_locale(locale) {
        "JPY"
    } else {
        "USD"
    }
}

fn localize_catalog_prices(mut catalog: CatalogData, locale: &str) -> CatalogData {
    let currency = locale_currency_code(locale);

    for material in &mut catalog.materials {
        if let Some(price) = resolve_amount_for_currency(&material.price_by_currency, currency) {
            material.price = price;
            material.price_display = format_currency_amount(price, currency);
        }
    }

    for country in &mut catalog.countries {
        if let Some(shipping) =
            resolve_amount_for_currency(&country.shipping_fee_by_currency, currency)
        {
            country.shipping = shipping;
        }
    }

    catalog
}

fn resolve_amount_for_currency(values: &HashMap<String, i64>, currency: &str) -> Option<i64> {
    let code = currency.trim().to_ascii_uppercase();
    if let Some(amount) = values.get(&code).copied() {
        return Some(amount.max(0));
    }
    if let Some(amount) = values.get("USD").copied() {
        return Some(amount.max(0));
    }
    if let Some(amount) = values.get("JPY").copied() {
        return Some(amount.max(0));
    }

    let mut keys = values.keys().cloned().collect::<Vec<_>>();
    keys.sort();
    keys.into_iter()
        .find_map(|key| values.get(&key).copied())
        .map(|amount| amount.max(0))
}

fn read_string_map_field(data: &BTreeMap<String, JsonValue>, key: &str) -> HashMap<String, String> {
    let Some(value) = data.get(key) else {
        return HashMap::new();
    };

    let Some(fields) = value
        .get("mapValue")
        .and_then(|map_value| map_value.get("fields"))
        .and_then(JsonValue::as_object)
    else {
        return HashMap::new();
    };

    let mut result = HashMap::new();
    for (map_key, map_value) in fields {
        let text = map_value
            .get("stringValue")
            .and_then(JsonValue::as_str)
            .map(str::trim)
            .filter(|value| !value.is_empty());
        if let Some(text) = text {
            result.insert(map_key.clone(), text.to_owned());
        }
    }

    result
}

fn read_string_array_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Vec<String> {
    read_array_field(data, key)
        .into_iter()
        .filter_map(|value| {
            value
                .get("stringValue")
                .and_then(JsonValue::as_str)
                .or_else(|| value.as_str())
                .map(str::trim)
                .filter(|value| !value.is_empty())
                .map(ToOwned::to_owned)
        })
        .collect::<Vec<_>>()
}

fn read_map_field(data: &BTreeMap<String, JsonValue>, key: &str) -> BTreeMap<String, JsonValue> {
    data.get(key)
        .and_then(|value| value.get("mapValue"))
        .and_then(|map_value| map_value.get("fields"))
        .and_then(JsonValue::as_object)
        .map(|fields| {
            fields
                .iter()
                .map(|(key, value)| (key.clone(), value.clone()))
                .collect::<BTreeMap<_, _>>()
        })
        .unwrap_or_default()
}

fn read_material_photos(data: &BTreeMap<String, JsonValue>) -> Vec<MaterialPhoto> {
    let mut photos = read_array_field(data, "photos")
        .into_iter()
        .filter_map(|photo| {
            let fields = photo
                .get("mapValue")
                .and_then(|map| map.get("fields"))
                .and_then(JsonValue::as_object)?;

            let fields = fields
                .iter()
                .map(|(key, value)| (key.clone(), value.clone()))
                .collect::<BTreeMap<_, _>>();

            Some(MaterialPhoto {
                asset_id: read_string_field(&fields, "asset_id"),
                storage_path: read_string_field(&fields, "storage_path"),
                alt_i18n: read_string_map_field(&fields, "alt_i18n"),
                sort_order: read_int_field(&fields, "sort_order").unwrap_or_default(),
                is_primary: read_bool_field(&fields, "is_primary").unwrap_or(false),
            })
        })
        .collect::<Vec<_>>();

    photos.sort_by(|left, right| {
        left.sort_order
            .cmp(&right.sort_order)
            .then(left.asset_id.cmp(&right.asset_id))
    });
    photos
}

#[cfg(test)]
fn resolve_material_photo(
    data: &BTreeMap<String, JsonValue>,
    storage_assets_bucket: &str,
    locale: &str,
    default_locale: &str,
) -> (String, String, bool) {
    if let Some((storage_path, storage_alt)) =
        select_primary_material_photo(data, locale, default_locale)
    {
        let photo_url = build_storage_media_url(storage_assets_bucket, &storage_path);
        if !photo_url.is_empty() {
            let photo_alt = if storage_alt.is_empty() {
                resolve_localized_field(data, "photo_alt_i18n", locale, default_locale, "")
            } else {
                storage_alt
            };
            return (photo_url, photo_alt, true);
        }
    }

    let photo_alt = resolve_localized_field(data, "photo_alt_i18n", locale, default_locale, "");
    (String::new(), photo_alt, false)
}

fn resolve_listing_photo(
    photos: &[MaterialPhoto],
    storage_assets_bucket: &str,
    locale: &str,
    default_locale: &str,
) -> (String, String, bool) {
    let Some(photo) = select_primary_listing_photo(photos) else {
        return (String::new(), String::new(), false);
    };

    let photo_url = build_storage_media_url(storage_assets_bucket, &photo.storage_path);
    let photo_alt = resolve_localized_text(&photo.alt_i18n, locale, default_locale);
    let has_photo = !photo_url.is_empty();
    (photo_url, photo_alt, has_photo)
}

fn select_primary_listing_photo(photos: &[MaterialPhoto]) -> Option<&MaterialPhoto> {
    let mut selected: Option<&MaterialPhoto> = None;
    for photo in photos {
        if photo.storage_path.trim().is_empty() {
            continue;
        }
        let replace = match selected {
            Some(current) => {
                if photo.is_primary != current.is_primary {
                    photo.is_primary && !current.is_primary
                } else if photo.sort_order != current.sort_order {
                    photo.sort_order < current.sort_order
                } else {
                    photo.asset_id < current.asset_id
                }
            }
            None => true,
        };
        if replace {
            selected = Some(photo);
        }
    }
    selected
}

#[cfg(test)]
fn select_primary_material_photo(
    data: &BTreeMap<String, JsonValue>,
    locale: &str,
    default_locale: &str,
) -> Option<(String, String)> {
    let mut selected_rank: Option<(i32, i64, usize)> = None;
    let mut selected_path = String::new();
    let mut selected_alt = String::new();

    for (index, photo) in read_array_field(data, "photos").into_iter().enumerate() {
        let Some(fields) = photo
            .get("mapValue")
            .and_then(|map| map.get("fields"))
            .and_then(JsonValue::as_object)
        else {
            continue;
        };

        let fields = fields
            .iter()
            .map(|(key, value)| (key.clone(), value.clone()))
            .collect::<BTreeMap<_, _>>();

        let storage_path = normalize_storage_path(&read_string_field(&fields, "storage_path"));
        if storage_path.is_empty() {
            continue;
        }

        let alt = resolve_localized_text(
            &read_string_map_field(&fields, "alt_i18n"),
            locale,
            default_locale,
        );
        let is_primary = read_bool_field(&fields, "is_primary").unwrap_or(false);
        let sort_order = read_int_field(&fields, "sort_order").unwrap_or_default();
        let rank = (if is_primary { 0 } else { 1 }, sort_order, index);

        let should_replace = match selected_rank {
            Some(current_rank) => rank < current_rank,
            None => true,
        };
        if should_replace {
            selected_rank = Some(rank);
            selected_path = storage_path;
            selected_alt = alt;
        }
    }

    if selected_path.is_empty() {
        None
    } else {
        Some((selected_path, selected_alt))
    }
}

fn build_storage_media_url(bucket_name: &str, storage_path: &str) -> String {
    let normalized_bucket = normalize_storage_bucket_name(bucket_name);
    let normalized_path = normalize_storage_path(storage_path);
    if normalized_bucket.is_empty() || normalized_path.is_empty() {
        return String::new();
    }

    format!(
        "https://storage.googleapis.com/{}/{}",
        normalized_bucket, normalized_path
    )
}

fn normalize_storage_bucket_name(value: &str) -> String {
    value
        .trim()
        .trim_start_matches("gs://")
        .trim_start_matches("GS://")
        .trim_matches('/')
        .to_owned()
}

fn normalize_storage_path(value: &str) -> String {
    value.trim().trim_start_matches('/').to_owned()
}

fn read_bool_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Option<bool> {
    let value = data.get(key)?;
    value
        .get("booleanValue")
        .and_then(JsonValue::as_bool)
        .or_else(|| value.as_bool())
}

fn read_array_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Vec<JsonValue> {
    let Some(value) = data.get(key) else {
        return Vec::new();
    };

    value
        .get("arrayValue")
        .and_then(|array| array.get("values"))
        .and_then(JsonValue::as_array)
        .cloned()
        .or_else(|| value.as_array().cloned())
        .unwrap_or_default()
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::{Body, to_bytes};
    use axum::extract::Form;
    use axum::http::Request;
    use serde_json::json;
    use std::collections::{BTreeMap, HashMap};
    use std::sync::Arc;
    use tower::ServiceExt;

    const TEST_SITE_BASE_URL: &str = "https://finitefield.org";
    const TEST_ALT_SITE_BASE_URL: &str = "https://inkanfield.org";
    const ADDED_BLOG_ARTICLE_SLUGS: [&str; 12] = [
        "custom-stone-seal-gift",
        "custom-jade-seal",
        "japanese-hanko-souvenir",
        "how-to-choose-stone-seal",
        "jade-agate-qingtian-stone-seal",
        "personal-seal-symbol-of-identity",
        "luxury-personal-seal",
        "english-name-kanji-seal",
        "what-to-engrave-on-seal",
        "personal-seals-for-artists",
        "chinese-chop-seal-vs-japanese-hanko",
        "one-of-a-kind-stone-seal",
    ];

    fn mock_state() -> AppState {
        AppState {
            source: Arc::new(CatalogSource::Mock(new_mock_catalog_source("ja"))),
            kanji_api: Arc::new(KanjiApiClient {
                base_url: "http://localhost:3050".to_owned(),
                http_client: reqwest::Client::new(),
            }),
            admin_proxy: Arc::new(AdminProxyClient {
                base_url: "http://localhost:3051".to_owned(),
                http_client: reqwest::Client::new(),
            }),
            mode: RunMode::Mock,
            locale: "ja".to_owned(),
            default_locale: "ja".to_owned(),
            site_base_url: TEST_SITE_BASE_URL.to_owned(),
        }
    }

    async fn route_get(path: &str) -> Response {
        build_router(mock_state())
            .oneshot(
                Request::builder()
                    .method("GET")
                    .uri(path)
                    .body(Body::empty())
                    .expect("test request should build"),
            )
            .await
            .expect("router should respond")
    }

    async fn route_get_html(path: &str) -> (StatusCode, String) {
        let response = route_get(path).await;
        let status = response.status();
        let body = to_bytes(response.into_body(), usize::MAX)
            .await
            .expect("response body should be readable");
        (
            status,
            String::from_utf8(body.to_vec()).expect("response body should be utf-8"),
        )
    }

    fn language_registry_entry(
        route_code: &str,
        bcp47: &str,
        native_name: &str,
        english_name: &str,
        text_direction: RegistryTextDirection,
        enabled: bool,
        indexed: bool,
        url_prefix: &str,
    ) -> LanguageRegistryEntry {
        LanguageRegistryEntry {
            route_code: route_code.to_owned(),
            bcp47: bcp47.to_owned(),
            native_name: native_name.to_owned(),
            english_name: english_name.to_owned(),
            text_direction,
            web: LanguageRegistryWebConfig {
                enabled,
                indexed,
                url_prefix: url_prefix.to_owned(),
            },
        }
    }

    async fn render_blog_article_html_for_test(slug: &str, locale: &str) -> String {
        let response = if is_japanese_locale(locale) {
            handle_localized_blog_article(
                Path(("ja".to_owned(), slug.to_owned())),
                State(mock_state()),
                Query(LocaleQuery::default()),
            )
            .await
        } else {
            handle_blog_article(
                State(mock_state()),
                Path(slug.to_owned()),
                Query(LocaleQuery {
                    lang: Some("en".to_owned()),
                }),
            )
            .await
        };

        assert_eq!(response.status(), StatusCode::OK);
        String::from_utf8(
            to_bytes(response.into_body(), usize::MAX)
                .await
                .expect("blog article body should be readable")
                .to_vec(),
        )
        .expect("blog article body should be utf-8")
    }

    fn assert_sitemap_has_localized_article_entry(sitemap_xml: &str, slug: &str, lastmod: &str) {
        let en_url = blog_article_url(TEST_SITE_BASE_URL, slug, "en");
        let ja_url = blog_article_url(TEST_SITE_BASE_URL, slug, "ja");

        assert!(
            sitemap_xml.contains(&format!("<loc>{en_url}</loc>")),
            "missing sitemap loc for {slug}"
        );
        assert!(
            sitemap_xml.contains(&format!("<loc>{ja_url}</loc>")),
            "missing localized sitemap loc for {slug}"
        );
        assert!(
            sitemap_xml.contains(&format!("<lastmod>{lastmod}</lastmod>")),
            "missing sitemap lastmod for {slug}"
        );
        assert!(
            sitemap_xml.contains(&format!(
                r#"<xhtml:link rel="alternate" hreflang="en" href="{en_url}" />"#
            )),
            "missing English alternate for {slug}"
        );
        assert!(
            sitemap_xml.contains(&format!(
                r#"<xhtml:link rel="alternate" hreflang="ja" href="{ja_url}" />"#
            )),
            "missing Japanese alternate for {slug}"
        );
        assert!(
            sitemap_xml.contains(&format!(
                r#"<xhtml:link rel="alternate" hreflang="x-default" href="{en_url}" />"#
            )),
            "missing x-default alternate for {slug}"
        );
    }

    fn valid_purchase_form() -> HashMap<String, String> {
        HashMap::from([
            ("locale".to_owned(), "ja".to_owned()),
            ("seal_line1".to_owned(), "黒".to_owned()),
            ("seal_line2".to_owned(), String::new()),
            ("font".to_owned(), "zen_maru_gothic".to_owned()),
            ("shape".to_owned(), "square".to_owned()),
            ("listing_id".to_owned(), "rose_quartz".to_owned()),
            ("recipient_name".to_owned(), "小野光".to_owned()),
            ("phone".to_owned(), "+81 80 6242 2597".to_owned()),
            ("country".to_owned(), "JP".to_owned()),
            ("postal_code".to_owned(), "5500012".to_owned()),
            ("state".to_owned(), "大阪府".to_owned()),
            ("city".to_owned(), "大阪市西区".to_owned()),
            ("address_line1".to_owned(), "立売堀5丁目5-9".to_owned()),
            (
                "address_line2".to_owned(),
                "第二レジデンス春日井503".to_owned(),
            ),
            ("email".to_owned(), "ono@finitefield.org".to_owned()),
            ("terms_agreed".to_owned(), "on".to_owned()),
        ])
    }

    #[test]
    fn create_order_api_request_serializes_legacy_web_payload_without_app_fields() {
        let request = CreateOrderApiRequest {
            channel: "web".to_owned(),
            locale: "ja".to_owned(),
            idempotency_key: "demo_key_123".to_owned(),
            terms_agreed: true,
            seal: CreateOrderSealApiRequest {
                line1: "田中".to_owned(),
                line2: "太郎".to_owned(),
                shape: "square".to_owned(),
                font_key: "zen_maru_gothic".to_owned(),
            },
            listing_id: "rose_quartz_01".to_owned(),
            shipping: CreateOrderShippingApiRequest {
                country_code: "JP".to_owned(),
                recipient_name: "田中 太郎".to_owned(),
                phone: "09000001111".to_owned(),
                postal_code: "1000001".to_owned(),
                state: "東京都".to_owned(),
                city: "千代田区".to_owned(),
                address_line1: "1-1-1".to_owned(),
                address_line2: String::new(),
            },
            contact: CreateOrderContactApiRequest {
                email: "taro@example.com".to_owned(),
                preferred_locale: "ja".to_owned(),
            },
        };

        let payload = serde_json::to_value(&request).expect("request should serialize");

        assert_eq!(
            payload,
            json!({
                "channel": "web",
                "locale": "ja",
                "idempotency_key": "demo_key_123",
                "terms_agreed": true,
                "seal": {
                    "line1": "田中",
                    "line2": "太郎",
                    "shape": "square",
                    "font_key": "zen_maru_gothic"
                },
                "listing_id": "rose_quartz_01",
                "shipping": {
                    "country_code": "JP",
                    "recipient_name": "田中 太郎",
                    "phone": "09000001111",
                    "postal_code": "1000001",
                    "state": "東京都",
                    "city": "千代田区",
                    "address_line1": "1-1-1",
                    "address_line2": ""
                },
                "contact": {
                    "email": "taro@example.com",
                    "preferred_locale": "ja"
                }
            })
        );
        assert!(payload.get("customer_confirmation").is_none());
        assert!(payload.get("order_note").is_none());
        assert!(payload["seal"].get("ai_generation_id").is_none());
        assert!(payload["seal"].get("ai_variant_id").is_none());
        assert!(payload["seal"].get("preview_image").is_none());
        assert!(payload["seal"].get("style").is_none());
    }

    #[tokio::test]
    async fn catalog_load_uses_requested_locale_for_mock_source() {
        let source = CatalogSource::Mock(new_mock_catalog_source("ja"));
        let catalog = load_catalog_with_timeout(&source, "en")
            .await
            .expect("catalog should load");

        let rose_quartz = catalog
            .materials
            .iter()
            .find(|material| material.key == "rose_quartz")
            .expect("rose_quartz material should exist");

        assert_eq!(rose_quartz.label, "Rose Quartz");
        assert_eq!(
            rose_quartz.description,
            "A soft-toned stone with a warm, approachable presence"
        );
        assert_eq!(rose_quartz.shape_label, "Square seal");
        assert_eq!(rose_quartz.color_tag_labels, vec!["Soft Pink"]);
        assert_eq!(rose_quartz.pattern_tag_labels, vec!["Cloud"]);
        assert!(rose_quartz.has_color_tag_labels);
        assert!(rose_quartz.has_pattern_tag_labels);

        let color_filters = catalog
            .material_filters
            .color_options
            .iter()
            .map(|option| (option.value.as_str(), option.label.as_str()))
            .collect::<Vec<_>>();
        assert!(color_filters.contains(&("pink", "Soft Pink")));
    }

    #[test]
    fn stone_listing_status_helper_requires_published() {
        assert!(stone_listing_is_published("published"));
        assert!(stone_listing_is_published(" Published "));
        assert!(!stone_listing_is_published("draft"));
    }

    #[test]
    fn stone_listing_catalog_visibility_requires_active_and_published() {
        assert!(stone_listing_is_catalog_visible(true, "published"));
        assert!(stone_listing_is_catalog_visible(true, " Published "));
        assert!(!stone_listing_is_catalog_visible(false, "published"));
        assert!(!stone_listing_is_catalog_visible(true, "draft"));
    }

    #[test]
    fn stone_listing_color_family_uses_scalar_facet_key() {
        let facets = BTreeMap::from([("color_family".to_owned(), json!(" Green "))]);
        assert_eq!(stone_listing_color_family_from_facets(&facets), "green");
    }

    #[test]
    fn stone_shape_helpers_preserve_unknown_values() {
        assert_eq!(normalize_stone_shape("freeform"), "freeform");
    }

    #[test]
    fn material_filter_state_preserves_color_and_pattern() {
        let filters = material_filter_state_from_query(&PaymentRedirectQuery {
            color_family: Some("Green".to_owned()),
            pattern_primary: Some("Cloud".to_owned()),
            ..PaymentRedirectQuery::default()
        });
        assert_eq!(filters.color_family, "green");
        assert_eq!(filters.pattern_primary, "cloud");
        assert_eq!(
            design_url_with_filters(TEST_SITE_BASE_URL, "ja", &filters),
            "https://finitefield.org/ja/design?color_family=green&pattern_primary=cloud"
        );
    }

    #[test]
    fn stone_listing_tag_labels_use_facet_tag_master() {
        let facet_tag_labels = {
            let mut labels = FacetTagLabels::default();
            labels.insert(
                "color",
                "deep_green",
                "濃緑",
                &["forest_green".to_owned(), "dark_green".to_owned()],
            );
            labels.insert("pattern", "cloud", "雲状", &["cloudy".to_owned()]);
            labels
        };

        let category = MaterialCategory {
            label: "翡翠".to_owned(),
        };
        let listing = StoneListingRecord {
            key: "listing-1".to_owned(),
            material_key: "jade".to_owned(),
            title: "翡翠の一点物".to_owned(),
            description: "個体説明".to_owned(),
            story: "作品紹介".to_owned(),
            price_by_currency: HashMap::from([("JPY".to_owned(), 150000)]),
            color_family: "green".to_owned(),
            pattern_primary: "cloud".to_owned(),
            color_tags: vec!["forest_green".to_owned()],
            pattern_tags: vec!["cloudy".to_owned(), "cloud".to_owned()],
            stone_shape: "square".to_owned(),
            photos: vec![],
        };

        let option = build_material_option_from_listing(
            &category,
            &listing,
            &facet_tag_labels,
            "ja",
            "ja",
            "bucket",
        );

        assert_eq!(option.color_tag_labels, vec!["濃緑"]);
        assert_eq!(option.pattern_tag_labels, vec!["雲状"]);
        assert_eq!(option.description, "個体説明");
        assert_eq!(option.story, "作品紹介");
        assert!(option.has_description);
        assert!(option.has_story);
        assert!(option.has_color_tag_labels);
        assert!(option.has_pattern_tag_labels);
    }

    #[test]
    fn material_option_uses_stone_shape_for_listing_shape() {
        let category = MaterialCategory {
            label: "翡翠".to_owned(),
        };
        let listing = StoneListingRecord {
            key: "listing-1".to_owned(),
            material_key: "jade".to_owned(),
            title: "翡翠の一点物".to_owned(),
            description: "個体説明".to_owned(),
            story: "作品紹介".to_owned(),
            price_by_currency: HashMap::from([("JPY".to_owned(), 150000)]),
            color_family: "green".to_owned(),
            pattern_primary: "cloud".to_owned(),
            color_tags: vec![],
            pattern_tags: vec![],
            stone_shape: "round".to_owned(),
            photos: vec![],
        };

        let option = build_material_option_from_listing(
            &category,
            &listing,
            &FacetTagLabels::default(),
            "ja",
            "ja",
            "bucket",
        );

        assert_eq!(option.shape_label, "丸印");
    }

    #[test]
    fn material_option_preserves_unknown_stone_shape() {
        let category = MaterialCategory {
            label: "翡翠".to_owned(),
        };
        let listing = StoneListingRecord {
            key: "listing-1".to_owned(),
            material_key: "jade".to_owned(),
            title: "翡翠の一点物".to_owned(),
            description: "個体説明".to_owned(),
            story: "作品紹介".to_owned(),
            price_by_currency: HashMap::from([("JPY".to_owned(), 150000)]),
            color_family: "green".to_owned(),
            pattern_primary: "cloud".to_owned(),
            color_tags: vec![],
            pattern_tags: vec![],
            stone_shape: "triangle".to_owned(),
            photos: vec![],
        };

        let option = build_material_option_from_listing(
            &category,
            &listing,
            &FacetTagLabels::default(),
            "ja",
            "ja",
            "bucket",
        );

        assert_eq!(option.shape, "triangle");
        assert_eq!(option.shape_label, "triangle");
    }

    #[test]
    fn material_option_uses_english_photo_alt_fallback() {
        let category = MaterialCategory {
            label: "翡翠".to_owned(),
        };
        let listing = StoneListingRecord {
            key: "listing-1".to_owned(),
            material_key: "jade".to_owned(),
            title: "One-of-a-kind Jade 101".to_owned(),
            description: "A refined piece with calm green flowing patterns.".to_owned(),
            story: "A quiet jade piece selected for its flowing green expression.".to_owned(),
            price_by_currency: HashMap::from([("USD".to_owned(), 92800)]),
            color_family: "green".to_owned(),
            pattern_primary: "cloud".to_owned(),
            color_tags: vec![],
            pattern_tags: vec![],
            stone_shape: "square".to_owned(),
            photos: vec![MaterialPhoto {
                asset_id: "mat_jade_01".to_owned(),
                storage_path: "materials/jade/mat_jade_01.webp".to_owned(),
                alt_i18n: HashMap::new(),
                sort_order: 0,
                is_primary: true,
            }],
        };

        let option = build_material_option_from_listing(
            &category,
            &listing,
            &FacetTagLabels::default(),
            "en",
            "ja",
            "bucket",
        );

        assert_eq!(option.photo_alt, "One-of-a-kind Jade 101 photo");
        assert!(option.has_photo);
    }

    #[test]
    fn material_photo_uses_firestore_photos() {
        let mut data = BTreeMap::new();
        data.insert(
            "photos".to_owned(),
            json!({
                "arrayValue": {
                    "values": [
                        {
                            "mapValue": {
                                "fields": {
                                    "asset_id": { "stringValue": "mat_rose_quartz_01" },
                                    "storage_path": {
                                        "stringValue": "materials/rose_quartz/mat_rose_quartz_01.webp"
                                    },
                                    "alt_i18n": {
                                        "mapValue": {
                                            "fields": {
                                                "ja": { "stringValue": "ローズクオーツの出品個体サンプル" }
                                            }
                                        }
                                    },
                                    "sort_order": { "integerValue": "0" },
                                    "is_primary": { "booleanValue": true },
                                    "width": { "integerValue": "1200" },
                                    "height": { "integerValue": "1200" }
                                }
                            }
                        }
                    ]
                }
            }),
        );

        let (photo_url, photo_alt, has_photo) =
            resolve_material_photo(&data, "hanko-field-prod", "ja", "ja");

        assert_eq!(
            photo_url,
            "https://storage.googleapis.com/hanko-field-prod/materials/rose_quartz/mat_rose_quartz_01.webp"
        );
        assert_eq!(photo_alt, "ローズクオーツの出品個体サンプル");
        assert!(has_photo);
    }

    #[tokio::test]
    async fn mock_purchase_returns_confirmation_without_api() {
        let response =
            handle_purchase_impl(mock_state(), Ok(Form(valid_purchase_form())), false).await;

        assert_eq!(response.status(), StatusCode::OK);

        let body = to_bytes(response.into_body(), usize::MAX)
            .await
            .expect("response body should be readable");
        let html = String::from_utf8(body.to_vec()).expect("response body should be utf-8");

        assert!(html.contains("注文を受け付けました（モック）"));
    }

    #[tokio::test]
    async fn unprefixed_stone_signature_pages_render_english() {
        let top_response =
            handle_top(State(mock_state()), Query(PaymentRedirectQuery::default())).await;
        assert_eq!(top_response.status(), StatusCode::OK);
        let top_html = String::from_utf8(
            to_bytes(top_response.into_body(), usize::MAX)
                .await
                .expect("top body should be readable")
                .to_vec(),
        )
        .expect("top body should be utf-8");

        assert!(top_html.contains(r#"<html lang="en" dir="ltr">"#));
        assert!(top_html.contains(r#"<span class="top-brand__subtitle">Seal Field</span>"#));
        assert!(top_html.contains("A gemstone seal made just for you."));
        assert!(!top_html.contains("あなただけの宝石印鑑"));

        let about_response = handle_about(State(mock_state()), Query(LocaleQuery::default())).await;
        assert_eq!(about_response.status(), StatusCode::OK);
        let about_html = String::from_utf8(
            to_bytes(about_response.into_body(), usize::MAX)
                .await
                .expect("about body should be readable")
                .to_vec(),
        )
        .expect("about body should be utf-8");

        assert!(about_html.contains(r#"<html lang="en" dir="ltr">"#));
        assert!(about_html.contains(r#"<span class="top-brand__subtitle">Seal Field</span>"#));
        assert!(about_html.contains("Your seal, made from gemstone"));
        assert!(
            about_html.contains("STONE SIGNATURE is a service for choosing a gemstone seal online")
        );
        assert!(!about_html.contains("宝石でつくる、あなたの印鑑"));
    }

    #[tokio::test]
    async fn web_router_resolves_supported_and_unknown_locale_prefixes() {
        let (about_status, about_html) = route_get_html("/about").await;
        assert_eq!(about_status, StatusCode::OK);
        assert!(about_html.contains(r#"<html lang="en" dir="ltr">"#));
        assert!(
            about_html.contains(r#"<link rel="canonical" href="https://finitefield.org/about">"#)
        );
        assert!(about_html.contains("Your seal, made from gemstone"));

        let (ja_about_status, ja_about_html) = route_get_html("/ja/about").await;
        assert_eq!(ja_about_status, StatusCode::OK);
        assert!(ja_about_html.contains(r#"<html lang="ja" dir="ltr">"#));
        assert!(
            ja_about_html
                .contains(r#"<link rel="canonical" href="https://finitefield.org/ja/about">"#)
        );
        assert!(ja_about_html.contains("宝石でつくる、あなたの印鑑"));

        let (en_about_status, en_about_html) = route_get_html("/en/about").await;
        assert_eq!(en_about_status, StatusCode::OK);
        assert!(en_about_html.contains(r#"<html lang="en" dir="ltr">"#));
        assert!(
            en_about_html
                .contains(r#"<link rel="canonical" href="https://finitefield.org/about">"#),
            "/en/about must remain non-canonical English compatibility"
        );

        for (path, html_lang, html_dir, expected_title, expected_seo_title, expected_meta) in [
            (
                "/zh/about",
                "zh",
                "ltr",
                "用宝石制作你的印章",
                "关于 STONE SIGNATURE | STONE SIGNATURE",
                "了解 STONE SIGNATURE 如何让你在线选择宝石印章、设计印面并完成下单。",
            ),
            (
                "/zhtw/about",
                "zhtw",
                "ltr",
                "用寶石製作你的印章",
                "關於 STONE SIGNATURE | STONE SIGNATURE",
                "了解 STONE SIGNATURE 如何讓你線上選擇寶石印章、設計印面並完成下單。",
            ),
            (
                "/ar/about",
                "ar",
                "rtl",
                "ختمك مصنوع من حجر كريم",
                "عن STONE SIGNATURE | STONE SIGNATURE",
                "تعرف على طريقة اختيار ختم من حجر كريم عبر STONE SIGNATURE، وتصميم طبعة الختم، وإرسال طلبك.",
            ),
        ] {
            let (status, body) = route_get_html(path).await;
            assert_eq!(status, StatusCode::OK, "{path} should render for QA");
            assert!(body.contains(&format!(r#"<html lang="{html_lang}" dir="{html_dir}">"#)));
            assert!(
                body.contains(expected_title),
                "{path} must render pilot locale page content"
            );
            assert!(
                body.contains(&format!("<title>{expected_seo_title}</title>")),
                "{path} must render pilot locale SEO title"
            );
            assert!(
                body.contains(&format!(
                    r#"<meta name="description" content="{expected_meta}">"#
                )),
                "{path} must render pilot locale meta description"
            );
            assert!(
                body.contains(r#"<meta name="robots" content="noindex,follow">"#),
                "{path} must remain non-indexed while in pilot"
            );
            assert!(
                body.contains(r#"<link rel="canonical" href="https://finitefield.org/about">"#),
                "{path} should canonicalize to the indexed default until public launch"
            );
        }

        for path in ["/xx/about"] {
            let (status, body) = route_get_html(path).await;
            assert_eq!(status, StatusCode::NOT_FOUND, "{path} should 404");
            assert!(
                !body.contains("Your seal, made from gemstone"),
                "{path} must not fall back to English content"
            );
        }
    }

    #[test]
    fn top_page_uses_locale_aware_privacy_policy_url() {
        let template = TopPageTemplate {
            selected_locale: "en".to_owned(),
            page_title: "Custom gemstone seals | STONE SIGNATURE".to_owned(),
            meta_description:
                "Design custom hand-carved gemstone seals online and order in English or Japanese."
                    .to_owned(),
            robots_meta: "index,follow".to_owned(),
            canonical_url: top_url(TEST_SITE_BASE_URL, "en"),
            x_default_url: top_url(TEST_SITE_BASE_URL, "en"),
            seo_language_links: indexed_hreflang_links_for_path(TEST_SITE_BASE_URL, "/"),
            language_links: language_links_for_path(TEST_SITE_BASE_URL, "/"),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "en"),
            about_url: about_url(TEST_SITE_BASE_URL, "en"),
            design_url: design_url(TEST_SITE_BASE_URL, "en"),
            blog_index_url: blog_index_url(TEST_SITE_BASE_URL, "en"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "en"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "en"),
            blog_posts: blog_post_cards(
                &load_blog_posts().expect("blog posts should load"),
                TEST_SITE_BASE_URL,
                "en",
            ),
        };

        let html = render_html(&template).expect("top page should render");

        assert!(html.contains(r#"<link rel="canonical" href="https://finitefield.org/">"#));
        assert!(html.contains(r#"<title>Custom gemstone seals | STONE SIGNATURE</title>"#));
        assert!(html.contains(
            r#"<meta name="description" content="Design custom hand-carved gemstone seals online and order in English or Japanese.">"#
        ));
        assert!(html.contains(r#"<meta name="robots" content="index,follow">"#));
        assert!(html.contains(
            r#"<link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/">"#
        ));
        assert!(
            html.contains(
                r#"<link rel="alternate" hreflang="en" href="https://finitefield.org/">"#
            )
        );
        assert!(html.contains(
            r#"<link rel="alternate" hreflang="x-default" href="https://finitefield.org/">"#
        ));
        assert!(html.contains("href=\"https://finitefield.org/en/privacy/\""));
        assert!(html.contains("href=\"https://finitefield.org/company/\""));
    }

    #[test]
    fn language_switcher_renders_registry_language_links() {
        let registry = WebLanguageRegistry::from_entries(vec![
            language_registry_entry(
                "en",
                "en",
                "English",
                "English",
                RegistryTextDirection::Ltr,
                true,
                true,
                "",
            ),
            language_registry_entry(
                "fr",
                "fr",
                "Français",
                "French",
                RegistryTextDirection::Ltr,
                true,
                false,
                "fr",
            ),
            language_registry_entry(
                "ja",
                "ja",
                "日本語",
                "Japanese",
                RegistryTextDirection::Ltr,
                true,
                true,
                "ja",
            ),
        ])
        .expect("fixture registry should load");
        let template = TopPageTemplate {
            selected_locale: "fr".to_owned(),
            page_title: "Custom gemstone seals | STONE SIGNATURE".to_owned(),
            meta_description:
                "Design custom hand-carved gemstone seals online and order in English or Japanese."
                    .to_owned(),
            robots_meta: "index,follow".to_owned(),
            canonical_url: top_url(TEST_SITE_BASE_URL, "en"),
            x_default_url: top_url(TEST_SITE_BASE_URL, "en"),
            seo_language_links: indexed_hreflang_links_for_path_with_registry(
                &registry,
                TEST_SITE_BASE_URL,
                "/",
            ),
            language_links: language_links_for_path_with_registry(
                &registry,
                TEST_SITE_BASE_URL,
                "/",
            ),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "en"),
            about_url: about_url(TEST_SITE_BASE_URL, "en"),
            design_url: design_url(TEST_SITE_BASE_URL, "en"),
            blog_index_url: blog_index_url(TEST_SITE_BASE_URL, "en"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "en"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "en"),
            blog_posts: Vec::new(),
        };

        let html = render_html(&template).expect("top page should render");

        assert_eq!(html.matches("data-language-option=").count(), 3);
        assert!(!html.contains(
            r#"<link rel="alternate" hreflang="fr" href="https://finitefield.org/fr/">"#
        ));
        assert!(html.contains(
            r#"class="top-language-switcher__item is-active" href="https://finitefield.org/fr/" hreflang="fr" data-language-option="fr">Français</a>"#
        ));
    }

    #[test]
    fn top_page_renders_journal_cards() {
        let template = TopPageTemplate {
            selected_locale: "en".to_owned(),
            page_title: "Custom gemstone seals | STONE SIGNATURE".to_owned(),
            meta_description:
                "Design custom hand-carved gemstone seals online and order in English or Japanese."
                    .to_owned(),
            robots_meta: "index,follow".to_owned(),
            canonical_url: top_url(TEST_SITE_BASE_URL, "en"),
            x_default_url: top_url(TEST_SITE_BASE_URL, "en"),
            seo_language_links: indexed_hreflang_links_for_path(TEST_SITE_BASE_URL, "/"),
            language_links: language_links_for_path(TEST_SITE_BASE_URL, "/"),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "en"),
            about_url: about_url(TEST_SITE_BASE_URL, "en"),
            design_url: design_url(TEST_SITE_BASE_URL, "en"),
            blog_index_url: blog_index_url(TEST_SITE_BASE_URL, "en"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "en"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "en"),
            blog_posts: blog_post_cards(
                &load_blog_posts().expect("blog posts should load"),
                TEST_SITE_BASE_URL,
                "en",
            ),
        };

        let html = render_html(&template).expect("top page should render");

        assert!(html.contains(r#"<section id="journal" class="journal-section">"#));
        assert!(html.contains(r#"<div class="journal-grid">"#));
        assert!(html.contains("What Is a Hanko? A Complete Guide to Japanese Personal Seals"));
        assert!(html.contains("Hanko vs Inkan: What&#39;s the Difference?"));
        assert!(html.contains("What Is a Personal Seal? History, Meaning, and Modern Uses"));
        assert!(html.contains("Why a Custom Stone Seal Makes a Meaningful Gift"));
        assert!(html.contains("Custom Jade Seal: Meaning, Materials, and How to Choose One"));
        assert!(html.contains("Japanese Hanko as a Souvenir: A Personal Piece of Japan"));
        assert!(html.contains("How to Choose the Right Stone for Your Personal Seal"));
        assert!(html.contains("Jade, Agate, or Qingtian Stone: Which Seal Material Is Best?"));
        assert!(html.contains("A Personal Seal as a Symbol of Identity"));
        assert!(html.contains("Luxury Personal Seals: A New Way to Express Your Signature"));
        assert!(html.contains("How to Turn Your English Name into a Japanese or Chinese Seal"));
        assert!(html.contains("What to Engrave on a Custom Personal Seal"));
        assert!(html.contains("Personal Seals for Artists, Writers, and Creators"));
        assert!(html.contains("Chinese Chop Seal vs Japanese Hanko: Similarities and Differences"));
        assert!(html.contains("The Beauty of One-of-a-Kind Stone Seals"));
        assert!(html.contains(r#"href="https://finitefield.org/blog/what-is-a-hanko""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/hanko-vs-inkan""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/what-is-a-personal-seal""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/custom-stone-seal-gift""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/custom-jade-seal""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/japanese-hanko-souvenir""#));
        assert!(html.contains(r#"href="https://finitefield.org/blog/how-to-choose-stone-seal""#));
        assert!(
            html.contains(r#"href="https://finitefield.org/blog/jade-agate-qingtian-stone-seal""#)
        );
        assert!(
            html.contains(
                r#"href="https://finitefield.org/blog/personal-seal-symbol-of-identity""#
            )
        );
        assert!(html.contains(r#"href="https://finitefield.org/blog/luxury-personal-seal""#));
        assert!(html.contains(&format!(
            r#"href="{}""#,
            blog_article_url(TEST_SITE_BASE_URL, "english-name-kanji-seal", "en")
        )));
        assert!(html.contains(&format!(
            r#"href="{}""#,
            blog_article_url(TEST_SITE_BASE_URL, "what-to-engrave-on-seal", "en")
        )));
        assert!(html.contains(&format!(
            r#"href="{}""#,
            blog_article_url(TEST_SITE_BASE_URL, "personal-seals-for-artists", "en")
        )));
        assert!(html.contains(&format!(
            r#"href="{}""#,
            blog_article_url(
                TEST_SITE_BASE_URL,
                "chinese-chop-seal-vs-japanese-hanko",
                "en"
            )
        )));
        assert!(html.contains(&format!(
            r#"href="{}""#,
            blog_article_url(TEST_SITE_BASE_URL, "one-of-a-kind-stone-seal", "en")
        )));
        assert!(!html.contains("The Art of Selection: Identifying the Finest Jadeite for Carving"));
        assert!(!html.contains(r#"href="https://finitefield.org/blog/art-of-selection-jadeite""#));
        assert!(!html.contains("journal-card__category"));
        assert!(!html.contains("journal-card__date"));
        assert!(!html.contains("journal-card__author"));
    }

    #[test]
    fn top_page_uses_logo_image_left_of_title() {
        let template = TopPageTemplate {
            selected_locale: "ja".to_owned(),
            page_title: "宝石印鑑をオンラインでデザイン | STONE SIGNATURE".to_owned(),
            meta_description:
                "宝石印鑑をオンラインでデザインして、日本語または英語で注文できます。".to_owned(),
            robots_meta: "index,follow".to_owned(),
            canonical_url: top_url(TEST_SITE_BASE_URL, "en"),
            x_default_url: top_url(TEST_SITE_BASE_URL, "en"),
            seo_language_links: indexed_hreflang_links_for_path(TEST_SITE_BASE_URL, "/"),
            language_links: language_links_for_path(TEST_SITE_BASE_URL, "/"),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "ja"),
            about_url: about_url(TEST_SITE_BASE_URL, "ja"),
            design_url: design_url(TEST_SITE_BASE_URL, "ja"),
            blog_index_url: blog_index_url(TEST_SITE_BASE_URL, "ja"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "ja"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "ja"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "ja"),
            blog_posts: blog_post_cards(
                &load_blog_posts().expect("blog posts should load"),
                TEST_SITE_BASE_URL,
                "ja",
            ),
        };

        let html = render_html(&template).expect("top page should render");

        assert!(html.contains(r#"<link rel="icon" type="image/png" href="/static/favicon.png">"#));

        let header_logo = html
            .find(r#"<img class="top-brand__logo" src="/static/site-logo.png" alt="" aria-hidden="true">"#)
            .expect("header logo should be rendered");
        let header_title = html
            .find(r#"<h1 class="top-brand__title">STONE SIGNATURE</h1>"#)
            .expect("header title should be rendered");
        assert!(header_logo < header_title);

        let footer_logo = html
            .find(r#"<img class="top-footer__brand-logo" src="/static/site-logo.png" alt="" aria-hidden="true">"#)
            .expect("footer logo should be rendered");
        let footer_title = html
            .find(r#"<div class="top-footer__brand-title">STONE SIGNATURE</div>"#)
            .expect("footer title should be rendered");
        assert!(footer_logo < footer_title);
    }

    #[tokio::test]
    async fn blog_article_page_renders_for_known_slug() {
        let response = handle_blog_article(
            State(mock_state()),
            Path("hanko-vs-inkan".to_owned()),
            Query(LocaleQuery {
                lang: Some("en".to_owned()),
            }),
        )
        .await;

        assert_eq!(response.status(), StatusCode::OK);

        let html = String::from_utf8(
            to_bytes(response.into_body(), usize::MAX)
                .await
                .expect("blog body should be readable")
                .to_vec(),
        )
        .expect("blog body should be utf-8");

        assert!(html.contains("Hanko vs Inkan: What&#39;s the Difference?"));
        assert!(html.contains(
            r#"<link rel="canonical" href="https://finitefield.org/blog/hanko-vs-inkan">"#
        ));
        assert!(!html.contains("slug = \"hanko-vs-inkan\""));
    }

    #[tokio::test]
    async fn hanko_guide_article_uses_optimized_localized_metadata() {
        let english_response = handle_blog_article(
            State(mock_state()),
            Path("what-is-a-hanko".to_owned()),
            Query(LocaleQuery::default()),
        )
        .await;
        assert_eq!(english_response.status(), StatusCode::OK);

        let english_html = String::from_utf8(
            to_bytes(english_response.into_body(), usize::MAX)
                .await
                .expect("blog body should be readable")
                .to_vec(),
        )
        .expect("blog body should be utf-8");

        assert!(english_html.contains(
            r#"<title>What Is a Hanko? A Complete Guide to Japanese Personal Seals | STONE SIGNATURE</title>"#
        ));
        assert!(english_html.contains(
            r#"<meta name="description" content="What is a hanko? Learn how Japanese personal seals differ from inkan, how hanko stamps are used, and why custom stone seals make meaningful gifts.">"#
        ));
        assert!(english_html.contains(
            r#"<meta property="og:image" content="https://finitefield.org/static/blog/what-is-a-hanko.svg">"#
        ));
        assert!(
            english_html
                .contains(r#"<meta property="article:published_time" content="2026-05-07">"#)
        );
        assert!(!english_html.contains("meta_description ="));

        let japanese_response = handle_localized_blog_article(
            Path(("ja".to_owned(), "what-is-a-hanko".to_owned())),
            State(mock_state()),
            Query(LocaleQuery::default()),
        )
        .await;
        assert_eq!(japanese_response.status(), StatusCode::OK);

        let japanese_html = String::from_utf8(
            to_bytes(japanese_response.into_body(), usize::MAX)
                .await
                .expect("blog body should be readable")
                .to_vec(),
        )
        .expect("blog body should be utf-8");

        assert!(japanese_html.contains(
            r#"<title>ハンコとは？日本のパーソナルシール完全ガイド | STONE SIGNATURE</title>"#
        ));
        assert!(japanese_html.contains(
            r#"<meta name="description" content="ハンコとは何かを解説。印鑑との違い、実印・銀行印・認印の用途、海外での使い方、天然石ハンコの魅力を紹介します。">"#
        ));
        assert!(japanese_html.contains(
            r#"<meta property="og:image:alt" content="手漉き紙の赤い印影と朱肉のそばに置かれた石のハンコ。">"#
        ));
    }

    #[tokio::test]
    async fn added_blog_articles_use_localized_seo_tags() {
        let posts = load_blog_posts().expect("blog posts should load");
        for slug in ADDED_BLOG_ARTICLE_SLUGS {
            let post = find_blog_post(&posts, slug).expect("added blog article should load");
            let en_url = blog_article_url(TEST_SITE_BASE_URL, slug, "en");
            let ja_url = blog_article_url(TEST_SITE_BASE_URL, slug, "ja");

            let english_html = render_blog_article_html_for_test(slug, "en").await;
            assert!(english_html.contains(&format!(
                r#"<meta name="description" content="{}">"#,
                post.meta_description
            )));
            assert!(english_html.contains(&format!(r#"<link rel="canonical" href="{en_url}">"#)));
            assert!(english_html.contains(&format!(
                r#"<link rel="alternate" hreflang="ja" href="{ja_url}">"#
            )));
            assert!(english_html.contains(&format!(
                r#"<link rel="alternate" hreflang="en" href="{en_url}">"#
            )));
            assert!(english_html.contains(&format!(
                r#"<link rel="alternate" hreflang="x-default" href="{en_url}">"#
            )));
            assert!(
                english_html.contains(&format!(r#"<meta property="og:url" content="{en_url}">"#))
            );
            assert!(english_html.contains(&format!(r#"href="{ja_url}" hreflang="ja""#)));

            let japanese_html = render_blog_article_html_for_test(slug, "ja").await;
            assert!(japanese_html.contains(&format!(
                r#"<meta name="description" content="{}">"#,
                post.meta_description_ja
            )));
            assert!(japanese_html.contains(&format!(r#"<link rel="canonical" href="{ja_url}">"#)));
            assert!(japanese_html.contains(&format!(
                r#"<link rel="alternate" hreflang="ja" href="{ja_url}">"#
            )));
            assert!(japanese_html.contains(&format!(
                r#"<link rel="alternate" hreflang="en" href="{en_url}">"#
            )));
            assert!(japanese_html.contains(&format!(
                r#"<link rel="alternate" hreflang="x-default" href="{en_url}">"#
            )));
            assert!(
                japanese_html.contains(&format!(r#"<meta property="og:url" content="{ja_url}">"#))
            );
            assert!(japanese_html.contains(&format!(r#"href="{en_url}" hreflang="en""#)));
        }
    }

    #[tokio::test]
    async fn localized_blog_pages_render_japanese_copy() {
        let index_response = handle_localized_blog_index(
            Path("ja".to_owned()),
            State(mock_state()),
            Query(LocaleQuery::default()),
        )
        .await;
        assert_eq!(index_response.status(), StatusCode::OK);

        let index_html = String::from_utf8(
            to_bytes(index_response.into_body(), usize::MAX)
                .await
                .expect("blog index body should be readable")
                .to_vec(),
        )
        .expect("blog index body should be utf-8");

        assert!(index_html.contains(r#"<html lang="ja" dir="ltr">"#));
        assert!(index_html.contains("<title>ジャーナル | STONE SIGNATURE</title>"));
        assert!(index_html.contains("ハンコとは？日本のパーソナルシール完全ガイド"));
        assert!(index_html.contains("ハンコと印鑑の違いとは？"));
        assert!(index_html.contains("パーソナルシールとは？歴史・意味・現代の使い方"));
        assert!(index_html.contains("カスタム石印が心に残るギフトになる理由"));
        assert!(index_html.contains("カスタム翡翠印とは？意味・素材・選び方"));
        assert!(index_html.contains("日本のハンコをお土産に: 日本を個人的に感じる一品"));
        assert!(index_html.contains("パーソナルシールに合う石の選び方"));
        assert!(index_html.contains("翡翠・瑪瑙・青田石: 印材はどれがよい？"));
        assert!(index_html.contains("本人性の象徴としてのパーソナルシール"));
        assert!(index_html.contains("ラグジュアリーパーソナルシール: 署名を表現する新しい方法"));
        assert!(index_html.contains("英語の名前を日本語・中国語の印にする方法"));
        assert!(index_html.contains("カスタムパーソナルシールには何を彫刻するべき？"));
        assert!(index_html.contains("アーティスト・作家・クリエイターのためのパーソナルシール"));
        assert!(index_html.contains("中国のチョップシールと日本のハンコ: 共通点と違い"));
        assert!(index_html.contains("一点ものの天然石印の美しさ"));
        assert!(
            !index_html
                .contains("The Art of Selection: Identifying the Finest Jadeite for Carving")
        );

        let article_response = handle_localized_blog_article(
            Path(("ja".to_owned(), "hanko-vs-inkan".to_owned())),
            State(mock_state()),
            Query(LocaleQuery::default()),
        )
        .await;
        assert_eq!(article_response.status(), StatusCode::OK);

        let article_html = String::from_utf8(
            to_bytes(article_response.into_body(), usize::MAX)
                .await
                .expect("blog article body should be readable")
                .to_vec(),
        )
        .expect("blog article body should be utf-8");

        assert!(article_html.contains("ハンコと印鑑の違いとは？"));
        assert!(article_html.contains("2026年5月7日"));
        assert!(article_html.contains("ジャーナル一覧へ"));
        assert!(article_html.contains("Hanko vs Inkan: シンプルな違い"));
        assert!(article_html.contains("ハンコはスタンプそのもの。印鑑は印影"));
        assert!(article_html.contains(
            r#"<link rel="canonical" href="https://finitefield.org/ja/blog/hanko-vs-inkan">"#
        ));
        assert!(article_html.contains(
            r#"<link rel="alternate" hreflang="x-default" href="https://finitefield.org/blog/hanko-vs-inkan">"#
        ));
        assert!(!article_html.contains("Hanko vs Inkan: The Simple Difference"));
    }

    #[tokio::test]
    async fn unsupported_localized_page_prefix_returns_not_found() {
        let response = handle_localized_about(
            Path("fr".to_owned()),
            State(mock_state()),
            Query(LocaleQuery::default()),
        )
        .await;

        assert_eq!(response.status(), StatusCode::NOT_FOUND);
    }

    #[tokio::test]
    async fn blog_article_page_returns_not_found_for_unknown_slug() {
        let response = handle_blog_article(
            State(mock_state()),
            Path("missing".to_owned()),
            Query(LocaleQuery::default()),
        )
        .await;

        assert_eq!(response.status(), StatusCode::NOT_FOUND);
    }

    #[tokio::test]
    async fn about_page_renders_localized_copy_and_footer_navigation() {
        let english_response = handle_about(
            State(mock_state()),
            Query(LocaleQuery {
                lang: Some("en".to_owned()),
            }),
        )
        .await;
        assert_eq!(english_response.status(), StatusCode::OK);
        let english_html = String::from_utf8(
            to_bytes(english_response.into_body(), usize::MAX)
                .await
                .expect("about body should be readable")
                .to_vec(),
        )
        .expect("about body should be utf-8");

        assert!(english_html.contains(r#"<title>About STONE SIGNATURE | STONE SIGNATURE</title>"#));
        assert!(english_html.contains("Your seal, made from gemstone"));
        assert!(
            english_html
                .contains("Choose a stone, design the seal impression, and place your order")
        );
        assert!(english_html.contains(r#"<span class="top-cta__label">Design</span>"#));
        assert!(english_html.contains("An easier way to choose a gemstone seal."));
        assert!(english_html.contains("choosing a gemstone seal online"));
        assert!(english_html.contains("Gemstone"));
        assert!(english_html.contains("Seal design"));
        assert!(english_html.contains("One of a kind"));
        assert!(
            english_html.contains(r#"<link rel="canonical" href="https://finitefield.org/about">"#)
        );
        assert!(english_html.contains(
            r#"<link rel="alternate" hreflang="x-default" href="https://finitefield.org/about">"#
        ));
        assert!(english_html.contains(r#"href="https://finitefield.org/about""#));
        assert!(english_html.contains("window.location.href='https://finitefield.org/design'"));

        let japanese_response = handle_localized_about(
            Path("ja".to_owned()),
            State(mock_state()),
            Query(LocaleQuery::default()),
        )
        .await;
        assert_eq!(japanese_response.status(), StatusCode::OK);
        let japanese_html = String::from_utf8(
            to_bytes(japanese_response.into_body(), usize::MAX)
                .await
                .expect("about body should be readable")
                .to_vec(),
        )
        .expect("about body should be utf-8");

        assert!(japanese_html.contains("STONE SIGNATUREとは"));
        assert!(japanese_html.contains("宝石でつくる、あなたの印鑑"));
        assert!(japanese_html.contains("石を選び、印影をデザインして注文できます"));
        assert!(japanese_html.contains(r#"<span class="top-cta__label">デザインする</span>"#));
        assert!(japanese_html.contains("宝石印鑑を、もっと選びやすく。"));
        assert!(japanese_html.contains("宝石を使った印鑑をオンラインで選び"));
        assert!(japanese_html.contains("天然石ならではの色や模様"));
        assert!(
            japanese_html
                .contains(r#"<link rel="canonical" href="https://finitefield.org/ja/about">"#)
        );
        assert!(japanese_html.contains(
            r#"<link rel="alternate" hreflang="x-default" href="https://finitefield.org/about">"#
        ));
        assert!(japanese_html.contains(r#"href="https://finitefield.org/ja/about""#));
        assert!(japanese_html.contains("window.location.href='https://finitefield.org/ja/design'"));
    }

    #[test]
    fn seo_metadata_marks_payment_pages_noindex() {
        let success_template = PaymentSuccessTemplate {
            has_session_id: true,
            session_id: "sess_123".to_owned(),
            has_order_id: true,
            order_id: "ord_456".to_owned(),
            has_app_redirect_url: false,
            app_redirect_url: String::new(),
            selected_locale: "en".to_owned(),
            page_title: "Payment complete | STONE SIGNATURE".to_owned(),
            meta_description: "Your payment was received. Check your Stripe payment receipt for order details and next steps.".to_owned(),
            robots_meta: "noindex,follow".to_owned(),
            canonical_url: payment_result_locale_url(
                TEST_SITE_BASE_URL,
                "/payment/success",
                &PaymentRedirectQuery::default(),
                "en",
            ),
            x_default_url: payment_result_locale_url(
                TEST_SITE_BASE_URL,
                "/payment/success",
                &PaymentRedirectQuery::default(),
                "en",
            ),
            seo_language_links: seo_language_links_with_urls(|language| {
                payment_result_locale_url(
                    TEST_SITE_BASE_URL,
                    "/payment/success",
                    &PaymentRedirectQuery::default(),
                    &language.route_code,
                )
            }),
            language_links: language_links_with_urls(|language| {
                payment_result_locale_url(
                    TEST_SITE_BASE_URL,
                    "/payment/success",
                    &PaymentRedirectQuery::default(),
                    &language.route_code,
                )
            }),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "en"),
            about_url: about_url(TEST_SITE_BASE_URL, "en"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "en"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            contact_url: inquiry_url(TEST_SITE_BASE_URL, "en"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "en"),
        };

        let success_html = render_html(&success_template).expect("payment success should render");
        assert!(success_html.contains(r#"<title>Payment complete | STONE SIGNATURE</title>"#));
        assert!(success_html.contains(r#"<meta name="robots" content="noindex,follow">"#));

        let failure_template = PaymentFailureTemplate {
            has_order_id: true,
            order_id: "ord_456".to_owned(),
            has_app_redirect_url: false,
            app_redirect_url: String::new(),
            selected_locale: "en".to_owned(),
            page_title: "Payment incomplete | STONE SIGNATURE".to_owned(),
            meta_description: "Payment did not complete. Check your card details and return to the purchase page to try again.".to_owned(),
            robots_meta: "noindex,follow".to_owned(),
            canonical_url: payment_result_locale_url(
                TEST_SITE_BASE_URL,
                "/payment/failure",
                &PaymentRedirectQuery::default(),
                "en",
            ),
            x_default_url: payment_result_locale_url(
                TEST_SITE_BASE_URL,
                "/payment/failure",
                &PaymentRedirectQuery::default(),
                "en",
            ),
            seo_language_links: seo_language_links_with_urls(|language| {
                payment_result_locale_url(
                    TEST_SITE_BASE_URL,
                    "/payment/failure",
                    &PaymentRedirectQuery::default(),
                    &language.route_code,
                )
            }),
            language_links: language_links_with_urls(|language| {
                payment_result_locale_url(
                    TEST_SITE_BASE_URL,
                    "/payment/failure",
                    &PaymentRedirectQuery::default(),
                    &language.route_code,
                )
            }),
            company_url: company_url(TEST_SITE_BASE_URL),
            top_url: top_url(TEST_SITE_BASE_URL, "en"),
            about_url: about_url(TEST_SITE_BASE_URL, "en"),
            design_url: design_url(TEST_SITE_BASE_URL, "en"),
            terms_url: terms_url(TEST_SITE_BASE_URL, "en"),
            commercial_transactions_url: commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            contact_url: inquiry_url(TEST_SITE_BASE_URL, "en"),
            privacy_policy_url: privacy_policy_url(TEST_SITE_BASE_URL, "en"),
        };

        let failure_html = render_html(&failure_template).expect("payment failure should render");
        assert!(failure_html.contains(r#"<title>Payment incomplete | STONE SIGNATURE</title>"#));
        assert!(failure_html.contains(r#"<meta name="robots" content="noindex,follow">"#));
    }

    #[test]
    fn web_language_registry_loads_checked_in_route_model() {
        let entries: Vec<LanguageRegistryEntry> =
            serde_json::from_str(LANGUAGE_REGISTRY_JSON).expect("registry should parse");
        assert_eq!(entries.len(), 68);

        let registry =
            WebLanguageRegistry::from_json(LANGUAGE_REGISTRY_JSON).expect("registry should load");
        assert_eq!(
            registry
                .enabled_languages()
                .iter()
                .map(|language| language.route_code.as_str())
                .collect::<Vec<_>>(),
            vec!["ar", "en", "ja", "zh", "zhtw"]
        );

        let english = registry
            .enabled_language_exact("en")
            .expect("English should be enabled for web");
        assert_eq!(english.bcp47, "en");
        assert_eq!(english.url_prefix, "");
        assert_eq!(english.text_direction, RegistryTextDirection::Ltr);

        let japanese = registry
            .enabled_language_for_path_segment("jp")
            .expect("legacy jp path segment should resolve to Japanese");
        assert_eq!(japanese.route_code, "ja");
        assert_eq!(japanese.url_prefix, "ja");
        assert_eq!(japanese.english_name, "Japanese");

        assert_eq!(parse_supported_locale("ja-JP"), Some("ja"));
        assert_eq!(parse_path_locale("ja"), Some("ja"));
        assert_eq!(parse_path_locale("zhtw"), Some("zhtw"));
        assert_eq!(parse_path_locale("ar"), Some("ar"));
        assert_eq!(robots_meta_for_locale("ar"), "noindex,follow");
        assert_eq!(robots_meta_for_locale("en"), "index,follow");

        let links = language_links_for_path(TEST_SITE_BASE_URL, "/about");
        assert_eq!(
            links
                .iter()
                .map(|link| (
                    link.route_code.as_str(),
                    link.label.as_str(),
                    link.is_default
                ))
                .collect::<Vec<_>>(),
            vec![
                ("ar", "العربية", false),
                ("en", "English", true),
                ("ja", "日本語", false),
                ("zh", "简体中文", false),
                ("zhtw", "繁體中文", false),
            ]
        );
    }

    #[test]
    fn registry_backed_links_include_non_indexed_enabled_languages() {
        let registry = WebLanguageRegistry::from_entries(vec![
            language_registry_entry(
                "en",
                "en",
                "English",
                "English",
                RegistryTextDirection::Ltr,
                true,
                true,
                "",
            ),
            language_registry_entry(
                "fr",
                "fr",
                "Français",
                "French",
                RegistryTextDirection::Ltr,
                true,
                false,
                "fr",
            ),
            language_registry_entry(
                "ja",
                "ja",
                "日本語",
                "Japanese",
                RegistryTextDirection::Ltr,
                true,
                true,
                "ja",
            ),
        ])
        .expect("fixture registry should load");

        let links = language_links_for_path_with_registry(&registry, TEST_SITE_BASE_URL, "/about");
        assert_eq!(
            links
                .iter()
                .map(|link| (link.route_code.as_str(), link.url.as_str(), link.is_indexed))
                .collect::<Vec<_>>(),
            vec![
                ("en", "https://finitefield.org/about", true),
                ("fr", "https://finitefield.org/fr/about", false),
                ("ja", "https://finitefield.org/ja/about", true),
            ]
        );

        let indexed_links =
            indexed_hreflang_links_for_path_with_registry(&registry, TEST_SITE_BASE_URL, "/about");
        assert_eq!(
            indexed_links
                .iter()
                .map(|link| (link.bcp47.as_str(), link.url.as_str()))
                .collect::<Vec<_>>(),
            vec![
                ("en", "https://finitefield.org/about"),
                ("ja", "https://finitefield.org/ja/about"),
            ]
        );

        assert_eq!(
            canonical_url_for_path_with_registry(&registry, TEST_SITE_BASE_URL, "/about", "fr"),
            "https://finitefield.org/about"
        );
        assert_eq!(
            canonical_url_for_path_with_registry(&registry, TEST_SITE_BASE_URL, "/about", "ja"),
            "https://finitefield.org/ja/about"
        );

        let sitemap_entry =
            sitemap_url_entry_with_registry(&registry, TEST_SITE_BASE_URL, "/about", "2026-05-11")
                .expect("fixture sitemap entry should build");
        assert!(sitemap_entry.contains("<loc>https://finitefield.org/about</loc>"));
        assert!(sitemap_entry.contains("<loc>https://finitefield.org/ja/about</loc>"));
        assert!(!sitemap_entry.contains("https://finitefield.org/fr/about"));
    }

    #[test]
    fn web_copy_document_loads_typed_sections() {
        let copy = web_copy_document();
        assert_eq!(copy.common.en["brand_subtitle"], "Seal Field");
        assert_eq!(copy.common.ja["brand_subtitle"], "印鑑フィールド");
        assert_eq!(copy.common.zh["brand_subtitle"], "印章设计");
        assert_eq!(copy.common.zhtw["brand_subtitle"], "印章設計");
        assert_eq!(copy.common.ar["brand_subtitle"], "تصميم الأختام");
        assert_eq!(
            web_copy_text("top", "en", "seo_title"),
            "Custom gemstone seals | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("design", "ja", "purchase_note_live"),
            "Stripe Checkout に遷移して決済します。"
        );
        assert_eq!(
            web_copy_text("commercial_transactions", "en", "seo_title"),
            "Legal Notice | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("top", "zh-Hans", "seo_title"),
            "在线定制宝石印章 | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("top", "zh-Hant", "seo_title"),
            "線上訂製寶石印章 | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("top", "zhtw", "seo_title"),
            "線上訂製寶石印章 | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("top", "ar", "seo_title"),
            "أختام أحجار كريمة مخصصة | STONE SIGNATURE"
        );
        assert_eq!(
            web_copy_text("top", "fr", "seo_title"),
            "Custom gemstone seals | STONE SIGNATURE"
        );
    }

    #[test]
    fn web_copy_locale_files_keep_matching_keys() {
        let copy = web_copy_document();
        for (section_name, section) in [
            ("common", &copy.common),
            ("about", &copy.about),
            ("blog_article", &copy.blog_article),
            ("blog_index", &copy.blog_index),
            ("commercial_transactions", &copy.commercial_transactions),
            ("design", &copy.design),
            ("kanji_suggestions", &copy.kanji_suggestions),
            ("payment_failure", &copy.payment_failure),
            ("payment_success", &copy.payment_success),
            ("purchase_result", &copy.purchase_result),
            ("terms", &copy.terms),
            ("top", &copy.top),
        ] {
            let en_keys = section.en.keys().collect::<HashSet<_>>();
            assert_eq!(
                section.ja.keys().collect::<HashSet<_>>(),
                en_keys,
                "{section_name}/ja keys must match en"
            );
            assert_eq!(
                section.zh.keys().collect::<HashSet<_>>(),
                en_keys,
                "{section_name}/zh keys must match en"
            );
            assert_eq!(
                section.zhtw.keys().collect::<HashSet<_>>(),
                en_keys,
                "{section_name}/zhtw keys must match en"
            );
            assert_eq!(
                section.ar.keys().collect::<HashSet<_>>(),
                en_keys,
                "{section_name}/ar keys must match en"
            );
        }
    }

    #[tokio::test]
    async fn pilot_payment_routes_render_localized_copy() {
        for (path, expected_dir, expected_title, expected_seo_title, expected_meta) in [
            (
                "/zh/payment/success",
                "ltr",
                "付款已完成",
                "付款完成 | STONE SIGNATURE",
                "你的付款已收到。请查看 Stripe 付款收据，确认订单详情和下一步。",
            ),
            (
                "/zhtw/payment/success",
                "ltr",
                "付款已完成",
                "付款完成 | STONE SIGNATURE",
                "你的付款已收到。請查看 Stripe 付款收據，確認訂單詳情和下一步。",
            ),
            (
                "/ar/payment/success",
                "rtl",
                "اكتمل الدفع",
                "اكتمل الدفع | STONE SIGNATURE",
                "تم استلام دفعتك. راجع إيصال الدفع من Stripe لمعرفة تفاصيل الطلب والخطوات التالية.",
            ),
            (
                "/zh/payment/failure",
                "ltr",
                "付款未完成",
                "付款未完成 | STONE SIGNATURE",
                "付款未完成。请检查银行卡信息，并返回购买页面重试。",
            ),
            (
                "/zhtw/payment/failure",
                "ltr",
                "付款未完成",
                "付款未完成 | STONE SIGNATURE",
                "付款未完成。請檢查卡片資訊，並返回購買頁面重試。",
            ),
            (
                "/ar/payment/failure",
                "rtl",
                "لم يكتمل الدفع",
                "الدفع غير مكتمل | STONE SIGNATURE",
                "لم يكتمل الدفع. تحقق من بيانات البطاقة وعد إلى صفحة الشراء للمحاولة مرة أخرى.",
            ),
        ] {
            let (status, body) = route_get_html(path).await;
            assert_eq!(status, StatusCode::OK, "{path} should render for QA");
            assert!(
                body.contains(&format!(r#"dir="{expected_dir}""#)),
                "{path} must render the expected text direction"
            );
            assert!(
                body.contains(expected_title),
                "{path} must render localized payment result copy"
            );
            assert!(
                body.contains(&format!("<title>{expected_seo_title}</title>")),
                "{path} must render localized payment SEO title"
            );
            assert!(
                body.contains(&format!(
                    r#"<meta name="description" content="{expected_meta}">"#
                )),
                "{path} must render localized payment meta description"
            );
            assert!(
                body.contains(r#"<meta name="robots" content="noindex,follow">"#),
                "{path} must remain non-indexed while in pilot"
            );
        }
    }

    #[test]
    fn locale_urls_use_english_as_the_main_variant() {
        assert_eq!(
            top_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/"
        );
        assert_eq!(
            top_url(TEST_SITE_BASE_URL, "ja"),
            "https://finitefield.org/ja/"
        );
        assert_eq!(
            top_url(TEST_SITE_BASE_URL, "jp"),
            "https://finitefield.org/ja/"
        );
        assert_eq!(
            top_url(TEST_SITE_BASE_URL, "fr"),
            "https://finitefield.org/"
        );
        assert_eq!(
            about_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/about"
        );
        assert_eq!(
            about_url(TEST_SITE_BASE_URL, "ja"),
            "https://finitefield.org/ja/about"
        );
        assert_eq!(
            design_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/design"
        );
        assert_eq!(
            design_url(TEST_SITE_BASE_URL, "ja"),
            "https://finitefield.org/ja/design"
        );
        assert_eq!(
            design_url(TEST_SITE_BASE_URL, "zhtw"),
            "https://finitefield.org/zhtw/design"
        );
        assert_eq!(
            design_url(TEST_SITE_BASE_URL, "ar"),
            "https://finitefield.org/ar/design"
        );
        assert_eq!(
            blog_index_url(TEST_ALT_SITE_BASE_URL, "en"),
            "https://inkanfield.org/blog"
        );
        assert_eq!(
            blog_article_url(TEST_ALT_SITE_BASE_URL, "hanko-vs-inkan", "en"),
            "https://inkanfield.org/blog/hanko-vs-inkan"
        );
        assert_eq!(
            blog_article_url(TEST_ALT_SITE_BASE_URL, "hanko-vs-inkan", "ja"),
            "https://inkanfield.org/ja/blog/hanko-vs-inkan"
        );
        assert_eq!(
            terms_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/terms"
        );
        assert_eq!(
            commercial_transactions_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/commercial-transactions"
        );
        assert_eq!(
            privacy_policy_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/en/privacy/"
        );
        assert_eq!(
            inquiry_url(TEST_SITE_BASE_URL, "en"),
            "https://finitefield.org/en/contact/"
        );
        assert_eq!(
            company_url(TEST_SITE_BASE_URL),
            "https://finitefield.org/company/"
        );
    }

    #[test]
    fn design_url_with_filters_preserves_selected_facets() {
        let filters = MaterialFilterState {
            color_family: "green".to_owned(),
            pattern_primary: "cloud".to_owned(),
        };

        assert_eq!(
            design_url_with_filters(TEST_SITE_BASE_URL, "en", &filters),
            "https://finitefield.org/design?color_family=green&pattern_primary=cloud"
        );
        assert_eq!(
            design_url_with_filters(TEST_SITE_BASE_URL, "ja", &filters),
            "https://finitefield.org/ja/design?color_family=green&pattern_primary=cloud"
        );
    }

    #[test]
    fn navigation_urls_preserve_the_selected_locale() {
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/", "en"),
            "https://finitefield.org/"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/about", "en"),
            "https://finitefield.org/about"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/design", "en"),
            "https://finitefield.org/design"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/terms", "en"),
            "https://finitefield.org/terms"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/commercial-transactions", "en"),
            "https://finitefield.org/commercial-transactions"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/", "ja"),
            "https://finitefield.org/ja/"
        );
        assert_eq!(
            localized_navigation_page_url(TEST_SITE_BASE_URL, "/commercial-transactions", "ja"),
            "https://finitefield.org/ja/commercial-transactions"
        );

        let query = PaymentRedirectQuery {
            checkout: Some("success".to_owned()),
            session_id: Some("sess_123".to_owned()),
            order_id: Some("ord_456".to_owned()),
            ..PaymentRedirectQuery::default()
        };
        assert_eq!(
            payment_result_navigation_url(TEST_SITE_BASE_URL, "/payment/success", &query, "en",),
            "https://finitefield.org/payment/success?checkout=success&session_id=sess_123&order_id=ord_456"
        );
    }

    #[test]
    fn legal_urls_still_point_to_finitefield_org_on_other_hosts() {
        assert_eq!(
            privacy_policy_url(TEST_ALT_SITE_BASE_URL, "ja"),
            "https://finitefield.org/privacy/"
        );
        assert_eq!(
            privacy_policy_url(TEST_ALT_SITE_BASE_URL, "en"),
            "https://finitefield.org/en/privacy/"
        );
        assert_eq!(
            inquiry_url(TEST_ALT_SITE_BASE_URL, "ja"),
            "https://finitefield.org/contact/"
        );
        assert_eq!(
            inquiry_url(TEST_ALT_SITE_BASE_URL, "en"),
            "https://finitefield.org/en/contact/"
        );
        assert_eq!(
            company_url(TEST_ALT_SITE_BASE_URL),
            "https://finitefield.org/company/"
        );
    }

    #[tokio::test]
    async fn legal_pages_back_buttons_point_to_top() {
        let commercial_response = handle_commercial_transactions(
            State(mock_state()),
            Query(LocaleQuery {
                lang: Some("en".to_owned()),
            }),
        )
        .await;
        let commercial_html = String::from_utf8(
            to_bytes(commercial_response.into_body(), usize::MAX)
                .await
                .expect("commercial transactions body should be readable")
                .to_vec(),
        )
        .expect("commercial transactions body should be utf-8");

        assert!(commercial_html.contains("Back to TOP"));
        assert!(commercial_html.contains("window.location.href='https://finitefield.org/'"));

        let terms_response = handle_terms(
            State(mock_state()),
            Query(LocaleQuery {
                lang: Some("en".to_owned()),
            }),
        )
        .await;
        let terms_html = String::from_utf8(
            to_bytes(terms_response.into_body(), usize::MAX)
                .await
                .expect("terms body should be readable")
                .to_vec(),
        )
        .expect("terms body should be utf-8");

        assert!(terms_html.contains("Back to TOP"));
        assert!(terms_html.contains("window.location.href='https://finitefield.org/'"));
    }

    #[test]
    fn payment_result_locale_url_uses_clean_english_urls() {
        let query = PaymentRedirectQuery {
            checkout: Some("success".to_owned()),
            session_id: Some("sess_123".to_owned()),
            order_id: Some("ord_456".to_owned()),
            ..PaymentRedirectQuery::default()
        };

        assert_eq!(
            payment_result_locale_url(TEST_SITE_BASE_URL, "/payment/success", &query, "en"),
            "https://finitefield.org/payment/success?checkout=success&session_id=sess_123&order_id=ord_456"
        );
        assert_eq!(
            payment_result_locale_url(TEST_SITE_BASE_URL, "/payment/success", &query, "ja"),
            "https://finitefield.org/ja/payment/success?checkout=success&session_id=sess_123&order_id=ord_456"
        );
    }

    #[test]
    fn payment_result_urls_preserve_app_return_marker() {
        let query = PaymentRedirectQuery {
            checkout: Some("success".to_owned()),
            session_id: Some("sess_123".to_owned()),
            order_id: Some("ord_456".to_owned()),
            return_to: Some("app".to_owned()),
            ..PaymentRedirectQuery::default()
        };

        assert_eq!(
            payment_result_locale_url(TEST_SITE_BASE_URL, "/payment/success", &query, "en"),
            "https://finitefield.org/payment/success?checkout=success&session_id=sess_123&order_id=ord_456&return_to=app"
        );
        assert_eq!(
            app_checkout_return_url("success", &query, "ja").as_deref(),
            Some(
                "hankofield://checkout/success?checkout=success&order_id=ord_456&session_id=sess_123&lang=ja"
            )
        );
    }

    #[tokio::test]
    async fn app_checkout_success_page_does_not_auto_redirect_to_custom_scheme() {
        let response = handle_payment_success(
            State(mock_state()),
            Query(PaymentRedirectQuery {
                checkout: Some("success".to_owned()),
                session_id: Some("cs_test_001".to_owned()),
                order_id: Some("ord_001".to_owned()),
                lang: Some("ja".to_owned()),
                return_to: Some("app".to_owned()),
                ..PaymentRedirectQuery::default()
            }),
        )
        .await;

        let html = String::from_utf8(
            to_bytes(response.into_body(), usize::MAX)
                .await
                .expect("payment success body should be readable")
                .to_vec(),
        )
        .expect("payment success body should be utf-8");

        assert!(!html.contains("http-equiv=\"refresh\""));
        assert!(!html.contains("window.location.replace"));
        assert!(html.contains("hankofield://checkout/success"));
        assert!(html.contains("order_id=ord_001"));
        assert!(html.contains("session_id=cs_test_001"));
    }

    #[tokio::test]
    async fn robots_txt_is_served_as_plain_text() {
        let response = handle_robots_txt(State(mock_state())).await;

        assert_eq!(response.status(), StatusCode::OK);
        assert_eq!(
            response
                .headers()
                .get("content-type")
                .and_then(|value| value.to_str().ok()),
            Some("text/plain; charset=utf-8")
        );

        let body = to_bytes(response.into_body(), usize::MAX)
            .await
            .expect("response body should be readable");
        let robots_txt = String::from_utf8(body.to_vec()).expect("response body should be utf-8");

        assert!(robots_txt.contains("User-agent: *"));
        assert!(robots_txt.contains("Disallow: /admin"));
        assert!(robots_txt.contains("Disallow: /mock"));
        assert!(robots_txt.contains("Disallow: /kanji"));
        assert!(robots_txt.contains("Disallow: /purchase"));
        assert!(robots_txt.contains("Disallow: /payment/"));
        assert!(robots_txt.contains("Sitemap: https://finitefield.org/sitemap.xml"));
        assert!(
            build_robots_txt(TEST_ALT_SITE_BASE_URL)
                .contains("Sitemap: https://inkanfield.org/sitemap.xml")
        );
    }

    #[tokio::test]
    async fn sitemap_xml_is_served_as_xml() {
        let response = handle_sitemap_xml(State(mock_state())).await;

        assert_eq!(response.status(), StatusCode::OK);
        assert_eq!(
            response
                .headers()
                .get("content-type")
                .and_then(|value| value.to_str().ok()),
            Some("application/xml; charset=utf-8")
        );

        let body = to_bytes(response.into_body(), usize::MAX)
            .await
            .expect("response body should be readable");
        let sitemap_xml = String::from_utf8(body.to_vec()).expect("response body should be utf-8");

        assert!(sitemap_xml.contains("<loc>https://finitefield.org/</loc>"));
        assert!(sitemap_xml.contains("<lastmod>2026-05-11</lastmod>"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="en" href="https://finitefield.org/" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="x-default" href="https://finitefield.org/" />"#
        ));
        assert!(sitemap_xml.contains("<loc>https://finitefield.org/about</loc>"));
        assert!(!sitemap_xml.contains("https://finitefield.org/about/"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="en" href="https://finitefield.org/about" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/about" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="x-default" href="https://finitefield.org/about" />"#
        ));
        assert!(sitemap_xml.contains("<loc>https://finitefield.org/design</loc>"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="en" href="https://finitefield.org/design" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/design" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="x-default" href="https://finitefield.org/design" />"#
        ));
        assert!(sitemap_xml.contains("<loc>https://finitefield.org/blog</loc>"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/blog" />"#
        ));
        let posts = load_blog_posts().expect("blog posts should load");
        for slug in ADDED_BLOG_ARTICLE_SLUGS {
            let post = find_blog_post(&posts, slug).expect("blog post should load");
            assert_sitemap_has_localized_article_entry(
                &sitemap_xml,
                slug,
                &post.last_modified_date,
            );
        }
        assert!(
            sitemap_xml.contains("<loc>https://finitefield.org/blog/what-is-a-personal-seal</loc>")
        );
        assert!(sitemap_xml.contains("<lastmod>2026-05-07</lastmod>"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/blog/what-is-a-personal-seal" />"#
        ));
        assert!(
            !sitemap_xml
                .contains("<loc>https://finitefield.org/blog/art-of-selection-jadeite</loc>")
        );
        assert!(sitemap_xml.contains("<loc>https://finitefield.org/terms</loc>"));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="en" href="https://finitefield.org/terms" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/terms" />"#
        ));
        assert!(sitemap_xml.contains(
            r#"<xhtml:link rel="alternate" hreflang="x-default" href="https://finitefield.org/terms" />"#
        ));
        assert!(sitemap_xml.contains("<loc>https://finitefield.org/commercial-transactions</loc>"));
        assert!(
            sitemap_xml
                .contains(r#"<xhtml:link rel="alternate" hreflang="en" href="https://finitefield.org/commercial-transactions" />"#)
        );
        assert!(
            sitemap_xml
                .contains(r#"<xhtml:link rel="alternate" hreflang="ja" href="https://finitefield.org/ja/commercial-transactions" />"#)
        );
        assert!(
            sitemap_xml
                .contains(r#"<xhtml:link rel="alternate" hreflang="x-default" href="https://finitefield.org/commercial-transactions" />"#)
        );
        assert!(!sitemap_xml.contains("https://finitefield.org/payment/"));
        assert!(!sitemap_xml.contains("https://finitefield.org/purchase"));
        assert!(!sitemap_xml.contains("https://finitefield.org/kanji"));
        assert!(!sitemap_xml.contains("https://finitefield.org/admin"));
    }
}
