use std::{
    collections::{BTreeMap, BTreeSet, HashMap, HashSet},
    env,
    sync::{Arc, OnceLock},
    time::Duration,
};

use anyhow::{Context, Result, anyhow, bail};
use askama::Template;
use axum::{
    Router,
    extract::{Form, Multipart, Path, Query, State, rejection::FormRejection},
    http::{HeaderMap, HeaderValue, StatusCode, header},
    middleware,
    response::{IntoResponse, Redirect, Response},
    routing::{get, patch},
};
use chrono::{DateTime, Local, SecondsFormat, Utc};
use firebase_sdk_rust::firebase_firestore::{
    CreateDocumentOptions, DeleteDocumentOptions, Document, FirebaseFirestoreClient,
    GetDocumentOptions, PatchDocumentOptions, RunQueryRequest,
};
use gcp_auth::{CustomServiceAccount, TokenProvider, provider};
use jsonwebtoken::{Algorithm, DecodingKey, EncodingKey, Header, Validation, decode, encode};
use serde::Deserialize;
use serde::Serialize;
use serde_json::{Value as JsonValue, json};
use tokio::{net::TcpListener, sync::RwLock};
use tower_http::services::ServeDir;
use uuid::Uuid;

const DATASTORE_SCOPE: &str = "https://www.googleapis.com/auth/datastore";
const STORAGE_SCOPE: &str = "https://www.googleapis.com/auth/devstorage.read_write";
const MAX_PHOTO_UPLOAD_BYTES: usize = 10 * 1024 * 1024;
const ADMIN_SOURCE_LOAD_TIMEOUT: Duration = Duration::from_secs(20);
const ADMIN_SESSION_COOKIE_NAME: &str = "hanko_admin_session";
const ADMIN_SESSION_DURATION_SECONDS: i64 = 60 * 60 * 24 * 7;
const ADMIN_SESSION_ISSUER: &str = "hanko-field-admin";
const HX_REDIRECT_HEADER: &str = "hx-redirect";
const ADMIN_EDITABLE_LOCALE_KEYS: [&str; 2] = ["ja", "en"];
const LANGUAGE_REGISTRY_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../config/languages.json"
));

static ADMIN_LOCALIZED_EDITOR_LANGUAGES: OnceLock<Vec<AdminLocalizedEditorLanguage>> =
    OnceLock::new();

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

#[derive(Debug, Clone)]
struct AppConfig {
    http_addr: String,
    mode: RunMode,
    locale: String,
    default_locale: String,
    firestore_project_id: Option<String>,
    storage_assets_bucket: Option<String>,
    credentials_file: Option<String>,
    login_passphrase: Option<String>,
}

#[derive(Debug, Clone)]
struct Order {
    id: String,
    order_no: String,
    channel: String,
    locale: String,
    currency: String,
    listing_key: String,
    status: String,
    status_updated_at: DateTime<Utc>,
    payment_status: String,
    fulfillment_status: String,
    tracking_no: String,
    carrier: String,
    country_code: String,
    contact_email: String,
    seal_line1: String,
    seal_line2: String,
    engraving_text: String,
    ai_generation_id: String,
    ai_variant_id: String,
    seal_preview_image_storage_path: String,
    seal_style_name: String,
    seal_style_stroke_weight: String,
    seal_style_balance: String,
    seal_style_prompt_summary: String,
    customer_confirmation_kanji_and_design: Option<bool>,
    customer_confirmation_custom_made_policy: Option<bool>,
    customer_confirmation_confirmed_at: Option<DateTime<Utc>>,
    customer_confirmation_confirmed_seal_text: String,
    listing_label_ja: String,
    total: i64,
    created_at: DateTime<Utc>,
    updated_at: DateTime<Utc>,
    events: Vec<OrderEvent>,
}

#[derive(Debug, Clone)]
struct OrderEvent {
    kind: String,
    actor_type: String,
    actor_id: String,
    before_status: String,
    after_status: String,
    note: String,
    created_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
struct MaterialPhoto {
    asset_id: String,
    storage_path: String,
    alt_i18n: HashMap<String, String>,
    sort_order: i64,
    is_primary: bool,
    width: i64,
    height: i64,
}

#[derive(Debug, Clone)]
struct StoneListingFacets {
    color_family: String,
    color_tags: Vec<String>,
    pattern_primary: String,
    pattern_tags: Vec<String>,
    stone_shape: String,
    translucency: String,
}

#[derive(Debug, Clone)]
struct StoneListing {
    key: String,
    listing_code: String,
    material_key: String,
    size: String,
    title_i18n: HashMap<String, String>,
    description_i18n: HashMap<String, String>,
    story_i18n: HashMap<String, String>,
    facets: StoneListingFacets,
    photos: Vec<MaterialPhoto>,
    price_by_currency: HashMap<String, i64>,
    status: String,
    is_active: bool,
    published_at: Option<DateTime<Utc>>,
    sort_order: i64,
    version: i64,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy)]
struct MaterialComparisonProfile {
    texture_ja: &'static str,
    texture_en: &'static str,
    weight_ja: &'static str,
    weight_en: &'static str,
    usage_ja: &'static str,
    usage_en: &'static str,
}

#[derive(Debug, Clone, Copy)]
struct MaterialMasterSeed {
    key: &'static str,
    label_ja: &'static str,
    label_en: &'static str,
    description_ja: &'static str,
    description_en: &'static str,
    sort_order: i64,
}

#[derive(Debug, Clone)]
struct Material {
    key: String,
    label_i18n: HashMap<String, String>,
    description_i18n: HashMap<String, String>,
    comparison_texture_ja: String,
    comparison_texture_en: String,
    comparison_weight_ja: String,
    comparison_weight_en: String,
    comparison_usage_ja: String,
    comparison_usage_en: String,
    shape: String,
    photos: Vec<MaterialPhoto>,
    price_usd: i64,
    price_jpy: i64,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
struct Font {
    key: String,
    label: String,
    font_family: String,
    font_stylesheet_url: String,
    kanji_style: String,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
struct Country {
    code: String,
    label_i18n: HashMap<String, String>,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
struct FacetTag {
    key: String,
    facet_type: String,
    label_i18n: HashMap<String, String>,
    aliases: Vec<String>,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: DateTime<Utc>,
}

#[derive(Debug, Clone)]
struct AdminSnapshot {
    orders: HashMap<String, Order>,
    order_ids: Vec<String>,
    fonts: HashMap<String, Font>,
    font_ids: Vec<String>,
    materials: HashMap<String, Material>,
    material_ids: Vec<String>,
    stone_listings: HashMap<String, StoneListing>,
    stone_listing_ids: Vec<String>,
    facet_tags: HashMap<String, FacetTag>,
    facet_tag_ids: Vec<String>,
    countries: HashMap<String, Country>,
    country_ids: Vec<String>,
}

impl AdminSnapshot {
    fn refresh_order_ids(&mut self) {
        let mut ids = self.orders.keys().cloned().collect::<Vec<_>>();
        ids.sort_by(|left, right| {
            let left_created = self
                .orders
                .get(left)
                .map(|order| order.created_at)
                .unwrap_or(DateTime::<Utc>::UNIX_EPOCH);
            let right_created = self
                .orders
                .get(right)
                .map(|order| order.created_at)
                .unwrap_or(DateTime::<Utc>::UNIX_EPOCH);
            right_created.cmp(&left_created)
        });
        self.order_ids = ids;
    }

    fn refresh_material_ids(&mut self) {
        let mut keys = self.materials.keys().cloned().collect::<Vec<_>>();
        keys.sort_by(|left, right| {
            let left_material = self.materials.get(left);
            let right_material = self.materials.get(right);
            match (left_material, right_material) {
                (Some(left_material), Some(right_material)) => left_material
                    .sort_order
                    .cmp(&right_material.sort_order)
                    .then_with(|| left_material.key.cmp(&right_material.key)),
                _ => left.cmp(right),
            }
        });
        self.material_ids = keys;
    }

    fn refresh_stone_listing_ids(&mut self) {
        let mut keys = self.stone_listings.keys().cloned().collect::<Vec<_>>();
        keys.sort_by(|left, right| {
            let left_listing = self.stone_listings.get(left);
            let right_listing = self.stone_listings.get(right);
            match (left_listing, right_listing) {
                (Some(left_listing), Some(right_listing)) => left_listing
                    .sort_order
                    .cmp(&right_listing.sort_order)
                    .then_with(|| {
                        stone_listing_published_at_sort_key(left_listing)
                            .cmp(&stone_listing_published_at_sort_key(right_listing))
                            .reverse()
                    })
                    .then_with(|| left_listing.key.cmp(&right_listing.key)),
                _ => left.cmp(right),
            }
        });
        self.stone_listing_ids = keys;
    }

    fn refresh_facet_tag_ids(&mut self) {
        let mut keys = self.facet_tags.keys().cloned().collect::<Vec<_>>();
        keys.sort_by(|left, right| {
            let left_tag = self.facet_tags.get(left);
            let right_tag = self.facet_tags.get(right);
            match (left_tag, right_tag) {
                (Some(left_tag), Some(right_tag)) => left_tag
                    .facet_type
                    .cmp(&right_tag.facet_type)
                    .then_with(|| left_tag.sort_order.cmp(&right_tag.sort_order))
                    .then_with(|| left_tag.key.cmp(&right_tag.key)),
                _ => left.cmp(right),
            }
        });
        self.facet_tag_ids = keys;
    }

    fn refresh_font_ids(&mut self) {
        let mut keys = self.fonts.keys().cloned().collect::<Vec<_>>();
        keys.sort_by(|left, right| {
            let left_font = self.fonts.get(left);
            let right_font = self.fonts.get(right);
            match (left_font, right_font) {
                (Some(left_font), Some(right_font)) => left_font
                    .sort_order
                    .cmp(&right_font.sort_order)
                    .then_with(|| left_font.key.cmp(&right_font.key)),
                _ => left.cmp(right),
            }
        });
        self.font_ids = keys;
    }

    fn refresh_country_ids(&mut self) {
        let mut codes = self.countries.keys().cloned().collect::<Vec<_>>();
        codes.sort_by(|left, right| {
            let left_country = self.countries.get(left);
            let right_country = self.countries.get(right);
            match (left_country, right_country) {
                (Some(left_country), Some(right_country)) => left_country
                    .sort_order
                    .cmp(&right_country.sort_order)
                    .then_with(|| left_country.code.cmp(&right_country.code)),
                _ => left.cmp(right),
            }
        });
        self.country_ids = codes;
    }
}

#[derive(Clone)]
enum DataSource {
    Mock,
    Firestore(FirestoreAdminSource),
}

impl DataSource {
    fn is_mock(&self) -> bool {
        matches!(self, Self::Mock)
    }

    fn label(&self) -> &str {
        match self {
            Self::Mock => "Mock",
            Self::Firestore(source) => &source.label,
        }
    }

    async fn load_snapshot(&self) -> Result<AdminSnapshot> {
        match self {
            Self::Mock => Ok(new_mock_snapshot()),
            Self::Firestore(source) => source.load_snapshot().await,
        }
    }

    async fn persist_order_mutation(&self, order: &Order, events: &[OrderEvent]) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_order_mutation(order, events).await,
        }
    }

    async fn persist_material_mutation(&self, material: &Material) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_material_mutation(material).await,
        }
    }

    async fn persist_stone_listing_mutation(&self, listing: &StoneListing) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_stone_listing_mutation(listing).await,
        }
    }

    async fn persist_facet_tag_mutation(&self, tag: &FacetTag) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_facet_tag_mutation(tag).await,
        }
    }

    async fn persist_font_mutation(&self, font: &Font) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_font_mutation(font).await,
        }
    }

    async fn persist_country_mutation(&self, country: &Country) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_country_mutation(country).await,
        }
    }

    async fn persist_country_deletion(&self, country_code: &str) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_country_deletion(country_code).await,
        }
    }

    async fn persist_material_deletion(&self, material_key: &str) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_material_deletion(material_key).await,
        }
    }

    async fn persist_stone_listing_deletion(&self, listing_key: &str) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_stone_listing_deletion(listing_key).await,
        }
    }

    async fn persist_facet_tag_deletion(&self, tag_id: &str) -> Result<()> {
        match self {
            Self::Mock => Ok(()),
            Self::Firestore(source) => source.persist_facet_tag_deletion(tag_id).await,
        }
    }

    async fn upload_material_photo(
        &self,
        material_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> Result<String> {
        match self {
            Self::Mock => Ok(build_storage_path_for_uploaded_photo(
                material_key,
                file_name,
                content_type,
            )),
            Self::Firestore(source) => {
                source
                    .upload_material_photo(material_key, file_name, content_type, bytes)
                    .await
            }
        }
    }

    async fn upload_stone_listing_photo(
        &self,
        stone_listing_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> Result<String> {
        match self {
            Self::Mock => Ok(build_storage_path_for_uploaded_stone_listing_photo(
                stone_listing_key,
                file_name,
                content_type,
            )),
            Self::Firestore(source) => {
                source
                    .upload_stone_listing_photo(stone_listing_key, file_name, content_type, bytes)
                    .await
            }
        }
    }
}

#[derive(Clone)]
struct FirestoreAdminSource {
    default_locale: String,
    label: String,
    parent: String,
    storage_assets_bucket: String,
    token_provider: Arc<dyn TokenProvider>,
}

#[derive(Clone)]
struct AuthState {
    enabled: bool,
    login_passphrase: String,
}

#[derive(Clone)]
struct AppState {
    server: Arc<ServerState>,
    auth: Arc<AuthState>,
}

struct ServerState {
    source_label: String,
    storage_assets_bucket: String,
    source: DataSource,
    data: RwLock<AdminSnapshot>,
}

#[derive(Debug, Clone)]
struct OrderFilter {
    status: String,
    country: String,
    email: String,
}

#[derive(Debug, Clone, Default)]
struct StoneListingFilter {
    color_family: String,
    color_tags: String,
    pattern_primary: String,
    pattern_tags: String,
    stone_shape: String,
}

#[derive(Debug, Default, Deserialize)]
struct OrdersPageQuery {
    status: Option<String>,
    country: Option<String>,
    email: Option<String>,
    order_id: Option<String>,
}

#[derive(Debug, Default, Deserialize)]
struct StoneListingsPageQuery {
    color_family: Option<String>,
    color_tags: Option<String>,
    pattern_primary: Option<String>,
    pattern_tags: Option<String>,
    stone_shape: Option<String>,
}

#[derive(Debug, Default, Deserialize)]
struct CountriesPageQuery {
    country_code: Option<String>,
}

#[derive(Debug, Clone)]
struct StoneListingFacetOptionView {
    value: String,
    label: String,
}

#[derive(Debug, Clone)]
struct StoneListingFilterOptions {
    color_family_options: Vec<StoneListingFacetOptionView>,
    pattern_primary_options: Vec<StoneListingFacetOptionView>,
    stone_shape_options: Vec<StoneListingFacetOptionView>,
}

#[derive(Debug, Clone)]
struct StoneListingTagOptions {
    color_tag_options: Vec<StoneListingFacetOptionView>,
    pattern_tag_options: Vec<StoneListingFacetOptionView>,
}

#[derive(Debug, Clone)]
struct StoneListingFilterBadgeView {
    label: String,
    value: String,
}

#[derive(Debug, Clone)]
struct StatusOptionView {
    value: String,
    label: String,
}

#[derive(Debug, Clone)]
struct CountryOptionView {
    code: String,
    label: String,
}

#[derive(Debug, Clone)]
struct OrderListItemView {
    id: String,
    order_no: String,
    created_at: String,
    channel_label: String,
    is_app_order: bool,
    engraving_text: String,
    has_engraving_text: bool,
    ai_variant_id: String,
    has_ai_variant_id: bool,
    status_label: String,
    payment_status_label: String,
    fulfillment_status_label: String,
    country_label: String,
    total: String,
}

#[derive(Debug, Clone)]
struct OrderEventView {
    kind: String,
    actor_type: String,
    actor_id: String,
    has_actor_id: bool,
    before_status_label: String,
    after_status_label: String,
    has_before_status: bool,
    note: String,
    has_note: bool,
    created_at: String,
}

#[derive(Debug, Clone)]
struct OrderDetailView {
    id: String,
    order_no: String,
    created_at: String,
    updated_at: String,
    status_label: String,
    payment_status_label: String,
    fulfillment_status_label: String,
    tracking_no: String,
    has_tracking_no: bool,
    carrier: String,
    country_code: String,
    country_label: String,
    contact_email: String,
    channel: String,
    locale: String,
    seal_line1: String,
    seal_line2: String,
    has_seal_line2: bool,
    seal_preview_image_storage_path: String,
    seal_preview_image_url: String,
    has_seal_preview_image: bool,
    has_seal_preview_image_url: bool,
    ai_generation_id: String,
    has_ai_generation_id: bool,
    ai_variant_id: String,
    has_ai_variant_id: bool,
    seal_style_name: String,
    seal_style_stroke_weight: String,
    seal_style_balance: String,
    seal_style_prompt_summary: String,
    has_seal_style_name: bool,
    has_seal_style_stroke_weight: bool,
    has_seal_style_balance: bool,
    has_seal_style_prompt_summary: bool,
    has_ai_seal_metadata: bool,
    customer_confirmation_kanji_and_design_label: String,
    customer_confirmation_custom_made_policy_label: String,
    customer_confirmation_confirmed_at: String,
    customer_confirmation_confirmed_seal_text: String,
    has_customer_confirmation_kanji_and_design: bool,
    has_customer_confirmation_custom_made_policy: bool,
    has_customer_confirmation_confirmed_at: bool,
    has_customer_confirmation_confirmed_seal_text: bool,
    has_customer_confirmation: bool,
    listing_label_ja: String,
    total: String,
    next_statuses: Vec<StatusOptionView>,
    has_next_statuses: bool,
    shipping_transitions: Vec<StatusOptionView>,
    events: Vec<OrderEventView>,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct MaterialListItemView {
    key: String,
    label_ja: String,
    is_active: bool,
    version: i64,
    updated_at: String,
}

#[derive(Debug, Clone)]
struct FontListItemView {
    key: String,
    label: String,
    font_family: String,
    font_stylesheet_url: String,
    kanji_style_label: String,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: String,
}

#[derive(Debug, Clone)]
struct LocalizedEditorGroupView {
    label: String,
    values: Vec<LocalizedEditorValueView>,
}

#[derive(Debug, Clone)]
struct LocalizedEditorValueView {
    route_code: String,
    language_label: String,
    text_direction: String,
    input_name: String,
    value: String,
}

#[derive(Debug, Clone)]
struct MaterialDetailView {
    key: String,
    label_ja: String,
    label_en: String,
    description_ja: String,
    description_en: String,
    localized_editors: Vec<LocalizedEditorGroupView>,
    has_localized_editors: bool,
    is_active: bool,
    version: i64,
    updated_at: String,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct StoneListingListItemView {
    key: String,
    listing_code: String,
    title_ja: String,
    material_key: String,
    size: String,
    color_family: String,
    pattern_primary: String,
    stone_shape_label: String,
    primary_photo_url: String,
    has_photo: bool,
    price_usd: String,
    price_jpy: String,
    status_label: String,
    is_active: bool,
    version: i64,
    updated_at: String,
}

#[derive(Debug, Clone)]
struct StoneListingDetailView {
    key: String,
    listing_code: String,
    material_key: String,
    size: String,
    title_ja: String,
    title_en: String,
    description_ja: String,
    description_en: String,
    story_ja: String,
    story_en: String,
    color_family: String,
    color_tags: String,
    pattern_primary: String,
    pattern_tags: String,
    stone_shape: String,
    stone_shape_label: String,
    translucency: String,
    price_usd: i64,
    price_jpy: i64,
    status: String,
    status_label: String,
    is_active: bool,
    sort_order: i64,
    photo_storage_path: String,
    primary_photo_url: String,
    photo_alt_ja: String,
    photo_alt_en: String,
    localized_editors: Vec<LocalizedEditorGroupView>,
    has_localized_editors: bool,
    has_photo: bool,
    version: i64,
    updated_at: String,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct FontDetailView {
    key: String,
    label: String,
    font_family: String,
    font_stylesheet_url: String,
    kanji_style: String,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: String,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct CountryListItemView {
    code: String,
    label_ja: String,
    label_en: String,
    shipping_fee_usd: String,
    shipping_fee_jpy: String,
    is_active: bool,
    version: i64,
    updated_at: String,
}

#[derive(Debug, Clone)]
struct CountryDetailView {
    code: String,
    label_ja: String,
    label_en: String,
    localized_editors: Vec<LocalizedEditorGroupView>,
    has_localized_editors: bool,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    is_active: bool,
    sort_order: i64,
    version: i64,
    updated_at: String,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct CountryCreateView {
    code: String,
    label_ja: String,
    label_en: String,
    shipping_fee_usd: String,
    shipping_fee_jpy: String,
    sort_order: String,
    is_active: bool,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct FacetTagListItemView {
    id: String,
    key: String,
    facet_type_label: String,
    label_ja: String,
    label_en: String,
    aliases: String,
    is_active: bool,
    version: i64,
    updated_at: String,
}

#[derive(Debug, Clone)]
struct FacetTagDetailView {
    id: String,
    key: String,
    facet_type_label: String,
    label_ja: String,
    label_en: String,
    localized_editors: Vec<LocalizedEditorGroupView>,
    has_localized_editors: bool,
    aliases: String,
    sort_order: i64,
    is_active: bool,
    version: i64,
    updated_at: String,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
    key_error: String,
    has_key_error: bool,
    aliases_error: String,
    has_aliases_error: bool,
}

#[derive(Debug, Clone)]
struct FacetTagCreateView {
    key: String,
    facet_type: String,
    label_ja: String,
    label_en: String,
    aliases: String,
    sort_order: String,
    is_active: bool,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
    key_error: String,
    has_key_error: bool,
    aliases_error: String,
    has_aliases_error: bool,
}

#[derive(Debug, Clone)]
struct FacetTagTypeOptionView {
    value: String,
    label: String,
}

#[derive(Debug, Default, Deserialize)]
struct FacetTagsPageQuery {
    facet_tag_id: Option<String>,
}

#[derive(Debug, Clone)]
struct MaterialCreateView {
    key: String,
    label_ja: String,
    label_en: String,
    description_ja: String,
    description_en: String,
    is_active: bool,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct StoneListingCreateView {
    stone_listing_key: String,
    listing_code: String,
    material_key: String,
    size: String,
    title_ja: String,
    title_en: String,
    description_ja: String,
    description_en: String,
    story_ja: String,
    story_en: String,
    color_family: String,
    color_tags: String,
    pattern_primary: String,
    pattern_tags: String,
    stone_shape: String,
    translucency: String,
    price_usd: String,
    price_jpy: String,
    sort_order: String,
    photo_storage_path: String,
    photo_alt_ja: String,
    photo_alt_en: String,
    status: String,
    is_active: bool,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct FontCreateView {
    key: String,
    label: String,
    font_family: String,
    kanji_style: String,
    sort_order: String,
    is_active: bool,
    message: String,
    has_message: bool,
    error: String,
    has_error: bool,
}

#[derive(Debug, Clone)]
struct MaterialCreateInput {
    key: String,
    label_ja: String,
    label_en: String,
    description_ja: String,
    description_en: String,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct StoneListingCreateInput {
    stone_listing_key: String,
    listing_code: String,
    material_key: String,
    size: String,
    title_ja: String,
    title_en: String,
    description_ja: String,
    description_en: String,
    story_ja: String,
    story_en: String,
    color_family: String,
    color_tags: Vec<String>,
    pattern_primary: String,
    pattern_tags: Vec<String>,
    stone_shape: String,
    translucency: String,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
    photo_storage_path: String,
    photo_alt_ja: String,
    photo_alt_en: String,
    status: String,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct FontCreateInput {
    key: String,
    label: String,
    font_family: String,
    kanji_style: String,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct MaterialPatchInput {
    label_ja: String,
    label_en: String,
    description_ja: String,
    description_en: String,
    label_i18n_updates: HashMap<String, String>,
    description_i18n_updates: HashMap<String, String>,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct StoneListingPatchInput {
    listing_code: String,
    material_key: String,
    size: String,
    title_ja: String,
    title_en: String,
    description_ja: String,
    description_en: String,
    story_ja: String,
    story_en: String,
    color_family: String,
    color_tags: Vec<String>,
    pattern_primary: String,
    pattern_tags: Vec<String>,
    stone_shape: String,
    translucency: String,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
    photo_storage_path: String,
    photo_alt_ja: String,
    photo_alt_en: String,
    title_i18n_updates: HashMap<String, String>,
    description_i18n_updates: HashMap<String, String>,
    story_i18n_updates: HashMap<String, String>,
    photo_alt_i18n_updates: HashMap<String, String>,
    status: String,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct FontPatchInput {
    label: String,
    font_family: String,
    kanji_style: String,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct CountryPatchInput {
    label_ja: String,
    label_en: String,
    label_i18n_updates: HashMap<String, String>,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct CountryCreateInput {
    code: String,
    label_ja: String,
    label_en: String,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct FacetTagCreateInput {
    key: String,
    facet_type: String,
    label_ja: String,
    label_en: String,
    aliases: Vec<String>,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Clone)]
struct FacetTagPatchInput {
    label_ja: String,
    label_en: String,
    label_i18n_updates: HashMap<String, String>,
    aliases: Vec<String>,
    sort_order: i64,
    is_active: bool,
}

#[derive(Debug, Deserialize)]
struct AdminLanguageRegistryEntry {
    route_code: String,
    native_name: String,
    english_name: String,
    text_direction: String,
}

#[derive(Debug, Clone)]
struct AdminLocalizedEditorLanguage {
    route_code: String,
    native_name: String,
    english_name: String,
    text_direction: String,
}

#[derive(Template)]
#[template(path = "orders_page.html")]
struct OrdersPageTemplate {
    filters: OrderFilter,
    status_options: Vec<StatusOptionView>,
    country_options: Vec<CountryOptionView>,
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    orders_list_html: String,
    order_detail_html: String,
    has_order_detail: bool,
}

#[derive(Template)]
#[template(path = "materials_page.html")]
struct MaterialsPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    materials_list_html: String,
}

#[derive(Template)]
#[template(path = "stone_listings_page.html")]
struct StoneListingsPageTemplate {
    filters: StoneListingFilter,
    color_family_options: Vec<StoneListingFacetOptionView>,
    pattern_primary_options: Vec<StoneListingFacetOptionView>,
    stone_shape_options: Vec<StoneListingFacetOptionView>,
    color_tag_options: Vec<StoneListingFacetOptionView>,
    pattern_tag_options: Vec<StoneListingFacetOptionView>,
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    stone_listings_list_html: String,
}

#[derive(Template)]
#[template(path = "fonts_page.html")]
struct FontsPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    fonts_list_html: String,
}

#[derive(Template)]
#[template(path = "material_edit_page.html")]
struct MaterialEditPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    material_key: String,
    material_detail_html: String,
}

#[derive(Template)]
#[template(path = "stone_listing_edit_page.html")]
struct StoneListingEditPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    color_tag_options: Vec<StoneListingFacetOptionView>,
    pattern_tag_options: Vec<StoneListingFacetOptionView>,
    stone_listing_key: String,
    stone_listing_detail_html: String,
}

#[derive(Template)]
#[template(path = "font_edit_page.html")]
struct FontEditPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    font_key: String,
    font_detail_html: String,
}

#[derive(Template)]
#[template(path = "font_create_page.html")]
struct FontCreatePageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    font_create_html: String,
}

#[derive(Template)]
#[template(path = "material_create_page.html")]
struct MaterialCreatePageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    material_create_html: String,
}

#[derive(Template)]
#[template(path = "stone_listing_create_page.html")]
struct StoneListingCreatePageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    color_tag_options: Vec<StoneListingFacetOptionView>,
    pattern_tag_options: Vec<StoneListingFacetOptionView>,
    stone_listing_create_html: String,
}

#[derive(Template)]
#[template(path = "countries_page.html")]
struct CountriesPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    country_create_html: String,
    countries_list_html: String,
    country_detail_html: String,
    has_country_detail: bool,
}

#[derive(Template)]
#[template(path = "facet_tags_page.html")]
struct FacetTagsPageTemplate {
    font_stylesheet_urls: Vec<String>,
    source_label: String,
    is_mock: bool,
    facet_tag_create_html: String,
    facet_tags_list_html: String,
    facet_tag_detail_html: String,
    has_facet_tag_detail: bool,
}

#[derive(Template)]
#[template(path = "orders_list.html")]
struct OrdersListTemplate {
    orders: Vec<OrderListItemView>,
    has_orders: bool,
}

#[derive(Template)]
#[template(path = "order_detail.html")]
struct OrderDetailTemplate {
    detail: OrderDetailView,
}

#[derive(Template)]
#[template(path = "materials_list.html")]
struct MaterialsListTemplate {
    materials: Vec<MaterialListItemView>,
    has_materials: bool,
}

#[derive(Template)]
#[template(path = "facet_tags_list.html")]
struct FacetTagsListTemplate {
    facet_tags: Vec<FacetTagListItemView>,
    has_facet_tags: bool,
}

#[derive(Template)]
#[template(path = "stone_listings_list.html")]
struct StoneListingsListTemplate {
    stone_listings: Vec<StoneListingListItemView>,
    has_stone_listings: bool,
    matching_count: usize,
    has_active_filters: bool,
    filter_badges: Vec<StoneListingFilterBadgeView>,
}

#[derive(Template)]
#[template(path = "fonts_list.html")]
struct FontsListTemplate {
    fonts: Vec<FontListItemView>,
    has_fonts: bool,
}

#[derive(Template)]
#[template(path = "material_detail.html")]
struct MaterialDetailTemplate {
    detail: MaterialDetailView,
}

#[derive(Template)]
#[template(path = "stone_listing_detail.html")]
struct StoneListingDetailTemplate {
    detail: StoneListingDetailView,
}

#[derive(Template)]
#[template(path = "facet_tag_detail.html")]
struct FacetTagDetailTemplate {
    detail: FacetTagDetailView,
}

#[derive(Template)]
#[template(path = "font_detail.html")]
struct FontDetailTemplate {
    detail: FontDetailView,
}

#[derive(Template)]
#[template(path = "material_create.html")]
struct MaterialCreateTemplate {
    view: MaterialCreateView,
}

#[derive(Template)]
#[template(path = "stone_listing_create.html")]
struct StoneListingCreateTemplate {
    view: StoneListingCreateView,
}

#[derive(Template)]
#[template(path = "font_create.html")]
struct FontCreateTemplate {
    view: FontCreateView,
}

#[derive(Template)]
#[template(path = "countries_list.html")]
struct CountriesListTemplate {
    countries: Vec<CountryListItemView>,
    has_countries: bool,
}

#[derive(Template)]
#[template(path = "country_detail.html")]
struct CountryDetailTemplate {
    detail: CountryDetailView,
}

#[derive(Template)]
#[template(path = "country_create.html")]
struct CountryCreateTemplate {
    view: CountryCreateView,
}

#[derive(Template)]
#[template(path = "facet_tag_create.html")]
struct FacetTagCreateTemplate {
    view: FacetTagCreateView,
    facet_tag_type_options: Vec<FacetTagTypeOptionView>,
}

#[derive(Template)]
#[template(path = "admin_login.html")]
struct AdminLoginTemplate {
    error: String,
    has_error: bool,
}

#[derive(Debug, Deserialize)]
struct AdminLoginForm {
    passphrase: String,
}

#[derive(Debug, Deserialize, Serialize)]
struct AdminSessionClaims {
    exp: usize,
    iat: usize,
    iss: String,
    sub: String,
    #[serde(default)]
    admin: bool,
}

#[tokio::main]
async fn main() {
    if let Err(error) = run().await {
        eprintln!("failed to start admin server: {error:#}");
        std::process::exit(1);
    }
}

async fn run() -> Result<()> {
    let cfg = load_config().context("failed to load config")?;
    let server = build_server(&cfg).await?;
    let auth = Arc::new(AuthState::from_config(&cfg));
    let app_state = AppState {
        server: Arc::clone(&server),
        auth: Arc::clone(&auth),
    };

    let public_router = Router::new()
        .route("/", get(handle_root))
        .route(
            "/admin-login",
            get(handle_admin_login_page).post(handle_admin_login_submit),
        )
        .nest_service("/admin/static", ServeDir::new("static"));

    let protected_router = Router::new()
        .route("/admin", get(handle_admin_root))
        .route("/admin/orders", get(handle_orders_page))
        .route("/admin/orders/list", get(handle_orders_list))
        .route("/admin/orders/{order_id}", get(handle_order_detail))
        .route(
            "/admin/orders/{order_id}/status",
            patch(handle_order_status_patch),
        )
        .route(
            "/admin/orders/{order_id}/shipping",
            patch(handle_order_shipping_patch),
        )
        .route(
            "/admin/materials",
            get(handle_materials_page).post(handle_material_create),
        )
        .route(
            "/admin/materials/photo-upload",
            axum::routing::post(handle_material_photo_upload),
        )
        .route("/admin/materials/list", get(handle_materials_list))
        .route("/admin/materials/new", get(handle_material_create_page))
        .route(
            "/admin/materials/{material_key}/edit",
            get(handle_material_edit_page),
        )
        .route(
            "/admin/materials/{material_key}",
            get(handle_material_detail)
                .patch(handle_material_patch)
                .delete(handle_material_delete),
        )
        .route(
            "/admin/stone-listings",
            get(handle_stone_listings_page).post(handle_stone_listing_create),
        )
        .route(
            "/admin/stone-listings/photo-upload",
            axum::routing::post(handle_stone_listing_photo_upload),
        )
        .route(
            "/admin/stone-listings/list",
            get(handle_stone_listings_list),
        )
        .route(
            "/admin/stone-listings/new",
            get(handle_stone_listing_create_page),
        )
        .route(
            "/admin/stone-listings/{stone_listing_key}/edit",
            get(handle_stone_listing_edit_page),
        )
        .route(
            "/admin/stone-listings/{stone_listing_key}",
            get(handle_stone_listing_detail)
                .patch(handle_stone_listing_patch)
                .delete(handle_stone_listing_delete),
        )
        .route(
            "/admin/fonts",
            get(handle_fonts_page).post(handle_font_create),
        )
        .route("/admin/fonts/list", get(handle_fonts_list))
        .route("/admin/fonts/new", get(handle_font_create_page))
        .route("/admin/fonts/{font_key}/edit", get(handle_font_edit_page))
        .route(
            "/admin/fonts/{font_key}",
            get(handle_font_detail).patch(handle_font_patch),
        )
        .route(
            "/admin/countries",
            get(handle_countries_page).post(handle_country_create),
        )
        .route("/admin/countries/list", get(handle_countries_list))
        .route(
            "/admin/countries/{country_code}",
            get(handle_country_detail)
                .patch(handle_country_patch)
                .delete(handle_country_delete),
        )
        .route(
            "/admin/facet-tags",
            get(handle_facet_tags_page).post(handle_facet_tag_create),
        )
        .route("/admin/facet-tags/list", get(handle_facet_tags_list))
        .route(
            "/admin/facet-tags/{facet_tag_id}",
            get(handle_facet_tag_detail)
                .patch(handle_facet_tag_patch)
                .delete(handle_facet_tag_delete),
        )
        .layer(middleware::from_fn_with_state(
            Arc::clone(&auth),
            require_admin_session,
        ));

    let app = public_router.merge(protected_router).with_state(app_state);

    let bind_addr = normalize_bind_addr(&cfg.http_addr);
    let listener = TcpListener::bind(&bind_addr)
        .await
        .with_context(|| format!("failed to bind {bind_addr}"))?;

    let public_url = format!("http://localhost{}", cfg.http_addr);
    if let Some(project_id) = cfg.firestore_project_id.as_deref() {
        println!(
            "hanko admin listening on {public_url} mode={} source={} project={}",
            cfg.mode.as_str(),
            server.source_label,
            project_id
        );
    } else {
        println!(
            "hanko admin listening on {public_url} mode={} source={}",
            cfg.mode.as_str(),
            server.source_label
        );
    }

    axum::serve(listener, app)
        .with_graceful_shutdown(async {
            let _ = tokio::signal::ctrl_c().await;
        })
        .await
        .context("admin server terminated unexpectedly")
}

fn normalize_bind_addr(http_addr: &str) -> String {
    let trimmed = http_addr.trim();
    if trimmed.starts_with(':') {
        format!("0.0.0.0{trimmed}")
    } else if trimmed.contains(':') {
        trimmed.to_owned()
    } else {
        format!("0.0.0.0:{trimmed}")
    }
}

fn load_config() -> Result<AppConfig> {
    let mut cfg = AppConfig {
        http_addr: env_first(&["ADMIN_HTTP_ADDR", "PORT"]),
        mode: RunMode::Mock,
        locale: env::var("HANKO_ADMIN_LOCALE")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        default_locale: env::var("HANKO_ADMIN_DEFAULT_LOCALE")
            .unwrap_or_default()
            .trim()
            .to_owned(),
        firestore_project_id: None,
        storage_assets_bucket: None,
        credentials_file: None,
        login_passphrase: env::var("HANKO_ADMIN_LOGIN_PASSPHRASE")
            .or_else(|_| env::var("HANKO_ADMIN_LOGIN_PASSPHRASE_PROD"))
            .ok()
            .map(|value| value.trim().to_owned())
            .filter(|value| !value.is_empty()),
    };

    if cfg.http_addr.is_empty() {
        cfg.http_addr = ":3051".to_owned();
    }
    if cfg.locale.is_empty() {
        cfg.locale = "ja".to_owned();
    }
    if cfg.default_locale.is_empty() {
        cfg.default_locale = "ja".to_owned();
    }

    let mut mode_value = env_first(&["HANKO_ADMIN_MODE", "HANKO_ADMIN_ENV"]).to_lowercase();
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
        _ => bail!("invalid HANKO_ADMIN_MODE {mode_value:?}: use mock, dev, or prod"),
    }

    if matches!(cfg.mode, RunMode::Prod) {
        if cfg.login_passphrase.is_none() {
            bail!("prod admin login requires HANKO_ADMIN_LOGIN_PASSPHRASE[_PROD]");
        }
    }

    let (project_id_keys, credential_keys, storage_bucket_keys): (&[&str], &[&str], &[&str]) =
        match cfg.mode {
            RunMode::Dev => (
                &[
                    "HANKO_ADMIN_FIREBASE_PROJECT_ID_DEV",
                    "HANKO_ADMIN_FIREBASE_PROJECT_ID",
                    "FIRESTORE_PROJECT_ID",
                    "FIREBASE_PROJECT_ID",
                    "GOOGLE_CLOUD_PROJECT",
                ],
                &[
                    "HANKO_ADMIN_FIREBASE_CREDENTIALS_FILE_DEV",
                    "HANKO_ADMIN_FIREBASE_CREDENTIALS_FILE",
                    "GOOGLE_APPLICATION_CREDENTIALS",
                ],
                &[
                    "HANKO_ADMIN_STORAGE_ASSETS_BUCKET_DEV",
                    "HANKO_ADMIN_STORAGE_ASSETS_BUCKET",
                    "API_STORAGE_ASSETS_BUCKET",
                ],
            ),
            RunMode::Prod => (
                &[
                    "HANKO_ADMIN_FIREBASE_PROJECT_ID_PROD",
                    "HANKO_ADMIN_FIREBASE_PROJECT_ID",
                    "FIRESTORE_PROJECT_ID",
                    "FIREBASE_PROJECT_ID",
                    "GOOGLE_CLOUD_PROJECT",
                ],
                &[
                    "HANKO_ADMIN_FIREBASE_CREDENTIALS_FILE_PROD",
                    "HANKO_ADMIN_FIREBASE_CREDENTIALS_FILE",
                    "GOOGLE_APPLICATION_CREDENTIALS",
                ],
                &[
                    "HANKO_ADMIN_STORAGE_ASSETS_BUCKET_PROD",
                    "HANKO_ADMIN_STORAGE_ASSETS_BUCKET",
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

    let credentials_file = env_first(credential_keys);
    if !credentials_file.is_empty() {
        cfg.credentials_file = Some(credentials_file);
    }

    let storage_assets_bucket = env_first(storage_bucket_keys);
    if !storage_assets_bucket.is_empty() {
        cfg.storage_assets_bucket = Some(storage_assets_bucket);
    } else if matches!(cfg.mode, RunMode::Dev | RunMode::Mock) {
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

async fn build_server(cfg: &AppConfig) -> Result<Arc<ServerState>> {
    let source = match cfg.mode {
        RunMode::Mock => DataSource::Mock,
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

            let project_id = cfg
                .firestore_project_id
                .clone()
                .context("firestore project id is empty")?;

            DataSource::Firestore(FirestoreAdminSource {
                parent: format!("projects/{project_id}/databases/(default)/documents"),
                default_locale: cfg.default_locale.clone(),
                label,
                storage_assets_bucket: cfg.storage_assets_bucket.clone().unwrap_or_default(),
                token_provider,
            })
        }
    };

    let snapshot = if source.is_mock() {
        new_mock_snapshot()
    } else {
        tokio::time::timeout(ADMIN_SOURCE_LOAD_TIMEOUT, source.load_snapshot())
            .await
            .context("admin data load timed out after 20s")??
    };

    Ok(Arc::new(ServerState {
        source_label: source.label().to_owned(),
        storage_assets_bucket: cfg.storage_assets_bucket.clone().unwrap_or_default(),
        source,
        data: RwLock::new(snapshot),
    }))
}

impl AuthState {
    fn from_config(cfg: &AppConfig) -> Self {
        let enabled = matches!(cfg.mode, RunMode::Prod);
        Self {
            enabled,
            login_passphrase: cfg.login_passphrase.clone().unwrap_or_default(),
        }
    }

    async fn login_session_cookie(&self, passphrase: &str) -> Result<String> {
        if !self.enabled {
            return Ok(String::new());
        }

        if passphrase.trim() != self.login_passphrase {
            bail!("invalid admin login passphrase");
        }

        let now = Utc::now().timestamp() as usize;
        let claims = AdminSessionClaims {
            exp: (now as i64 + ADMIN_SESSION_DURATION_SECONDS) as usize,
            iat: now,
            iss: ADMIN_SESSION_ISSUER.to_owned(),
            sub: "admin".to_owned(),
            admin: true,
        };

        encode(
            &Header::new(Algorithm::HS256),
            &claims,
            &EncodingKey::from_secret(self.login_passphrase.as_bytes()),
        )
        .context("failed to sign admin session cookie")
    }

    async fn is_authenticated(&self, headers: &HeaderMap) -> Result<bool> {
        if !self.enabled {
            return Ok(true);
        }

        let Some(cookie_value) = extract_cookie_value(headers, ADMIN_SESSION_COOKIE_NAME) else {
            return Ok(false);
        };

        match self.verify_session_cookie(&cookie_value).await {
            Ok(claims) => Ok(claims.admin),
            Err(error) => {
                eprintln!("failed to verify admin session cookie: {error:#}");
                Ok(false)
            }
        }
    }

    async fn verify_session_cookie(&self, cookie_value: &str) -> Result<AdminSessionClaims> {
        let decoding_key = DecodingKey::from_secret(self.login_passphrase.as_bytes());
        let mut validation = Validation::new(Algorithm::HS256);
        validation.validate_exp = true;
        validation.set_issuer(&[ADMIN_SESSION_ISSUER]);

        let token = decode::<AdminSessionClaims>(cookie_value, &decoding_key, &validation)
            .context("failed to verify admin session cookie")?;
        if !token.claims.admin {
            bail!("admin session cookie is missing admin claim");
        }

        Ok(token.claims)
    }
}

async fn handle_root() -> Redirect {
    Redirect::to("/admin-login")
}

async fn handle_admin_login_page(State(state): State<AppState>, headers: HeaderMap) -> Response {
    if state.auth.is_authenticated(&headers).await.unwrap_or(false) {
        return Redirect::to("/admin/orders").into_response();
    }

    render_admin_login_page("")
}

async fn handle_admin_login_submit(
    State(state): State<AppState>,
    headers: HeaderMap,
    form: Result<Form<AdminLoginForm>, FormRejection>,
) -> Response {
    if state.auth.is_authenticated(&headers).await.unwrap_or(false) {
        return Redirect::to("/admin/orders").into_response();
    }

    let form = match form {
        Ok(Form(form)) => form,
        Err(error) => {
            return plain_error(
                StatusCode::BAD_REQUEST,
                format!("invalid admin login form: {error}"),
            );
        }
    };

    let session_cookie = match state.auth.login_session_cookie(&form.passphrase).await {
        Ok(cookie) => cookie,
        Err(error) => {
            eprintln!("admin login failed: {error:#}");
            return render_admin_login_page("パスフレーズが正しくありません。");
        }
    };

    let cookie_header = format!(
        "{name}={value}; Path=/; HttpOnly; Secure; SameSite=Lax; Max-Age={max_age}",
        name = ADMIN_SESSION_COOKIE_NAME,
        value = session_cookie,
        max_age = ADMIN_SESSION_DURATION_SECONDS
    );

    (
        [(header::SET_COOKIE, cookie_header)],
        Redirect::to("/admin/orders"),
    )
        .into_response()
}

async fn require_admin_session(
    State(auth): State<Arc<AuthState>>,
    headers: HeaderMap,
    request: axum::extract::Request,
    next: middleware::Next,
) -> Response {
    if !auth.enabled {
        return next.run(request).await;
    }

    if auth.is_authenticated(&headers).await.unwrap_or(false) {
        return next.run(request).await;
    }

    redirect_to_admin_login(headers.contains_key("hx-request"))
}

fn render_admin_login_page(error: &str) -> Response {
    let template = AdminLoginTemplate {
        error: error.to_owned(),
        has_error: !error.trim().is_empty(),
    };

    match template.render() {
        Ok(html) => (
            StatusCode::OK,
            [("content-type", "text/html; charset=utf-8")],
            html,
        )
            .into_response(),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render admin login page: {error}"),
        ),
    }
}

fn redirect_to_admin_login(hx_request: bool) -> Response {
    if hx_request {
        (
            StatusCode::UNAUTHORIZED,
            [(
                header::HeaderName::from_static(HX_REDIRECT_HEADER),
                HeaderValue::from_static("/admin-login"),
            )],
            "",
        )
            .into_response()
    } else {
        Redirect::to("/admin-login").into_response()
    }
}

fn extract_cookie_value(headers: &HeaderMap, cookie_name: &str) -> Option<String> {
    let cookie_header = headers.get(header::COOKIE)?.to_str().ok()?;
    for cookie in cookie_header.split(';') {
        let (name, value) = cookie.split_once('=')?;
        if name.trim() == cookie_name {
            return Some(value.trim().to_owned());
        }
    }

    None
}

async fn handle_admin_root() -> Redirect {
    Redirect::to("/admin/orders")
}

async fn handle_orders_page(
    State(state): State<AppState>,
    Query(query): Query<OrdersPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load orders: {error}"),
        );
    }

    let filters = OrderFilter {
        status: normalize_query_value(query.status),
        country: normalize_query_value(query.country),
        email: normalize_query_value(query.email),
    };

    let orders = state.server.filter_orders(&filters).await;
    let font_stylesheet_urls = state.server.font_stylesheet_urls().await;
    let orders_list_html = match render_orders_list(&orders) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render orders list: {error}"),
            );
        }
    };

    let mut selected_order_id = normalize_query_value(query.order_id);
    if selected_order_id.is_empty() {
        if let Some(first) = orders.first() {
            selected_order_id = first.id.clone();
        }
    }

    let order_detail = if selected_order_id.is_empty() {
        None
    } else {
        state
            .server
            .get_order_detail(&selected_order_id, "", "")
            .await
    };

    let order_detail_html = if let Some(detail) = order_detail.as_ref() {
        match render_order_detail(detail) {
            Ok(html) => html,
            Err(error) => {
                return plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render order detail: {error}"),
                );
            }
        }
    } else {
        String::new()
    };

    let page = OrdersPageTemplate {
        filters,
        status_options: status_options(),
        country_options: state.server.country_options().await,
        font_stylesheet_urls,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        orders_list_html,
        order_detail_html,
        has_order_detail: order_detail.is_some(),
    };

    render_template(&page)
}

async fn handle_orders_list(
    State(state): State<AppState>,
    Query(query): Query<OrdersPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load orders: {error}"),
        );
    }

    let filters = OrderFilter {
        status: normalize_query_value(query.status),
        country: normalize_query_value(query.country),
        email: normalize_query_value(query.email),
    };

    let orders = state.server.filter_orders(&filters).await;
    match render_orders_list(&orders) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render orders list: {error}"),
        ),
    }
}

async fn handle_order_detail(
    State(state): State<AppState>,
    Path(order_id): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load orders: {error}"),
        );
    }

    let Some(detail) = state.server.get_order_detail(&order_id, "", "").await else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_order_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render order detail: {error}"),
        ),
    }
}

async fn handle_order_status_patch(
    State(state): State<AppState>,
    Path(order_id): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load orders: {error}"),
        );
    }

    let next_status = form_value(&form, "next_status");
    let actor_id = {
        let value = form_value(&form, "actor_id");
        if value.is_empty() {
            "admin.console".to_owned()
        } else {
            value
        }
    };

    match state
        .server
        .update_order_status(&order_id, &next_status, &actor_id)
        .await
    {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_order_detail(&order_id, "ステータスを更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_order_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "order-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render order detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_order_detail(&order_id, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_order_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render order detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_order_shipping_patch(
    State(state): State<AppState>,
    Path(order_id): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load orders: {error}"),
        );
    }

    let carrier = form_value(&form, "carrier");
    let tracking_no = form_value(&form, "tracking_no");
    let transition = form_value(&form, "shipping_transition");
    let actor_id = {
        let value = form_value(&form, "actor_id");
        if value.is_empty() {
            "admin.console".to_owned()
        } else {
            value
        }
    };

    match state
        .server
        .update_shipping(&order_id, &carrier, &tracking_no, &transition, &actor_id)
        .await
    {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_order_detail(&order_id, "出荷情報を更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_order_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "order-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render order detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_order_detail(&order_id, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_order_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render order detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_materials_page(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let materials = state.server.list_materials().await;
    let font_stylesheet_urls = state.server.font_stylesheet_urls().await;
    let materials_list_html = match render_materials_list(&materials) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render materials list: {error}"),
            );
        }
    };

    let page = MaterialsPageTemplate {
        font_stylesheet_urls,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        materials_list_html,
    };

    render_template(&page)
}

async fn handle_materials_list(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let materials = state.server.list_materials().await;
    match render_materials_list(&materials) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render materials list: {error}"),
        ),
    }
}

async fn handle_material_create_page(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let material_create_html = match render_material_create(&new_material_create_view("", "")) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render material create: {error}"),
            );
        }
    };

    let page = MaterialCreatePageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        material_create_html,
    };

    render_template(&page)
}

async fn handle_material_edit_page(
    State(state): State<AppState>,
    Path(material_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_material_detail(&material_key, "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    let material_detail_html = match render_material_detail(&detail) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render material detail: {error}"),
            );
        }
    };

    let page = MaterialEditPageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        material_key,
        material_detail_html,
    };

    render_template(&page)
}

async fn handle_material_detail(
    State(state): State<AppState>,
    Path(material_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_material_detail(&material_key, "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_material_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render material detail: {error}"),
        ),
    }
}

async fn handle_material_photo_upload(
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return json_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("failed to load materials: {error}"),
        );
    }

    let mut material_key = String::new();
    let mut file_name = String::new();
    let mut content_type: Option<String> = None;
    let mut bytes = Vec::new();

    loop {
        let next = match multipart.next_field().await {
            Ok(next) => next,
            Err(error) => {
                return json_error(
                    StatusCode::BAD_REQUEST,
                    &format!("invalid multipart: {error}"),
                );
            }
        };

        let Some(field) = next else {
            break;
        };

        let Some(name) = field.name() else {
            continue;
        };

        match name {
            "material_key" => {
                material_key = match field.text().await {
                    Ok(text) => text.trim().to_owned(),
                    Err(error) => {
                        return json_error(
                            StatusCode::BAD_REQUEST,
                            &format!("invalid material key: {error}"),
                        );
                    }
                };
            }
            "photo_file" => {
                if let Some(value) = field.file_name() {
                    file_name = value.to_owned();
                }
                content_type = field.content_type().map(ToOwned::to_owned);

                let payload = match field.bytes().await {
                    Ok(payload) => payload,
                    Err(error) => {
                        return json_error(
                            StatusCode::BAD_REQUEST,
                            &format!("failed to read upload file: {error}"),
                        );
                    }
                };

                bytes = payload.to_vec();
            }
            _ => {}
        }
    }

    if material_key.is_empty() {
        return json_error(
            StatusCode::BAD_REQUEST,
            "材質キーを入力してから画像をアップロードしてください。",
        );
    }

    if bytes.is_empty() {
        return json_error(StatusCode::BAD_REQUEST, "画像ファイルを選択してください。");
    }

    match state
        .server
        .upload_material_photo(&material_key, &file_name, content_type.as_deref(), &bytes)
        .await
    {
        Ok(storage_path) => json_response(
            StatusCode::OK,
            json!({
                "storage_path": storage_path,
            }),
        ),
        Err(error_message) => json_error(StatusCode::BAD_REQUEST, &error_message),
    }
}

async fn handle_material_create(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let input = MaterialCreateInput {
        key: form_value(&form, "key"),
        label_ja: form_value(&form, "label_ja"),
        label_en: form_value(&form, "label_en"),
        description_ja: form_value(&form, "description_ja"),
        description_en: form_value(&form, "description_en"),
        is_active: form.contains_key("is_active"),
    };
    let created_key = input.key.clone();

    match state.server.create_material(input).await {
        Ok(()) => render_material_create_response_with_trigger(
            StatusCode::CREATED,
            &new_material_create_view(&format!("材質「{created_key}」を作成しました。"), ""),
            "material-updated",
        ),
        Err(error_message) => render_material_create_response(
            StatusCode::BAD_REQUEST,
            &material_create_view_from_form(&form, "", &error_message),
        ),
    }
}

async fn handle_material_delete(
    State(state): State<AppState>,
    Path(material_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    match state.server.delete_material(&material_key).await {
        Ok(()) => {
            let materials = state.server.list_materials().await;
            match render_materials_list(&materials) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "material-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render materials list: {error}"),
                ),
            }
        }
        Err(error_message) => plain_error(StatusCode::BAD_REQUEST, error_message),
    }
}

async fn handle_material_patch(
    State(state): State<AppState>,
    Path(material_key): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load materials: {error}"),
        );
    }

    let input = MaterialPatchInput {
        label_ja: form_value(&form, "label_ja"),
        label_en: form_value(&form, "label_en"),
        description_ja: form_value(&form, "description_ja"),
        description_en: form_value(&form, "description_en"),
        label_i18n_updates: parse_localized_form_updates(&form, "label_i18n"),
        description_i18n_updates: parse_localized_form_updates(&form, "description_i18n"),
        is_active: form.contains_key("is_active"),
    };

    match state.server.update_material(&material_key, input).await {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_material_detail(&material_key, "材質マスタを更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_material_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "material-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render material detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_material_detail(&material_key, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_material_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render material detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_stone_listings_page(
    State(state): State<AppState>,
    Query(query): Query<StoneListingsPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let filters = stone_listing_filter_from_query(query);
    let filters = state.server.normalize_stone_listing_filter(&filters).await;
    let stone_listings = state.server.filter_stone_listings(&filters).await;
    let filter_options = state.server.stone_listing_filter_options().await;
    let tag_options = state.server.stone_listing_tag_options().await;
    let stone_listings_list_html = match render_stone_listings_list(&stone_listings, &filters) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render stone listings list: {error}"),
            );
        }
    };

    let page = StoneListingsPageTemplate {
        filters,
        color_family_options: filter_options.color_family_options,
        pattern_primary_options: filter_options.pattern_primary_options,
        stone_shape_options: filter_options.stone_shape_options,
        color_tag_options: tag_options.color_tag_options,
        pattern_tag_options: tag_options.pattern_tag_options,
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        stone_listings_list_html,
    };

    render_template(&page)
}

async fn handle_stone_listings_list(
    State(state): State<AppState>,
    Query(query): Query<StoneListingsPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let filters = stone_listing_filter_from_query(query);
    let filters = state.server.normalize_stone_listing_filter(&filters).await;
    let stone_listings = state.server.filter_stone_listings(&filters).await;
    match render_stone_listings_list(&stone_listings, &filters) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render stone listings list: {error}"),
        ),
    }
}

async fn handle_stone_listing_create_page(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let tag_options = state.server.stone_listing_tag_options().await;
    let stone_listing_create_html =
        match render_stone_listing_create(&new_stone_listing_create_view("", "")) {
            Ok(html) => html,
            Err(error) => {
                return plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing create: {error}"),
                );
            }
        };

    let page = StoneListingCreatePageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        color_tag_options: tag_options.color_tag_options,
        pattern_tag_options: tag_options.pattern_tag_options,
        stone_listing_create_html,
    };

    render_template(&page)
}

async fn handle_stone_listing_edit_page(
    State(state): State<AppState>,
    Path(stone_listing_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_stone_listing_detail(&stone_listing_key, "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    let stone_listing_detail_html = match render_stone_listing_detail(&detail) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render stone listing detail: {error}"),
            );
        }
    };

    let tag_options = state.server.stone_listing_tag_options().await;
    let page = StoneListingEditPageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        color_tag_options: tag_options.color_tag_options,
        pattern_tag_options: tag_options.pattern_tag_options,
        stone_listing_key,
        stone_listing_detail_html,
    };

    render_template(&page)
}

async fn handle_stone_listing_detail(
    State(state): State<AppState>,
    Path(stone_listing_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_stone_listing_detail(&stone_listing_key, "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_stone_listing_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render stone listing detail: {error}"),
        ),
    }
}

async fn handle_stone_listing_photo_upload(
    State(state): State<AppState>,
    mut multipart: Multipart,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return json_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            &format!("failed to load stone listings: {error}"),
        );
    }

    let mut stone_listing_key = String::new();
    let mut file_name = String::new();
    let mut content_type: Option<String> = None;
    let mut bytes = Vec::new();

    loop {
        let next = match multipart.next_field().await {
            Ok(next) => next,
            Err(error) => {
                return json_error(
                    StatusCode::BAD_REQUEST,
                    &format!("invalid multipart: {error}"),
                );
            }
        };

        let Some(field) = next else {
            break;
        };

        let Some(name) = field.name() else {
            continue;
        };

        match name {
            "stone_listing_key" => {
                stone_listing_key = match field.text().await {
                    Ok(text) => text.trim().to_owned(),
                    Err(error) => {
                        return json_error(
                            StatusCode::BAD_REQUEST,
                            &format!("invalid stone listing key: {error}"),
                        );
                    }
                };
            }
            "photo_file" => {
                if let Some(value) = field.file_name() {
                    file_name = value.to_owned();
                }
                content_type = field.content_type().map(ToOwned::to_owned);

                let payload = match field.bytes().await {
                    Ok(payload) => payload,
                    Err(error) => {
                        return json_error(
                            StatusCode::BAD_REQUEST,
                            &format!("failed to read upload file: {error}"),
                        );
                    }
                };

                bytes = payload.to_vec();
            }
            _ => {}
        }
    }

    if stone_listing_key.is_empty() {
        return json_error(
            StatusCode::BAD_REQUEST,
            "一点物キーを入力してから画像をアップロードしてください。",
        );
    }

    if bytes.is_empty() {
        return json_error(StatusCode::BAD_REQUEST, "画像ファイルを選択してください。");
    }

    match state
        .server
        .upload_stone_listing_photo(
            &stone_listing_key,
            &file_name,
            content_type.as_deref(),
            &bytes,
        )
        .await
    {
        Ok(storage_path) => json_response(
            StatusCode::OK,
            json!({
                "storage_path": storage_path,
            }),
        ),
        Err(error_message) => json_error(StatusCode::BAD_REQUEST, &error_message),
    }
}

async fn handle_stone_listing_create(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let price_usd = match parse_usd_cents_input(&form_value(&form, "price_usd")) {
        Ok(value) => value,
        Err(_) => {
            return render_stone_listing_create_response(
                StatusCode::BAD_REQUEST,
                &stone_listing_create_view_from_form(
                    &form,
                    "",
                    "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。",
                ),
            );
        }
    };

    let price_jpy = match form_value(&form, "price_jpy").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_stone_listing_create_response(
                StatusCode::BAD_REQUEST,
                &stone_listing_create_view_from_form(
                    &form,
                    "",
                    "価格（JPY）は整数で入力してください。",
                ),
            );
        }
    };

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_stone_listing_create_response(
                StatusCode::BAD_REQUEST,
                &stone_listing_create_view_from_form(&form, "", "表示順は整数で入力してください。"),
            );
        }
    };

    let input = stone_listing_create_input_from_form(&form, price_usd, price_jpy, sort_order);
    let created_key = input.stone_listing_key.clone();

    match state.server.create_stone_listing(input).await {
        Ok(()) => render_stone_listing_create_response_with_trigger(
            StatusCode::CREATED,
            &new_stone_listing_create_view(&format!("一点物「{created_key}」を作成しました。"), ""),
            "stone-listing-updated",
        ),
        Err(error_message) => render_stone_listing_create_response(
            StatusCode::BAD_REQUEST,
            &stone_listing_create_view_from_form(&form, "", &error_message),
        ),
    }
}

async fn handle_stone_listing_delete(
    State(state): State<AppState>,
    headers: HeaderMap,
    Path(stone_listing_key): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let hx_target = headers
        .get("hx-target")
        .and_then(|value| value.to_str().ok())
        .map(str::trim)
        .unwrap_or("");
    let is_list_target = hx_target == "stone-listings-list" || hx_target == "#stone-listings-list";

    let form = match form {
        Ok(Form(form)) => form,
        Err(_) if is_list_target => {
            return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned());
        }
        Err(_) => HashMap::new(),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    match state.server.delete_stone_listing(&stone_listing_key).await {
        Ok(()) => {
            if !is_list_target {
                return html_response_with_trigger(
                    StatusCode::OK,
                    "<p class=\"p-5 text-sm font-semibold text-admin-muted\">一点物を削除しました。</p>"
                        .to_owned(),
                    "stone-listing-updated",
                );
            }

            let filters = stone_listing_filter_from_form(&form);
            let filters = state.server.normalize_stone_listing_filter(&filters).await;
            let stone_listings = state.server.filter_stone_listings(&filters).await;
            match render_stone_listings_list(&stone_listings, &filters) {
                Ok(html) => html_response(StatusCode::OK, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listings list: {error}"),
                ),
            }
        }
        Err(error_message) => plain_error(StatusCode::BAD_REQUEST, error_message),
    }
}

async fn handle_stone_listing_patch(
    State(state): State<AppState>,
    Path(stone_listing_key): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load stone listings: {error}"),
        );
    }

    let price_usd = match parse_usd_cents_input(&form_value(&form, "price_usd")) {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_stone_listing_detail(
                    &stone_listing_key,
                    "",
                    "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_stone_listing_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing detail: {error}"),
                ),
            };
        }
    };

    let price_jpy = match form_value(&form, "price_jpy").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_stone_listing_detail(
                    &stone_listing_key,
                    "",
                    "価格（JPY）は整数で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_stone_listing_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing detail: {error}"),
                ),
            };
        }
    };

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_stone_listing_detail(
                    &stone_listing_key,
                    "",
                    "表示順は整数で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_stone_listing_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing detail: {error}"),
                ),
            };
        }
    };

    let input = stone_listing_patch_input_from_form(&form, price_usd, price_jpy, sort_order);

    match state
        .server
        .update_stone_listing(&stone_listing_key, input)
        .await
    {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_stone_listing_detail(&stone_listing_key, "一点物を更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_stone_listing_detail(&detail) {
                Ok(html) => {
                    html_response_with_trigger(StatusCode::OK, html, "stone-listing-updated")
                }
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_stone_listing_detail(&stone_listing_key, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_stone_listing_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render stone listing detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_fonts_page(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let fonts = state.server.list_fonts().await;
    let font_stylesheet_urls = state.server.font_stylesheet_urls().await;
    let fonts_list_html = match render_fonts_list(&fonts) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render fonts list: {error}"),
            );
        }
    };

    let page = FontsPageTemplate {
        font_stylesheet_urls,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        fonts_list_html,
    };

    render_template(&page)
}

async fn handle_fonts_list(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let fonts = state.server.list_fonts().await;
    match render_fonts_list(&fonts) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render fonts list: {error}"),
        ),
    }
}

async fn handle_font_create_page(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let font_create_html = match render_font_create(&new_font_create_view("", "")) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render font create: {error}"),
            );
        }
    };

    let page = FontCreatePageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        font_create_html,
    };

    render_template(&page)
}

async fn handle_font_create(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_font_create_response(
                StatusCode::BAD_REQUEST,
                &font_create_view_from_form(&form, "", "表示順は整数で入力してください。"),
            );
        }
    };

    let input = FontCreateInput {
        key: form_value(&form, "key"),
        label: form_value(&form, "label"),
        font_family: form_value(&form, "font_family"),
        kanji_style: form_value(&form, "kanji_style"),
        sort_order,
        is_active: form.contains_key("is_active"),
    };
    let created_key = input.key.clone();

    match state.server.create_font(input).await {
        Ok(()) => render_font_create_response_with_trigger(
            StatusCode::CREATED,
            &new_font_create_view(&format!("フォント「{created_key}」を作成しました。"), ""),
            "font-updated",
        ),
        Err(error_message) => render_font_create_response(
            StatusCode::BAD_REQUEST,
            &font_create_view_from_form(&form, "", &error_message),
        ),
    }
}

async fn handle_font_edit_page(
    State(state): State<AppState>,
    Path(font_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let Some(detail) = state.server.get_font_detail(&font_key, "", "").await else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    let font_detail_html = match render_font_detail(&detail) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render font detail: {error}"),
            );
        }
    };

    let page = FontEditPageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        font_key,
        font_detail_html,
    };

    render_template(&page)
}

async fn handle_font_detail(
    State(state): State<AppState>,
    Path(font_key): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let Some(detail) = state.server.get_font_detail(&font_key, "", "").await else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_font_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render font detail: {error}"),
        ),
    }
}

async fn handle_font_patch(
    State(state): State<AppState>,
    Path(font_key): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load fonts: {error}"),
        );
    }

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_font_detail(&font_key, "", "表示順は整数で入力してください。")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_font_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render font detail: {error}"),
                ),
            };
        }
    };

    let input = FontPatchInput {
        label: form_value(&form, "label"),
        font_family: form_value(&form, "font_family"),
        kanji_style: form_value(&form, "kanji_style"),
        sort_order,
        is_active: form.contains_key("is_active"),
    };

    match state.server.update_font(&font_key, input).await {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_font_detail(&font_key, "フォントマスタを更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_font_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "font-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render font detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_font_detail(&font_key, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_font_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render font detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_countries_page(
    State(state): State<AppState>,
    Query(query): Query<CountriesPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    let countries = state.server.list_countries().await;
    let country_create_html = match render_country_create(&new_country_create_view("", "")) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render country create: {error}"),
            );
        }
    };
    let countries_list_html = match render_countries_list(&countries) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render countries list: {error}"),
            );
        }
    };

    let mut selected_country_code = normalize_query_value(query.country_code).to_uppercase();
    if selected_country_code.is_empty() {
        if let Some(first) = countries.first() {
            selected_country_code = first.code.clone();
        }
    }

    let country_detail = if selected_country_code.is_empty() {
        None
    } else {
        state
            .server
            .get_country_detail(&selected_country_code, "", "")
            .await
    };

    let country_detail_html = if let Some(detail) = country_detail.as_ref() {
        match render_country_detail(detail) {
            Ok(html) => html,
            Err(error) => {
                return plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                );
            }
        }
    } else {
        String::new()
    };

    let page = CountriesPageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        country_create_html,
        countries_list_html,
        country_detail_html,
        has_country_detail: country_detail.is_some(),
    };

    render_template(&page)
}

async fn handle_facet_tags_page(
    State(state): State<AppState>,
    Query(query): Query<FacetTagsPageQuery>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    let facet_tags = state.server.list_facet_tags().await;
    let facet_tag_create_html =
        match render_facet_tag_create(&new_facet_tag_create_view("", "", "", "")) {
            Ok(html) => html,
            Err(error) => {
                return plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render facet tag create: {error}"),
                );
            }
        };
    let facet_tags_list_html = match render_facet_tags_list(&facet_tags) {
        Ok(html) => html,
        Err(error) => {
            return plain_error(
                StatusCode::INTERNAL_SERVER_ERROR,
                format!("failed to render facet tags list: {error}"),
            );
        }
    };

    let mut selected_facet_tag_id = normalize_query_value(query.facet_tag_id);
    if selected_facet_tag_id.is_empty()
        && let Some(first) = facet_tags.first()
    {
        selected_facet_tag_id = first.id.clone();
    }

    let facet_tag_detail = if selected_facet_tag_id.is_empty() {
        None
    } else {
        state
            .server
            .get_facet_tag_detail(&selected_facet_tag_id, "", "")
            .await
    };

    let facet_tag_detail_html = if let Some(detail) = facet_tag_detail.as_ref() {
        match render_facet_tag_detail(detail) {
            Ok(html) => html,
            Err(error) => {
                return plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render facet tag detail: {error}"),
                );
            }
        }
    } else {
        String::new()
    };

    let page = FacetTagsPageTemplate {
        font_stylesheet_urls: state.server.font_stylesheet_urls().await,
        source_label: state.server.source_label.clone(),
        is_mock: state.server.source.is_mock(),
        facet_tag_create_html,
        facet_tags_list_html,
        facet_tag_detail_html,
        has_facet_tag_detail: facet_tag_detail.is_some(),
    };

    render_template(&page)
}

async fn handle_facet_tags_list(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    let facet_tags = state.server.list_facet_tags().await;
    match render_facet_tags_list(&facet_tags) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render facet tags list: {error}"),
        ),
    }
}

async fn handle_facet_tag_detail(
    State(state): State<AppState>,
    Path(facet_tag_id): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_facet_tag_detail(&facet_tag_id, "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_facet_tag_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render facet tag detail: {error}"),
        ),
    }
}

async fn handle_facet_tag_create(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_facet_tag_create_response(
                StatusCode::BAD_REQUEST,
                &facet_tag_create_view_from_form(
                    &form,
                    "",
                    "表示順は整数で入力してください。",
                    "",
                    "",
                ),
            );
        }
    };

    let input = FacetTagCreateInput {
        key: form_value(&form, "key"),
        facet_type: form_value(&form, "facet_type"),
        label_ja: form_value(&form, "label_ja"),
        label_en: form_value(&form, "label_en"),
        aliases: parse_comma_separated_values(&form_value(&form, "aliases")),
        sort_order,
        is_active: form.contains_key("is_active"),
    };
    let created_key = input.key.clone();

    match state.server.create_facet_tag(input).await {
        Ok(()) => render_facet_tag_create_response_with_trigger(
            StatusCode::CREATED,
            &new_facet_tag_create_view(
                &format!("タグ「{created_key}」を作成しました。"),
                "",
                "",
                "",
            ),
            "facet-tag-updated",
        ),
        Err(error_message) => {
            let (key_error, aliases_error) = facet_tag_field_errors_from_message(&error_message);
            render_facet_tag_create_response(
                StatusCode::BAD_REQUEST,
                &facet_tag_create_view_from_form(
                    &form,
                    "",
                    &error_message,
                    &key_error,
                    &aliases_error,
                ),
            )
        }
    }
}

async fn handle_facet_tag_patch(
    State(state): State<AppState>,
    Path(facet_tag_id): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_facet_tag_detail(&facet_tag_id, "", "表示順は整数で入力してください。")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_facet_tag_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render facet tag detail: {error}"),
                ),
            };
        }
    };

    let input = facet_tag_patch_input_from_form(&form, sort_order);

    match state.server.update_facet_tag(&facet_tag_id, input).await {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_facet_tag_detail(&facet_tag_id, "タグを更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_facet_tag_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "facet-tag-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render facet tag detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let (key_error, aliases_error) = facet_tag_field_errors_from_message(&error_message);
            let Some(mut detail) = state
                .server
                .get_facet_tag_detail(&facet_tag_id, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            detail.key_error = key_error;
            detail.has_key_error = !detail.key_error.is_empty();
            detail.aliases_error = aliases_error;
            detail.has_aliases_error = !detail.aliases_error.is_empty();
            match render_facet_tag_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render facet tag detail: {error}"),
                ),
            }
        }
    }
}

async fn handle_facet_tag_delete(
    State(state): State<AppState>,
    Path(facet_tag_id): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load facet tags: {error}"),
        );
    }

    match state.server.delete_facet_tag(&facet_tag_id).await {
        Ok(()) => html_response_with_trigger(
            StatusCode::OK,
            "<p class=\"p-5 text-sm font-semibold text-admin-muted\">タグを削除しました。</p>"
                .to_owned(),
            "facet-tag-updated",
        ),
        Err(error_message) => plain_error(StatusCode::BAD_REQUEST, error_message),
    }
}

async fn handle_country_create(
    State(state): State<AppState>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    let shipping_fee_usd = match form_value(&form, "shipping_fee_usd").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_country_create_response(
                StatusCode::BAD_REQUEST,
                &country_create_view_from_form(
                    &form,
                    "",
                    "送料（USD cents）は整数で入力してください。",
                ),
            );
        }
    };

    let shipping_fee_jpy = match form_value(&form, "shipping_fee_jpy").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_country_create_response(
                StatusCode::BAD_REQUEST,
                &country_create_view_from_form(&form, "", "送料（JPY）は整数で入力してください。"),
            );
        }
    };

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            return render_country_create_response(
                StatusCode::BAD_REQUEST,
                &country_create_view_from_form(&form, "", "表示順は整数で入力してください。"),
            );
        }
    };

    let input = CountryCreateInput {
        code: form_value(&form, "code"),
        label_ja: form_value(&form, "label_ja"),
        label_en: form_value(&form, "label_en"),
        shipping_fee_usd,
        shipping_fee_jpy,
        sort_order,
        is_active: form.contains_key("is_active"),
    };
    let created_code = input.code.to_uppercase();

    match state.server.create_country(input).await {
        Ok(()) => render_country_create_response_with_trigger(
            StatusCode::CREATED,
            &new_country_create_view(&format!("配送国「{created_code}」を作成しました。"), ""),
            "country-updated",
        ),
        Err(error_message) => render_country_create_response(
            StatusCode::BAD_REQUEST,
            &country_create_view_from_form(&form, "", &error_message),
        ),
    }
}

async fn handle_countries_list(State(state): State<AppState>) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    let countries = state.server.list_countries().await;
    match render_countries_list(&countries) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render countries list: {error}"),
        ),
    }
}

async fn handle_country_delete(
    State(state): State<AppState>,
    Path(country_code): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    match state.server.delete_country(&country_code).await {
        Ok(()) => {
            let countries = state.server.list_countries().await;
            match render_countries_list(&countries) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "country-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render countries list: {error}"),
                ),
            }
        }
        Err(error_message) => plain_error(StatusCode::BAD_REQUEST, error_message),
    }
}

async fn handle_country_detail(
    State(state): State<AppState>,
    Path(country_code): Path<String>,
) -> Response {
    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    let Some(detail) = state
        .server
        .get_country_detail(&country_code.to_uppercase(), "", "")
        .await
    else {
        return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
    };

    match render_country_detail(&detail) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render country detail: {error}"),
        ),
    }
}

async fn handle_country_patch(
    State(state): State<AppState>,
    Path(country_code): Path<String>,
    form: std::result::Result<Form<HashMap<String, String>>, FormRejection>,
) -> Response {
    let Form(form) = match form {
        Ok(form) => form,
        Err(_) => return plain_error(StatusCode::BAD_REQUEST, "invalid request".to_owned()),
    };

    if let Err(error) = state.server.refresh_from_source().await {
        return plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to load countries: {error}"),
        );
    }

    let shipping_fee_usd = match form_value(&form, "shipping_fee_usd").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_country_detail(
                    &country_code.to_uppercase(),
                    "",
                    "送料（USD cents）は整数で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_country_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                ),
            };
        }
    };

    let shipping_fee_jpy = match form_value(&form, "shipping_fee_jpy").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_country_detail(
                    &country_code.to_uppercase(),
                    "",
                    "送料（JPY）は整数で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_country_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                ),
            };
        }
    };

    let sort_order = match form_value(&form, "sort_order").parse::<i64>() {
        Ok(value) => value,
        Err(_) => {
            let Some(detail) = state
                .server
                .get_country_detail(
                    &country_code.to_uppercase(),
                    "",
                    "表示順は整数で入力してください。",
                )
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            return match render_country_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                ),
            };
        }
    };

    let input = CountryPatchInput {
        label_ja: form_value(&form, "label_ja"),
        label_en: form_value(&form, "label_en"),
        label_i18n_updates: parse_localized_form_updates(&form, "label_i18n"),
        shipping_fee_usd,
        shipping_fee_jpy,
        sort_order,
        is_active: form.contains_key("is_active"),
    };

    let normalized_code = country_code.to_uppercase();
    match state.server.update_country(&normalized_code, input).await {
        Ok(()) => {
            let Some(detail) = state
                .server
                .get_country_detail(&normalized_code, "配送国マスタを更新しました。", "")
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_country_detail(&detail) {
                Ok(html) => html_response_with_trigger(StatusCode::OK, html, "country-updated"),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                ),
            }
        }
        Err(error_message) => {
            let Some(detail) = state
                .server
                .get_country_detail(&normalized_code, "", &error_message)
                .await
            else {
                return plain_error(StatusCode::NOT_FOUND, "not found".to_owned());
            };
            match render_country_detail(&detail) {
                Ok(html) => html_response(StatusCode::BAD_REQUEST, html),
                Err(error) => plain_error(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    format!("failed to render country detail: {error}"),
                ),
            }
        }
    }
}

impl ServerState {
    async fn refresh_from_source(&self) -> Result<()> {
        if self.source.is_mock() {
            return Ok(());
        }

        let snapshot = tokio::time::timeout(ADMIN_SOURCE_LOAD_TIMEOUT, self.source.load_snapshot())
            .await
            .context("admin data load timed out after 20s")??;

        let mut data = self.data.write().await;
        *data = snapshot;
        Ok(())
    }

    async fn filter_orders(&self, filters: &OrderFilter) -> Vec<OrderListItemView> {
        let data = self.data.read().await;

        let mut items = Vec::with_capacity(data.order_ids.len());
        for id in &data.order_ids {
            let Some(order) = data.orders.get(id) else {
                continue;
            };

            if !filters.status.is_empty() && order.status != filters.status {
                continue;
            }
            if !filters.country.is_empty()
                && !order.country_code.eq_ignore_ascii_case(&filters.country)
            {
                continue;
            }
            if !filters.email.is_empty()
                && !order
                    .contact_email
                    .to_lowercase()
                    .contains(&filters.email.to_lowercase())
            {
                continue;
            }

            items.push(OrderListItemView {
                id: order.id.clone(),
                order_no: order.order_no.clone(),
                created_at: format_datetime(order.created_at),
                channel_label: order_channel_label(&order.channel),
                is_app_order: order.channel.eq_ignore_ascii_case("app"),
                engraving_text: order.engraving_text.clone(),
                has_engraving_text: !order.engraving_text.is_empty(),
                ai_variant_id: order.ai_variant_id.clone(),
                has_ai_variant_id: !order.ai_variant_id.is_empty(),
                status_label: order_status_label(&order.status).to_owned(),
                payment_status_label: payment_status_label(&order.payment_status).to_owned(),
                fulfillment_status_label: fulfillment_status_label(&order.fulfillment_status)
                    .to_owned(),
                country_label: country_label(&data.countries, &order.country_code),
                total: format_order_amount(order.total, &order.currency),
            });
        }

        items
    }

    async fn get_order_detail(
        &self,
        order_id: &str,
        message: &str,
        render_error: &str,
    ) -> Option<OrderDetailView> {
        let data = self.data.read().await;
        let order = data.orders.get(order_id)?;

        let mut events = order.events.clone();
        events.sort_by(|left, right| right.created_at.cmp(&left.created_at));

        let event_views = events
            .into_iter()
            .map(|event| {
                let has_before_status = !event.before_status.is_empty();
                let has_note = !event.note.is_empty();
                let has_actor_id = !event.actor_id.is_empty();
                OrderEventView {
                    kind: event.kind,
                    actor_type: event.actor_type,
                    actor_id: event.actor_id,
                    has_actor_id,
                    before_status_label: order_status_label(&event.before_status).to_owned(),
                    after_status_label: order_status_label(&event.after_status).to_owned(),
                    has_before_status,
                    note: event.note,
                    has_note,
                    created_at: format_datetime(event.created_at),
                }
            })
            .collect::<Vec<_>>();

        let next_statuses = next_status_options(&order.status);
        let shipping_transitions = shipping_transition_options(&order.status);
        let seal_preview_image_url = build_storage_media_url(
            &self.storage_assets_bucket,
            &order.seal_preview_image_storage_path,
        );
        let has_ai_generation_id = !order.ai_generation_id.is_empty();
        let has_ai_variant_id = !order.ai_variant_id.is_empty();
        let has_seal_style_name = !order.seal_style_name.is_empty();
        let has_seal_style_stroke_weight = !order.seal_style_stroke_weight.is_empty();
        let has_seal_style_balance = !order.seal_style_balance.is_empty();
        let has_seal_style_prompt_summary = !order.seal_style_prompt_summary.is_empty();
        let has_ai_seal_metadata = has_ai_generation_id
            || has_ai_variant_id
            || has_seal_style_name
            || has_seal_style_stroke_weight
            || has_seal_style_balance
            || has_seal_style_prompt_summary;
        let has_customer_confirmation_kanji_and_design =
            order.customer_confirmation_kanji_and_design.is_some();
        let has_customer_confirmation_custom_made_policy =
            order.customer_confirmation_custom_made_policy.is_some();
        let has_customer_confirmation_confirmed_at =
            order.customer_confirmation_confirmed_at.is_some();
        let customer_confirmation_confirmed_at = order
            .customer_confirmation_confirmed_at
            .as_ref()
            .map(|value| format_datetime(value.to_owned()))
            .unwrap_or_default();
        let has_customer_confirmation_confirmed_seal_text =
            !order.customer_confirmation_confirmed_seal_text.is_empty();
        let has_customer_confirmation = has_customer_confirmation_kanji_and_design
            || has_customer_confirmation_custom_made_policy
            || has_customer_confirmation_confirmed_at
            || has_customer_confirmation_confirmed_seal_text;

        Some(OrderDetailView {
            id: order.id.clone(),
            order_no: order.order_no.clone(),
            created_at: format_datetime(order.created_at),
            updated_at: format_datetime(order.updated_at),
            status_label: order_status_label(&order.status).to_owned(),
            payment_status_label: payment_status_label(&order.payment_status).to_owned(),
            fulfillment_status_label: fulfillment_status_label(&order.fulfillment_status)
                .to_owned(),
            tracking_no: order.tracking_no.clone(),
            has_tracking_no: !order.tracking_no.is_empty(),
            carrier: order.carrier.clone(),
            country_code: order.country_code.clone(),
            country_label: country_label(&data.countries, &order.country_code),
            contact_email: order.contact_email.clone(),
            channel: order.channel.clone(),
            locale: order.locale.clone(),
            seal_line1: order.seal_line1.clone(),
            seal_line2: order.seal_line2.clone(),
            has_seal_line2: !order.seal_line2.is_empty(),
            seal_preview_image_storage_path: order.seal_preview_image_storage_path.clone(),
            seal_preview_image_url: seal_preview_image_url.clone(),
            has_seal_preview_image: !order.seal_preview_image_storage_path.is_empty(),
            has_seal_preview_image_url: !seal_preview_image_url.is_empty(),
            ai_generation_id: order.ai_generation_id.clone(),
            has_ai_generation_id,
            ai_variant_id: order.ai_variant_id.clone(),
            has_ai_variant_id,
            seal_style_name: order.seal_style_name.clone(),
            seal_style_stroke_weight: order.seal_style_stroke_weight.clone(),
            seal_style_balance: order.seal_style_balance.clone(),
            seal_style_prompt_summary: order.seal_style_prompt_summary.clone(),
            has_seal_style_name,
            has_seal_style_stroke_weight,
            has_seal_style_balance,
            has_seal_style_prompt_summary,
            has_ai_seal_metadata,
            customer_confirmation_kanji_and_design_label: confirmation_bool_label(
                order
                    .customer_confirmation_kanji_and_design
                    .unwrap_or_default(),
            )
            .to_owned(),
            customer_confirmation_custom_made_policy_label: confirmation_bool_label(
                order
                    .customer_confirmation_custom_made_policy
                    .unwrap_or_default(),
            )
            .to_owned(),
            customer_confirmation_confirmed_at,
            customer_confirmation_confirmed_seal_text: order
                .customer_confirmation_confirmed_seal_text
                .clone(),
            has_customer_confirmation_kanji_and_design,
            has_customer_confirmation_custom_made_policy,
            has_customer_confirmation_confirmed_at,
            has_customer_confirmation_confirmed_seal_text,
            has_customer_confirmation,
            listing_label_ja: order.listing_label_ja.clone(),
            total: format_order_amount(order.total, &order.currency),
            has_next_statuses: !next_statuses.is_empty(),
            next_statuses,
            shipping_transitions,
            events: event_views,
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
        })
    }

    async fn list_materials(&self) -> Vec<MaterialListItemView> {
        let data = self.data.read().await;
        let mut items = Vec::with_capacity(data.material_ids.len());

        for key in &data.material_ids {
            let Some(material) = data.materials.get(key) else {
                continue;
            };

            items.push(MaterialListItemView {
                key: material.key.clone(),
                label_ja: material.label_i18n.get("ja").cloned().unwrap_or_default(),
                is_active: material.is_active,
                version: material.version,
                updated_at: format_datetime(material.updated_at),
            });
        }

        items
    }

    async fn normalize_stone_listing_filter(
        &self,
        filters: &StoneListingFilter,
    ) -> StoneListingFilter {
        let data = self.data.read().await;
        normalize_stone_listing_filter_with_snapshot(filters, &data)
    }

    async fn filter_stone_listings(
        &self,
        filters: &StoneListingFilter,
    ) -> Vec<StoneListingListItemView> {
        let data = self.data.read().await;
        let mut items = Vec::with_capacity(data.stone_listing_ids.len());
        let filters = normalize_stone_listing_filter_with_snapshot(filters, &data);

        let color_family_filter = if filters.color_family.is_empty() {
            None
        } else {
            Some(filters.color_family.as_str())
        };
        let color_tag_filters = stone_listing_tag_values(&filters.color_tags);
        let pattern_primary_filter = if filters.pattern_primary.is_empty() {
            None
        } else {
            Some(filters.pattern_primary.as_str())
        };
        let pattern_tag_filters = stone_listing_tag_values(&filters.pattern_tags);
        let stone_shape_filter = if filters.stone_shape.is_empty() {
            None
        } else {
            Some(filters.stone_shape.as_str())
        };

        for key in &data.stone_listing_ids {
            let Some(listing) = data.stone_listings.get(key) else {
                continue;
            };
            if let Some(filter) = color_family_filter
                && listing.facets.color_family != filter
            {
                continue;
            }
            if !color_tag_filters.is_empty()
                && !color_tag_filters
                    .iter()
                    .all(|filter| listing.facets.color_tags.iter().any(|tag| tag == filter))
            {
                continue;
            }
            if let Some(filter) = pattern_primary_filter
                && listing.facets.pattern_primary != filter
            {
                continue;
            }
            if !pattern_tag_filters.is_empty()
                && !pattern_tag_filters
                    .iter()
                    .all(|filter| listing.facets.pattern_tags.iter().any(|tag| tag == filter))
            {
                continue;
            }
            if let Some(filter) = stone_shape_filter
                && listing.facets.stone_shape != filter
            {
                continue;
            }
            let primary_photo = select_primary_material_photo(&listing.photos);
            let primary_photo_path = primary_photo
                .map(|photo| photo.storage_path.clone())
                .unwrap_or_default();
            let primary_photo_url =
                build_storage_media_url(&self.storage_assets_bucket, &primary_photo_path);

            items.push(StoneListingListItemView {
                key: listing.key.clone(),
                listing_code: listing.listing_code.clone(),
                title_ja: listing.title_i18n.get("ja").cloned().unwrap_or_default(),
                material_key: listing.material_key.clone(),
                size: listing.size.clone(),
                color_family: listing.facets.color_family.clone(),
                pattern_primary: listing.facets.pattern_primary.clone(),
                stone_shape_label: stone_shape_label(&listing.facets.stone_shape).to_owned(),
                primary_photo_url: primary_photo_url.clone(),
                has_photo: !primary_photo_url.is_empty(),
                price_usd: format_usd(
                    listing
                        .price_by_currency
                        .get("USD")
                        .copied()
                        .unwrap_or_default(),
                ),
                price_jpy: format_jpy(
                    listing
                        .price_by_currency
                        .get("JPY")
                        .copied()
                        .unwrap_or_default(),
                ),
                status_label: stone_listing_status_label(&listing.status).to_owned(),
                is_active: listing.is_active,
                version: listing.version,
                updated_at: format_datetime(listing.updated_at),
            });
        }

        items
    }

    async fn stone_listing_filter_options(&self) -> StoneListingFilterOptions {
        let data = self.data.read().await;
        let mut color_families = BTreeSet::new();
        let mut pattern_primaries = BTreeSet::new();
        let mut stone_shapes = BTreeSet::new();

        for key in &data.stone_listing_ids {
            let Some(listing) = data.stone_listings.get(key) else {
                continue;
            };
            if !listing.facets.color_family.is_empty() {
                color_families.insert(listing.facets.color_family.clone());
            }
            if !listing.facets.pattern_primary.is_empty() {
                pattern_primaries.insert(listing.facets.pattern_primary.clone());
            }
            if !listing.facets.stone_shape.trim().is_empty() {
                stone_shapes.insert(listing.facets.stone_shape.clone());
            }
        }

        let mut stone_shape_options = stone_shape_filter_options();
        for known_value in ["square", "round"] {
            stone_shapes.remove(known_value);
        }
        // Unknown stone shapes are kept in the filter list so they can round-trip unchanged.
        stone_shape_options.extend(stone_shapes.into_iter().map(|value| {
            let label = stone_shape_label(&value);
            StoneListingFacetOptionView { value, label }
        }));

        StoneListingFilterOptions {
            color_family_options: color_families
                .into_iter()
                .map(|value| stone_listing_facet_option_view(&value))
                .collect(),
            pattern_primary_options: pattern_primaries
                .into_iter()
                .map(|value| stone_listing_facet_option_view(&value))
                .collect(),
            stone_shape_options,
        }
    }

    async fn stone_listing_tag_options(&self) -> StoneListingTagOptions {
        let data = self.data.read().await;
        let lookups = facet_tag_lookup_maps(&data);
        let mut color_tag_options = Vec::new();
        let mut pattern_tag_options = Vec::new();
        let mut seen_color_tags = HashSet::new();
        let mut seen_pattern_tags = HashSet::new();

        for id in &data.facet_tag_ids {
            let Some(tag) = data.facet_tags.get(id) else {
                continue;
            };
            if !tag.is_active {
                continue;
            }

            let option = StoneListingFacetOptionView {
                value: tag.key.clone(),
                label: facet_tag_display_label(tag),
            };

            match tag.facet_type.as_str() {
                "color" => {
                    if seen_color_tags.insert(option.value.clone()) {
                        color_tag_options.push(option);
                    }
                }
                "pattern" => {
                    if seen_pattern_tags.insert(option.value.clone()) {
                        pattern_tag_options.push(option);
                    }
                }
                _ => {}
            }
        }

        for id in &data.stone_listing_ids {
            let Some(listing) = data.stone_listings.get(id) else {
                continue;
            };

            for value in normalize_faceted_tag_values_with_lookup(
                &listing.facets.color_tags,
                lookups.get("color"),
            ) {
                if seen_color_tags.insert(value.clone()) {
                    color_tag_options.push(stone_listing_facet_option_view(&value));
                }
            }
            for value in normalize_faceted_tag_values_with_lookup(
                &listing.facets.pattern_tags,
                lookups.get("pattern"),
            ) {
                if seen_pattern_tags.insert(value.clone()) {
                    pattern_tag_options.push(stone_listing_facet_option_view(&value));
                }
            }
        }

        StoneListingTagOptions {
            color_tag_options,
            pattern_tag_options,
        }
    }

    async fn list_facet_tags(&self) -> Vec<FacetTagListItemView> {
        let data = self.data.read().await;
        let mut items = Vec::with_capacity(data.facet_tag_ids.len());

        for id in &data.facet_tag_ids {
            let Some(tag) = data.facet_tags.get(id) else {
                continue;
            };

            items.push(FacetTagListItemView {
                id: id.clone(),
                key: tag.key.clone(),
                facet_type_label: facet_tag_type_label(&tag.facet_type).to_owned(),
                label_ja: tag.label_i18n.get("ja").cloned().unwrap_or_default(),
                label_en: tag.label_i18n.get("en").cloned().unwrap_or_default(),
                aliases: tag.aliases.join(", "),
                is_active: tag.is_active,
                version: tag.version,
                updated_at: format_datetime(tag.updated_at),
            });
        }

        items
    }

    async fn get_facet_tag_detail(
        &self,
        id: &str,
        message: &str,
        render_error: &str,
    ) -> Option<FacetTagDetailView> {
        let data = self.data.read().await;
        let tag = data.facet_tags.get(id)?;
        let localized_editors = localized_editor_groups(&[("名称", "label_i18n", &tag.label_i18n)]);
        let has_localized_editors = !localized_editors.is_empty();

        Some(FacetTagDetailView {
            id: id.to_owned(),
            key: tag.key.clone(),
            facet_type_label: facet_tag_type_label(&tag.facet_type).to_owned(),
            label_ja: tag.label_i18n.get("ja").cloned().unwrap_or_default(),
            label_en: tag.label_i18n.get("en").cloned().unwrap_or_default(),
            localized_editors,
            has_localized_editors,
            aliases: tag.aliases.join(", "),
            sort_order: tag.sort_order,
            is_active: tag.is_active,
            version: tag.version,
            updated_at: format_datetime(tag.updated_at),
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
            key_error: String::new(),
            has_key_error: false,
            aliases_error: String::new(),
            has_aliases_error: false,
        })
    }

    async fn create_facet_tag(
        &self,
        input: FacetTagCreateInput,
    ) -> std::result::Result<(), String> {
        let key =
            normalize_faceted_token(&input.key).ok_or_else(|| "タグキーは必須です。".to_owned())?;
        validate_facet_tag_key(&key)?;

        let facet_type = normalize_facet_tag_type(&input.facet_type)
            .ok_or_else(|| "タグ種別を選択してください。".to_owned())?
            .to_owned();
        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        let aliases = normalize_facet_tag_aliases(&input.aliases);
        if label_ja.is_empty() || label_en.is_empty() {
            return Err("タグ名（ja/en）は必須です。".to_owned());
        }
        if input.sort_order < 0 {
            return Err("表示順は 0 以上で入力してください。".to_owned());
        }

        let tag = {
            let mut data = self.data.write().await;
            let id = facet_tag_document_id(&facet_type, &key);
            if data.facet_tags.contains_key(&id) {
                return Err("同じタグキーは既に存在します。".to_owned());
            }
            validate_facet_tag_alias_collisions(&data, &facet_type, &key, &aliases, None)?;

            let now = Utc::now();
            let tag = FacetTag {
                key: key.clone(),
                facet_type: facet_type.clone(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), label_ja),
                    ("en".to_owned(), label_en),
                ]),
                aliases,
                is_active: input.is_active,
                sort_order: input.sort_order,
                version: 1,
                updated_at: now,
            };

            data.facet_tags.insert(id, tag.clone());
            data.refresh_facet_tag_ids();
            tag
        };

        if let Err(error) = self.source.persist_facet_tag_mutation(&tag).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after facet tag create error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn update_facet_tag(
        &self,
        id: &str,
        input: FacetTagPatchInput,
    ) -> std::result::Result<(), String> {
        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        let aliases = normalize_facet_tag_aliases(&input.aliases);
        if label_ja.is_empty() || label_en.is_empty() {
            return Err("タグ名（ja/en）は必須です。".to_owned());
        }
        if input.sort_order < 0 {
            return Err("表示順は 0 以上で入力してください。".to_owned());
        }

        let updated_tag = {
            let mut data = self.data.write().await;
            let (facet_type, key) = {
                let Some(tag) = data.facet_tags.get(id) else {
                    return Err("タグが見つかりません。".to_owned());
                };
                (tag.facet_type.clone(), tag.key.clone())
            };
            validate_facet_tag_alias_collisions(&data, &facet_type, &key, &aliases, Some(id))?;
            let Some(tag) = data.facet_tags.get_mut(id) else {
                return Err("タグが見つかりません。".to_owned());
            };

            let now = Utc::now();
            merge_admin_localized_map_values(
                &mut tag.label_i18n,
                &[("ja", &label_ja), ("en", &label_en)],
            );
            merge_admin_localized_extra_map_values(&mut tag.label_i18n, &input.label_i18n_updates);
            tag.aliases = aliases;
            tag.is_active = input.is_active;
            tag.sort_order = input.sort_order;
            tag.version += 1;
            tag.updated_at = now;

            let updated = tag.clone();
            data.refresh_facet_tag_ids();
            updated
        };

        if let Err(error) = self.source.persist_facet_tag_mutation(&updated_tag).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after facet tag update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn delete_facet_tag(&self, id: &str) -> std::result::Result<(), String> {
        let normalized_id = id.trim().to_owned();
        if normalized_id.is_empty() {
            return Err("タグ ID が不正です。".to_owned());
        }

        {
            let mut data = self.data.write().await;
            let Some(tag) = data.facet_tags.get(&normalized_id).cloned() else {
                return Err("タグが見つかりません。".to_owned());
            };
            if let Some(listing_key) = data
                .stone_listings
                .values()
                .find(|listing| match tag.facet_type.as_str() {
                    "color" => listing
                        .facets
                        .color_tags
                        .iter()
                        .any(|value| value == &tag.key),
                    "pattern" => listing
                        .facets
                        .pattern_tags
                        .iter()
                        .any(|value| value == &tag.key),
                    _ => false,
                })
                .map(|listing| listing.key.clone())
            {
                return Err(format!(
                    "一点物 `{listing_key}` で参照されているため削除できません。"
                ));
            }
            let Some(_) = data.facet_tags.remove(&normalized_id) else {
                return Err("タグが見つかりません。".to_owned());
            };
            data.refresh_facet_tag_ids();
        }

        if let Err(error) = self.source.persist_facet_tag_deletion(&normalized_id).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after facet tag delete error: {refresh_error}"
                );
            }
            return Err(format!("firestore delete failed: {error}"));
        }

        Ok(())
    }

    async fn get_material_detail(
        &self,
        key: &str,
        message: &str,
        render_error: &str,
    ) -> Option<MaterialDetailView> {
        let data = self.data.read().await;
        let material = data.materials.get(key)?;
        let localized_editors = localized_editor_groups(&[
            ("材質名", "label_i18n", &material.label_i18n),
            ("説明", "description_i18n", &material.description_i18n),
        ]);
        let has_localized_editors = !localized_editors.is_empty();

        Some(MaterialDetailView {
            key: material.key.clone(),
            label_ja: material.label_i18n.get("ja").cloned().unwrap_or_default(),
            label_en: material.label_i18n.get("en").cloned().unwrap_or_default(),
            description_ja: material
                .description_i18n
                .get("ja")
                .cloned()
                .unwrap_or_default(),
            description_en: material
                .description_i18n
                .get("en")
                .cloned()
                .unwrap_or_default(),
            localized_editors,
            has_localized_editors,
            is_active: material.is_active,
            version: material.version,
            updated_at: format_datetime(material.updated_at),
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
        })
    }

    async fn get_stone_listing_detail(
        &self,
        key: &str,
        message: &str,
        render_error: &str,
    ) -> Option<StoneListingDetailView> {
        let data = self.data.read().await;
        let listing = data.stone_listings.get(key)?;
        let primary_photo = select_primary_material_photo(&listing.photos);
        let photo_storage_path = primary_photo
            .map(|photo| photo.storage_path.clone())
            .unwrap_or_default();
        let primary_photo_url =
            build_storage_media_url(&self.storage_assets_bucket, &photo_storage_path);
        let photo_alt_i18n = primary_photo
            .map(|photo| photo.alt_i18n.clone())
            .unwrap_or_default();
        let localized_editors = localized_editor_groups(&[
            ("一点物タイトル", "title_i18n", &listing.title_i18n),
            ("説明", "description_i18n", &listing.description_i18n),
            ("作品紹介", "story_i18n", &listing.story_i18n),
            ("写真代替テキスト", "photo_alt_i18n", &photo_alt_i18n),
        ]);
        let has_localized_editors = !localized_editors.is_empty();

        Some(StoneListingDetailView {
            key: listing.key.clone(),
            listing_code: listing.listing_code.clone(),
            material_key: listing.material_key.clone(),
            size: listing.size.clone(),
            title_ja: listing.title_i18n.get("ja").cloned().unwrap_or_default(),
            title_en: listing.title_i18n.get("en").cloned().unwrap_or_default(),
            description_ja: listing
                .description_i18n
                .get("ja")
                .cloned()
                .unwrap_or_default(),
            description_en: listing
                .description_i18n
                .get("en")
                .cloned()
                .unwrap_or_default(),
            story_ja: listing.story_i18n.get("ja").cloned().unwrap_or_default(),
            story_en: listing.story_i18n.get("en").cloned().unwrap_or_default(),
            color_family: listing.facets.color_family.clone(),
            color_tags: listing.facets.color_tags.join(", "),
            pattern_primary: listing.facets.pattern_primary.clone(),
            pattern_tags: listing.facets.pattern_tags.join(", "),
            stone_shape: listing.facets.stone_shape.clone(),
            stone_shape_label: stone_shape_label(&listing.facets.stone_shape).to_owned(),
            translucency: listing.facets.translucency.clone(),
            price_usd: listing
                .price_by_currency
                .get("USD")
                .copied()
                .unwrap_or_default(),
            price_jpy: listing
                .price_by_currency
                .get("JPY")
                .copied()
                .unwrap_or_default(),
            status: listing.status.clone(),
            status_label: stone_listing_status_label(&listing.status).to_owned(),
            is_active: listing.is_active,
            sort_order: listing.sort_order,
            photo_storage_path,
            primary_photo_url: primary_photo_url.clone(),
            photo_alt_ja: primary_photo
                .and_then(|photo| photo.alt_i18n.get("ja").cloned())
                .unwrap_or_default(),
            photo_alt_en: primary_photo
                .and_then(|photo| photo.alt_i18n.get("en").cloned())
                .unwrap_or_default(),
            localized_editors,
            has_localized_editors,
            has_photo: !primary_photo_url.is_empty(),
            version: listing.version,
            updated_at: format_datetime(listing.updated_at),
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
        })
    }

    async fn list_fonts(&self) -> Vec<FontListItemView> {
        let data = self.data.read().await;
        let mut items = Vec::with_capacity(data.font_ids.len());

        for key in &data.font_ids {
            let Some(font) = data.fonts.get(key) else {
                continue;
            };

            items.push(FontListItemView {
                key: font.key.clone(),
                label: font.label.clone(),
                font_family: font.font_family.clone(),
                font_stylesheet_url: font.font_stylesheet_url.clone(),
                kanji_style_label: kanji_style_label(&font.kanji_style).to_owned(),
                is_active: font.is_active,
                sort_order: font.sort_order,
                version: font.version,
                updated_at: format_datetime(font.updated_at),
            });
        }

        items
    }

    async fn get_font_detail(
        &self,
        key: &str,
        message: &str,
        render_error: &str,
    ) -> Option<FontDetailView> {
        let data = self.data.read().await;
        let font = data.fonts.get(key)?;

        Some(FontDetailView {
            key: font.key.clone(),
            label: font.label.clone(),
            font_family: font.font_family.clone(),
            font_stylesheet_url: font.font_stylesheet_url.clone(),
            kanji_style: normalize_kanji_style(&font.kanji_style)
                .unwrap_or("japanese")
                .to_owned(),
            is_active: font.is_active,
            sort_order: font.sort_order,
            version: font.version,
            updated_at: format_datetime(font.updated_at),
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
        })
    }

    async fn font_stylesheet_urls(&self) -> Vec<String> {
        let data = self.data.read().await;
        let mut seen = HashSet::new();
        let mut urls = Vec::new();

        for key in &data.font_ids {
            let Some(font) = data.fonts.get(key) else {
                continue;
            };
            let url = font.font_stylesheet_url.trim();
            if url.is_empty() {
                continue;
            }
            if seen.insert(url.to_owned()) {
                urls.push(url.to_owned());
            }
        }

        urls
    }

    async fn list_countries(&self) -> Vec<CountryListItemView> {
        let data = self.data.read().await;
        let mut items = Vec::with_capacity(data.country_ids.len());

        for code in &data.country_ids {
            let Some(country) = data.countries.get(code) else {
                continue;
            };

            items.push(CountryListItemView {
                code: country.code.clone(),
                label_ja: country.label_i18n.get("ja").cloned().unwrap_or_default(),
                label_en: country.label_i18n.get("en").cloned().unwrap_or_default(),
                shipping_fee_usd: format_usd(country.shipping_fee_usd),
                shipping_fee_jpy: format_jpy(country.shipping_fee_jpy),
                is_active: country.is_active,
                version: country.version,
                updated_at: format_datetime(country.updated_at),
            });
        }

        items
    }

    async fn get_country_detail(
        &self,
        code: &str,
        message: &str,
        render_error: &str,
    ) -> Option<CountryDetailView> {
        let data = self.data.read().await;
        let country = data.countries.get(code)?;
        let localized_editors =
            localized_editor_groups(&[("配送国名", "label_i18n", &country.label_i18n)]);
        let has_localized_editors = !localized_editors.is_empty();

        Some(CountryDetailView {
            code: country.code.clone(),
            label_ja: country.label_i18n.get("ja").cloned().unwrap_or_default(),
            label_en: country.label_i18n.get("en").cloned().unwrap_or_default(),
            localized_editors,
            has_localized_editors,
            shipping_fee_usd: country.shipping_fee_usd,
            shipping_fee_jpy: country.shipping_fee_jpy,
            is_active: country.is_active,
            sort_order: country.sort_order,
            version: country.version,
            updated_at: format_datetime(country.updated_at),
            message: message.to_owned(),
            has_message: !message.is_empty(),
            error: render_error.to_owned(),
            has_error: !render_error.is_empty(),
        })
    }

    async fn country_options(&self) -> Vec<CountryOptionView> {
        let data = self.data.read().await;
        data.country_ids
            .iter()
            .filter_map(|code| data.countries.get(code))
            .map(|country| CountryOptionView {
                code: country.code.clone(),
                label: country_display_label(country),
            })
            .collect::<Vec<_>>()
    }

    async fn create_country(&self, input: CountryCreateInput) -> std::result::Result<(), String> {
        let normalized_code = input.code.trim().to_uppercase();
        validate_country_code(&normalized_code)?;

        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        validate_country_values(
            &label_ja,
            &label_en,
            input.shipping_fee_usd,
            input.shipping_fee_jpy,
            input.sort_order,
        )?;

        let country = {
            let mut data = self.data.write().await;
            if data.countries.contains_key(&normalized_code) {
                return Err("同じ国コードは既に存在します。".to_owned());
            }

            let now = Utc::now();
            let country = Country {
                code: normalized_code.clone(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), label_ja),
                    ("en".to_owned(), label_en),
                ]),
                shipping_fee_usd: input.shipping_fee_usd,
                shipping_fee_jpy: input.shipping_fee_jpy,
                is_active: input.is_active,
                sort_order: input.sort_order,
                version: 1,
                updated_at: now,
            };

            data.countries.insert(normalized_code, country.clone());
            data.refresh_country_ids();
            country
        };

        if let Err(error) = self.source.persist_country_mutation(&country).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after country create error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn delete_country(&self, code: &str) -> std::result::Result<(), String> {
        let normalized_code = code.trim().to_uppercase();
        if normalized_code.is_empty() {
            return Err("国コードが不正です。".to_owned());
        }

        {
            let mut data = self.data.write().await;
            let Some(_) = data.countries.remove(&normalized_code) else {
                return Err("配送国が見つかりません。".to_owned());
            };
            data.refresh_country_ids();
        }

        if let Err(error) = self.source.persist_country_deletion(&normalized_code).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after country delete error: {refresh_error}"
                );
            }
            return Err(format!("firestore delete failed: {error}"));
        }

        Ok(())
    }

    async fn update_country(
        &self,
        code: &str,
        input: CountryPatchInput,
    ) -> std::result::Result<(), String> {
        let normalized_code = code.trim().to_uppercase();
        if normalized_code.is_empty() {
            return Err("国コードが不正です。".to_owned());
        }

        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        validate_country_values(
            &label_ja,
            &label_en,
            input.shipping_fee_usd,
            input.shipping_fee_jpy,
            input.sort_order,
        )?;

        let updated_country = {
            let mut data = self.data.write().await;
            let Some(country) = data.countries.get_mut(&normalized_code) else {
                return Err("配送国が見つかりません。".to_owned());
            };

            let now = Utc::now();
            merge_admin_localized_map_values(
                &mut country.label_i18n,
                &[("ja", &label_ja), ("en", &label_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut country.label_i18n,
                &input.label_i18n_updates,
            );
            country.shipping_fee_usd = input.shipping_fee_usd;
            country.shipping_fee_jpy = input.shipping_fee_jpy;
            country.sort_order = input.sort_order;
            country.is_active = input.is_active;
            country.version += 1;
            country.updated_at = now;

            let updated = country.clone();
            data.refresh_country_ids();
            updated
        };

        if let Err(error) = self.source.persist_country_mutation(&updated_country).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after country update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn update_order_status(
        &self,
        order_id: &str,
        next_status: &str,
        actor_id: &str,
    ) -> std::result::Result<(), String> {
        let next_status = next_status.trim();
        if next_status.is_empty() {
            return Err("更新先のステータスを選択してください。".to_owned());
        }

        let (updated_order, new_events) = {
            let mut data = self.data.write().await;
            let Some(order) = data.orders.get_mut(order_id) else {
                return Err("注文が見つかりません。".to_owned());
            };

            if order.status == next_status {
                return Err("現在と同じステータスには更新できません。".to_owned());
            }

            let allowed = status_transitions(&order.status);
            if !allowed.contains(&next_status) {
                return Err(format!(
                    "{} から {} には遷移できません。",
                    order_status_label(&order.status),
                    order_status_label(next_status)
                ));
            }

            let now = Utc::now();
            let before = order.status.clone();
            order.status = next_status.to_owned();
            order.status_updated_at = now;
            order.updated_at = now;
            apply_derived_statuses(order);

            let event = OrderEvent {
                kind: "status_changed".to_owned(),
                actor_type: "admin".to_owned(),
                actor_id: actor_id.trim().to_owned(),
                before_status: before,
                after_status: next_status.to_owned(),
                note: String::new(),
                created_at: now,
            };
            order.events.push(event.clone());

            (order.clone(), vec![event])
        };

        if let Err(error) = self
            .source
            .persist_order_mutation(&updated_order, &new_events)
            .await
        {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after status update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn update_shipping(
        &self,
        order_id: &str,
        carrier: &str,
        tracking_no: &str,
        transition: &str,
        actor_id: &str,
    ) -> std::result::Result<(), String> {
        if carrier.trim().is_empty() {
            return Err("配送業者を入力してください。".to_owned());
        }
        if tracking_no.trim().is_empty() {
            return Err("追跡番号を入力してください。".to_owned());
        }

        let (updated_order, new_events) = {
            let mut data = self.data.write().await;
            let Some(order) = data.orders.get_mut(order_id) else {
                return Err("注文が見つかりません。".to_owned());
            };

            let transition = transition.trim();
            if !transition.is_empty() && transition != "none" {
                if order.status == transition {
                    return Err("現在と同じステータスは指定できません。".to_owned());
                }

                let allowed = status_transitions(&order.status);
                if !allowed.contains(&transition) {
                    return Err(format!(
                        "{} から {} には遷移できません。",
                        order_status_label(&order.status),
                        order_status_label(transition)
                    ));
                }
            }

            let now = Utc::now();
            let before_status = order.status.clone();
            let mut events = Vec::new();

            order.carrier = carrier.trim().to_owned();
            order.tracking_no = tracking_no.trim().to_owned();
            order.updated_at = now;

            let shipment_event = OrderEvent {
                kind: "shipment_registered".to_owned(),
                actor_type: "admin".to_owned(),
                actor_id: actor_id.trim().to_owned(),
                before_status: String::new(),
                after_status: String::new(),
                note: format!("{} / {}", order.carrier, order.tracking_no),
                created_at: now,
            };
            order.events.push(shipment_event.clone());
            events.push(shipment_event);

            if !transition.is_empty() && transition != "none" {
                order.status = transition.to_owned();
                order.status_updated_at = now;
                apply_derived_statuses(order);

                let status_event = OrderEvent {
                    kind: "status_changed".to_owned(),
                    actor_type: "admin".to_owned(),
                    actor_id: actor_id.trim().to_owned(),
                    before_status,
                    after_status: transition.to_owned(),
                    note: String::new(),
                    created_at: now,
                };
                order.events.push(status_event.clone());
                events.push(status_event);
            }

            (order.clone(), events)
        };

        if let Err(error) = self
            .source
            .persist_order_mutation(&updated_order, &new_events)
            .await
        {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after shipping update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn update_material(
        &self,
        key: &str,
        input: MaterialPatchInput,
    ) -> std::result::Result<(), String> {
        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        let description_ja = input.description_ja.trim().to_owned();
        let description_en = input.description_en.trim().to_owned();
        validate_material_values(&label_ja, &label_en, &description_ja, &description_en)?;

        let updated_material = {
            let mut data = self.data.write().await;
            let Some(material) = data.materials.get_mut(key) else {
                return Err("材質が見つかりません。".to_owned());
            };

            let now = Utc::now();
            merge_admin_localized_map_values(
                &mut material.label_i18n,
                &[("ja", &label_ja), ("en", &label_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut material.label_i18n,
                &input.label_i18n_updates,
            );
            merge_admin_localized_map_values(
                &mut material.description_i18n,
                &[("ja", &description_ja), ("en", &description_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut material.description_i18n,
                &input.description_i18n_updates,
            );
            material.is_active = input.is_active;
            material.version += 1;
            material.updated_at = now;

            let updated = material.clone();
            data.refresh_material_ids();
            updated
        };

        if let Err(error) = self
            .source
            .persist_material_mutation(&updated_material)
            .await
        {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after material update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn update_font(
        &self,
        key: &str,
        input: FontPatchInput,
    ) -> std::result::Result<(), String> {
        let normalized_key = key.trim().to_owned();
        if normalized_key.is_empty() {
            return Err("フォントキーが不正です。".to_owned());
        }

        let label = input.label.trim().to_owned();
        let font_family = input.font_family.trim().to_owned();
        let kanji_style = normalize_kanji_style(&input.kanji_style)
            .ok_or_else(|| "スタイルは日本・中国・台湾から選択してください。".to_owned())?
            .to_owned();
        let font_stylesheet_url =
            validate_font_values(&label, &font_family, &kanji_style, input.sort_order)?;

        let updated_font = {
            let mut data = self.data.write().await;
            let Some(font) = data.fonts.get_mut(&normalized_key) else {
                return Err("フォントが見つかりません。".to_owned());
            };

            let now = Utc::now();
            font.label = label;
            font.font_family = font_family;
            font.font_stylesheet_url = font_stylesheet_url;
            font.kanji_style = kanji_style;
            font.sort_order = input.sort_order;
            font.is_active = input.is_active;
            font.version += 1;
            font.updated_at = now;

            let updated = font.clone();
            data.refresh_font_ids();
            updated
        };

        if let Err(error) = self.source.persist_font_mutation(&updated_font).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after font update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn create_font(&self, input: FontCreateInput) -> std::result::Result<(), String> {
        let key = input.key.trim().to_owned();
        validate_font_key(&key)?;

        let label = input.label.trim().to_owned();
        let font_family = input.font_family.trim().to_owned();
        let kanji_style = normalize_kanji_style(&input.kanji_style)
            .ok_or_else(|| "スタイルは日本・中国・台湾から選択してください。".to_owned())?
            .to_owned();
        let font_stylesheet_url =
            validate_font_values(&label, &font_family, &kanji_style, input.sort_order)?;

        let font = {
            let mut data = self.data.write().await;
            if data.fonts.contains_key(&key) {
                return Err("同じフォントキーは既に存在します。".to_owned());
            }

            let now = Utc::now();
            let font = Font {
                key: key.clone(),
                label,
                font_family,
                font_stylesheet_url,
                kanji_style,
                is_active: input.is_active,
                sort_order: input.sort_order,
                version: 1,
                updated_at: now,
            };

            data.fonts.insert(key, font.clone());
            data.refresh_font_ids();
            font
        };

        if let Err(error) = self.source.persist_font_mutation(&font).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after font create error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn create_material(&self, input: MaterialCreateInput) -> std::result::Result<(), String> {
        let key = input.key.trim().to_owned();
        validate_material_key(&key)?;

        let label_ja = input.label_ja.trim().to_owned();
        let label_en = input.label_en.trim().to_owned();
        let description_ja = input.description_ja.trim().to_owned();
        let description_en = input.description_en.trim().to_owned();
        validate_material_values(&label_ja, &label_en, &description_ja, &description_en)?;

        let material = {
            let mut data = self.data.write().await;
            if data.materials.contains_key(&key) {
                return Err("同じ材質キーは既に存在します。".to_owned());
            }

            let now = Utc::now();
            let sort_order = next_material_sort_order(&data.materials);
            let material = Material {
                key: key.clone(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), label_ja),
                    ("en".to_owned(), label_en),
                ]),
                description_i18n: HashMap::from([
                    ("ja".to_owned(), description_ja),
                    ("en".to_owned(), description_en),
                ]),
                comparison_texture_ja: String::new(),
                comparison_texture_en: String::new(),
                comparison_weight_ja: String::new(),
                comparison_weight_en: String::new(),
                comparison_usage_ja: String::new(),
                comparison_usage_en: String::new(),
                shape: "square".to_owned(),
                photos: Vec::new(),
                price_usd: 0,
                price_jpy: 0,
                is_active: input.is_active,
                sort_order,
                version: 1,
                updated_at: now,
            };

            data.materials.insert(key, material.clone());
            data.refresh_material_ids();
            material
        };

        if let Err(error) = self.source.persist_material_mutation(&material).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after material create error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn create_stone_listing(
        &self,
        input: StoneListingCreateInput,
    ) -> std::result::Result<(), String> {
        let key = input.stone_listing_key.trim().to_owned();
        validate_material_key(&key)?;

        let listing_code = input.listing_code.trim().to_uppercase();
        validate_stone_listing_code(&listing_code)?;

        let material_key = input.material_key.trim().to_owned();
        validate_material_key(&material_key)?;

        let size = input.size.trim().to_owned();
        let title_ja = input.title_ja.trim().to_owned();
        let title_en = input.title_en.trim().to_owned();
        let description_ja = input.description_ja.trim().to_owned();
        let description_en = input.description_en.trim().to_owned();
        let story_ja = input.story_ja.trim().to_owned();
        let story_en = input.story_en.trim().to_owned();
        let color_family = normalize_faceted_token(&input.color_family)
            .ok_or_else(|| "色ファミリーは必須です。".to_owned())?;
        let pattern_primary = normalize_faceted_token(&input.pattern_primary)
            .ok_or_else(|| "模様の代表値は必須です。".to_owned())?;
        let stone_shape = normalize_stone_shape_optional(&input.stone_shape)
            .ok_or_else(|| "石の形は必須です。".to_owned())?;
        let translucency = normalize_optional_faceted_token(&input.translucency);
        let photo_storage_path = normalize_storage_path(&input.photo_storage_path);
        let photo_alt_ja = input.photo_alt_ja.trim().to_owned();
        let photo_alt_en = input.photo_alt_en.trim().to_owned();
        let status = normalize_stone_listing_status(&input.status)
            .ok_or_else(|| "公開状態を選択してください。".to_owned())?
            .to_owned();
        self.validate_stone_listing_material_key(&material_key)
            .await?;
        let facet_tag_lookups = {
            let data = self.data.read().await;
            facet_tag_lookup_maps(&data)
        };
        let color_tags = normalize_faceted_tag_values_with_lookup_strict(
            &input.color_tags,
            facet_tag_lookups.get("color"),
            "色タグ",
        )?;
        let pattern_tags = normalize_faceted_tag_values_with_lookup_strict(
            &input.pattern_tags,
            facet_tag_lookups.get("pattern"),
            "模様タグ",
        )?;

        validate_stone_listing_values(
            &title_ja,
            &title_en,
            &description_ja,
            &description_en,
            &size,
            &color_family,
            &pattern_primary,
            &stone_shape,
            input.price_usd,
            input.price_jpy,
            input.sort_order,
            &status,
            &material_key,
            &listing_code,
            &photo_storage_path,
        )?;

        let listing = {
            let mut data = self.data.write().await;
            if data.stone_listings.contains_key(&key) {
                return Err("同じ一点物キーは既に存在します。".to_owned());
            }

            let now = Utc::now();
            let published_at = if stone_listing_is_published(&status) {
                Some(now)
            } else {
                None
            };
            let listing = StoneListing {
                key: key.clone(),
                listing_code,
                material_key,
                size,
                title_i18n: HashMap::from([
                    ("ja".to_owned(), title_ja),
                    ("en".to_owned(), title_en),
                ]),
                description_i18n: HashMap::from([
                    ("ja".to_owned(), description_ja),
                    ("en".to_owned(), description_en),
                ]),
                story_i18n: HashMap::from([
                    ("ja".to_owned(), story_ja),
                    ("en".to_owned(), story_en),
                ]),
                facets: StoneListingFacets {
                    color_family,
                    color_tags,
                    pattern_primary,
                    pattern_tags,
                    stone_shape,
                    translucency,
                },
                photos: build_single_stone_listing_photos(
                    &key,
                    &photo_storage_path,
                    &photo_alt_ja,
                    &photo_alt_en,
                ),
                price_by_currency: HashMap::from([
                    ("USD".to_owned(), input.price_usd),
                    ("JPY".to_owned(), input.price_jpy),
                ]),
                status,
                is_active: input.is_active,
                published_at,
                sort_order: input.sort_order,
                version: 1,
                updated_at: now,
            };

            data.stone_listings.insert(key, listing.clone());
            data.refresh_stone_listing_ids();
            listing
        };

        if let Err(error) = self.source.persist_stone_listing_mutation(&listing).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after stone listing create error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn upload_material_photo(
        &self,
        material_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> std::result::Result<String, String> {
        let normalized_key = material_key.trim().to_owned();
        validate_material_key(&normalized_key)?;

        if bytes.is_empty() {
            return Err("画像ファイルが空です。".to_owned());
        }
        if bytes.len() > MAX_PHOTO_UPLOAD_BYTES {
            return Err("画像サイズは 10MB 以下にしてください。".to_owned());
        }

        let normalized_content_type = normalize_image_content_type(file_name, content_type)
            .ok_or_else(|| {
                "対応していない画像形式です。png / jpg / webp / gif / avif を利用してください。"
                    .to_owned()
            })?;

        self.source
            .upload_material_photo(
                &normalized_key,
                file_name,
                Some(normalized_content_type),
                bytes,
            )
            .await
            .map_err(|error| format!("photo upload failed: {error}"))
    }

    async fn upload_stone_listing_photo(
        &self,
        stone_listing_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> std::result::Result<String, String> {
        let normalized_key = stone_listing_key.trim().to_owned();
        validate_material_key(&normalized_key)?;

        if bytes.is_empty() {
            return Err("画像ファイルが空です。".to_owned());
        }
        if bytes.len() > MAX_PHOTO_UPLOAD_BYTES {
            return Err("画像サイズは 10MB 以下にしてください。".to_owned());
        }

        let normalized_content_type = normalize_image_content_type(file_name, content_type)
            .ok_or_else(|| {
                "対応していない画像形式です。png / jpg / webp / gif / avif を利用してください。"
                    .to_owned()
            })?;

        self.source
            .upload_stone_listing_photo(
                &normalized_key,
                file_name,
                Some(normalized_content_type),
                bytes,
            )
            .await
            .map_err(|error| format!("photo upload failed: {error}"))
    }

    async fn validate_stone_listing_material_key(
        &self,
        material_key: &str,
    ) -> std::result::Result<(), String> {
        let data = self.data.read().await;
        match data.materials.get(material_key) {
            Some(material) if material.is_active => Ok(()),
            _ => Err("素材キーに対応する有効な材質が見つかりません。".to_owned()),
        }
    }

    async fn validate_stone_listing_material_key_for_update(
        &self,
        stone_listing_key: &str,
        material_key: &str,
    ) -> std::result::Result<(), String> {
        let data = self.data.read().await;
        if matches!(data.materials.get(material_key), Some(material) if material.is_active) {
            return Ok(());
        }

        if data
            .stone_listings
            .get(stone_listing_key)
            .is_some_and(|listing| listing.material_key == material_key)
        {
            return Ok(());
        }

        Err("素材キーに対応する有効な材質が見つかりません。".to_owned())
    }

    async fn delete_material(&self, key: &str) -> std::result::Result<(), String> {
        let normalized_key = key.trim().to_owned();
        if normalized_key.is_empty() {
            return Err("材質キーが不正です。".to_owned());
        }

        {
            let mut data = self.data.write().await;
            let Some(_) = data.materials.remove(&normalized_key) else {
                return Err("材質が見つかりません。".to_owned());
            };
            data.refresh_material_ids();
        }

        if let Err(error) = self.source.persist_material_deletion(&normalized_key).await {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after material delete error: {refresh_error}"
                );
            }
            return Err(format!("firestore delete failed: {error}"));
        }

        Ok(())
    }

    async fn update_stone_listing(
        &self,
        key: &str,
        input: StoneListingPatchInput,
    ) -> std::result::Result<(), String> {
        let normalized_key = key.trim().to_owned();
        if normalized_key.is_empty() {
            return Err("一点物キーが不正です。".to_owned());
        }

        let listing_code = input.listing_code.trim().to_uppercase();
        validate_stone_listing_code(&listing_code)?;

        let material_key = input.material_key.trim().to_owned();
        validate_material_key(&material_key)?;

        let size = input.size.trim().to_owned();
        let title_ja = input.title_ja.trim().to_owned();
        let title_en = input.title_en.trim().to_owned();
        let description_ja = input.description_ja.trim().to_owned();
        let description_en = input.description_en.trim().to_owned();
        let story_ja = input.story_ja.trim().to_owned();
        let story_en = input.story_en.trim().to_owned();
        let color_family = normalize_faceted_token(&input.color_family)
            .ok_or_else(|| "色ファミリーは必須です。".to_owned())?;
        let pattern_primary = normalize_faceted_token(&input.pattern_primary)
            .ok_or_else(|| "模様の代表値は必須です。".to_owned())?;
        let stone_shape = normalize_stone_shape_optional(&input.stone_shape)
            .ok_or_else(|| "石の形は必須です。".to_owned())?;
        let translucency = normalize_optional_faceted_token(&input.translucency);
        let photo_storage_path = normalize_storage_path(&input.photo_storage_path);
        let photo_alt_ja = input.photo_alt_ja.trim().to_owned();
        let photo_alt_en = input.photo_alt_en.trim().to_owned();
        let status = normalize_stone_listing_status(&input.status)
            .ok_or_else(|| "公開状態を選択してください。".to_owned())?
            .to_owned();
        self.validate_stone_listing_material_key_for_update(&normalized_key, &material_key)
            .await?;
        let facet_tag_lookups = {
            let data = self.data.read().await;
            facet_tag_lookup_maps(&data)
        };
        let color_tags = normalize_faceted_tag_values_with_lookup_strict(
            &input.color_tags,
            facet_tag_lookups.get("color"),
            "色タグ",
        )?;
        let pattern_tags = normalize_faceted_tag_values_with_lookup_strict(
            &input.pattern_tags,
            facet_tag_lookups.get("pattern"),
            "模様タグ",
        )?;

        validate_stone_listing_values(
            &title_ja,
            &title_en,
            &description_ja,
            &description_en,
            &size,
            &color_family,
            &pattern_primary,
            &stone_shape,
            input.price_usd,
            input.price_jpy,
            input.sort_order,
            &status,
            &material_key,
            &listing_code,
            &photo_storage_path,
        )?;

        let updated_listing = {
            let mut data = self.data.write().await;
            let Some(listing) = data.stone_listings.get_mut(&normalized_key) else {
                return Err("一点物が見つかりません。".to_owned());
            };

            let now = Utc::now();
            let was_published = stone_listing_is_published(&listing.status);
            listing.listing_code = listing_code;
            listing.material_key = material_key;
            listing.size = size;
            merge_admin_localized_map_values(
                &mut listing.title_i18n,
                &[("ja", &title_ja), ("en", &title_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut listing.title_i18n,
                &input.title_i18n_updates,
            );
            merge_admin_localized_map_values(
                &mut listing.description_i18n,
                &[("ja", &description_ja), ("en", &description_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut listing.description_i18n,
                &input.description_i18n_updates,
            );
            merge_admin_localized_map_values(
                &mut listing.story_i18n,
                &[("ja", &story_ja), ("en", &story_en)],
            );
            merge_admin_localized_extra_map_values(
                &mut listing.story_i18n,
                &input.story_i18n_updates,
            );
            listing.facets.color_family = color_family;
            listing.facets.color_tags = color_tags;
            listing.facets.pattern_primary = pattern_primary;
            listing.facets.pattern_tags = pattern_tags;
            listing.facets.stone_shape = stone_shape;
            listing.facets.translucency = translucency;
            listing.price_by_currency = HashMap::from([
                ("USD".to_owned(), input.price_usd),
                ("JPY".to_owned(), input.price_jpy),
            ]);
            if stone_listing_is_published(&status)
                && (!was_published || listing.published_at.is_none())
            {
                listing.published_at = Some(now);
            }
            listing.status = status;
            listing.is_active = input.is_active;
            listing.sort_order = input.sort_order;
            listing.photos = merge_primary_stone_listing_photo(
                &listing.photos,
                &listing.key,
                &photo_storage_path,
                &photo_alt_ja,
                &photo_alt_en,
            );
            merge_primary_stone_listing_photo_alt_updates(
                &mut listing.photos,
                &input.photo_alt_i18n_updates,
            );
            listing.version += 1;
            listing.updated_at = now;

            let updated = listing.clone();
            data.refresh_stone_listing_ids();
            updated
        };

        if let Err(error) = self
            .source
            .persist_stone_listing_mutation(&updated_listing)
            .await
        {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after stone listing update error: {refresh_error}"
                );
            }
            return Err(format!("firestore update failed: {error}"));
        }

        Ok(())
    }

    async fn delete_stone_listing(&self, key: &str) -> std::result::Result<(), String> {
        let normalized_key = key.trim().to_owned();
        if normalized_key.is_empty() {
            return Err("一点物キーが不正です。".to_owned());
        }

        {
            let mut data = self.data.write().await;
            if let Some(order_label) = data
                .orders
                .values()
                .find(|order| order.listing_key == normalized_key)
                .map(|order| {
                    if order.order_no.is_empty() {
                        order.id.clone()
                    } else {
                        order.order_no.clone()
                    }
                })
            {
                return Err(format!(
                    "注文 `{order_label}` で参照されているため削除できません。"
                ));
            }
            let Some(_) = data.stone_listings.remove(&normalized_key) else {
                return Err("一点物が見つかりません。".to_owned());
            };
            data.refresh_stone_listing_ids();
        }

        if let Err(error) = self
            .source
            .persist_stone_listing_deletion(&normalized_key)
            .await
        {
            if let Err(refresh_error) = self.refresh_from_source().await {
                eprintln!(
                    "failed to rollback from firestore after stone listing delete error: {refresh_error}"
                );
            }
            return Err(format!("firestore delete failed: {error}"));
        }

        Ok(())
    }
}

fn decode_order_fields(
    default_locale: &str,
    order_id: &str,
    data: &BTreeMap<String, JsonValue>,
) -> Order {
    let payment = read_map_field(data, "payment");
    let listing = read_map_field(data, "listing");
    let material = read_map_field(data, "material");
    let fulfillment = read_map_field(data, "fulfillment");
    let shipping = read_map_field(data, "shipping");
    let contact = read_map_field(data, "contact");
    let seal = read_map_field(data, "seal");
    let seal_style = read_map_field(&seal, "style");
    let customer_confirmation = read_map_field(data, "customer_confirmation");
    let pricing = read_map_field(data, "pricing");
    let raw_locale = read_string_field(data, "locale");
    let locale = if raw_locale.is_empty() {
        let contact_locale = read_string_field(&contact, "preferred_locale");
        if contact_locale.is_empty() {
            default_locale.trim().to_owned()
        } else {
            contact_locale
        }
    } else {
        raw_locale
    };
    let pricing_currency = resolve_order_currency(data, &pricing, &payment, &locale);
    let total = resolve_order_total(data, &pricing);
    let (listing_key, listing_label_ja) =
        FirestoreAdminSource::resolve_order_listing_fields(data, &listing, &material);
    let seal_line1 = read_string_field(&seal, "line1");
    let seal_line2 = read_string_field(&seal, "line2");
    let engraving_text =
        resolve_order_engraving_text(&customer_confirmation, &seal_line1, &seal_line2);

    let created_at = read_timestamp_field(data, "created_at").unwrap_or_else(Utc::now);
    let updated_at = read_timestamp_field(data, "updated_at").unwrap_or(created_at);
    let status_updated_at = read_timestamp_field(data, "status_updated_at").unwrap_or(updated_at);

    let mut order = Order {
        id: order_id.to_owned(),
        order_no: read_string_field(data, "order_no"),
        channel: read_string_field(data, "channel"),
        locale,
        currency: pricing_currency,
        listing_key,
        status: read_string_field(data, "status"),
        status_updated_at,
        payment_status: read_string_field(&payment, "status"),
        fulfillment_status: read_string_field(&fulfillment, "status"),
        tracking_no: read_string_field(&fulfillment, "tracking_no"),
        carrier: read_string_field(&fulfillment, "carrier"),
        country_code: read_string_field(&shipping, "country_code").to_uppercase(),
        contact_email: read_string_field(&contact, "email"),
        seal_line1,
        seal_line2,
        engraving_text,
        ai_generation_id: read_string_field(&seal, "ai_generation_id"),
        ai_variant_id: read_string_field(&seal, "ai_variant_id"),
        seal_preview_image_storage_path: read_seal_preview_image_storage_path(&seal),
        seal_style_name: read_string_field(&seal_style, "name"),
        seal_style_stroke_weight: read_string_field(&seal_style, "stroke_weight"),
        seal_style_balance: read_string_field(&seal_style, "balance"),
        seal_style_prompt_summary: read_string_field(&seal_style, "prompt_summary"),
        customer_confirmation_kanji_and_design: read_bool_field(
            &customer_confirmation,
            "kanji_and_design",
        ),
        customer_confirmation_custom_made_policy: read_bool_field(
            &customer_confirmation,
            "custom_made_policy",
        ),
        customer_confirmation_confirmed_at: read_timestamp_field(
            &customer_confirmation,
            "confirmed_at",
        ),
        customer_confirmation_confirmed_seal_text: read_string_field(
            &customer_confirmation,
            "confirmed_seal_text",
        ),
        listing_label_ja,
        total,
        created_at,
        updated_at,
        events: Vec::new(),
    };

    if order.order_no.is_empty() {
        order.order_no = order_id.to_owned();
    }
    if order.locale.is_empty() {
        order.locale = default_locale.trim().to_owned();
    }
    if order.currency.trim().is_empty() {
        order.currency = "USD".to_owned();
    }
    if order.status.is_empty() {
        order.status = "pending_payment".to_owned();
    }

    fill_derived_statuses(&mut order);
    order
}

impl FirestoreAdminSource {
    async fn firestore_client(&self) -> Result<FirebaseFirestoreClient> {
        let access_token = self
            .token_provider
            .token(&[DATASTORE_SCOPE])
            .await
            .context("failed to acquire firestore access token")?;

        firestore_client_from_access_token(access_token.as_str())
    }

    async fn load_snapshot(&self) -> Result<AdminSnapshot> {
        let client = self.firestore_client().await?;

        let orders = self.load_orders(&client).await?;
        let fonts = self.load_fonts(&client).await?;
        let materials = self.load_materials(&client).await?;
        let stone_listings = self.load_stone_listings(&client).await?;
        let facet_tags = self.load_facet_tags(&client).await?;
        let mut countries = self.load_countries(&client).await?;

        if countries.is_empty() {
            for order in orders.values() {
                if !order.country_code.is_empty() {
                    let code = order.country_code.to_uppercase();
                    countries.insert(
                        code.clone(),
                        Country {
                            code: code.clone(),
                            label_i18n: HashMap::from([("ja".to_owned(), code)]),
                            shipping_fee_usd: 0,
                            shipping_fee_jpy: 0,
                            is_active: true,
                            sort_order: 9999,
                            version: 1,
                            updated_at: Utc::now(),
                        },
                    );
                }
            }
        }

        let mut snapshot = AdminSnapshot {
            orders,
            order_ids: Vec::new(),
            fonts,
            font_ids: Vec::new(),
            materials,
            material_ids: Vec::new(),
            stone_listings,
            stone_listing_ids: Vec::new(),
            facet_tags,
            facet_tag_ids: Vec::new(),
            countries,
            country_ids: Vec::new(),
        };
        snapshot.refresh_order_ids();
        snapshot.refresh_font_ids();
        snapshot.refresh_material_ids();
        snapshot.refresh_stone_listing_ids();
        snapshot.refresh_facet_tag_ids();
        normalize_stone_listing_facets_in_snapshot(&mut snapshot);
        snapshot.refresh_country_ids();
        Ok(snapshot)
    }

    async fn load_orders(
        &self,
        client: &FirebaseFirestoreClient,
    ) -> Result<HashMap<String, Order>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "orders" }],
                "orderBy": [{
                    "field": { "fieldPath": "created_at" },
                    "direction": "DESCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load orders")?;

        let mut orders = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(order_id) = document_id(&document) else {
                continue;
            };

            let mut order = self.decode_order(&order_id, &document.fields);
            let order_name = document
                .name
                .clone()
                .unwrap_or_else(|| format!("{}/orders/{order_id}", self.parent));
            let events = self
                .load_order_events(client, &order_name, &order_id)
                .await?;
            order.events = events;
            orders.insert(order_id, order);
        }

        Ok(orders)
    }

    async fn load_order_events(
        &self,
        client: &FirebaseFirestoreClient,
        order_name: &str,
        order_id: &str,
    ) -> Result<Vec<OrderEvent>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "events" }],
                "orderBy": [{
                    "field": { "fieldPath": "created_at" },
                    "direction": "ASCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(order_name, &query)
            .await
            .with_context(|| format!("failed to load order events ({order_id})"))?;

        let mut events = Vec::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let data = &document.fields;

            let kind = read_string_field(data, "type");
            let note = {
                let note = read_string_field(data, "note");
                if !note.is_empty() {
                    note
                } else {
                    let payload = read_map_field(data, "payload");
                    let carrier = read_string_field(&payload, "carrier");
                    let tracking_no = read_string_field(&payload, "tracking_no");
                    if carrier.is_empty() && tracking_no.is_empty() {
                        String::new()
                    } else {
                        format!("{} / {}", carrier, tracking_no)
                    }
                    .trim()
                    .to_owned()
                }
            };

            events.push(OrderEvent {
                kind: if kind.is_empty() {
                    "event".to_owned()
                } else {
                    kind
                },
                actor_type: read_string_field(data, "actor_type"),
                actor_id: read_string_field(data, "actor_id"),
                before_status: read_string_field(data, "before_status"),
                after_status: read_string_field(data, "after_status"),
                note,
                created_at: read_timestamp_field(data, "created_at").unwrap_or_else(Utc::now),
            });
        }

        Ok(events)
    }

    fn decode_order(&self, order_id: &str, data: &BTreeMap<String, JsonValue>) -> Order {
        decode_order_fields(&self.default_locale, order_id, data)
    }

    fn resolve_order_listing_fields(
        data: &BTreeMap<String, JsonValue>,
        listing: &BTreeMap<String, JsonValue>,
        material: &BTreeMap<String, JsonValue>,
    ) -> (String, String) {
        let listing_key = {
            let key = read_string_field(listing, "key");
            if key.is_empty() {
                let legacy_listing_key = read_string_field(data, "listing_key");
                if legacy_listing_key.is_empty() {
                    read_string_field(material, "key")
                } else {
                    legacy_listing_key
                }
            } else {
                key
            }
        };
        let listing_label_ja = {
            let localized =
                resolve_localized_text(&read_string_map_field(listing, "title_i18n"), "ja", "ja");
            if !localized.is_empty() {
                localized
            } else {
                let legacy_localized = resolve_localized_text(
                    &read_string_map_field(material, "label_i18n"),
                    "ja",
                    "ja",
                );
                if !legacy_localized.is_empty() {
                    legacy_localized
                } else {
                    let legacy_label_ja = read_string_field(data, "material_label_ja");
                    if !legacy_label_ja.is_empty() {
                        legacy_label_ja
                    } else {
                        let listing_code = read_string_field(listing, "listing_code");
                        if !listing_code.is_empty() {
                            listing_code
                        } else {
                            let legacy_listing_key = read_string_field(data, "listing_key");
                            if !legacy_listing_key.is_empty() {
                                legacy_listing_key
                            } else {
                                read_string_field(material, "key")
                            }
                        }
                    }
                }
            }
        };

        (listing_key, listing_label_ja)
    }

    async fn load_materials(
        &self,
        client: &FirebaseFirestoreClient,
    ) -> Result<HashMap<String, Material>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "materials" }],
                "orderBy": [{
                    "field": { "fieldPath": "sort_order" },
                    "direction": "ASCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load materials")?;

        let mut materials = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(doc_id) = document_id(&document) else {
                continue;
            };

            let data = &document.fields;
            let price_by_currency = material_price_by_currency_from_fields(data);
            materials.insert(
                doc_id.clone(),
                Material {
                    key: doc_id,
                    label_i18n: read_string_map_field(data, "label_i18n"),
                    description_i18n: read_string_map_field(data, "description_i18n"),
                    comparison_texture_ja: read_string_field(data, "comparison_texture_ja"),
                    comparison_texture_en: read_string_field(data, "comparison_texture_en"),
                    comparison_weight_ja: read_string_field(data, "comparison_weight_ja"),
                    comparison_weight_en: read_string_field(data, "comparison_weight_en"),
                    comparison_usage_ja: read_string_field(data, "comparison_usage_ja"),
                    comparison_usage_en: read_string_field(data, "comparison_usage_en"),
                    shape: material_shape_or_default(&read_string_field(data, "shape")),
                    photos: read_material_photos(data),
                    price_usd: price_by_currency.get("USD").copied().unwrap_or_default(),
                    price_jpy: price_by_currency.get("JPY").copied().unwrap_or_default(),
                    is_active: read_bool_field(data, "is_active").unwrap_or(true),
                    sort_order: read_int_field(data, "sort_order").unwrap_or_default(),
                    version: read_int_field(data, "version").unwrap_or(1),
                    updated_at: read_timestamp_field(data, "updated_at").unwrap_or_else(Utc::now),
                },
            );
        }

        return Ok(materials);
    }

    async fn load_stone_listings(
        &self,
        client: &FirebaseFirestoreClient,
    ) -> Result<HashMap<String, StoneListing>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "stone_listings" }],
                "orderBy": [{
                    "field": { "fieldPath": "sort_order" },
                    "direction": "ASCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load stone_listings")?;

        let mut listings = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(doc_id) = document_id(&document) else {
                continue;
            };

            let data = &document.fields;
            let price_by_currency = stone_listing_price_by_currency_from_fields(data);
            if price_by_currency.is_empty() {
                continue;
            }

            let facets = read_map_field(data, "facets");
            let title_i18n = read_string_map_field(data, "title_i18n");
            let description_i18n = read_string_map_field(data, "description_i18n");
            let story_i18n = read_string_map_field(data, "story_i18n");

            listings.insert(
                doc_id.clone(),
                StoneListing {
                    key: doc_id,
                    listing_code: read_string_field(data, "listing_code"),
                    material_key: read_string_field(data, "material_key"),
                    size: read_string_field(data, "size"),
                    title_i18n,
                    description_i18n,
                    story_i18n,
                    facets: StoneListingFacets {
                        color_family: read_string_field(&facets, "color_family"),
                        color_tags: read_string_array_field(&facets, "color_tags"),
                        pattern_primary: read_string_field(&facets, "pattern_primary"),
                        pattern_tags: read_string_array_field(&facets, "pattern_tags"),
                        stone_shape: normalize_stone_shape(&read_string_field(
                            &facets,
                            "stone_shape",
                        )),
                        translucency: read_string_field(&facets, "translucency"),
                    },
                    photos: read_material_photos(data),
                    price_by_currency,
                    status: read_string_field(data, "status"),
                    is_active: read_bool_field(data, "is_active").unwrap_or(true),
                    published_at: read_timestamp_field(data, "published_at"),
                    sort_order: read_int_field(data, "sort_order").unwrap_or_default(),
                    version: read_int_field(data, "version").unwrap_or(1),
                    updated_at: read_timestamp_field(data, "updated_at").unwrap_or_else(Utc::now),
                },
            );
        }

        Ok(listings)
    }

    async fn load_facet_tags(
        &self,
        client: &FirebaseFirestoreClient,
    ) -> Result<HashMap<String, FacetTag>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "facet_tags" }],
                "orderBy": [
                    {
                        "field": { "fieldPath": "facet_type" },
                        "direction": "ASCENDING"
                    },
                    {
                        "field": { "fieldPath": "sort_order" },
                        "direction": "ASCENDING"
                    },
                    {
                        "field": { "fieldPath": "key" },
                        "direction": "ASCENDING"
                    }
                ]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load facet_tags")?;

        let mut tags = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(doc_id) = document_id(&document) else {
                continue;
            };

            let data = &document.fields;
            let mut facet_type = normalize_facet_tag_type(&read_string_field(data, "facet_type"))
                .unwrap_or_default()
                .to_owned();
            let mut key =
                normalize_faceted_token(&read_string_field(data, "key")).unwrap_or_default();

            if (facet_type.is_empty() || key.is_empty())
                && let Some((doc_type, doc_key)) = doc_id.split_once(':')
            {
                if facet_type.is_empty() {
                    facet_type = normalize_facet_tag_type(doc_type)
                        .unwrap_or_default()
                        .to_owned();
                }
                if key.is_empty() {
                    key = normalize_faceted_token(doc_key).unwrap_or_default();
                }
            }

            if facet_type.is_empty() || key.is_empty() {
                continue;
            }

            let label_i18n = {
                let mut values = read_string_map_field(data, "label_i18n");
                if values.is_empty() {
                    values.insert("ja".to_owned(), key.replace('_', " "));
                }
                values
            };
            let aliases = normalize_facet_tag_aliases(&read_string_array_field(data, "aliases"));
            let is_active = read_bool_field(data, "is_active").unwrap_or(true);
            let sort_order = read_int_field(data, "sort_order").unwrap_or_default();
            let version = read_int_field(data, "version").unwrap_or(1);
            let updated_at = read_timestamp_field(data, "updated_at").unwrap_or_else(Utc::now);

            let id = facet_tag_document_id(&facet_type, &key);
            tags.insert(
                id,
                FacetTag {
                    key,
                    facet_type,
                    label_i18n,
                    aliases,
                    is_active,
                    sort_order,
                    version,
                    updated_at,
                },
            );
        }

        Ok(tags)
    }

    async fn load_fonts(&self, client: &FirebaseFirestoreClient) -> Result<HashMap<String, Font>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "fonts" }],
                "orderBy": [{
                    "field": { "fieldPath": "sort_order" },
                    "direction": "ASCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load fonts")?;

        let mut fonts = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(doc_id) = document_id(&document) else {
                continue;
            };

            let data = &document.fields;
            let label = resolve_font_label_field(data, &doc_id);

            let font_family = read_string_field(data, "font_family");
            if font_family.is_empty() {
                continue;
            }
            let mut font_stylesheet_url = read_string_field(data, "font_stylesheet_url");
            if font_stylesheet_url.is_empty() {
                font_stylesheet_url =
                    build_google_fonts_stylesheet_url(&font_family).unwrap_or_default();
            }
            let kanji_style = read_string_field(data, "kanji_style");
            let kanji_style = normalize_kanji_style(&kanji_style)
                .unwrap_or("japanese")
                .to_owned();

            let is_active = read_bool_field(data, "is_active").unwrap_or(true);
            let sort_order = read_int_field(data, "sort_order").unwrap_or_default();
            let version = read_int_field(data, "version").unwrap_or(1);
            let updated_at = read_timestamp_field(data, "updated_at").unwrap_or_else(Utc::now);

            fonts.insert(
                doc_id.clone(),
                Font {
                    key: doc_id,
                    label,
                    font_family,
                    font_stylesheet_url,
                    kanji_style,
                    is_active,
                    sort_order,
                    version,
                    updated_at,
                },
            );
        }

        Ok(fonts)
    }

    async fn load_countries(
        &self,
        client: &FirebaseFirestoreClient,
    ) -> Result<HashMap<String, Country>> {
        let query = RunQueryRequest {
            structured_query: Some(json!({
                "from": [{ "collectionId": "countries" }],
                "orderBy": [{
                    "field": { "fieldPath": "sort_order" },
                    "direction": "ASCENDING"
                }]
            })),
            ..RunQueryRequest::default()
        };

        let rows = client
            .run_query(&self.parent, &query)
            .await
            .context("failed to load countries")?;

        let mut countries = HashMap::new();
        for row in rows {
            let Some(document) = row.document else {
                continue;
            };
            let Some(doc_id) = document_id(&document) else {
                continue;
            };

            let code = doc_id.to_uppercase();
            let data = &document.fields;
            let shipping_fee_by_currency = country_shipping_fee_by_currency_from_fields(data);
            let shipping_fee_usd = shipping_fee_by_currency
                .get("USD")
                .copied()
                .unwrap_or_default();
            let shipping_fee_jpy = shipping_fee_by_currency
                .get("JPY")
                .copied()
                .unwrap_or_default();
            let is_active = read_bool_field(data, "is_active").unwrap_or(true);
            let sort_order = read_int_field(data, "sort_order").unwrap_or_default();
            let version = read_int_field(data, "version").unwrap_or(1);
            let updated_at = read_timestamp_field(data, "updated_at").unwrap_or_else(Utc::now);

            let mut label_i18n = read_string_map_field(data, "label_i18n");
            if label_i18n.is_empty() {
                label_i18n.insert("ja".to_owned(), code.clone());
            }

            countries.insert(
                code.clone(),
                Country {
                    code,
                    label_i18n,
                    shipping_fee_usd,
                    shipping_fee_jpy,
                    is_active,
                    sort_order,
                    version,
                    updated_at,
                },
            );
        }

        Ok(countries)
    }

    async fn upload_material_photo(
        &self,
        material_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> Result<String> {
        let bucket = normalize_storage_bucket_name(&self.storage_assets_bucket);
        if bucket.is_empty() {
            bail!(
                "storage bucket is not configured (set HANKO_ADMIN_STORAGE_ASSETS_BUCKET[_DEV|_PROD] or API_STORAGE_ASSETS_BUCKET)"
            );
        }
        validate_storage_bucket_name(&bucket)
            .map_err(|error| anyhow!("invalid storage bucket `{}`: {}", bucket, error))?;

        let normalized_content_type = normalize_image_content_type(file_name, content_type)
            .ok_or_else(|| anyhow!("unsupported image type"))?;
        let storage_path = build_storage_path_for_uploaded_photo(
            material_key,
            file_name,
            Some(normalized_content_type),
        );

        let access_token = self
            .token_provider
            .token(&[STORAGE_SCOPE])
            .await
            .context("failed to acquire storage access token")?;

        let mut endpoint =
            reqwest::Url::parse("https://storage.googleapis.com/upload/storage/v1/b")
                .context("failed to construct storage upload endpoint")?;
        endpoint
            .path_segments_mut()
            .map_err(|_| anyhow!("failed to construct storage upload endpoint"))?
            .extend([bucket.as_str(), "o"]);

        let response = reqwest::Client::new()
            .post(endpoint)
            .bearer_auth(access_token.as_str())
            .query(&[("uploadType", "media"), ("name", storage_path.as_str())])
            .header(reqwest::header::CONTENT_TYPE, normalized_content_type)
            .body(bytes.to_vec())
            .send()
            .await
            .context("failed to upload photo to cloud storage")?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response
                .text()
                .await
                .unwrap_or_else(|_| "<unable to read response body>".to_owned());
            bail!("storage upload failed status={} body={}", status, body);
        }

        Ok(storage_path)
    }

    async fn upload_stone_listing_photo(
        &self,
        stone_listing_key: &str,
        file_name: &str,
        content_type: Option<&str>,
        bytes: &[u8],
    ) -> Result<String> {
        let bucket = normalize_storage_bucket_name(&self.storage_assets_bucket);
        if bucket.is_empty() {
            bail!(
                "storage bucket is not configured (set HANKO_ADMIN_STORAGE_ASSETS_BUCKET[_DEV|_PROD] or API_STORAGE_ASSETS_BUCKET)"
            );
        }
        validate_storage_bucket_name(&bucket)
            .map_err(|error| anyhow!("invalid storage bucket `{}`: {}", bucket, error))?;

        let normalized_content_type = normalize_image_content_type(file_name, content_type)
            .ok_or_else(|| anyhow!("unsupported image type"))?;
        let storage_path = build_storage_path_for_uploaded_stone_listing_photo(
            stone_listing_key,
            file_name,
            Some(normalized_content_type),
        );

        let access_token = self
            .token_provider
            .token(&[STORAGE_SCOPE])
            .await
            .context("failed to acquire storage access token")?;

        let mut endpoint =
            reqwest::Url::parse("https://storage.googleapis.com/upload/storage/v1/b")
                .context("failed to construct storage upload endpoint")?;
        endpoint
            .path_segments_mut()
            .map_err(|_| anyhow!("failed to construct storage upload endpoint"))?
            .extend([bucket.as_str(), "o"]);

        let response = reqwest::Client::new()
            .post(endpoint)
            .bearer_auth(access_token.as_str())
            .query(&[("uploadType", "media"), ("name", storage_path.as_str())])
            .header(reqwest::header::CONTENT_TYPE, normalized_content_type)
            .body(bytes.to_vec())
            .send()
            .await
            .context("failed to upload photo to cloud storage")?;

        if !response.status().is_success() {
            let status = response.status();
            let body = response
                .text()
                .await
                .unwrap_or_else(|_| "<unable to read response body>".to_owned());
            bail!("storage upload failed status={} body={}", status, body);
        }

        Ok(storage_path)
    }

    async fn persist_order_mutation(&self, order: &Order, events: &[OrderEvent]) -> Result<()> {
        let client = self.firestore_client().await?;

        let order_name = format!("{}/orders/{}", self.parent, order.id);
        let existing_order = client
            .get_document(&order_name, &GetDocumentOptions::default())
            .await
            .context("failed to load order for listing sync")?;
        let listing_key =
            read_string_field(&read_map_field(&existing_order.fields, "listing"), "key");

        let mut fulfillment_fields = btree_from_pairs(vec![
            (
                "status",
                fs_string(order.fulfillment_status.trim().to_owned()),
            ),
            ("carrier", fs_string(order.carrier.trim().to_owned())),
            (
                "tracking_no",
                fs_string(order.tracking_no.trim().to_owned()),
            ),
        ]);

        if order.fulfillment_status == "shipped" {
            fulfillment_fields.insert("shipped_at".to_owned(), fs_timestamp(order.updated_at));
        }
        if order.fulfillment_status == "delivered" {
            fulfillment_fields.insert("delivered_at".to_owned(), fs_timestamp(order.updated_at));
        }

        let document = Document {
            name: Some(order_name.clone()),
            fields: btree_from_pairs(vec![
                ("status", fs_string(order.status.trim().to_owned())),
                ("status_updated_at", fs_timestamp(order.status_updated_at)),
                ("updated_at", fs_timestamp(order.updated_at)),
                (
                    "payment",
                    fs_map(btree_from_pairs(vec![(
                        "status",
                        fs_string(order.payment_status.trim().to_owned()),
                    )])),
                ),
                ("fulfillment", fs_map(fulfillment_fields)),
            ]),
            ..Document::default()
        };

        client
            .patch_document(
                &order_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths: vec![
                        "status".to_owned(),
                        "status_updated_at".to_owned(),
                        "updated_at".to_owned(),
                        "payment".to_owned(),
                        "fulfillment".to_owned(),
                    ],
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist order mutation")?;

        if let Some(listing_status) = stone_listing_status_after_order_status(&order.status) {
            if !listing_key.is_empty() {
                let listing_name = format!("{}/stone_listings/{}", self.parent, listing_key);
                let mut listing_doc = client
                    .get_document(&listing_name, &GetDocumentOptions::default())
                    .await
                    .context("failed to load stone listing for order mutation")?;
                let current_listing_status = read_string_field(&listing_doc.fields, "status");
                let current_published_at =
                    read_timestamp_field(&listing_doc.fields, "published_at");
                let should_update_listing = if listing_status.eq_ignore_ascii_case("published") {
                    stone_listing_should_restore_after_canceled_order(
                        &current_listing_status,
                        current_published_at.is_none(),
                    )
                } else {
                    !current_listing_status
                        .trim()
                        .eq_ignore_ascii_case(listing_status)
                };
                if should_update_listing {
                    let listing_version =
                        read_int_field(&listing_doc.fields, "version").unwrap_or_default();
                    let published_at = current_published_at.unwrap_or(order.updated_at);
                    listing_doc
                        .fields
                        .insert("status".to_owned(), fs_string(listing_status));
                    if listing_status.eq_ignore_ascii_case("published") {
                        listing_doc
                            .fields
                            .insert("published_at".to_owned(), fs_timestamp(published_at));
                    }
                    listing_doc
                        .fields
                        .insert("version".to_owned(), fs_int(listing_version + 1));
                    listing_doc
                        .fields
                        .insert("updated_at".to_owned(), fs_timestamp(order.updated_at));
                    client
                        .patch_document(
                            &listing_name,
                            &listing_doc,
                            &PatchDocumentOptions::default(),
                        )
                        .await
                        .context("failed to persist stone listing status")?;
                }
            }
        }

        for event in events {
            let event_document = Document {
                fields: encode_order_event(event),
                ..Document::default()
            };

            client
                .create_document(
                    &order_name,
                    "events",
                    &event_document,
                    &CreateDocumentOptions {
                        document_id: Some(format!("evt_{}", Uuid::new_v4().simple())),
                        ..CreateDocumentOptions::default()
                    },
                )
                .await
                .context("failed to persist order event")?;
        }

        Ok(())
    }

    async fn persist_material_mutation(&self, material: &Material) -> Result<()> {
        let client = self.firestore_client().await?;

        let material_name = format!("{}/materials/{}", self.parent, material.key);
        let price_by_currency = HashMap::from([
            ("USD".to_owned(), material.price_usd.max(0)),
            ("JPY".to_owned(), material.price_jpy.max(0)),
        ]);
        let document = Document {
            name: Some(material_name.clone()),
            fields: btree_from_pairs(vec![
                ("label_i18n", fs_string_map(&material.label_i18n)),
                (
                    "description_i18n",
                    fs_string_map(&material.description_i18n),
                ),
                (
                    "comparison_texture_ja",
                    fs_string(material.comparison_texture_ja.clone()),
                ),
                (
                    "comparison_texture_en",
                    fs_string(material.comparison_texture_en.clone()),
                ),
                (
                    "comparison_weight_ja",
                    fs_string(material.comparison_weight_ja.clone()),
                ),
                (
                    "comparison_weight_en",
                    fs_string(material.comparison_weight_en.clone()),
                ),
                (
                    "comparison_usage_ja",
                    fs_string(material.comparison_usage_ja.clone()),
                ),
                (
                    "comparison_usage_en",
                    fs_string(material.comparison_usage_en.clone()),
                ),
                ("shape", fs_string(material.shape.clone())),
                ("photos", fs_material_photos(&material.photos)),
                ("price_by_currency", fs_int_map(&price_by_currency)),
                ("is_active", fs_bool(material.is_active)),
                ("sort_order", fs_int(material.sort_order)),
                ("version", fs_int(material.version)),
                ("updated_at", fs_timestamp(material.updated_at)),
            ]),
            ..Document::default()
        };

        let mut update_mask_field_paths = vec![
            "comparison_texture_ja".to_owned(),
            "comparison_texture_en".to_owned(),
            "comparison_weight_ja".to_owned(),
            "comparison_weight_en".to_owned(),
            "comparison_usage_ja".to_owned(),
            "comparison_usage_en".to_owned(),
            "shape".to_owned(),
            "photos".to_owned(),
            "price_by_currency".to_owned(),
            "is_active".to_owned(),
            "sort_order".to_owned(),
            "version".to_owned(),
            "updated_at".to_owned(),
        ];
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "label_i18n",
            &material.label_i18n,
        );
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "description_i18n",
            &material.description_i18n,
        );

        client
            .patch_document(
                &material_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths,
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist material mutation")?;

        Ok(())
    }

    async fn persist_stone_listing_mutation(&self, listing: &StoneListing) -> Result<()> {
        let client = self.firestore_client().await?;

        let listing_name = format!("{}/stone_listings/{}", self.parent, listing.key);
        let document = Document {
            name: Some(listing_name.clone()),
            fields: stone_listing_snapshot_fields(listing),
            ..Document::default()
        };

        let mut update_mask_field_paths = vec![
            "listing_code".to_owned(),
            "material_key".to_owned(),
            "size".to_owned(),
            "facets".to_owned(),
            "photos".to_owned(),
            "price_by_currency".to_owned(),
            "status".to_owned(),
            "is_active".to_owned(),
            "sort_order".to_owned(),
            "version".to_owned(),
            "updated_at".to_owned(),
        ];
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "title_i18n",
            &listing.title_i18n,
        );
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "description_i18n",
            &listing.description_i18n,
        );
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "story_i18n",
            &listing.story_i18n,
        );
        if listing.published_at.is_some() {
            update_mask_field_paths.push("published_at".to_owned());
        }

        client
            .patch_document(
                &listing_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths,
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist stone listing mutation")?;

        Ok(())
    }

    async fn persist_facet_tag_mutation(&self, tag: &FacetTag) -> Result<()> {
        let client = self.firestore_client().await?;

        let tag_name = format!(
            "{}/facet_tags/{}",
            self.parent,
            facet_tag_document_id(&tag.facet_type, &tag.key)
        );
        let document = Document {
            name: Some(tag_name.clone()),
            fields: facet_tag_snapshot_fields(tag),
            ..Document::default()
        };

        let mut update_mask_field_paths = vec![
            "key".to_owned(),
            "facet_type".to_owned(),
            "aliases".to_owned(),
            "is_active".to_owned(),
            "sort_order".to_owned(),
            "version".to_owned(),
            "updated_at".to_owned(),
        ];
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "label_i18n",
            &tag.label_i18n,
        );

        client
            .patch_document(
                &tag_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths,
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist facet tag mutation")?;

        Ok(())
    }

    async fn persist_font_mutation(&self, font: &Font) -> Result<()> {
        let client = self.firestore_client().await?;

        let font_name = format!("{}/fonts/{}", self.parent, font.key);
        let document = Document {
            name: Some(font_name.clone()),
            fields: btree_from_pairs(vec![
                ("label", fs_string(font.label.clone())),
                ("font_family", fs_string(font.font_family.clone())),
                (
                    "font_stylesheet_url",
                    fs_string(font.font_stylesheet_url.clone()),
                ),
                ("kanji_style", fs_string(font.kanji_style.clone())),
                ("is_active", fs_bool(font.is_active)),
                ("sort_order", fs_int(font.sort_order)),
                ("version", fs_int(font.version)),
                ("updated_at", fs_timestamp(font.updated_at)),
            ]),
            ..Document::default()
        };

        client
            .patch_document(
                &font_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths: vec![
                        "label".to_owned(),
                        "font_family".to_owned(),
                        "font_stylesheet_url".to_owned(),
                        "kanji_style".to_owned(),
                        "is_active".to_owned(),
                        "sort_order".to_owned(),
                        "version".to_owned(),
                        "updated_at".to_owned(),
                    ],
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist font mutation")?;

        Ok(())
    }

    async fn persist_country_mutation(&self, country: &Country) -> Result<()> {
        let client = self.firestore_client().await?;

        let country_name = format!("{}/countries/{}", self.parent, country.code);
        let shipping_fee_by_currency = HashMap::from([
            ("USD".to_owned(), country.shipping_fee_usd.max(0)),
            ("JPY".to_owned(), country.shipping_fee_jpy.max(0)),
        ]);
        let document = Document {
            name: Some(country_name.clone()),
            fields: btree_from_pairs(vec![
                ("label_i18n", fs_string_map(&country.label_i18n)),
                (
                    "shipping_fee_by_currency",
                    fs_int_map(&shipping_fee_by_currency),
                ),
                ("is_active", fs_bool(country.is_active)),
                ("sort_order", fs_int(country.sort_order)),
                ("version", fs_int(country.version)),
                ("updated_at", fs_timestamp(country.updated_at)),
            ]),
            ..Document::default()
        };

        let mut update_mask_field_paths = vec![
            "shipping_fee_by_currency".to_owned(),
            "is_active".to_owned(),
            "sort_order".to_owned(),
            "version".to_owned(),
            "updated_at".to_owned(),
        ];
        append_admin_localized_update_mask_paths_for_values(
            &mut update_mask_field_paths,
            "label_i18n",
            &country.label_i18n,
        );

        client
            .patch_document(
                &country_name,
                &document,
                &PatchDocumentOptions {
                    update_mask_field_paths,
                    ..PatchDocumentOptions::default()
                },
            )
            .await
            .context("failed to persist country mutation")?;

        Ok(())
    }

    async fn persist_country_deletion(&self, country_code: &str) -> Result<()> {
        let client = self.firestore_client().await?;

        let country_name = format!("{}/countries/{}", self.parent, country_code);
        client
            .delete_document(&country_name, &DeleteDocumentOptions::default())
            .await
            .context("failed to persist country deletion")?;

        Ok(())
    }

    async fn persist_material_deletion(&self, material_key: &str) -> Result<()> {
        let client = self.firestore_client().await?;

        let material_name = format!("{}/materials/{}", self.parent, material_key);
        client
            .delete_document(&material_name, &DeleteDocumentOptions::default())
            .await
            .context("failed to persist material deletion")?;

        Ok(())
    }

    async fn persist_stone_listing_deletion(&self, listing_key: &str) -> Result<()> {
        let client = self.firestore_client().await?;

        let listing_name = format!("{}/stone_listings/{}", self.parent, listing_key);
        client
            .delete_document(&listing_name, &DeleteDocumentOptions::default())
            .await
            .context("failed to persist stone listing deletion")?;

        Ok(())
    }

    async fn persist_facet_tag_deletion(&self, tag_id: &str) -> Result<()> {
        let client = self.firestore_client().await?;

        let tag_name = format!("{}/facet_tags/{}", self.parent, tag_id);
        client
            .delete_document(&tag_name, &DeleteDocumentOptions::default())
            .await
            .context("failed to persist facet tag deletion")?;

        Ok(())
    }
}

fn render_orders_list(orders: &[OrderListItemView]) -> Result<String> {
    let template = OrdersListTemplate {
        orders: orders.to_vec(),
        has_orders: !orders.is_empty(),
    };
    render_html(&template)
}

fn render_order_detail(detail: &OrderDetailView) -> Result<String> {
    render_html(&OrderDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_materials_list(materials: &[MaterialListItemView]) -> Result<String> {
    let template = MaterialsListTemplate {
        materials: materials.to_vec(),
        has_materials: !materials.is_empty(),
    };
    render_html(&template)
}

fn render_stone_listings_list(
    stone_listings: &[StoneListingListItemView],
    filters: &StoneListingFilter,
) -> Result<String> {
    let template = StoneListingsListTemplate {
        stone_listings: stone_listings.to_vec(),
        has_stone_listings: !stone_listings.is_empty(),
        matching_count: stone_listings.len(),
        has_active_filters: stone_listing_filter_is_active(filters),
        filter_badges: stone_listing_filter_badges(filters),
    };
    render_html(&template)
}

fn render_stone_listing_detail(detail: &StoneListingDetailView) -> Result<String> {
    render_html(&StoneListingDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_fonts_list(fonts: &[FontListItemView]) -> Result<String> {
    let template = FontsListTemplate {
        fonts: fonts.to_vec(),
        has_fonts: !fonts.is_empty(),
    };
    render_html(&template)
}

fn render_material_detail(detail: &MaterialDetailView) -> Result<String> {
    render_html(&MaterialDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_stone_listing_create(view: &StoneListingCreateView) -> Result<String> {
    render_html(&StoneListingCreateTemplate { view: view.clone() })
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

fn render_font_detail(detail: &FontDetailView) -> Result<String> {
    render_html(&FontDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_material_create(view: &MaterialCreateView) -> Result<String> {
    render_html(&MaterialCreateTemplate { view: view.clone() })
}

fn render_font_create(view: &FontCreateView) -> Result<String> {
    render_html(&FontCreateTemplate { view: view.clone() })
}

fn render_countries_list(countries: &[CountryListItemView]) -> Result<String> {
    let template = CountriesListTemplate {
        countries: countries.to_vec(),
        has_countries: !countries.is_empty(),
    };
    render_html(&template)
}

fn render_country_detail(detail: &CountryDetailView) -> Result<String> {
    render_html(&CountryDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_country_create(view: &CountryCreateView) -> Result<String> {
    render_html(&CountryCreateTemplate { view: view.clone() })
}

fn render_facet_tags_list(facet_tags: &[FacetTagListItemView]) -> Result<String> {
    let template = FacetTagsListTemplate {
        facet_tags: facet_tags.to_vec(),
        has_facet_tags: !facet_tags.is_empty(),
    };
    render_html(&template)
}

fn render_facet_tag_detail(detail: &FacetTagDetailView) -> Result<String> {
    render_html(&FacetTagDetailTemplate {
        detail: detail.clone(),
    })
}

fn render_facet_tag_create(view: &FacetTagCreateView) -> Result<String> {
    render_html(&FacetTagCreateTemplate {
        view: view.clone(),
        facet_tag_type_options: facet_tag_type_options(),
    })
}

fn render_template<T: Template>(template: &T) -> Response {
    match render_html(template) {
        Ok(html) => html_response(StatusCode::OK, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render template: {error}"),
        ),
    }
}

fn render_html<T: Template>(template: &T) -> Result<String> {
    template
        .render()
        .map_err(|error| anyhow!(error.to_string()))
}

fn firestore_client_from_access_token(access_token: &str) -> Result<FirebaseFirestoreClient> {
    Ok(FirebaseFirestoreClient::new(access_token.to_owned()))
}

fn html_response(status: StatusCode, body: String) -> Response {
    (
        status,
        [(header::CONTENT_TYPE, "text/html; charset=utf-8")],
        body,
    )
        .into_response()
}

fn html_response_with_trigger(status: StatusCode, body: String, trigger: &str) -> Response {
    let mut response = html_response(status, body);
    if let Ok(value) = HeaderValue::from_str(trigger) {
        response.headers_mut().insert("HX-Trigger", value);
    }
    response
}

fn plain_error(status: StatusCode, message: String) -> Response {
    (status, message).into_response()
}

fn json_response(status: StatusCode, payload: JsonValue) -> Response {
    (
        status,
        [(header::CONTENT_TYPE, "application/json; charset=utf-8")],
        payload.to_string(),
    )
        .into_response()
}

fn json_error(status: StatusCode, message: &str) -> Response {
    json_response(
        status,
        json!({
            "error": message,
        }),
    )
}

fn form_value(form: &HashMap<String, String>, key: &str) -> String {
    form.get(key)
        .map(|value| value.trim().to_owned())
        .unwrap_or_default()
}

fn merge_admin_localized_map_values(
    values: &mut HashMap<String, String>,
    updates: &[(&str, &str)],
) {
    for (locale, value) in updates {
        let locale = locale.trim();
        if locale.is_empty() {
            continue;
        }
        values.insert(locale.to_owned(), value.trim().to_owned());
    }
}

fn merge_optional_admin_localized_map_values(
    values: &mut HashMap<String, String>,
    updates: &[(&str, &str)],
) {
    for (locale, value) in updates {
        let locale = locale.trim();
        if locale.is_empty() {
            continue;
        }
        let value = value.trim();
        if value.is_empty() {
            values.remove(locale);
        } else {
            values.insert(locale.to_owned(), value.to_owned());
        }
    }
}

fn merge_admin_localized_extra_map_values(
    values: &mut HashMap<String, String>,
    updates: &HashMap<String, String>,
) {
    let mut locales = updates.keys().collect::<Vec<_>>();
    locales.sort();

    for locale in locales {
        let locale = locale.trim();
        if locale.is_empty() {
            continue;
        }
        let Some(value) = updates.get(locale) else {
            continue;
        };
        values.insert(locale.to_owned(), value.trim().to_owned());
    }
}

fn admin_localized_update_mask_paths(field: &str) -> Vec<String> {
    ADMIN_EDITABLE_LOCALE_KEYS
        .iter()
        .map(|locale| format!("{field}.{locale}"))
        .collect()
}

fn append_admin_localized_update_mask_paths(paths: &mut Vec<String>, field: &str) {
    paths.extend(admin_localized_update_mask_paths(field));
}

fn admin_localized_update_mask_paths_for_values(
    field: &str,
    values: &HashMap<String, String>,
) -> Vec<String> {
    let mut locales = values
        .keys()
        .map(|locale| locale.trim().to_lowercase())
        .filter(|locale| !locale.is_empty())
        .collect::<Vec<_>>();
    locales.sort();
    locales.dedup();
    locales
        .into_iter()
        .map(|locale| format!("{field}.{locale}"))
        .collect()
}

fn append_admin_localized_update_mask_paths_for_values(
    paths: &mut Vec<String>,
    field: &str,
    values: &HashMap<String, String>,
) {
    paths.extend(admin_localized_update_mask_paths_for_values(field, values));
}

fn admin_localized_editor_languages() -> &'static [AdminLocalizedEditorLanguage] {
    ADMIN_LOCALIZED_EDITOR_LANGUAGES
        .get_or_init(|| {
            let mut languages =
                serde_json::from_str::<Vec<AdminLanguageRegistryEntry>>(LANGUAGE_REGISTRY_JSON)
                    .expect("checked-in language registry must parse for admin localized editor")
                    .into_iter()
                    .filter_map(|entry| {
                        let route_code = entry.route_code.trim().to_lowercase();
                        if route_code.is_empty()
                            || ADMIN_EDITABLE_LOCALE_KEYS.contains(&route_code.as_str())
                        {
                            return None;
                        }

                        Some(AdminLocalizedEditorLanguage {
                            route_code,
                            native_name: entry.native_name.trim().to_owned(),
                            english_name: entry.english_name.trim().to_owned(),
                            text_direction: if entry
                                .text_direction
                                .trim()
                                .eq_ignore_ascii_case("rtl")
                            {
                                "rtl".to_owned()
                            } else {
                                "ltr".to_owned()
                            },
                        })
                    })
                    .collect::<Vec<_>>();
            languages.sort_by(|left, right| left.route_code.cmp(&right.route_code));
            languages
        })
        .as_slice()
}

fn localized_editor_group(
    label: &str,
    input_prefix: &str,
    values: &HashMap<String, String>,
) -> LocalizedEditorGroupView {
    LocalizedEditorGroupView {
        label: label.to_owned(),
        values: admin_localized_editor_languages()
            .iter()
            .map(|language| LocalizedEditorValueView {
                route_code: language.route_code.clone(),
                language_label: localized_editor_language_label(language),
                text_direction: language.text_direction.clone(),
                input_name: format!("{input_prefix}__{}", language.route_code),
                value: values
                    .get(&language.route_code)
                    .cloned()
                    .unwrap_or_default(),
            })
            .collect(),
    }
}

fn localized_editor_groups(
    groups: &[(&str, &str, &HashMap<String, String>)],
) -> Vec<LocalizedEditorGroupView> {
    groups
        .iter()
        .map(|(label, input_prefix, values)| localized_editor_group(label, input_prefix, values))
        .collect()
}

fn localized_editor_language_label(language: &AdminLocalizedEditorLanguage) -> String {
    if language.native_name == language.english_name {
        language.native_name.clone()
    } else {
        format!("{} / {}", language.native_name, language.english_name)
    }
}

fn parse_localized_form_updates(
    form: &HashMap<String, String>,
    input_prefix: &str,
) -> HashMap<String, String> {
    let allowed_routes = admin_localized_editor_languages()
        .iter()
        .map(|language| language.route_code.as_str())
        .collect::<BTreeSet<_>>();
    let field_prefix = format!("{input_prefix}__");

    form.iter()
        .filter_map(|(key, value)| {
            let route_code = key.strip_prefix(&field_prefix)?.trim().to_lowercase();
            let value = value.trim();
            if route_code.is_empty()
                || value.is_empty()
                || !allowed_routes.contains(route_code.as_str())
            {
                return None;
            }
            Some((route_code, value.to_owned()))
        })
        .collect()
}

fn next_material_sort_order(materials: &HashMap<String, Material>) -> i64 {
    materials
        .values()
        .map(|material| material.sort_order)
        .max()
        .unwrap_or(0)
        .saturating_add(10)
}

fn new_material_create_view(message: &str, render_error: &str) -> MaterialCreateView {
    MaterialCreateView {
        key: String::new(),
        label_ja: String::new(),
        label_en: String::new(),
        description_ja: String::new(),
        description_en: String::new(),
        is_active: true,
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn material_create_view_from_form(
    form: &HashMap<String, String>,
    message: &str,
    render_error: &str,
) -> MaterialCreateView {
    MaterialCreateView {
        key: form_value(form, "key"),
        label_ja: form_value(form, "label_ja"),
        label_en: form_value(form, "label_en"),
        description_ja: form_value(form, "description_ja"),
        description_en: form_value(form, "description_en"),
        is_active: form.contains_key("is_active"),
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn render_material_create_response(status: StatusCode, view: &MaterialCreateView) -> Response {
    match render_material_create(view) {
        Ok(html) => html_response(status, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render material create: {error}"),
        ),
    }
}

fn render_material_create_response_with_trigger(
    status: StatusCode,
    view: &MaterialCreateView,
    trigger: &str,
) -> Response {
    match render_material_create(view) {
        Ok(html) => html_response_with_trigger(status, html, trigger),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render material create: {error}"),
        ),
    }
}

fn new_stone_listing_create_view(message: &str, render_error: &str) -> StoneListingCreateView {
    StoneListingCreateView {
        stone_listing_key: String::new(),
        listing_code: String::new(),
        material_key: String::new(),
        size: String::new(),
        title_ja: String::new(),
        title_en: String::new(),
        description_ja: String::new(),
        description_en: String::new(),
        story_ja: String::new(),
        story_en: String::new(),
        color_family: String::new(),
        color_tags: String::new(),
        pattern_primary: String::new(),
        pattern_tags: String::new(),
        stone_shape: "square".to_owned(),
        translucency: String::new(),
        price_usd: "0".to_owned(),
        price_jpy: "0".to_owned(),
        sort_order: "0".to_owned(),
        photo_storage_path: String::new(),
        photo_alt_ja: String::new(),
        photo_alt_en: String::new(),
        status: "draft".to_owned(),
        is_active: true,
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn stone_listing_create_view_from_form(
    form: &HashMap<String, String>,
    message: &str,
    render_error: &str,
) -> StoneListingCreateView {
    StoneListingCreateView {
        stone_listing_key: form_value(form, "stone_listing_key"),
        listing_code: form_value(form, "listing_code"),
        material_key: form_value(form, "material_key"),
        size: form_value(form, "size"),
        title_ja: form_value(form, "title_ja"),
        title_en: form_value(form, "title_en"),
        description_ja: form_value(form, "description_ja"),
        description_en: form_value(form, "description_en"),
        story_ja: form_value(form, "story_ja"),
        story_en: form_value(form, "story_en"),
        color_family: form_value(form, "color_family"),
        color_tags: form_value(form, "color_tags"),
        pattern_primary: form_value(form, "pattern_primary"),
        pattern_tags: form_value(form, "pattern_tags"),
        stone_shape: normalize_stone_shape_optional(&form_value(form, "stone_shape"))
            .unwrap_or_else(|| "square".to_owned()),
        translucency: form_value(form, "translucency"),
        price_usd: form_value(form, "price_usd"),
        price_jpy: form_value(form, "price_jpy"),
        sort_order: form_value(form, "sort_order"),
        photo_storage_path: form_value(form, "photo_storage_path"),
        photo_alt_ja: form_value(form, "photo_alt_ja"),
        photo_alt_en: form_value(form, "photo_alt_en"),
        status: normalize_stone_listing_status(&form_value(form, "status"))
            .unwrap_or("draft")
            .to_owned(),
        is_active: form.contains_key("is_active"),
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn stone_listing_create_input_from_form(
    form: &HashMap<String, String>,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
) -> StoneListingCreateInput {
    let stone_listing_key = form_value(form, "stone_listing_key");
    let listing_code = form_value(form, "listing_code");
    let material_key = form_value(form, "material_key");
    let size = form_value(form, "size");
    let title_ja = form_value(form, "title_ja");
    let title_en = form_value(form, "title_en");
    let description_ja = form_value(form, "description_ja");
    let description_en = form_value(form, "description_en");
    let story_ja = form_value(form, "story_ja");
    let story_en = form_value(form, "story_en");
    let color_family = form_value(form, "color_family");
    let color_tags = parse_comma_separated_values(&form_value(form, "color_tags"))
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>();
    let pattern_primary = form_value(form, "pattern_primary");
    let pattern_tags = parse_comma_separated_values(&form_value(form, "pattern_tags"))
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>();
    let stone_shape = form_value(form, "stone_shape");
    let translucency = form_value(form, "translucency");
    let photo_storage_path = form_value(form, "photo_storage_path");
    let photo_alt_ja = form_value(form, "photo_alt_ja");
    let photo_alt_en = form_value(form, "photo_alt_en");
    let status = form_value(form, "status");
    let is_active = form.contains_key("is_active");

    StoneListingCreateInput {
        stone_listing_key,
        listing_code,
        material_key,
        size,
        title_ja,
        title_en,
        description_ja,
        description_en,
        story_ja,
        story_en,
        color_family,
        color_tags,
        pattern_primary,
        pattern_tags,
        stone_shape,
        translucency,
        price_usd,
        price_jpy,
        sort_order,
        photo_storage_path,
        photo_alt_ja,
        photo_alt_en,
        status,
        is_active,
    }
}

fn stone_listing_patch_input_from_form(
    form: &HashMap<String, String>,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
) -> StoneListingPatchInput {
    let listing_code = form_value(form, "listing_code");
    let material_key = form_value(form, "material_key");
    let size = form_value(form, "size");
    let title_ja = form_value(form, "title_ja");
    let title_en = form_value(form, "title_en");
    let description_ja = form_value(form, "description_ja");
    let description_en = form_value(form, "description_en");
    let story_ja = form_value(form, "story_ja");
    let story_en = form_value(form, "story_en");
    let color_family = form_value(form, "color_family");
    let color_tags = parse_comma_separated_values(&form_value(form, "color_tags"))
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>();
    let pattern_primary = form_value(form, "pattern_primary");
    let pattern_tags = parse_comma_separated_values(&form_value(form, "pattern_tags"))
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>();
    let stone_shape = form_value(form, "stone_shape");
    let translucency = form_value(form, "translucency");
    let photo_storage_path = form_value(form, "photo_storage_path");
    let photo_alt_ja = form_value(form, "photo_alt_ja");
    let photo_alt_en = form_value(form, "photo_alt_en");
    let status = form_value(form, "status");
    let is_active = form.contains_key("is_active");

    StoneListingPatchInput {
        listing_code,
        material_key,
        size,
        title_ja,
        title_en,
        description_ja,
        description_en,
        story_ja,
        story_en,
        color_family,
        color_tags,
        pattern_primary,
        pattern_tags,
        stone_shape,
        translucency,
        price_usd,
        price_jpy,
        sort_order,
        photo_storage_path,
        photo_alt_ja,
        photo_alt_en,
        title_i18n_updates: parse_localized_form_updates(form, "title_i18n"),
        description_i18n_updates: parse_localized_form_updates(form, "description_i18n"),
        story_i18n_updates: parse_localized_form_updates(form, "story_i18n"),
        photo_alt_i18n_updates: parse_localized_form_updates(form, "photo_alt_i18n"),
        status,
        is_active,
    }
}

fn render_stone_listing_create_response(
    status: StatusCode,
    view: &StoneListingCreateView,
) -> Response {
    match render_stone_listing_create(view) {
        Ok(html) => html_response(status, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render stone listing create: {error}"),
        ),
    }
}

fn render_stone_listing_create_response_with_trigger(
    status: StatusCode,
    view: &StoneListingCreateView,
    trigger: &str,
) -> Response {
    match render_stone_listing_create(view) {
        Ok(html) => html_response_with_trigger(status, html, trigger),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render stone listing create: {error}"),
        ),
    }
}

fn new_font_create_view(message: &str, render_error: &str) -> FontCreateView {
    FontCreateView {
        key: String::new(),
        label: String::new(),
        font_family: String::new(),
        kanji_style: "japanese".to_owned(),
        sort_order: "0".to_owned(),
        is_active: true,
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn font_create_view_from_form(
    form: &HashMap<String, String>,
    message: &str,
    render_error: &str,
) -> FontCreateView {
    FontCreateView {
        key: form_value(form, "key"),
        label: form_value(form, "label"),
        font_family: form_value(form, "font_family"),
        kanji_style: normalize_kanji_style(&form_value(form, "kanji_style"))
            .unwrap_or("japanese")
            .to_owned(),
        sort_order: form_value(form, "sort_order"),
        is_active: form.contains_key("is_active"),
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn render_font_create_response(status: StatusCode, view: &FontCreateView) -> Response {
    match render_font_create(view) {
        Ok(html) => html_response(status, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render font create: {error}"),
        ),
    }
}

fn render_font_create_response_with_trigger(
    status: StatusCode,
    view: &FontCreateView,
    trigger: &str,
) -> Response {
    match render_font_create(view) {
        Ok(html) => html_response_with_trigger(status, html, trigger),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render font create: {error}"),
        ),
    }
}

fn new_country_create_view(message: &str, render_error: &str) -> CountryCreateView {
    CountryCreateView {
        code: String::new(),
        label_ja: String::new(),
        label_en: String::new(),
        shipping_fee_usd: "0".to_owned(),
        shipping_fee_jpy: "0".to_owned(),
        sort_order: "0".to_owned(),
        is_active: true,
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn country_create_view_from_form(
    form: &HashMap<String, String>,
    message: &str,
    render_error: &str,
) -> CountryCreateView {
    CountryCreateView {
        code: form_value(form, "code"),
        label_ja: form_value(form, "label_ja"),
        label_en: form_value(form, "label_en"),
        shipping_fee_usd: form_value(form, "shipping_fee_usd"),
        shipping_fee_jpy: form_value(form, "shipping_fee_jpy"),
        sort_order: form_value(form, "sort_order"),
        is_active: form.contains_key("is_active"),
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
    }
}

fn render_country_create_response(status: StatusCode, view: &CountryCreateView) -> Response {
    match render_country_create(view) {
        Ok(html) => html_response(status, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render country create: {error}"),
        ),
    }
}

fn render_country_create_response_with_trigger(
    status: StatusCode,
    view: &CountryCreateView,
    trigger: &str,
) -> Response {
    match render_country_create(view) {
        Ok(html) => html_response_with_trigger(status, html, trigger),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render country create: {error}"),
        ),
    }
}

fn new_facet_tag_create_view(
    message: &str,
    render_error: &str,
    key_error: &str,
    aliases_error: &str,
) -> FacetTagCreateView {
    FacetTagCreateView {
        key: String::new(),
        facet_type: "color".to_owned(),
        label_ja: String::new(),
        label_en: String::new(),
        aliases: String::new(),
        sort_order: "0".to_owned(),
        is_active: true,
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
        key_error: key_error.to_owned(),
        has_key_error: !key_error.is_empty(),
        aliases_error: aliases_error.to_owned(),
        has_aliases_error: !aliases_error.is_empty(),
    }
}

fn facet_tag_create_view_from_form(
    form: &HashMap<String, String>,
    message: &str,
    render_error: &str,
    key_error: &str,
    aliases_error: &str,
) -> FacetTagCreateView {
    FacetTagCreateView {
        key: form_value(form, "key"),
        facet_type: normalize_facet_tag_type(&form_value(form, "facet_type"))
            .unwrap_or("color")
            .to_owned(),
        label_ja: form_value(form, "label_ja"),
        label_en: form_value(form, "label_en"),
        aliases: form_value(form, "aliases"),
        sort_order: form_value(form, "sort_order"),
        is_active: form.contains_key("is_active"),
        message: message.to_owned(),
        has_message: !message.is_empty(),
        error: render_error.to_owned(),
        has_error: !render_error.is_empty(),
        key_error: key_error.to_owned(),
        has_key_error: !key_error.is_empty(),
        aliases_error: aliases_error.to_owned(),
        has_aliases_error: !aliases_error.is_empty(),
    }
}

fn facet_tag_patch_input_from_form(
    form: &HashMap<String, String>,
    sort_order: i64,
) -> FacetTagPatchInput {
    FacetTagPatchInput {
        label_ja: form_value(form, "label_ja"),
        label_en: form_value(form, "label_en"),
        label_i18n_updates: parse_localized_form_updates(form, "label_i18n"),
        aliases: parse_comma_separated_values(&form_value(form, "aliases")),
        sort_order,
        is_active: form.contains_key("is_active"),
    }
}

fn render_facet_tag_create_response(status: StatusCode, view: &FacetTagCreateView) -> Response {
    match render_facet_tag_create(view) {
        Ok(html) => html_response(status, html),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render facet tag create: {error}"),
        ),
    }
}

fn render_facet_tag_create_response_with_trigger(
    status: StatusCode,
    view: &FacetTagCreateView,
    trigger: &str,
) -> Response {
    match render_facet_tag_create(view) {
        Ok(html) => html_response_with_trigger(status, html, trigger),
        Err(error) => plain_error(
            StatusCode::INTERNAL_SERVER_ERROR,
            format!("failed to render facet tag create: {error}"),
        ),
    }
}

fn validate_material_key(key: &str) -> std::result::Result<(), String> {
    if key.is_empty() {
        return Err("材質キーは必須です。".to_owned());
    }
    if key.len() > 64 {
        return Err("材質キーは 64 文字以内で入力してください。".to_owned());
    }
    if key.starts_with("__") && key.ends_with("__") {
        return Err("材質キーに `__...__` 形式は利用できません。".to_owned());
    }
    if !key
        .chars()
        .all(|ch| ch.is_ascii_lowercase() || ch.is_ascii_digit() || matches!(ch, '-' | '_'))
    {
        return Err(
            "材質キーは英小文字・数字・ハイフン・アンダースコアのみ使用できます。".to_owned(),
        );
    }
    Ok(())
}

fn validate_font_key(key: &str) -> std::result::Result<(), String> {
    if key.is_empty() {
        return Err("フォントキーは必須です。".to_owned());
    }
    if key.len() > 64 {
        return Err("フォントキーは 64 文字以内で入力してください。".to_owned());
    }
    if key.starts_with("__") && key.ends_with("__") {
        return Err("フォントキーに `__...__` 形式は利用できません。".to_owned());
    }
    if !key
        .chars()
        .all(|ch| ch.is_ascii_lowercase() || ch.is_ascii_digit() || matches!(ch, '-' | '_'))
    {
        return Err(
            "フォントキーは英小文字・数字・ハイフン・アンダースコアのみ使用できます。".to_owned(),
        );
    }
    Ok(())
}

fn validate_stone_listing_code(code: &str) -> std::result::Result<(), String> {
    if code.is_empty() {
        return Err("一点物コードは必須です。".to_owned());
    }
    if code.len() > 64 {
        return Err("一点物コードは 64 文字以内で入力してください。".to_owned());
    }
    if code.starts_with("__") && code.ends_with("__") {
        return Err("一点物コードに `__...__` 形式は利用できません。".to_owned());
    }
    if !code
        .chars()
        .all(|ch| ch.is_ascii_uppercase() || ch.is_ascii_digit() || matches!(ch, '-' | '_'))
    {
        return Err(
            "一点物コードは英大文字・数字・ハイフン・アンダースコアのみ使用できます。".to_owned(),
        );
    }
    Ok(())
}

fn normalize_faceted_token(raw: &str) -> Option<String> {
    let normalized = raw.trim();
    if normalized.is_empty() {
        return None;
    }

    let normalized = normalized
        .chars()
        .map(|ch| if ch.is_ascii_whitespace() { '_' } else { ch })
        .collect::<String>()
        .to_ascii_lowercase()
        .trim_matches('_')
        .to_owned();

    if normalized.is_empty() {
        None
    } else {
        Some(normalized)
    }
}

fn normalize_optional_faceted_token(raw: &str) -> String {
    normalize_faceted_token(raw).unwrap_or_default()
}

fn parse_comma_separated_values(raw: &str) -> Vec<String> {
    let mut values = raw
        .split(|ch| matches!(ch, ',' | '\n' | '\r' | '、' | '，'))
        .map(|value| value.trim().to_owned())
        .filter(|value| !value.is_empty())
        .collect::<Vec<_>>();
    values.sort();
    values.dedup();
    values
}

fn validate_stone_listing_values(
    title_ja: &str,
    title_en: &str,
    description_ja: &str,
    description_en: &str,
    size: &str,
    color_family: &str,
    pattern_primary: &str,
    stone_shape: &str,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
    status: &str,
    material_key: &str,
    listing_code: &str,
    photo_storage_path: &str,
) -> std::result::Result<(), String> {
    if title_ja.is_empty() || title_en.is_empty() {
        return Err("一点物タイトル（ja/en）は必須です。".to_owned());
    }
    if description_ja.is_empty() || description_en.is_empty() {
        return Err("一点物説明（ja/en）は必須です。".to_owned());
    }
    if size.chars().count() > 80 {
        return Err("サイズは 80 文字以内で入力してください。".to_owned());
    }
    if color_family.is_empty() {
        return Err("色ファミリーは必須です。".to_owned());
    }
    if pattern_primary.is_empty() {
        return Err("模様の代表値は必須です。".to_owned());
    }
    if stone_shape.trim().is_empty() {
        return Err("石の形は必須です。".to_owned());
    }
    if price_usd < 0 {
        return Err("価格（USD cents）は 0 以上で入力してください。".to_owned());
    }
    if price_jpy < 0 {
        return Err("価格（JPY）は 0 以上で入力してください。".to_owned());
    }
    if sort_order < 0 {
        return Err("表示順は 0 以上で入力してください。".to_owned());
    }
    if normalize_stone_listing_status(status).is_none() {
        return Err("公開状態を選択してください。".to_owned());
    }
    validate_material_key(material_key)?;
    validate_stone_listing_code(listing_code)?;
    validate_material_photo_storage_path(photo_storage_path)?;
    Ok(())
}

fn normalize_stone_listing_status(raw: &str) -> Option<&'static str> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "draft" => Some("draft"),
        "published" => Some("published"),
        "reserved" => Some("reserved"),
        "sold" => Some("sold"),
        "archived" => Some("archived"),
        _ => None,
    }
}

fn validate_country_code(code: &str) -> std::result::Result<(), String> {
    if code.is_empty() {
        return Err("国コードは必須です。".to_owned());
    }
    if code.chars().count() != 2 || !code.chars().all(|ch| ch.is_ascii_alphabetic()) {
        return Err("国コードは ISO alpha-2（英字2文字）で入力してください。".to_owned());
    }
    Ok(())
}

fn validate_material_values(
    label_ja: &str,
    label_en: &str,
    description_ja: &str,
    description_en: &str,
) -> std::result::Result<(), String> {
    if label_ja.is_empty() || label_en.is_empty() {
        return Err("材質名（ja/en）は必須です。".to_owned());
    }
    if description_ja.is_empty() || description_en.is_empty() {
        return Err("説明文（ja/en）は必須です。".to_owned());
    }
    Ok(())
}

fn parse_usd_cents_input(raw: &str) -> std::result::Result<i64, String> {
    let value = raw.trim();
    if value.is_empty() {
        return Err("価格（USD）は必須です。".to_owned());
    }

    if let Some((whole, fraction)) = value.split_once('.') {
        if whole.trim().is_empty() {
            return Err(
                "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。"
                    .to_owned(),
            );
        }
        let dollars = whole.trim().parse::<i64>().map_err(|_| {
            "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。"
                .to_owned()
        })?;
        if dollars < 0 {
            return Err("価格（USD）は 0 以上で入力してください。".to_owned());
        }

        let fraction = fraction.trim();
        if fraction.is_empty() {
            return dollars
                .checked_mul(100)
                .ok_or_else(|| "価格（USD）が大きすぎます。".to_owned());
        }
        if !fraction.chars().all(|ch| ch.is_ascii_digit()) {
            return Err(
                "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。"
                    .to_owned(),
            );
        }
        if fraction.len() > 2 {
            return Err("価格（USD）の小数は 2 桁までです。".to_owned());
        }

        let cents =
            match fraction.len() {
                1 => fraction.parse::<i64>().map_err(|_| {
                    "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。"
                        .to_owned()
                })? * 10,
                2 => fraction.parse::<i64>().map_err(|_| {
                    "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。"
                        .to_owned()
                })?,
                _ => 0,
            };

        return dollars
            .checked_mul(100)
            .and_then(|value| value.checked_add(cents))
            .ok_or_else(|| "価格（USD）が大きすぎます。".to_owned());
    }

    let cents = value.parse::<i64>().map_err(|_| {
        "価格（USD）は整数の cents、または 165.00 のような小数表記で入力してください。".to_owned()
    })?;
    if cents < 0 {
        return Err("価格（USD）は 0 以上で入力してください。".to_owned());
    }

    Ok(cents)
}

fn validate_font_values(
    label: &str,
    font_family: &str,
    kanji_style: &str,
    sort_order: i64,
) -> std::result::Result<String, String> {
    if label.is_empty() {
        return Err("フォント名は必須です。".to_owned());
    }
    if font_family.is_empty() {
        return Err("font-family は必須です。".to_owned());
    }
    if normalize_kanji_style(kanji_style).is_none() {
        return Err("スタイルは日本・中国・台湾から選択してください。".to_owned());
    }
    if sort_order < 0 {
        return Err("表示順は 0 以上で入力してください。".to_owned());
    }
    build_google_fonts_stylesheet_url(font_family)
}

fn build_google_fonts_stylesheet_url(font_family: &str) -> std::result::Result<String, String> {
    let first_font_name = extract_primary_font_name(font_family)
        .ok_or_else(|| "font-family の先頭にフォント名を指定してください。".to_owned())?;
    if is_generic_css_font_family(&first_font_name) {
        return Err(
            "font-family の先頭には Google Fonts のフォント名を指定してください。".to_owned(),
        );
    }

    let mut url = reqwest::Url::parse("https://fonts.googleapis.com/css2")
        .map_err(|_| "Google Fonts URL の生成に失敗しました。".to_owned())?;
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

fn validate_country_values(
    label_ja: &str,
    label_en: &str,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    sort_order: i64,
) -> std::result::Result<(), String> {
    if label_ja.is_empty() || label_en.is_empty() {
        return Err("配送国名（ja/en）は必須です。".to_owned());
    }
    if shipping_fee_usd < 0 {
        return Err("送料（USD cents）は 0 以上で入力してください。".to_owned());
    }
    if shipping_fee_jpy < 0 {
        return Err("送料（JPY）は 0 以上で入力してください。".to_owned());
    }
    if sort_order < 0 {
        return Err("表示順は 0 以上で入力してください。".to_owned());
    }
    Ok(())
}

fn normalize_storage_path(value: &str) -> String {
    value.trim().trim_start_matches('/').to_owned()
}

fn normalize_material_shape(raw: &str) -> Option<&'static str> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "square" => Some("square"),
        "round" => Some("round"),
        _ => None,
    }
}

fn normalize_stone_shape(raw: &str) -> String {
    // Preserve unknown values so existing Firestore data is not coerced to a known shape.
    match raw.trim().to_ascii_lowercase().as_str() {
        "round" => "round".to_owned(),
        "square" => "square".to_owned(),
        _ => raw.trim().to_owned(),
    }
}

fn normalize_stone_shape_optional(raw: &str) -> Option<String> {
    let normalized = normalize_stone_shape(raw);
    if normalized.is_empty() {
        None
    } else {
        Some(normalized)
    }
}

fn normalize_kanji_style(raw: &str) -> Option<&'static str> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "japanese" | "japan" | "jp" => Some("japanese"),
        "chinese" | "china" | "cn" => Some("chinese"),
        "taiwanese" | "taiwan" | "tw" => Some("taiwanese"),
        _ => None,
    }
}

fn kanji_style_label(style: &str) -> &'static str {
    match normalize_kanji_style(style) {
        Some("chinese") => "中国スタイル",
        Some("taiwanese") => "台湾スタイル",
        _ => "日本スタイル",
    }
}

fn material_shape_or_default(raw: &str) -> String {
    normalize_material_shape(raw).unwrap_or("square").to_owned()
}

fn stone_shape_label(shape: &str) -> String {
    match normalize_stone_shape(shape).as_str() {
        "square" => "四角形".to_owned(),
        "round" => "丸形".to_owned(),
        other => other.to_owned(),
    }
}

fn stone_listing_is_published(status: &str) -> bool {
    status.trim().eq_ignore_ascii_case("published")
}

fn stone_listing_status_label(status: &str) -> &'static str {
    match normalize_stone_listing_status(status).unwrap_or("draft") {
        "draft" => "下書き",
        "published" => "公開中",
        "reserved" => "予約中",
        "sold" => "売却済み",
        "archived" => "保管",
        _ => "下書き",
    }
}

fn stone_listing_status_after_order_status(status: &str) -> Option<&'static str> {
    match status.trim() {
        "paid" => Some("sold"),
        "canceled" => Some("published"),
        _ => None,
    }
}

fn stone_listing_should_restore_after_canceled_order(
    current_status: &str,
    current_published_at_is_none: bool,
) -> bool {
    let current_status = current_status.trim();
    current_status.eq_ignore_ascii_case("reserved")
        || (current_status.eq_ignore_ascii_case("published") && current_published_at_is_none)
}

fn stone_listing_published_at_sort_key(listing: &StoneListing) -> DateTime<Utc> {
    if stone_listing_is_published(&listing.status) {
        listing.published_at.unwrap_or(DateTime::<Utc>::UNIX_EPOCH)
    } else {
        DateTime::<Utc>::UNIX_EPOCH
    }
}

fn material_comparison_profile(material_key: &str) -> MaterialComparisonProfile {
    match material_key {
        "rose_quartz" => MaterialComparisonProfile {
            texture_ja: "淡い桃色のやわらかな透明感",
            texture_en: "Soft, translucent pink sheen",
            weight_ja: "やや軽やかで手になじみやすい",
            weight_en: "Light and gentle to handle",
            usage_ja: "やわらかな印象を出しやすい",
            usage_en: "A soft, friendly finish",
        },
        "lapis_lazuli" => MaterialComparisonProfile {
            texture_ja: "深い青にきらめきが入る石目",
            texture_en: "Deep blue stone with bright flecks",
            weight_ja: "ほどよい重さで存在感がある",
            weight_en: "Medium-heavy with a strong presence",
            usage_ja: "印象を強めやすい",
            usage_en: "A vivid, distinctive finish",
        },
        "jade" => MaterialComparisonProfile {
            texture_ja: "しっとりした緑石の艶感",
            texture_en: "Polished green stone with a calm sheen",
            weight_ja: "ほどよく重く、落ち着いた安定感",
            weight_en: "Substantial and steady",
            usage_ja: "落ち着いた格調を出しやすい",
            usage_en: "A calm, dignified finish",
        },
        "boxwood" => MaterialComparisonProfile {
            texture_ja: "木目がやわらかく見える素朴な質感",
            texture_en: "A simple wood grain with a gentle feel",
            weight_ja: "軽くて扱いやすい",
            weight_en: "Light and easy to handle",
            usage_ja: "日常使いしやすい定番",
            usage_en: "A dependable everyday standard",
        },
        "black_buffalo" => MaterialComparisonProfile {
            texture_ja: "深い黒のしっとりした質感",
            texture_en: "Smooth, deep black finish",
            weight_ja: "ほどよい重さで落ち着きがある",
            weight_en: "Moderately weighted and steady",
            usage_ja: "落ち着いた印象を出したいときに向く",
            usage_en: "Well suited to a calm, formal look",
        },
        "a_maru" => MaterialComparisonProfile {
            texture_ja: "標準的な質感",
            texture_en: "Balanced, neutral texture",
            weight_ja: "中程度の重さ",
            weight_en: "Medium weight",
            usage_ja: "汎用的",
            usage_en: "Versatile",
        },
        _ => MaterialComparisonProfile {
            texture_ja: "標準的な質感",
            texture_en: "Balanced, neutral texture",
            weight_ja: "中程度の重さ",
            weight_en: "Medium weight",
            usage_ja: "汎用的",
            usage_en: "Versatile",
        },
    }
}

fn material_master_seeds() -> Vec<MaterialMasterSeed> {
    vec![
        MaterialMasterSeed {
            key: "wood",
            label_ja: "木材",
            label_en: "Wood",
            description_ja: "自然な木目と軽さがあり、あたたかみのある印象に仕上がる材質です。",
            description_en: "A lightweight material with natural grain that gives the seal a warm, organic feel.",
            sort_order: 10,
        },
        MaterialMasterSeed {
            key: "qingtian_stone",
            label_ja: "青田石",
            label_en: "Qingtian Stone",
            description_ja: "中国浙江省青田産として知られる代表的な篆刻石で、きめ細かく彫りやすい石材です。",
            description_en: "A classic seal-carving stone associated with Qingtian, Zhejiang, known for its fine texture and ease of carving.",
            sort_order: 20,
        },
        MaterialMasterSeed {
            key: "shoushan_stone",
            label_ja: "寿山石",
            label_en: "Shoushan Stone",
            description_ja: "中国福建省寿山産として知られる印材石で、色味の幅と滑らかな彫り心地が特徴です。",
            description_en: "A seal stone associated with Shoushan, Fujian, valued for its range of colors and smooth carving feel.",
            sort_order: 30,
        },
        MaterialMasterSeed {
            key: "balin_stone",
            label_ja: "巴林石",
            label_en: "Balin Stone",
            description_ja: "中国内モンゴル産として知られる印材石で、色柄の変化とほどよい硬さを持つ材質です。",
            description_en: "A seal stone associated with Inner Mongolia, known for varied colors and patterns with moderate hardness.",
            sort_order: 40,
        },
        MaterialMasterSeed {
            key: "yili_stone",
            label_ja: "伊犁石",
            label_en: "Yili Stone",
            description_ja: "中国新疆ウイグル自治区の伊犁地域に由来する印材石で、落ち着いた色味と素朴な石質が特徴です。",
            description_en: "A seal stone associated with the Yili region of Xinjiang, with subdued colors and a natural stone texture.",
            sort_order: 50,
        },
        MaterialMasterSeed {
            key: "laos_stone",
            label_ja: "ラオス石",
            label_en: "Laos Stone",
            description_ja: "ラオス産として流通する印材石で、やわらかめの石質と彫刻しやすさが特徴です。",
            description_en: "A seal stone commonly traded as Laos stone, known for its softer texture and ease of carving.",
            sort_order: 60,
        },
        MaterialMasterSeed {
            key: "xixia_stone",
            label_ja: "西峡石",
            label_en: "Xixia Stone",
            description_ja: "中国河南省西峡産として知られる印材石で、緻密で落ち着いた質感を持つ材質です。",
            description_en: "A seal stone associated with Xixia, Henan, with a dense structure and calm, refined texture.",
            sort_order: 70,
        },
        MaterialMasterSeed {
            key: "frozen_stone",
            label_ja: "凍石",
            label_en: "Frozen Stone",
            description_ja: "凍ったような半透明感としっとりした質感が特徴の、篆刻向けの石材です。",
            description_en: "A seal-carving stone with a moist, semi-translucent appearance reminiscent of frozen stone.",
            sort_order: 80,
        },
    ]
}

fn normalize_storage_bucket_name(value: &str) -> String {
    value
        .trim()
        .trim_start_matches("gs://")
        .trim_start_matches("GS://")
        .trim_matches('/')
        .to_owned()
}

fn validate_storage_bucket_name(bucket: &str) -> std::result::Result<(), String> {
    if bucket.is_empty() {
        return Err("bucket name is empty".to_owned());
    }
    if bucket.chars().any(|ch| ch.is_whitespace()) {
        return Err("bucket name must not contain whitespace".to_owned());
    }
    if bucket.contains('/') {
        return Err("bucket must be a bucket name only (no path segments)".to_owned());
    }
    Ok(())
}

fn normalize_image_content_type(
    file_name: &str,
    content_type: Option<&str>,
) -> Option<&'static str> {
    let normalized_content_type = content_type
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(|value| value.to_ascii_lowercase())
        .and_then(|value| match value.as_str() {
            "image/jpeg" | "image/jpg" => Some("image/jpeg"),
            "image/png" => Some("image/png"),
            "image/webp" => Some("image/webp"),
            "image/gif" => Some("image/gif"),
            "image/avif" => Some("image/avif"),
            _ => None,
        });

    if normalized_content_type.is_some() {
        return normalized_content_type;
    }

    let extension = file_name
        .rsplit_once('.')
        .map(|(_, ext)| ext.trim().to_ascii_lowercase())
        .unwrap_or_default();

    match extension.as_str() {
        "jpg" | "jpeg" => Some("image/jpeg"),
        "png" => Some("image/png"),
        "webp" => Some("image/webp"),
        "gif" => Some("image/gif"),
        "avif" => Some("image/avif"),
        _ => None,
    }
}

fn image_extension_for_content_type(content_type: &str) -> &'static str {
    match content_type {
        "image/jpeg" => "jpg",
        "image/png" => "png",
        "image/gif" => "gif",
        "image/avif" => "avif",
        _ => "webp",
    }
}

fn build_storage_path_for_uploaded_photo(
    material_key: &str,
    file_name: &str,
    content_type: Option<&str>,
) -> String {
    let normalized_content_type =
        normalize_image_content_type(file_name, content_type).unwrap_or("image/webp");
    let extension = image_extension_for_content_type(normalized_content_type);
    let asset_id = format!("mat_{}", Uuid::new_v4().simple());
    format!("materials/{material_key}/{asset_id}.{extension}")
}

fn select_primary_material_photo(photos: &[MaterialPhoto]) -> Option<&MaterialPhoto> {
    photos
        .iter()
        .find(|photo| photo.is_primary)
        .or_else(|| photos.first())
}

fn build_storage_path_for_uploaded_stone_listing_photo(
    stone_listing_key: &str,
    file_name: &str,
    content_type: Option<&str>,
) -> String {
    let normalized_content_type =
        normalize_image_content_type(file_name, content_type).unwrap_or("image/webp");
    let extension = image_extension_for_content_type(normalized_content_type);
    let asset_id = format!("lst_{}", Uuid::new_v4().simple());
    format!("stone_listings/{stone_listing_key}/{asset_id}.{extension}")
}

fn build_single_stone_listing_photos(
    stone_listing_key: &str,
    storage_path: &str,
    alt_ja: &str,
    alt_en: &str,
) -> Vec<MaterialPhoto> {
    if storage_path.is_empty() {
        return Vec::new();
    }

    let mut alt_i18n = HashMap::new();
    merge_optional_admin_localized_map_values(&mut alt_i18n, &[("ja", alt_ja), ("en", alt_en)]);

    vec![MaterialPhoto {
        asset_id: format!("lst_{}_01", stone_listing_key),
        storage_path: storage_path.to_owned(),
        alt_i18n,
        sort_order: 0,
        is_primary: true,
        width: 0,
        height: 0,
    }]
}

fn merge_primary_stone_listing_photo(
    existing: &[MaterialPhoto],
    stone_listing_key: &str,
    storage_path: &str,
    alt_ja: &str,
    alt_en: &str,
) -> Vec<MaterialPhoto> {
    if storage_path.is_empty() {
        return existing.to_vec();
    }

    if existing.is_empty() {
        return build_single_stone_listing_photos(stone_listing_key, storage_path, alt_ja, alt_en);
    }

    let mut photos = existing.to_vec();
    let primary_index = photos
        .iter()
        .position(|photo| photo.is_primary)
        .unwrap_or(0);

    {
        let primary = &mut photos[primary_index];
        primary.storage_path = storage_path.to_owned();

        merge_optional_admin_localized_map_values(
            &mut primary.alt_i18n,
            &[("ja", alt_ja), ("en", alt_en)],
        );

        if primary.asset_id.trim().is_empty() {
            primary.asset_id = format!("lst_{}_01", stone_listing_key);
        }
        primary.is_primary = true;
    }

    for (index, photo) in photos.iter_mut().enumerate() {
        if index != primary_index {
            photo.is_primary = false;
        }
    }

    photos
}

fn merge_primary_stone_listing_photo_alt_updates(
    photos: &mut [MaterialPhoto],
    updates: &HashMap<String, String>,
) {
    if updates.is_empty() || photos.is_empty() {
        return;
    }

    let primary_index = photos
        .iter()
        .position(|photo| photo.is_primary)
        .unwrap_or(0);
    merge_admin_localized_extra_map_values(&mut photos[primary_index].alt_i18n, updates);
}

fn validate_material_photo_storage_path(storage_path: &str) -> std::result::Result<(), String> {
    if storage_path.is_empty() {
        return Ok(());
    }

    let lowered = storage_path.to_lowercase();
    if lowered.starts_with("http://")
        || lowered.starts_with("https://")
        || lowered.starts_with("gs://")
    {
        return Err(
            "写真は URL ではなく Storage パス（例: materials/jade/mat_jade_01.webp）を入力してください。".to_owned(),
        );
    }

    if storage_path.chars().any(|ch| ch.is_whitespace()) {
        return Err("写真パスに空白文字は使用できません。".to_owned());
    }

    Ok(())
}

fn normalize_query_value(value: Option<String>) -> String {
    value
        .map(|value| value.trim().to_owned())
        .unwrap_or_default()
}

fn stone_listing_filter_from_query(query: StoneListingsPageQuery) -> StoneListingFilter {
    stone_listing_filter_from_raw(
        query.color_family.as_deref().unwrap_or_default(),
        query.color_tags.as_deref().unwrap_or_default(),
        query.pattern_primary.as_deref().unwrap_or_default(),
        query.pattern_tags.as_deref().unwrap_or_default(),
        query.stone_shape.as_deref().unwrap_or_default(),
    )
}

fn stone_listing_filter_from_form(form: &HashMap<String, String>) -> StoneListingFilter {
    stone_listing_filter_from_raw(
        &form_value(form, "color_family"),
        &form_value(form, "color_tags"),
        &form_value(form, "pattern_primary"),
        &form_value(form, "pattern_tags"),
        &form_value(form, "stone_shape"),
    )
}

fn stone_listing_filter_from_raw(
    color_family: &str,
    color_tags: &str,
    pattern_primary: &str,
    pattern_tags: &str,
    stone_shape: &str,
) -> StoneListingFilter {
    StoneListingFilter {
        color_family: normalize_faceted_token(color_family).unwrap_or_default(),
        color_tags: normalize_faceted_tag_input(color_tags),
        pattern_primary: normalize_faceted_token(pattern_primary).unwrap_or_default(),
        pattern_tags: normalize_faceted_tag_input(pattern_tags),
        stone_shape: normalize_stone_shape_optional(stone_shape).unwrap_or_default(),
    }
}

fn stone_listing_filter_is_active(filters: &StoneListingFilter) -> bool {
    !filters.color_family.is_empty()
        || !filters.color_tags.is_empty()
        || !filters.pattern_primary.is_empty()
        || !filters.pattern_tags.is_empty()
        || !filters.stone_shape.is_empty()
}

fn stone_listing_filter_badges(filters: &StoneListingFilter) -> Vec<StoneListingFilterBadgeView> {
    let mut badges = Vec::new();

    if !filters.color_family.is_empty() {
        badges.push(StoneListingFilterBadgeView {
            label: "色".to_owned(),
            value: filters.color_family.replace('_', " "),
        });
    }
    if !filters.color_tags.is_empty() {
        badges.push(StoneListingFilterBadgeView {
            label: "色タグ".to_owned(),
            value: stone_listing_tag_value_label(&filters.color_tags),
        });
    }
    if !filters.pattern_primary.is_empty() {
        badges.push(StoneListingFilterBadgeView {
            label: "模様".to_owned(),
            value: filters.pattern_primary.replace('_', " "),
        });
    }
    if !filters.pattern_tags.is_empty() {
        badges.push(StoneListingFilterBadgeView {
            label: "模様タグ".to_owned(),
            value: stone_listing_tag_value_label(&filters.pattern_tags),
        });
    }
    if !filters.stone_shape.is_empty() {
        badges.push(StoneListingFilterBadgeView {
            label: "石の形".to_owned(),
            value: stone_shape_label(&filters.stone_shape).to_owned(),
        });
    }

    badges
}

fn stone_listing_facet_option_view(value: &str) -> StoneListingFacetOptionView {
    StoneListingFacetOptionView {
        value: value.to_owned(),
        label: value.replace('_', " "),
    }
}

fn facet_tag_display_label(tag: &FacetTag) -> String {
    let label = resolve_localized_text(&tag.label_i18n, "ja", "ja");
    if label.is_empty() {
        tag.key.replace('_', " ")
    } else {
        label
    }
}

fn facet_tag_document_id(facet_type: &str, key: &str) -> String {
    format!("{}:{}", facet_type.trim(), key.trim())
}

fn facet_tag_type_label(facet_type: &str) -> &str {
    match facet_type {
        "color" => "色",
        "pattern" => "模様",
        _ => facet_type,
    }
}

fn facet_tag_type_options() -> Vec<FacetTagTypeOptionView> {
    vec![
        FacetTagTypeOptionView {
            value: "color".to_owned(),
            label: "色".to_owned(),
        },
        FacetTagTypeOptionView {
            value: "pattern".to_owned(),
            label: "模様".to_owned(),
        },
    ]
}

fn normalize_facet_tag_type(raw: &str) -> Option<&'static str> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "color" => Some("color"),
        "pattern" => Some("pattern"),
        _ => None,
    }
}

fn normalize_facet_tag_aliases(values: &[String]) -> Vec<String> {
    let mut aliases = values
        .iter()
        .filter_map(|value| normalize_faceted_token(value))
        .collect::<Vec<_>>();
    aliases.sort();
    aliases.dedup();
    aliases
}

fn validate_facet_tag_alias_collisions(
    data: &AdminSnapshot,
    facet_type: &str,
    key: &str,
    aliases: &[String],
    excluded_id: Option<&str>,
) -> std::result::Result<(), String> {
    let mut occupied_tokens = HashMap::<String, String>::new();

    for id in &data.facet_tag_ids {
        if excluded_id.is_some_and(|excluded_id| id == excluded_id) {
            continue;
        }
        let Some(tag) = data.facet_tags.get(id) else {
            continue;
        };
        if tag.facet_type != facet_type {
            continue;
        }

        let owner = facet_tag_conflict_owner(tag);
        if let Some(token) = normalize_faceted_token(&tag.key) {
            occupied_tokens
                .entry(token)
                .or_insert_with(|| owner.clone());
        }
        for alias in &tag.aliases {
            if let Some(token) = normalize_faceted_token(alias) {
                occupied_tokens
                    .entry(token)
                    .or_insert_with(|| owner.clone());
            }
        }
    }

    let type_label = facet_tag_type_label(facet_type);
    if let Some(owner) = occupied_tokens.get(key) {
        return Err(format!(
            "{}タグのキー `{key}` は既に {owner} と重複しています。",
            type_label
        ));
    }

    for alias in aliases {
        if alias == key {
            return Err(format!(
                "{}タグの別名 `{alias}` はキーと重複しています。",
                type_label
            ));
        }
        if let Some(owner) = occupied_tokens.get(alias) {
            return Err(format!(
                "{}タグの別名 `{alias}` は既に {owner} と重複しています。",
                type_label
            ));
        }
    }

    Ok(())
}

fn facet_tag_conflict_owner(tag: &FacetTag) -> String {
    let summary = format!("{}:{}", tag.facet_type, tag.key);
    let label = facet_tag_display_label(tag);
    if label == tag.key.replace('_', " ") {
        summary
    } else {
        format!("{summary} ({label})")
    }
}

fn facet_tag_field_errors_from_message(message: &str) -> (String, String) {
    let message = message.trim();
    if message.is_empty() {
        return (String::new(), String::new());
    }

    if message.contains("別名") {
        (String::new(), message.to_owned())
    } else if message.contains("タグキー") || message.contains("キー") {
        (message.to_owned(), String::new())
    } else {
        (String::new(), String::new())
    }
}

fn validate_facet_tag_key(key: &str) -> std::result::Result<(), String> {
    if key.is_empty() {
        return Err("タグキーは必須です。".to_owned());
    }
    if key.len() > 64 {
        return Err("タグキーは 64 文字以内で入力してください。".to_owned());
    }
    if key.starts_with("__") && key.ends_with("__") {
        return Err("タグキーに `__...__` 形式は利用できません。".to_owned());
    }
    if !key
        .chars()
        .all(|ch| ch.is_ascii_lowercase() || ch.is_ascii_digit() || matches!(ch, '-' | '_'))
    {
        return Err(
            "タグキーは英小文字・数字・ハイフン・アンダースコアのみ使用できます。".to_owned(),
        );
    }
    Ok(())
}

fn normalize_faceted_tag_input(raw: &str) -> String {
    parse_comma_separated_values(raw)
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>()
        .join(", ")
}

fn facet_tag_lookup_maps(snapshot: &AdminSnapshot) -> HashMap<String, HashMap<String, String>> {
    let mut lookups = HashMap::new();

    for id in &snapshot.facet_tag_ids {
        let Some(tag) = snapshot.facet_tags.get(id) else {
            continue;
        };

        let lookup = lookups
            .entry(tag.facet_type.clone())
            .or_insert_with(HashMap::new);
        lookup
            .entry(tag.key.clone())
            .or_insert_with(|| tag.key.clone());
        for alias in &tag.aliases {
            lookup
                .entry(alias.clone())
                .or_insert_with(|| tag.key.clone());
        }
    }

    lookups
}

fn normalize_faceted_token_with_lookup(
    raw: &str,
    lookup: Option<&HashMap<String, String>>,
) -> Option<String> {
    normalize_faceted_token(raw).map(|token| {
        lookup
            .and_then(|lookup| lookup.get(&token).cloned())
            .unwrap_or(token)
    })
}

fn normalize_faceted_token_with_lookup_strict(
    raw: &str,
    lookup: Option<&HashMap<String, String>>,
    facet_label: &str,
) -> std::result::Result<Option<String>, String> {
    let Some(token) = normalize_faceted_token(raw) else {
        return Ok(None);
    };

    lookup
        .and_then(|lookup| lookup.get(&token).cloned())
        .map(Some)
        .ok_or_else(|| format!("{facet_label} `{token}` はタグマスタに存在しません。"))
}

fn normalize_faceted_tag_values_with_lookup(
    values: &[String],
    lookup: Option<&HashMap<String, String>>,
) -> Vec<String> {
    let mut normalized = values
        .iter()
        .filter_map(|value| normalize_faceted_token_with_lookup(value, lookup))
        .collect::<Vec<_>>();
    normalized.sort();
    normalized.dedup();
    normalized
}

fn normalize_faceted_tag_values_with_lookup_strict(
    values: &[String],
    lookup: Option<&HashMap<String, String>>,
    facet_label: &str,
) -> std::result::Result<Vec<String>, String> {
    let mut normalized = values
        .iter()
        .map(|value| normalize_faceted_token_with_lookup_strict(value, lookup, facet_label))
        .collect::<std::result::Result<Vec<_>, _>>()?
        .into_iter()
        .flatten()
        .collect::<Vec<_>>();
    normalized.sort();
    normalized.dedup();
    Ok(normalized)
}

fn normalize_stone_listing_filter_with_snapshot(
    filters: &StoneListingFilter,
    snapshot: &AdminSnapshot,
) -> StoneListingFilter {
    let lookups = facet_tag_lookup_maps(snapshot);

    StoneListingFilter {
        color_family: normalize_faceted_token(&filters.color_family).unwrap_or_default(),
        color_tags: normalize_faceted_tag_values_with_lookup(
            &parse_comma_separated_values(&filters.color_tags),
            lookups.get("color"),
        )
        .join(", "),
        pattern_primary: normalize_faceted_token(&filters.pattern_primary).unwrap_or_default(),
        pattern_tags: normalize_faceted_tag_values_with_lookup(
            &parse_comma_separated_values(&filters.pattern_tags),
            lookups.get("pattern"),
        )
        .join(", "),
        stone_shape: normalize_stone_shape_optional(&filters.stone_shape).unwrap_or_default(),
    }
}

fn normalize_stone_listing_facets_in_snapshot(snapshot: &mut AdminSnapshot) {
    let lookups = facet_tag_lookup_maps(snapshot);
    for listing in snapshot.stone_listings.values_mut() {
        listing.facets.color_tags = normalize_faceted_tag_values_with_lookup(
            &listing.facets.color_tags,
            lookups.get("color"),
        );
        listing.facets.pattern_tags = normalize_faceted_tag_values_with_lookup(
            &listing.facets.pattern_tags,
            lookups.get("pattern"),
        );
    }
}

fn stone_listing_tag_values(raw: &str) -> Vec<String> {
    parse_comma_separated_values(raw)
        .into_iter()
        .filter_map(|value| normalize_faceted_token(&value))
        .collect::<Vec<_>>()
}

fn stone_listing_tag_value_label(raw: &str) -> String {
    stone_listing_tag_values(raw)
        .into_iter()
        .map(|value| value.replace('_', " "))
        .collect::<Vec<_>>()
        .join(" / ")
}

fn stone_shape_filter_options() -> Vec<StoneListingFacetOptionView> {
    [("square", "四角形"), ("round", "丸形")]
        .into_iter()
        .map(|(value, label)| StoneListingFacetOptionView {
            value: value.to_owned(),
            label: label.to_owned(),
        })
        .collect()
}

fn order_status_label(status: &str) -> &str {
    match status {
        "pending_payment" => "支払い待ち",
        "paid" => "支払い済み",
        "manufacturing" => "製造中",
        "shipped" => "出荷済み",
        "delivered" => "配達完了",
        "canceled" => "キャンセル",
        "refunded" => "返金済み",
        _ => status,
    }
}

fn order_channel_label(channel: &str) -> String {
    match channel.trim().to_ascii_lowercase().as_str() {
        "app" => "App".to_owned(),
        "web" => "Web".to_owned(),
        "" => "不明".to_owned(),
        _ => channel.trim().to_owned(),
    }
}

fn resolve_order_engraving_text(
    customer_confirmation: &BTreeMap<String, JsonValue>,
    seal_line1: &str,
    seal_line2: &str,
) -> String {
    let confirmed_seal_text = read_string_field(customer_confirmation, "confirmed_seal_text");
    if !confirmed_seal_text.is_empty() {
        return confirmed_seal_text;
    }

    format!("{}{}", seal_line1.trim(), seal_line2.trim())
}

fn read_seal_preview_image_storage_path(seal: &BTreeMap<String, JsonValue>) -> String {
    let preview_image = read_map_field(seal, "preview_image");
    normalize_storage_path(&read_string_field(&preview_image, "storage_path"))
}

fn confirmation_bool_label(value: bool) -> &'static str {
    if value { "確認済み" } else { "未確認" }
}

fn payment_status_label(status: &str) -> &str {
    match status {
        "unpaid" => "未払い",
        "processing" => "処理中",
        "paid" => "支払い済み",
        "failed" => "失敗",
        "refunded" => "返金済み",
        _ => status,
    }
}

fn fulfillment_status_label(status: &str) -> &str {
    match status {
        "pending" => "未着手",
        "manufacturing" => "製造中",
        "shipped" => "出荷済み",
        "delivered" => "配達完了",
        _ => status,
    }
}

fn status_transitions(current: &str) -> &'static [&'static str] {
    match current {
        "pending_payment" => &["paid", "canceled"],
        "paid" => &["manufacturing", "refunded"],
        "manufacturing" => &["shipped", "refunded"],
        "shipped" => &["delivered", "refunded"],
        "delivered" => &[],
        "canceled" => &[],
        "refunded" => &[],
        _ => &[],
    }
}

fn status_options() -> Vec<StatusOptionView> {
    let mut keys = vec![
        "pending_payment",
        "paid",
        "manufacturing",
        "shipped",
        "delivered",
        "canceled",
        "refunded",
    ];
    keys.sort();
    keys.into_iter()
        .map(|value| StatusOptionView {
            value: value.to_owned(),
            label: order_status_label(value).to_owned(),
        })
        .collect()
}

fn next_status_options(current: &str) -> Vec<StatusOptionView> {
    status_transitions(current)
        .iter()
        .map(|value| StatusOptionView {
            value: (*value).to_owned(),
            label: order_status_label(value).to_owned(),
        })
        .collect()
}

fn shipping_transition_options(current: &str) -> Vec<StatusOptionView> {
    let mut options = vec![StatusOptionView {
        value: "none".to_owned(),
        label: "ステータス変更なし".to_owned(),
    }];

    for next in status_transitions(current) {
        if matches!(*next, "manufacturing" | "shipped" | "delivered") {
            options.push(StatusOptionView {
                value: (*next).to_owned(),
                label: order_status_label(next).to_owned(),
            });
        }
    }

    options
}

fn country_display_label(country: &Country) -> String {
    let label = resolve_localized_text(&country.label_i18n, "ja", "ja");
    if label.is_empty() {
        country.code.clone()
    } else {
        label
    }
}

fn country_label(countries: &HashMap<String, Country>, code: &str) -> String {
    let normalized = code.trim().to_uppercase();
    countries
        .get(&normalized)
        .map(country_display_label)
        .unwrap_or_else(|| {
            if normalized.is_empty() {
                code.to_owned()
            } else {
                normalized
            }
        })
}

fn apply_derived_statuses(order: &mut Order) {
    match order.status.as_str() {
        "pending_payment" => {
            order.payment_status = "unpaid".to_owned();
            order.fulfillment_status = "pending".to_owned();
        }
        "paid" => {
            order.payment_status = "paid".to_owned();
            order.fulfillment_status = "pending".to_owned();
        }
        "manufacturing" => {
            order.payment_status = "paid".to_owned();
            order.fulfillment_status = "manufacturing".to_owned();
        }
        "shipped" => {
            order.payment_status = "paid".to_owned();
            order.fulfillment_status = "shipped".to_owned();
        }
        "delivered" => {
            order.payment_status = "paid".to_owned();
            order.fulfillment_status = "delivered".to_owned();
        }
        "canceled" => {
            order.payment_status = "failed".to_owned();
            order.fulfillment_status = "pending".to_owned();
        }
        "refunded" => {
            order.payment_status = "refunded".to_owned();
            if order.fulfillment_status.is_empty() {
                order.fulfillment_status = "pending".to_owned();
            }
        }
        _ => {
            order.payment_status = "processing".to_owned();
            if order.fulfillment_status.is_empty() {
                order.fulfillment_status = "pending".to_owned();
            }
        }
    }
}

fn fill_derived_statuses(order: &mut Order) {
    if !order.payment_status.is_empty() && !order.fulfillment_status.is_empty() {
        return;
    }

    let mut derived = order.clone();
    apply_derived_statuses(&mut derived);

    if order.payment_status.is_empty() {
        order.payment_status = derived.payment_status;
    }
    if order.fulfillment_status.is_empty() {
        order.fulfillment_status = derived.fulfillment_status;
    }
}

fn resolve_order_currency(
    data: &BTreeMap<String, JsonValue>,
    pricing: &BTreeMap<String, JsonValue>,
    payment: &BTreeMap<String, JsonValue>,
    locale: &str,
) -> String {
    let mut resolved = [
        read_string_field(pricing, "currency"),
        read_string_field(pricing, "pricing_currency"),
        read_string_field(payment, "currency"),
        read_string_field(data, "currency"),
        read_string_field(data, "pricing_currency"),
    ]
    .into_iter()
    .find_map(|value| normalize_currency_code(&value));

    if let Some(expected) = currency_hint_for_locale(locale) {
        match resolved.as_deref() {
            Some(current) if current != expected => {
                eprintln!(
                    "warning: order currency mismatch locale={locale} stored={current}; using {expected}"
                );
                resolved = Some(expected.to_owned());
            }
            None => resolved = Some(expected.to_owned()),
            _ => {}
        }
    }

    resolved.unwrap_or_else(|| "USD".to_owned())
}

fn resolve_order_total(
    _data: &BTreeMap<String, JsonValue>,
    pricing: &BTreeMap<String, JsonValue>,
) -> i64 {
    if let Some(total) = read_int_field(pricing, "total") {
        return total;
    }

    0
}

fn currency_hint_for_locale(locale: &str) -> Option<&'static str> {
    let normalized = locale.trim().to_lowercase();
    if normalized.is_empty() {
        return None;
    }
    if normalized == "ja" || normalized.starts_with("ja-") {
        return Some("JPY");
    }
    if normalized == "en" || normalized.starts_with("en-") {
        return Some("USD");
    }
    None
}

fn format_datetime(value: DateTime<Utc>) -> String {
    value
        .with_timezone(&Local)
        .format("%Y-%m-%d %H:%M")
        .to_string()
}

fn format_usd(value_cents: i64) -> String {
    let sign = if value_cents < 0 { "-" } else { "" };
    let cents = value_cents.abs();
    let whole = cents / 100;
    let fraction = cents % 100;
    let whole_display = format_with_grouping(whole);
    format!("{sign}USD {whole_display}.{fraction:02}")
}

fn format_jpy(value_yen: i64) -> String {
    let sign = if value_yen < 0 { "-" } else { "" };
    let whole_display = format_with_grouping(value_yen.abs());
    format!("{sign}{whole_display}円")
}

fn format_order_amount(value: i64, currency: &str) -> String {
    if currency.trim().eq_ignore_ascii_case("JPY") {
        return format_jpy(value);
    }
    format_usd(value)
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

fn resolve_font_label_field(data: &BTreeMap<String, JsonValue>, fallback: &str) -> String {
    let label = read_string_field(data, "label");
    if !label.is_empty() {
        return label;
    }

    let localized = resolve_localized_text(&read_string_map_field(data, "label_i18n"), "ja", "ja");
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

fn encode_order_event(event: &OrderEvent) -> BTreeMap<String, JsonValue> {
    let mut fields = btree_from_pairs(vec![
        ("type", fs_string(event.kind.clone())),
        ("actor_type", fs_string(event.actor_type.clone())),
        ("created_at", fs_timestamp(event.created_at)),
    ]);

    if !event.actor_id.trim().is_empty() {
        fields.insert("actor_id".to_owned(), fs_string(event.actor_id.clone()));
    }
    if !event.before_status.trim().is_empty() {
        fields.insert(
            "before_status".to_owned(),
            fs_string(event.before_status.clone()),
        );
    }
    if !event.after_status.trim().is_empty() {
        fields.insert(
            "after_status".to_owned(),
            fs_string(event.after_status.clone()),
        );
    }
    if !event.note.trim().is_empty() {
        fields.insert("note".to_owned(), fs_string(event.note.clone()));
    }

    if event.kind == "shipment_registered" {
        let mut payload_fields = BTreeMap::new();
        let parts = event.note.splitn(2, " / ").collect::<Vec<_>>();
        if let Some(carrier) = parts.first() {
            let carrier = carrier.trim();
            if !carrier.is_empty() {
                payload_fields.insert("carrier".to_owned(), fs_string(carrier.to_owned()));
            }
        }
        if let Some(tracking_no) = parts.get(1) {
            let tracking_no = tracking_no.trim();
            if !tracking_no.is_empty() {
                payload_fields.insert("tracking_no".to_owned(), fs_string(tracking_no.to_owned()));
            }
        }

        if !payload_fields.is_empty() {
            fields.insert("payload".to_owned(), fs_map(payload_fields));
        }
    }

    fields
}

fn document_id(document: &Document) -> Option<String> {
    document
        .name
        .as_deref()
        .and_then(|name| name.rsplit('/').next())
        .map(ToOwned::to_owned)
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
        if let Some(parsed) = integer_value
            .as_u64()
            .and_then(|value| i64::try_from(value).ok())
        {
            return Some(parsed);
        }
    }

    if let Some(double_value) = value.get("doubleValue").and_then(JsonValue::as_f64) {
        return Some(double_value as i64);
    }

    value
        .as_i64()
        .or_else(|| value.as_u64().and_then(|value| i64::try_from(value).ok()))
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
        let mut container = BTreeMap::new();
        container.insert("amount".to_owned(), map_value.clone());
        if let Some(amount) = read_int_field(&container, "amount")
            && let Some(currency) = normalize_currency_map_key(map_key)
        {
            result.insert(currency, amount.max(0));
        }
    }

    result
}

fn read_bool_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Option<bool> {
    let value = data.get(key)?;
    if let Some(boolean_value) = value.get("booleanValue").and_then(JsonValue::as_bool) {
        return Some(boolean_value);
    }
    value.as_bool()
}

fn read_timestamp_field(data: &BTreeMap<String, JsonValue>, key: &str) -> Option<DateTime<Utc>> {
    let value = data.get(key)?;

    let raw = value
        .get("timestampValue")
        .and_then(JsonValue::as_str)
        .or_else(|| value.as_str())?;

    DateTime::parse_from_rfc3339(raw)
        .ok()
        .map(|value| value.with_timezone(&Utc))
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
                storage_path: normalize_storage_path(&read_string_field(&fields, "storage_path")),
                alt_i18n: read_string_map_field(&fields, "alt_i18n"),
                sort_order: read_int_field(&fields, "sort_order").unwrap_or_default(),
                is_primary: read_bool_field(&fields, "is_primary").unwrap_or(false),
                width: read_int_field(&fields, "width").unwrap_or_default(),
                height: read_int_field(&fields, "height").unwrap_or_default(),
            })
        })
        .filter(|photo| !photo.storage_path.is_empty())
        .collect::<Vec<_>>();

    photos.sort_by(|left, right| {
        let left_primary = if left.is_primary { 0 } else { 1 };
        let right_primary = if right.is_primary { 0 } else { 1 };
        left_primary
            .cmp(&right_primary)
            .then(left.sort_order.cmp(&right.sort_order))
            .then(left.asset_id.cmp(&right.asset_id))
    });

    photos
}

fn read_map_field(data: &BTreeMap<String, JsonValue>, key: &str) -> BTreeMap<String, JsonValue> {
    let Some(value) = data.get(key) else {
        return BTreeMap::new();
    };

    if let Some(fields) = value
        .get("mapValue")
        .and_then(|map_value| map_value.get("fields"))
        .and_then(JsonValue::as_object)
    {
        return fields
            .iter()
            .map(|(map_key, map_value)| (map_key.clone(), map_value.clone()))
            .collect();
    }

    value
        .as_object()
        .map(|fields| {
            fields
                .iter()
                .map(|(map_key, map_value)| (map_key.clone(), map_value.clone()))
                .collect::<BTreeMap<_, _>>()
        })
        .unwrap_or_default()
}

fn read_string_map_field(data: &BTreeMap<String, JsonValue>, key: &str) -> HashMap<String, String> {
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
        let text = map_value
            .get("stringValue")
            .and_then(JsonValue::as_str)
            .or_else(|| map_value.as_str())
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

fn stone_listing_price_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    read_int_map_field(data, "price_by_currency")
}

fn material_price_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    read_int_map_field(data, "price_by_currency")
}

fn country_shipping_fee_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    read_int_map_field(data, "shipping_fee_by_currency")
}

fn normalize_currency_code(key: &str) -> Option<String> {
    let normalized = key.trim().to_ascii_uppercase();
    if normalized.len() != 3 || !normalized.chars().all(|ch| ch.is_ascii_alphabetic()) {
        return None;
    }
    Some(normalized)
}

fn normalize_currency_map_key(key: &str) -> Option<String> {
    normalize_currency_code(key)
}

fn fs_string(value: impl Into<String>) -> JsonValue {
    json!({ "stringValue": value.into() })
}

fn fs_bool(value: bool) -> JsonValue {
    json!({ "booleanValue": value })
}

fn fs_int(value: i64) -> JsonValue {
    json!({ "integerValue": value.to_string() })
}

fn fs_timestamp(value: DateTime<Utc>) -> JsonValue {
    json!({ "timestampValue": value.to_rfc3339_opts(SecondsFormat::Secs, true) })
}

fn fs_map(fields: BTreeMap<String, JsonValue>) -> JsonValue {
    json!({ "mapValue": { "fields": fields } })
}

fn fs_array(values: Vec<JsonValue>) -> JsonValue {
    json!({ "arrayValue": { "values": values } })
}

fn fs_string_map(values: &HashMap<String, String>) -> JsonValue {
    if values.is_empty() {
        return fs_map(BTreeMap::new());
    }

    let mut keys = values.keys().cloned().collect::<Vec<_>>();
    keys.sort();

    let mut fields = BTreeMap::new();
    for key in keys {
        if let Some(value) = values.get(&key) {
            fields.insert(key, fs_string(value.clone()));
        }
    }

    fs_map(fields)
}

fn fs_string_array(values: &[String]) -> JsonValue {
    if values.is_empty() {
        return fs_array(Vec::new());
    }

    let mut items = values
        .iter()
        .map(|value| value.trim())
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
        .collect::<Vec<_>>();
    items.sort();
    items.dedup();

    fs_array(items.into_iter().map(fs_string).collect::<Vec<_>>())
}

fn fs_int_map(values: &HashMap<String, i64>) -> JsonValue {
    if values.is_empty() {
        return fs_map(BTreeMap::new());
    }

    let mut keys = values.keys().cloned().collect::<Vec<_>>();
    keys.sort();

    let mut fields = BTreeMap::new();
    for key in keys {
        if let Some(value) = values.get(&key) {
            fields.insert(key, fs_int((*value).max(0)));
        }
    }

    fs_map(fields)
}

fn fs_material_photos(photos: &[MaterialPhoto]) -> JsonValue {
    let values = photos
        .iter()
        .map(|photo| {
            let mut fields = btree_from_pairs(vec![
                ("asset_id", fs_string(photo.asset_id.clone())),
                ("storage_path", fs_string(photo.storage_path.clone())),
                ("alt_i18n", fs_string_map(&photo.alt_i18n)),
                ("sort_order", fs_int(photo.sort_order)),
                ("is_primary", fs_bool(photo.is_primary)),
            ]);

            if photo.width > 0 {
                fields.insert("width".to_owned(), fs_int(photo.width));
            }
            if photo.height > 0 {
                fields.insert("height".to_owned(), fs_int(photo.height));
            }

            fs_map(fields)
        })
        .collect::<Vec<_>>();

    fs_array(values)
}

fn fs_stone_listing_facets(facets: &StoneListingFacets) -> JsonValue {
    let mut fields = BTreeMap::new();

    if !facets.color_family.trim().is_empty() {
        fields.insert(
            "color_family".to_owned(),
            fs_string(facets.color_family.clone()),
        );
    }
    if !facets.color_tags.is_empty() {
        fields.insert("color_tags".to_owned(), fs_string_array(&facets.color_tags));
    }
    if !facets.pattern_primary.trim().is_empty() {
        fields.insert(
            "pattern_primary".to_owned(),
            fs_string(facets.pattern_primary.clone()),
        );
    }
    if !facets.pattern_tags.is_empty() {
        fields.insert(
            "pattern_tags".to_owned(),
            fs_string_array(&facets.pattern_tags),
        );
    }
    if !facets.stone_shape.trim().is_empty() {
        fields.insert(
            "stone_shape".to_owned(),
            fs_string(facets.stone_shape.clone()),
        );
    }
    if !facets.translucency.trim().is_empty() {
        fields.insert(
            "translucency".to_owned(),
            fs_string(facets.translucency.clone()),
        );
    }

    fs_map(fields)
}

fn stone_listing_snapshot_fields(listing: &StoneListing) -> BTreeMap<String, JsonValue> {
    let mut fields = btree_from_pairs(vec![
        ("listing_code", fs_string(listing.listing_code.clone())),
        ("material_key", fs_string(listing.material_key.clone())),
        ("size", fs_string(listing.size.clone())),
        ("title_i18n", fs_string_map(&listing.title_i18n)),
        ("description_i18n", fs_string_map(&listing.description_i18n)),
        ("story_i18n", fs_string_map(&listing.story_i18n)),
        ("facets", fs_stone_listing_facets(&listing.facets)),
        ("photos", fs_material_photos(&listing.photos)),
        ("price_by_currency", fs_int_map(&listing.price_by_currency)),
        ("status", fs_string(listing.status.clone())),
        ("is_active", fs_bool(listing.is_active)),
        ("sort_order", fs_int(listing.sort_order)),
        ("version", fs_int(listing.version)),
        ("updated_at", fs_timestamp(listing.updated_at)),
    ]);

    if let Some(published_at) = listing.published_at {
        fields.insert("published_at".to_owned(), fs_timestamp(published_at));
    }

    if listing.listing_code.trim().is_empty() {
        fields.remove("listing_code");
    }
    if listing.material_key.trim().is_empty() {
        fields.remove("material_key");
    }
    if listing.size.trim().is_empty() {
        fields.remove("size");
    }

    fields
}

fn facet_tag_snapshot_fields(tag: &FacetTag) -> BTreeMap<String, JsonValue> {
    btree_from_pairs(vec![
        ("key", fs_string(tag.key.clone())),
        ("facet_type", fs_string(tag.facet_type.clone())),
        ("label_i18n", fs_string_map(&tag.label_i18n)),
        ("aliases", fs_string_array(&tag.aliases)),
        ("is_active", fs_bool(tag.is_active)),
        ("sort_order", fs_int(tag.sort_order)),
        ("version", fs_int(tag.version)),
        ("updated_at", fs_timestamp(tag.updated_at)),
    ])
}

fn btree_from_pairs(pairs: Vec<(&str, JsonValue)>) -> BTreeMap<String, JsonValue> {
    pairs
        .into_iter()
        .map(|(key, value)| (key.to_owned(), value))
        .collect::<BTreeMap<_, _>>()
}

fn new_mock_snapshot() -> AdminSnapshot {
    let now = Utc::now();

    let mut orders = HashMap::from([
        (
            "ord_1007".to_owned(),
            Order {
                id: "ord_1007".to_owned(),
                order_no: "HF-20260209-1007".to_owned(),
                channel: "web".to_owned(),
                locale: "ja".to_owned(),
                currency: "JPY".to_owned(),
                listing_key: String::new(),
                status: "manufacturing".to_owned(),
                status_updated_at: now - chrono::Duration::hours(4),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: String::new(),
                carrier: String::new(),
                country_code: "JP".to_owned(),
                contact_email: "ito@example.com".to_owned(),
                seal_line1: "伊".to_owned(),
                seal_line2: "藤".to_owned(),
                engraving_text: "伊藤".to_owned(),
                ai_generation_id: String::new(),
                ai_variant_id: String::new(),
                seal_preview_image_storage_path: String::new(),
                seal_style_name: String::new(),
                seal_style_stroke_weight: String::new(),
                seal_style_balance: String::new(),
                seal_style_prompt_summary: String::new(),
                customer_confirmation_kanji_and_design: None,
                customer_confirmation_custom_made_policy: None,
                customer_confirmation_confirmed_at: None,
                customer_confirmation_confirmed_seal_text: String::new(),
                listing_label_ja: "黒水牛".to_owned(),
                total: 5400,
                created_at: now - chrono::Duration::hours(9),
                updated_at: now - chrono::Duration::hours(4),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "注文を受付".to_owned(),
                        created_at: now - chrono::Duration::hours(9),
                    },
                    OrderEvent {
                        kind: "payment_paid".to_owned(),
                        actor_type: "webhook".to_owned(),
                        actor_id: "stripe".to_owned(),
                        before_status: "pending_payment".to_owned(),
                        after_status: "paid".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(6),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: "paid".to_owned(),
                        after_status: "manufacturing".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(4),
                    },
                ],
            },
        ),
        (
            "ord_1006".to_owned(),
            Order {
                id: "ord_1006".to_owned(),
                order_no: "HF-20260209-1006".to_owned(),
                channel: "app".to_owned(),
                locale: "en".to_owned(),
                currency: "USD".to_owned(),
                listing_key: String::new(),
                status: "paid".to_owned(),
                status_updated_at: now - chrono::Duration::hours(2),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: String::new(),
                carrier: String::new(),
                country_code: "US".to_owned(),
                contact_email: "jane.smith@example.com".to_owned(),
                seal_line1: "JA".to_owned(),
                seal_line2: "NE".to_owned(),
                engraving_text: "JANE".to_owned(),
                ai_generation_id: "seal_request_006".to_owned(),
                ai_variant_id: "seal_variant_006".to_owned(),
                seal_preview_image_storage_path:
                    "seal_designs/seal_request_006/seal_variant_006.png".to_owned(),
                seal_style_name: "elegant".to_owned(),
                seal_style_stroke_weight: "standard".to_owned(),
                seal_style_balance: "balanced".to_owned(),
                seal_style_prompt_summary: "Latin initials with refined spacing".to_owned(),
                customer_confirmation_kanji_and_design: Some(true),
                customer_confirmation_custom_made_policy: Some(true),
                customer_confirmation_confirmed_at: Some(now - chrono::Duration::hours(13)),
                customer_confirmation_confirmed_seal_text: "JANE".to_owned(),
                listing_label_ja: "チタン".to_owned(),
                total: 11600,
                created_at: now - chrono::Duration::hours(12),
                updated_at: now - chrono::Duration::hours(2),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "Order accepted".to_owned(),
                        created_at: now - chrono::Duration::hours(12),
                    },
                    OrderEvent {
                        kind: "payment_paid".to_owned(),
                        actor_type: "webhook".to_owned(),
                        actor_id: "stripe".to_owned(),
                        before_status: "pending_payment".to_owned(),
                        after_status: "paid".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(2),
                    },
                ],
            },
        ),
        (
            "ord_1005".to_owned(),
            Order {
                id: "ord_1005".to_owned(),
                order_no: "HF-20260209-1005".to_owned(),
                channel: "web".to_owned(),
                locale: "ja".to_owned(),
                currency: "JPY".to_owned(),
                listing_key: String::new(),
                status: "shipped".to_owned(),
                status_updated_at: now - chrono::Duration::hours(26),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: "SGP-824901".to_owned(),
                carrier: "DHL".to_owned(),
                country_code: "SG".to_owned(),
                contact_email: "tanaka@example.com".to_owned(),
                seal_line1: "田".to_owned(),
                seal_line2: "中".to_owned(),
                engraving_text: "田中".to_owned(),
                ai_generation_id: String::new(),
                ai_variant_id: String::new(),
                seal_preview_image_storage_path: String::new(),
                seal_style_name: String::new(),
                seal_style_stroke_weight: String::new(),
                seal_style_balance: String::new(),
                seal_style_prompt_summary: String::new(),
                customer_confirmation_kanji_and_design: None,
                customer_confirmation_custom_made_policy: None,
                customer_confirmation_confirmed_at: None,
                customer_confirmation_confirmed_seal_text: String::new(),
                listing_label_ja: "柘植".to_owned(),
                total: 4900,
                created_at: now - chrono::Duration::hours(36),
                updated_at: now - chrono::Duration::hours(26),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "注文を受付".to_owned(),
                        created_at: now - chrono::Duration::hours(36),
                    },
                    OrderEvent {
                        kind: "payment_paid".to_owned(),
                        actor_type: "webhook".to_owned(),
                        actor_id: "stripe".to_owned(),
                        before_status: "pending_payment".to_owned(),
                        after_status: "paid".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(31),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: "paid".to_owned(),
                        after_status: "manufacturing".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(29),
                    },
                    OrderEvent {
                        kind: "shipment_registered".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "DHL / SGP-824901".to_owned(),
                        created_at: now - chrono::Duration::hours(26),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: "manufacturing".to_owned(),
                        after_status: "shipped".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(26),
                    },
                ],
            },
        ),
        (
            "ord_1004".to_owned(),
            Order {
                id: "ord_1004".to_owned(),
                order_no: "HF-20260208-1004".to_owned(),
                channel: "app".to_owned(),
                locale: "ja".to_owned(),
                currency: "JPY".to_owned(),
                listing_key: String::new(),
                status: "delivered".to_owned(),
                status_updated_at: now - chrono::Duration::hours(72),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: "YMT-99120".to_owned(),
                carrier: "ヤマト運輸".to_owned(),
                country_code: "JP".to_owned(),
                contact_email: "kato@example.com".to_owned(),
                seal_line1: "加".to_owned(),
                seal_line2: "藤".to_owned(),
                engraving_text: "加藤".to_owned(),
                ai_generation_id: "seal_request_004".to_owned(),
                ai_variant_id: "seal_variant_004".to_owned(),
                seal_preview_image_storage_path:
                    "seal_designs/seal_request_004/seal_variant_004.png".to_owned(),
                seal_style_name: "traditional".to_owned(),
                seal_style_stroke_weight: "bold".to_owned(),
                seal_style_balance: "dense".to_owned(),
                seal_style_prompt_summary: "Classic kanji seal with dense balance".to_owned(),
                customer_confirmation_kanji_and_design: Some(true),
                customer_confirmation_custom_made_policy: Some(true),
                customer_confirmation_confirmed_at: Some(now - chrono::Duration::hours(98)),
                customer_confirmation_confirmed_seal_text: "加藤".to_owned(),
                listing_label_ja: "柘植".to_owned(),
                total: 4200,
                created_at: now - chrono::Duration::hours(96),
                updated_at: now - chrono::Duration::hours(72),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "注文を受付".to_owned(),
                        created_at: now - chrono::Duration::hours(96),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: "shipped".to_owned(),
                        after_status: "delivered".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(72),
                    },
                ],
            },
        ),
        (
            "ord_1003".to_owned(),
            Order {
                id: "ord_1003".to_owned(),
                order_no: "HF-20260208-1003".to_owned(),
                channel: "web".to_owned(),
                locale: "en".to_owned(),
                currency: "USD".to_owned(),
                listing_key: String::new(),
                status: "pending_payment".to_owned(),
                status_updated_at: now - chrono::Duration::hours(8),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: String::new(),
                carrier: String::new(),
                country_code: "CA".to_owned(),
                contact_email: "chris@example.com".to_owned(),
                seal_line1: "CH".to_owned(),
                seal_line2: "RI".to_owned(),
                engraving_text: "CHRI".to_owned(),
                ai_generation_id: String::new(),
                ai_variant_id: String::new(),
                seal_preview_image_storage_path: String::new(),
                seal_style_name: String::new(),
                seal_style_stroke_weight: String::new(),
                seal_style_balance: String::new(),
                seal_style_prompt_summary: String::new(),
                customer_confirmation_kanji_and_design: None,
                customer_confirmation_custom_made_policy: None,
                customer_confirmation_confirmed_at: None,
                customer_confirmation_confirmed_seal_text: String::new(),
                listing_label_ja: "チタン".to_owned(),
                total: 11800,
                created_at: now - chrono::Duration::hours(30),
                updated_at: now - chrono::Duration::hours(8),
                events: vec![OrderEvent {
                    kind: "order_created".to_owned(),
                    actor_type: "system".to_owned(),
                    actor_id: "api".to_owned(),
                    before_status: String::new(),
                    after_status: String::new(),
                    note: "Order accepted".to_owned(),
                    created_at: now - chrono::Duration::hours(30),
                }],
            },
        ),
        (
            "ord_1002".to_owned(),
            Order {
                id: "ord_1002".to_owned(),
                order_no: "HF-20260207-1002".to_owned(),
                channel: "app".to_owned(),
                locale: "ja".to_owned(),
                currency: "JPY".to_owned(),
                listing_key: String::new(),
                status: "refunded".to_owned(),
                status_updated_at: now - chrono::Duration::hours(130),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: "GB-12400".to_owned(),
                carrier: "Royal Mail".to_owned(),
                country_code: "GB".to_owned(),
                contact_email: "suzuki@example.com".to_owned(),
                seal_line1: "鈴".to_owned(),
                seal_line2: "木".to_owned(),
                engraving_text: "鈴木".to_owned(),
                ai_generation_id: "seal_request_002".to_owned(),
                ai_variant_id: "seal_variant_002".to_owned(),
                seal_preview_image_storage_path:
                    "seal_designs/seal_request_002/seal_variant_002.png".to_owned(),
                seal_style_name: "soft".to_owned(),
                seal_style_stroke_weight: "standard".to_owned(),
                seal_style_balance: "airy".to_owned(),
                seal_style_prompt_summary: "Soft kanji seal with open spacing".to_owned(),
                customer_confirmation_kanji_and_design: Some(true),
                customer_confirmation_custom_made_policy: Some(true),
                customer_confirmation_confirmed_at: Some(now - chrono::Duration::hours(152)),
                customer_confirmation_confirmed_seal_text: "鈴木".to_owned(),
                listing_label_ja: "黒水牛".to_owned(),
                total: 6900,
                created_at: now - chrono::Duration::hours(150),
                updated_at: now - chrono::Duration::hours(130),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "注文を受付".to_owned(),
                        created_at: now - chrono::Duration::hours(150),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "admin".to_owned(),
                        actor_id: "admin.console".to_owned(),
                        before_status: "shipped".to_owned(),
                        after_status: "refunded".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(130),
                    },
                ],
            },
        ),
        (
            "ord_1001".to_owned(),
            Order {
                id: "ord_1001".to_owned(),
                order_no: "HF-20260207-1001".to_owned(),
                channel: "web".to_owned(),
                locale: "ja".to_owned(),
                currency: "JPY".to_owned(),
                listing_key: String::new(),
                status: "canceled".to_owned(),
                status_updated_at: now - chrono::Duration::hours(80),
                payment_status: String::new(),
                fulfillment_status: String::new(),
                tracking_no: String::new(),
                carrier: String::new(),
                country_code: "AU".to_owned(),
                contact_email: "yamada@example.com".to_owned(),
                seal_line1: "山".to_owned(),
                seal_line2: "田".to_owned(),
                engraving_text: "山田".to_owned(),
                ai_generation_id: String::new(),
                ai_variant_id: String::new(),
                seal_preview_image_storage_path: String::new(),
                seal_style_name: String::new(),
                seal_style_stroke_weight: String::new(),
                seal_style_balance: String::new(),
                seal_style_prompt_summary: String::new(),
                customer_confirmation_kanji_and_design: None,
                customer_confirmation_custom_made_policy: None,
                customer_confirmation_confirmed_at: None,
                customer_confirmation_confirmed_seal_text: String::new(),
                listing_label_ja: "柘植".to_owned(),
                total: 5600,
                created_at: now - chrono::Duration::hours(120),
                updated_at: now - chrono::Duration::hours(80),
                events: vec![
                    OrderEvent {
                        kind: "order_created".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "api".to_owned(),
                        before_status: String::new(),
                        after_status: String::new(),
                        note: "注文を受付".to_owned(),
                        created_at: now - chrono::Duration::hours(120),
                    },
                    OrderEvent {
                        kind: "status_changed".to_owned(),
                        actor_type: "system".to_owned(),
                        actor_id: "scheduler".to_owned(),
                        before_status: "pending_payment".to_owned(),
                        after_status: "canceled".to_owned(),
                        note: String::new(),
                        created_at: now - chrono::Duration::hours(80),
                    },
                ],
            },
        ),
    ]);

    for order in orders.values_mut() {
        apply_derived_statuses(order);
    }

    let fonts = HashMap::from([
        (
            "zen_maru_gothic".to_owned(),
            Font {
                key: "zen_maru_gothic".to_owned(),
                label: "Zen Maru Gothic".to_owned(),
                font_family: "'Zen Maru Gothic', sans-serif".to_owned(),
                font_stylesheet_url:
                    "https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic:wght@400;700&display=swap"
                        .to_owned(),
                kanji_style: "japanese".to_owned(),
                is_active: true,
                sort_order: 10,
                version: 4,
                updated_at: now - chrono::Duration::hours(28),
            },
        ),
        (
            "potta_one".to_owned(),
            Font {
                key: "potta_one".to_owned(),
                label: "Potta One".to_owned(),
                font_family: "'Potta One', cursive".to_owned(),
                font_stylesheet_url:
                    "https://fonts.googleapis.com/css2?family=Potta+One&display=swap".to_owned(),
                kanji_style: "japanese".to_owned(),
                is_active: true,
                sort_order: 20,
                version: 2,
                updated_at: now - chrono::Duration::hours(52),
            },
        ),
        (
            "wdxl_lubrifont_jp_n".to_owned(),
            Font {
                key: "wdxl_lubrifont_jp_n".to_owned(),
                label: "WDXL Lubrifont JP N".to_owned(),
                font_family: "'WDXL Lubrifont JP N', sans-serif".to_owned(),
                font_stylesheet_url:
                    "https://fonts.googleapis.com/css2?family=WDXL+Lubrifont+JP+N&display=swap"
                        .to_owned(),
                kanji_style: "japanese".to_owned(),
                is_active: false,
                sort_order: 30,
                version: 1,
                updated_at: now - chrono::Duration::hours(72),
            },
        ),
    ]);

    let rose_quartz = material_comparison_profile("rose_quartz");
    let lapis_lazuli = material_comparison_profile("lapis_lazuli");
    let jade = material_comparison_profile("jade");
    let mut materials = HashMap::from([
        (
            "rose_quartz".to_owned(),
            Material {
                key: "rose_quartz".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "ローズクオーツ".to_owned()),
                    ("en".to_owned(), "Rose Quartz".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "やわらかな色合いで、親しみやすい印象の石材".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A soft-toned stone with a warm, approachable presence.".to_owned(),
                    ),
                ]),
                comparison_texture_ja: rose_quartz.texture_ja.to_owned(),
                comparison_texture_en: rose_quartz.texture_en.to_owned(),
                comparison_weight_ja: rose_quartz.weight_ja.to_owned(),
                comparison_weight_en: rose_quartz.weight_en.to_owned(),
                comparison_usage_ja: rose_quartz.usage_ja.to_owned(),
                comparison_usage_en: rose_quartz.usage_en.to_owned(),
                shape: "square".to_owned(),
                photos: vec![MaterialPhoto {
                    asset_id: "mat_rose_quartz_01".to_owned(),
                    storage_path: "materials/rose_quartz/mat_rose_quartz_01.webp".to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "ローズクオーツの材質サンプル".to_owned()),
                        ("en".to_owned(), "Rose quartz material sample".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_usd: 16500,
                price_jpy: 28000,
                is_active: true,
                sort_order: 110,
                version: 1,
                updated_at: now - chrono::Duration::hours(36),
            },
        ),
        (
            "lapis_lazuli".to_owned(),
            Material {
                key: "lapis_lazuli".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "ラピスラビリ".to_owned()),
                    ("en".to_owned(), "Lapis Lazuli".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "深い青が印象的な、存在感のある石材".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A deep-blue stone with a strong, distinctive presence.".to_owned(),
                    ),
                ]),
                comparison_texture_ja: lapis_lazuli.texture_ja.to_owned(),
                comparison_texture_en: lapis_lazuli.texture_en.to_owned(),
                comparison_weight_ja: lapis_lazuli.weight_ja.to_owned(),
                comparison_weight_en: lapis_lazuli.weight_en.to_owned(),
                comparison_usage_ja: lapis_lazuli.usage_ja.to_owned(),
                comparison_usage_en: lapis_lazuli.usage_en.to_owned(),
                shape: "round".to_owned(),
                photos: vec![MaterialPhoto {
                    asset_id: "mat_lapis_lazuli_01".to_owned(),
                    storage_path: "materials/lapis_lazuli/mat_lapis_lazuli_01.webp".to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "ラピスラビリの材質サンプル".to_owned()),
                        ("en".to_owned(), "Lapis lazuli material sample".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_usd: 32500,
                price_jpy: 55000,
                is_active: true,
                sort_order: 120,
                version: 1,
                updated_at: now - chrono::Duration::hours(24),
            },
        ),
        (
            "jade".to_owned(),
            Material {
                key: "jade".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "翡翠".to_owned()),
                    ("en".to_owned(), "Jade".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "落ち着いた緑の艶が映える、格調ある石材".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A dignified stone with a calm green sheen.".to_owned(),
                    ),
                ]),
                comparison_texture_ja: jade.texture_ja.to_owned(),
                comparison_texture_en: jade.texture_en.to_owned(),
                comparison_weight_ja: jade.weight_ja.to_owned(),
                comparison_weight_en: jade.weight_en.to_owned(),
                comparison_usage_ja: jade.usage_ja.to_owned(),
                comparison_usage_en: jade.usage_en.to_owned(),
                shape: "square".to_owned(),
                photos: vec![MaterialPhoto {
                    asset_id: "mat_jade_01".to_owned(),
                    storage_path: "materials/jade/mat_jade_01.webp".to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "翡翠の材質サンプル".to_owned()),
                        ("en".to_owned(), "Jade material sample".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_usd: 88500,
                price_jpy: 150000,
                is_active: true,
                sort_order: 130,
                version: 1,
                updated_at: now - chrono::Duration::hours(12),
            },
        ),
    ]);
    for seed in material_master_seeds() {
        materials.insert(
            seed.key.to_owned(),
            Material {
                key: seed.key.to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), seed.label_ja.to_owned()),
                    ("en".to_owned(), seed.label_en.to_owned()),
                ]),
                description_i18n: HashMap::from([
                    ("ja".to_owned(), seed.description_ja.to_owned()),
                    ("en".to_owned(), seed.description_en.to_owned()),
                ]),
                comparison_texture_ja: String::new(),
                comparison_texture_en: String::new(),
                comparison_weight_ja: String::new(),
                comparison_weight_en: String::new(),
                comparison_usage_ja: String::new(),
                comparison_usage_en: String::new(),
                shape: "square".to_owned(),
                photos: Vec::new(),
                price_usd: 0,
                price_jpy: 0,
                is_active: true,
                sort_order: seed.sort_order,
                version: 1,
                updated_at: now,
            },
        );
    }

    let stone_listings = HashMap::from([
        (
            "rose_quartz_01".to_owned(),
            StoneListing {
                key: "rose_quartz_01".to_owned(),
                listing_code: "RQT-0001".to_owned(),
                material_key: "rose_quartz".to_owned(),
                size: "15mm x 15mm x 60mm".to_owned(),
                title_i18n: HashMap::from([
                    ("ja".to_owned(), "ローズクオーツの一点物 01".to_owned()),
                    ("en".to_owned(), "One-of-a-kind Rose Quartz 01".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "やわらかなピンクに細かな模様が入った個体です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A soft pink piece with delicate natural patterning.".to_owned(),
                    ),
                ]),
                story_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "春の光を思わせる、穏やかな表情が魅力です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "Its calm look evokes the softness of spring light.".to_owned(),
                    ),
                ]),
                facets: StoneListingFacets {
                    color_family: "pink".to_owned(),
                    color_tags: vec!["soft_pink".to_owned(), "light_rose".to_owned()],
                    pattern_primary: "cloud".to_owned(),
                    pattern_tags: vec!["cloud".to_owned(), "speckled".to_owned()],
                    stone_shape: "square".to_owned(),
                    translucency: "semi_translucent".to_owned(),
                },
                photos: vec![MaterialPhoto {
                    asset_id: "lst_rose_quartz_01".to_owned(),
                    storage_path: "stone_listings/rose_quartz/rose_quartz_01/main.webp".to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "ローズクオーツの一点物 01".to_owned()),
                        ("en".to_owned(), "One-of-a-kind Rose Quartz 01".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_by_currency: HashMap::from([
                    ("USD".to_owned(), 16800),
                    ("JPY".to_owned(), 28500),
                ]),
                status: "published".to_owned(),
                is_active: true,
                published_at: Some(now - chrono::Duration::hours(40)),
                sort_order: 10,
                version: 1,
                updated_at: now - chrono::Duration::hours(34),
            },
        ),
        (
            "lapis_lazuli_01".to_owned(),
            StoneListing {
                key: "lapis_lazuli_01".to_owned(),
                listing_code: "LPS-0001".to_owned(),
                material_key: "lapis_lazuli".to_owned(),
                size: "18mm x 18mm x 60mm".to_owned(),
                title_i18n: HashMap::from([
                    ("ja".to_owned(), "ラピスラズリの一点物 01".to_owned()),
                    ("en".to_owned(), "One-of-a-kind Lapis Lazuli 01".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "深い青に金色の粒が映える、存在感のある個体です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A strong blue piece with visible golden flecks.".to_owned(),
                    ),
                ]),
                story_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "夜空のような深さを持つ展示向けの石です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A display-worthy stone with the depth of a night sky.".to_owned(),
                    ),
                ]),
                facets: StoneListingFacets {
                    color_family: "blue".to_owned(),
                    color_tags: vec!["deep_blue".to_owned(), "navy".to_owned()],
                    pattern_primary: "speckled".to_owned(),
                    pattern_tags: vec!["speckled".to_owned(), "gold_fleck".to_owned()],
                    stone_shape: "square".to_owned(),
                    translucency: "opaque".to_owned(),
                },
                photos: vec![MaterialPhoto {
                    asset_id: "lst_lapis_lazuli_01".to_owned(),
                    storage_path: "stone_listings/lapis_lazuli/lapis_lazuli_01/main.webp"
                        .to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "ラピスラズリの一点物 01".to_owned()),
                        ("en".to_owned(), "One-of-a-kind Lapis Lazuli 01".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_by_currency: HashMap::from([
                    ("USD".to_owned(), 34800),
                    ("JPY".to_owned(), 58000),
                ]),
                status: "published".to_owned(),
                is_active: true,
                published_at: Some(now - chrono::Duration::hours(30)),
                sort_order: 20,
                version: 1,
                updated_at: now - chrono::Duration::hours(28),
            },
        ),
        (
            "jade_01".to_owned(),
            StoneListing {
                key: "jade_01".to_owned(),
                listing_code: "JDE-0001".to_owned(),
                material_key: "jade".to_owned(),
                size: "18mm x 60mm".to_owned(),
                title_i18n: HashMap::from([
                    ("ja".to_owned(), "翡翠の一点物 01".to_owned()),
                    ("en".to_owned(), "One-of-a-kind Jade 01".to_owned()),
                ]),
                description_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "落ち着いた緑の流れが入った、端正な個体です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A refined piece with calm green flowing patterns.".to_owned(),
                    ),
                ]),
                story_i18n: HashMap::from([
                    (
                        "ja".to_owned(),
                        "格調ある色味が魅力の定番人気の石です。".to_owned(),
                    ),
                    (
                        "en".to_owned(),
                        "A classic favorite with a dignified color tone.".to_owned(),
                    ),
                ]),
                facets: StoneListingFacets {
                    color_family: "green".to_owned(),
                    color_tags: vec!["deep_green".to_owned(), "mottled".to_owned()],
                    pattern_primary: "banded".to_owned(),
                    pattern_tags: vec!["banded".to_owned(), "cloud".to_owned()],
                    stone_shape: "round".to_owned(),
                    translucency: "semi_translucent".to_owned(),
                },
                photos: vec![MaterialPhoto {
                    asset_id: "lst_jade_01".to_owned(),
                    storage_path: "stone_listings/jade/jade_01/main.webp".to_owned(),
                    alt_i18n: HashMap::from([
                        ("ja".to_owned(), "翡翠の一点物 01".to_owned()),
                        ("en".to_owned(), "One-of-a-kind Jade 01".to_owned()),
                    ]),
                    sort_order: 0,
                    is_primary: true,
                    width: 1200,
                    height: 1200,
                }],
                price_by_currency: HashMap::from([
                    ("USD".to_owned(), 92800),
                    ("JPY".to_owned(), 155000),
                ]),
                status: "reserved".to_owned(),
                is_active: true,
                published_at: None,
                sort_order: 30,
                version: 1,
                updated_at: now - chrono::Duration::hours(14),
            },
        ),
    ]);

    let facet_tags = HashMap::from([
        (
            "color:deep_green".to_owned(),
            FacetTag {
                key: "deep_green".to_owned(),
                facet_type: "color".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "濃緑".to_owned()),
                    ("en".to_owned(), "Deep Green".to_owned()),
                ]),
                aliases: vec!["dark_green".to_owned(), "forest_green".to_owned()],
                is_active: true,
                sort_order: 10,
                version: 1,
                updated_at: now - chrono::Duration::hours(12),
            },
        ),
        (
            "color:soft_pink".to_owned(),
            FacetTag {
                key: "soft_pink".to_owned(),
                facet_type: "color".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "淡桃".to_owned()),
                    ("en".to_owned(), "Soft Pink".to_owned()),
                ]),
                aliases: vec!["light_pink".to_owned()],
                is_active: true,
                sort_order: 20,
                version: 1,
                updated_at: now - chrono::Duration::hours(14),
            },
        ),
        (
            "color:deep_blue".to_owned(),
            FacetTag {
                key: "deep_blue".to_owned(),
                facet_type: "color".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "深青".to_owned()),
                    ("en".to_owned(), "Deep Blue".to_owned()),
                ]),
                aliases: vec!["navy_blue".to_owned()],
                is_active: true,
                sort_order: 30,
                version: 1,
                updated_at: now - chrono::Duration::hours(16),
            },
        ),
        (
            "color:mottled".to_owned(),
            FacetTag {
                key: "mottled".to_owned(),
                facet_type: "color".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "斑".to_owned()),
                    ("en".to_owned(), "Mottled".to_owned()),
                ]),
                aliases: vec!["marbled".to_owned()],
                is_active: true,
                sort_order: 40,
                version: 1,
                updated_at: now - chrono::Duration::hours(18),
            },
        ),
        (
            "pattern:banded".to_owned(),
            FacetTag {
                key: "banded".to_owned(),
                facet_type: "pattern".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "縞".to_owned()),
                    ("en".to_owned(), "Banded".to_owned()),
                ]),
                aliases: vec!["striped".to_owned()],
                is_active: true,
                sort_order: 10,
                version: 1,
                updated_at: now - chrono::Duration::hours(10),
            },
        ),
        (
            "pattern:cloud".to_owned(),
            FacetTag {
                key: "cloud".to_owned(),
                facet_type: "pattern".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "雲状".to_owned()),
                    ("en".to_owned(), "Cloud".to_owned()),
                ]),
                aliases: vec!["cloudy".to_owned()],
                is_active: true,
                sort_order: 20,
                version: 1,
                updated_at: now - chrono::Duration::hours(11),
            },
        ),
        (
            "pattern:speckled".to_owned(),
            FacetTag {
                key: "speckled".to_owned(),
                facet_type: "pattern".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "点状".to_owned()),
                    ("en".to_owned(), "Speckled".to_owned()),
                ]),
                aliases: vec!["spotted".to_owned()],
                is_active: true,
                sort_order: 30,
                version: 1,
                updated_at: now - chrono::Duration::hours(13),
            },
        ),
        (
            "pattern:gold_fleck".to_owned(),
            FacetTag {
                key: "gold_fleck".to_owned(),
                facet_type: "pattern".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "金斑".to_owned()),
                    ("en".to_owned(), "Gold Fleck".to_owned()),
                ]),
                aliases: vec!["gold_speck".to_owned()],
                is_active: true,
                sort_order: 40,
                version: 1,
                updated_at: now - chrono::Duration::hours(15),
            },
        ),
    ]);

    let countries = HashMap::from([
        (
            "JP".to_owned(),
            Country {
                code: "JP".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "日本".to_owned()),
                    ("en".to_owned(), "Japan".to_owned()),
                ]),
                shipping_fee_usd: 600,
                shipping_fee_jpy: 600,
                is_active: true,
                sort_order: 10,
                version: 3,
                updated_at: now - chrono::Duration::hours(24),
            },
        ),
        (
            "US".to_owned(),
            Country {
                code: "US".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "アメリカ合衆国".to_owned()),
                    ("en".to_owned(), "United States".to_owned()),
                ]),
                shipping_fee_usd: 1800,
                shipping_fee_jpy: 1800,
                is_active: true,
                sort_order: 20,
                version: 4,
                updated_at: now - chrono::Duration::hours(18),
            },
        ),
        (
            "CA".to_owned(),
            Country {
                code: "CA".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "カナダ".to_owned()),
                    ("en".to_owned(), "Canada".to_owned()),
                ]),
                shipping_fee_usd: 1900,
                shipping_fee_jpy: 1900,
                is_active: true,
                sort_order: 30,
                version: 2,
                updated_at: now - chrono::Duration::hours(32),
            },
        ),
        (
            "GB".to_owned(),
            Country {
                code: "GB".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "イギリス".to_owned()),
                    ("en".to_owned(), "United Kingdom".to_owned()),
                ]),
                shipping_fee_usd: 2000,
                shipping_fee_jpy: 2000,
                is_active: true,
                sort_order: 40,
                version: 2,
                updated_at: now - chrono::Duration::hours(30),
            },
        ),
        (
            "AU".to_owned(),
            Country {
                code: "AU".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "オーストラリア".to_owned()),
                    ("en".to_owned(), "Australia".to_owned()),
                ]),
                shipping_fee_usd: 2100,
                shipping_fee_jpy: 2100,
                is_active: true,
                sort_order: 50,
                version: 1,
                updated_at: now - chrono::Duration::hours(54),
            },
        ),
        (
            "SG".to_owned(),
            Country {
                code: "SG".to_owned(),
                label_i18n: HashMap::from([
                    ("ja".to_owned(), "シンガポール".to_owned()),
                    ("en".to_owned(), "Singapore".to_owned()),
                ]),
                shipping_fee_usd: 1300,
                shipping_fee_jpy: 1300,
                is_active: true,
                sort_order: 60,
                version: 3,
                updated_at: now - chrono::Duration::hours(20),
            },
        ),
    ]);

    let mut snapshot = AdminSnapshot {
        orders,
        order_ids: Vec::new(),
        fonts,
        font_ids: Vec::new(),
        materials,
        material_ids: Vec::new(),
        stone_listings,
        stone_listing_ids: Vec::new(),
        facet_tags,
        facet_tag_ids: Vec::new(),
        countries,
        country_ids: Vec::new(),
    };
    snapshot.refresh_order_ids();
    snapshot.refresh_font_ids();
    snapshot.refresh_material_ids();
    snapshot.refresh_stone_listing_ids();
    snapshot.refresh_facet_tag_ids();
    snapshot.refresh_country_ids();
    snapshot
}

#[cfg(test)]
mod tests {
    use super::*;
    use axum::body::to_bytes;
    use axum::extract::{Form, State};
    use std::sync::Arc;

    fn mock_server_state() -> ServerState {
        ServerState {
            source_label: "Mock".to_owned(),
            storage_assets_bucket: "hanko-field-dev".to_owned(),
            source: DataSource::Mock,
            data: RwLock::new(new_mock_snapshot()),
        }
    }

    fn mock_auth_state() -> Arc<AuthState> {
        Arc::new(AuthState {
            enabled: false,
            login_passphrase: String::new(),
        })
    }

    fn mock_app_state() -> AppState {
        AppState {
            server: Arc::new(mock_server_state()),
            auth: mock_auth_state(),
        }
    }

    fn valid_stone_listing_create_input() -> StoneListingCreateInput {
        StoneListingCreateInput {
            stone_listing_key: "jade_variant_01".to_owned(),
            listing_code: "JDE-0101".to_owned(),
            material_key: "jade".to_owned(),
            size: "18mm x 60mm".to_owned(),
            title_ja: "翡翠の一点物 101".to_owned(),
            title_en: "One-of-a-kind Jade 101".to_owned(),
            description_ja: "落ち着いた緑の流れが入った個体です。".to_owned(),
            description_en: "A refined piece with calm green flowing patterns.".to_owned(),
            story_ja: "格調ある色味が魅力の石です。".to_owned(),
            story_en: "A stone with a dignified color tone.".to_owned(),
            color_family: "green".to_owned(),
            color_tags: vec!["deep_green".to_owned(), "mottled".to_owned()],
            pattern_primary: "banded".to_owned(),
            pattern_tags: vec!["banded".to_owned(), "cloud".to_owned()],
            stone_shape: "round".to_owned(),
            translucency: "semi_translucent".to_owned(),
            price_usd: 92800,
            price_jpy: 155000,
            sort_order: 41,
            photo_storage_path: "stone_listings/jade/jade_variant_01/main.webp".to_owned(),
            photo_alt_ja: "翡翠の一点物 101".to_owned(),
            photo_alt_en: "One-of-a-kind Jade 101".to_owned(),
            status: "published".to_owned(),
            is_active: true,
        }
    }

    fn valid_stone_listing_patch_input() -> StoneListingPatchInput {
        StoneListingPatchInput {
            listing_code: "JDE-0101".to_owned(),
            material_key: "jade".to_owned(),
            size: "18mm x 60mm".to_owned(),
            title_ja: "翡翠の一点物 101".to_owned(),
            title_en: "One-of-a-kind Jade 101".to_owned(),
            description_ja: "落ち着いた緑の流れが入った個体です。".to_owned(),
            description_en: "A refined piece with calm green flowing patterns.".to_owned(),
            story_ja: "格調ある色味が魅力の石です。".to_owned(),
            story_en: "A stone with a dignified color tone.".to_owned(),
            color_family: "green".to_owned(),
            color_tags: vec!["deep_green".to_owned(), "mottled".to_owned()],
            pattern_primary: "banded".to_owned(),
            pattern_tags: vec!["banded".to_owned(), "cloud".to_owned()],
            stone_shape: "round".to_owned(),
            translucency: "semi_translucent".to_owned(),
            price_usd: 92800,
            price_jpy: 155000,
            sort_order: 41,
            photo_storage_path: "stone_listings/jade/jade_01/main.webp".to_owned(),
            photo_alt_ja: "翡翠の一点物 01".to_owned(),
            photo_alt_en: "One-of-a-kind Jade 01".to_owned(),
            title_i18n_updates: HashMap::new(),
            description_i18n_updates: HashMap::new(),
            story_i18n_updates: HashMap::new(),
            photo_alt_i18n_updates: HashMap::new(),
            status: "published".to_owned(),
            is_active: true,
        }
    }

    #[tokio::test]
    async fn filter_orders_by_status() {
        let state = mock_server_state();
        let orders = state
            .filter_orders(&OrderFilter {
                status: "paid".to_owned(),
                country: String::new(),
                email: String::new(),
            })
            .await;

        assert!(!orders.is_empty());
    }

    #[tokio::test]
    async fn filter_orders_exposes_app_metadata_for_list() {
        let state = mock_server_state();
        let orders = state
            .filter_orders(&OrderFilter {
                status: "paid".to_owned(),
                country: String::new(),
                email: String::new(),
            })
            .await;

        let app_order = orders
            .iter()
            .find(|order| order.id == "ord_1006")
            .expect("paid app order should be listed");

        assert!(app_order.is_app_order);
        assert_eq!(app_order.channel_label, "App");
        assert_eq!(app_order.engraving_text, "JANE");
        assert!(app_order.has_engraving_text);
        assert_eq!(app_order.ai_variant_id, "seal_variant_006");
        assert!(app_order.has_ai_variant_id);
        assert_eq!(app_order.payment_status_label, "支払い済み");
        assert_eq!(app_order.fulfillment_status_label, "未着手");
    }

    #[tokio::test]
    async fn render_orders_list_includes_engraving_and_ai_variant() {
        let state = mock_server_state();
        let orders = state
            .filter_orders(&OrderFilter {
                status: "paid".to_owned(),
                country: String::new(),
                email: String::new(),
            })
            .await;

        let html = render_orders_list(&orders).expect("orders list should render");

        assert!(html.contains("彫刻 / AI"));
        assert!(html.contains("JANE"));
        assert!(html.contains("AI Variant: seal_variant_006"));
        assert!(html.contains(">App</span>"));
    }

    #[tokio::test]
    async fn get_order_detail_exposes_ai_seal_preview_image() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        assert!(detail.has_seal_preview_image);
        assert!(detail.has_seal_preview_image_url);
        assert_eq!(
            detail.seal_preview_image_storage_path,
            "seal_designs/seal_request_006/seal_variant_006.png"
        );
        assert_eq!(
            detail.seal_preview_image_url,
            "https://storage.googleapis.com/hanko-field-dev/seal_designs/seal_request_006/seal_variant_006.png"
        );
    }

    #[tokio::test]
    async fn get_order_detail_exposes_ai_seal_metadata() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        assert!(detail.has_ai_seal_metadata);
        assert!(detail.has_ai_generation_id);
        assert!(detail.has_ai_variant_id);
        assert_eq!(detail.ai_generation_id, "seal_request_006");
        assert_eq!(detail.ai_variant_id, "seal_variant_006");
        assert_eq!(detail.seal_style_name, "elegant");
        assert_eq!(detail.seal_style_stroke_weight, "standard");
        assert_eq!(detail.seal_style_balance, "balanced");
        assert_eq!(
            detail.seal_style_prompt_summary,
            "Latin initials with refined spacing"
        );
        assert!(detail.has_seal_style_name);
        assert!(detail.has_seal_style_stroke_weight);
        assert!(detail.has_seal_style_balance);
        assert!(detail.has_seal_style_prompt_summary);
    }

    #[tokio::test]
    async fn render_order_detail_includes_generated_seal_preview_image() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        let html = render_order_detail(&detail).expect("order detail should render");

        assert!(html.contains("生成印影プレビュー"));
        assert!(html.contains(
            "AIは構造化レシピ提案のみを担い、最終PNGは実在フォントを使ったプログラム描画です。"
        ));
        assert!(html.contains("seal_designs/seal_request_006/seal_variant_006.png"));
        assert!(html.contains("https://storage.googleapis.com/hanko-field-dev/seal_designs/seal_request_006/seal_variant_006.png"));
        assert!(html.contains("HF-20260209-1006 の生成印影プレビュー"));
    }

    #[tokio::test]
    async fn render_order_detail_includes_generated_seal_recipe_metadata() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        let html = render_order_detail(&detail).expect("order detail should render");

        assert!(html.contains("生成印影レシピ/スタイル"));
        assert!(html.contains(
            "制作確認ではStorageパス、AI Variant ID、選択スタイル、顧客確認を正本として扱います。"
        ));
        assert!(html.contains("AIレシピ生成ID"));
        assert!(html.contains("seal_request_006"));
        assert!(html.contains("AI Variant ID"));
        assert!(html.contains("seal_variant_006"));
        assert!(html.contains("選択スタイル"));
        assert!(html.contains("elegant"));
        assert!(html.contains("線の太さ"));
        assert!(html.contains("standard"));
        assert!(html.contains("余白バランス"));
        assert!(html.contains("balanced"));
        assert!(html.contains("レシピ要約"));
        assert!(html.contains("Latin initials with refined spacing"));
        assert!(!html.contains("AI画像生成"));
        assert!(!html.contains("AI印影プレビュー"));
        assert!(!html.contains("AI印影メタデータ"));
    }

    #[tokio::test]
    async fn get_order_detail_exposes_customer_confirmation() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        assert!(detail.has_customer_confirmation);
        assert!(detail.has_customer_confirmation_kanji_and_design);
        assert!(detail.has_customer_confirmation_custom_made_policy);
        assert!(detail.has_customer_confirmation_confirmed_at);
        assert!(detail.has_customer_confirmation_confirmed_seal_text);
        assert_eq!(
            detail.customer_confirmation_kanji_and_design_label,
            "確認済み"
        );
        assert_eq!(
            detail.customer_confirmation_custom_made_policy_label,
            "確認済み"
        );
        assert!(!detail.customer_confirmation_confirmed_at.is_empty());
        assert_eq!(detail.customer_confirmation_confirmed_seal_text, "JANE");
    }

    #[tokio::test]
    async fn render_order_detail_includes_customer_confirmation() {
        let state = mock_server_state();
        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");

        let html = render_order_detail(&detail).expect("order detail should render");

        assert!(html.contains("顧客確認チェック"));
        assert!(html.contains("漢字/生成印影確認"));
        assert!(html.contains("カスタムメイド規約"));
        assert!(html.contains("確認時刻"));
        assert!(html.contains("確認時彫刻文字"));
        assert!(html.contains("確認済み"));
        assert!(html.contains("JANE"));
    }

    #[test]
    fn resolve_order_engraving_text_prefers_customer_confirmation() {
        let customer_confirmation =
            btree_from_pairs(vec![("confirmed_seal_text", fs_string(" confirmed text "))]);

        let engraving_text = resolve_order_engraving_text(&customer_confirmation, "AA", "BB");

        assert_eq!(engraving_text, "confirmed text");
    }

    #[test]
    fn read_seal_preview_image_storage_path_reads_nested_storage_path() {
        let seal = btree_from_pairs(vec![(
            "preview_image",
            fs_map(btree_from_pairs(vec![(
                "storage_path",
                fs_string(" /seal_designs/seal_request_001/seal_variant_001.png "),
            )])),
        )]);

        let storage_path = read_seal_preview_image_storage_path(&seal);

        assert_eq!(
            storage_path,
            "seal_designs/seal_request_001/seal_variant_001.png"
        );
    }

    #[test]
    fn decode_order_fields_defaults_missing_app_fields_for_legacy_web_order() {
        let now = Utc::now();
        let data = btree_from_pairs(vec![
            ("order_no", fs_string("HF-LEGACY-001")),
            ("channel", fs_string("web")),
            ("status", fs_string("paid")),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
            (
                "seal",
                fs_map(btree_from_pairs(vec![
                    ("line1", fs_string("佐")),
                    ("line2", fs_string("藤")),
                ])),
            ),
            (
                "pricing",
                fs_map(btree_from_pairs(vec![("total", fs_int(4200))])),
            ),
        ]);

        let order = decode_order_fields("ja", "legacy_web_001", &data);

        assert_eq!(order.order_no, "HF-LEGACY-001");
        assert_eq!(order.channel, "web");
        assert_eq!(order.locale, "ja");
        assert_eq!(order.currency, "JPY");
        assert_eq!(order.status, "paid");
        assert_eq!(order.payment_status, "paid");
        assert_eq!(order.fulfillment_status, "pending");
        assert_eq!(order.engraving_text, "佐藤");
        assert!(order.ai_generation_id.is_empty());
        assert!(order.ai_variant_id.is_empty());
        assert!(order.seal_preview_image_storage_path.is_empty());
        assert!(order.seal_style_name.is_empty());
        assert!(order.customer_confirmation_kanji_and_design.is_none());
        assert!(order.customer_confirmation_custom_made_policy.is_none());
        assert!(order.customer_confirmation_confirmed_at.is_none());
        assert!(order.customer_confirmation_confirmed_seal_text.is_empty());
    }

    #[tokio::test]
    async fn render_order_detail_handles_legacy_web_order_missing_app_fields() {
        let now = Utc::now();
        let data = btree_from_pairs(vec![
            ("order_no", fs_string("HF-LEGACY-002")),
            ("channel", fs_string("web")),
            ("status", fs_string("paid")),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
            (
                "contact",
                fs_map(btree_from_pairs(vec![(
                    "email",
                    fs_string("legacy@example.com"),
                )])),
            ),
            (
                "shipping",
                fs_map(btree_from_pairs(vec![("country_code", fs_string("JP"))])),
            ),
            (
                "seal",
                fs_map(btree_from_pairs(vec![
                    ("line1", fs_string("佐")),
                    ("line2", fs_string("藤")),
                ])),
            ),
            (
                "pricing",
                fs_map(btree_from_pairs(vec![("total", fs_int(4200))])),
            ),
        ]);
        let mut snapshot = new_mock_snapshot();
        snapshot.orders.insert(
            "legacy_web_002".to_owned(),
            decode_order_fields("ja", "legacy_web_002", &data),
        );
        snapshot.refresh_order_ids();
        let state = ServerState {
            source_label: "Mock".to_owned(),
            storage_assets_bucket: "hanko-field-dev".to_owned(),
            source: DataSource::Mock,
            data: RwLock::new(snapshot),
        };

        let detail = state
            .get_order_detail("legacy_web_002", "", "")
            .await
            .expect("legacy web order should exist");
        let html = render_order_detail(&detail).expect("legacy web order detail should render");

        assert_eq!(detail.order_no, "HF-LEGACY-002");
        assert_eq!(detail.status_label, "支払い済み");
        assert_eq!(detail.payment_status_label, "支払い済み");
        assert_eq!(detail.fulfillment_status_label, "未着手");
        assert!(!detail.has_seal_preview_image);
        assert!(!detail.has_ai_seal_metadata);
        assert!(!detail.has_customer_confirmation);
        assert!(html.contains("HF-LEGACY-002"));
        assert!(html.contains("佐 / 藤"));
        assert!(!html.contains("生成印影プレビュー"));
        assert!(!html.contains("生成印影レシピ/スタイル"));
        assert!(!html.contains("顧客確認チェック"));
    }

    #[tokio::test]
    async fn filter_stone_listings_by_facets() {
        let state = mock_server_state();
        let stone_listings = state
            .filter_stone_listings(&StoneListingFilter {
                color_family: "green".to_owned(),
                color_tags: String::new(),
                pattern_primary: "banded".to_owned(),
                pattern_tags: String::new(),
                stone_shape: "round".to_owned(),
            })
            .await;

        assert_eq!(stone_listings.len(), 1);
        assert_eq!(stone_listings[0].key, "jade_01");
    }

    #[tokio::test]
    async fn filter_stone_listings_by_tag_aliases() {
        let state = mock_server_state();
        let stone_listings = state
            .filter_stone_listings(&StoneListingFilter {
                color_family: String::new(),
                color_tags: "dark_green, marbled".to_owned(),
                pattern_primary: String::new(),
                pattern_tags: "striped, cloudy".to_owned(),
                stone_shape: String::new(),
            })
            .await;

        assert_eq!(stone_listings.len(), 1);
        assert_eq!(stone_listings[0].key, "jade_01");
    }

    #[tokio::test]
    async fn create_stone_listing_allows_missing_photo() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.stone_listing_key = "jade_variant_without_photo".to_owned();
        input.listing_code = "JDE-0102".to_owned();
        input.status = "draft".to_owned();
        input.photo_storage_path = String::new();

        let result = state.create_stone_listing(input).await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_variant_without_photo")
            .expect("listing should exist after creation");
        assert!(listing.photos.is_empty());
        assert_eq!(listing.status, "draft");
        assert!(listing.published_at.is_none());
    }

    #[tokio::test]
    async fn create_stone_listing_stores_size() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.stone_listing_key = "jade_variant_sized".to_owned();
        input.listing_code = "JDE-0103".to_owned();
        input.size = "21mm x 21mm x 60mm".to_owned();

        let result = state.create_stone_listing(input).await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_variant_sized")
            .expect("listing should exist after creation");
        assert_eq!(listing.size, "21mm x 21mm x 60mm");
    }

    #[tokio::test]
    async fn create_stone_listing_rejects_size_over_limit() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.size = "x".repeat(81);

        let result = state.create_stone_listing(input).await;

        assert!(matches!(result, Err(message) if message.contains("サイズ")));
    }

    #[tokio::test]
    async fn create_stone_listing_honors_status_and_publication_time() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.stone_listing_key = "jade_variant_02".to_owned();
        input.status = "published".to_owned();

        let result = state.create_stone_listing(input).await;

        assert!(result.is_ok());

        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_variant_02")
            .expect("listing should exist after creation");
        assert_eq!(listing.status, "published");
        assert!(listing.published_at.is_some());
    }

    #[tokio::test]
    async fn update_stone_listing_updates_size() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_patch_input();
        input.size = "15mm x 60mm".to_owned();

        let result = state.update_stone_listing("jade_01", input).await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_01")
            .expect("listing should exist after update");
        assert_eq!(listing.size, "15mm x 60mm");
    }

    #[tokio::test]
    async fn create_stone_listing_rejects_unknown_facet_tags() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.color_tags = vec!["missing_green_tag".to_owned()];

        let result = state.create_stone_listing(input).await;

        assert!(
            matches!(result, Err(message) if message.contains("色タグ") && message.contains("missing_green_tag"))
        );
    }

    #[tokio::test]
    async fn update_stone_listing_rejects_unknown_facet_tags() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_patch_input();
        input.pattern_tags = vec!["missing_pattern_tag".to_owned()];

        let result = state.update_stone_listing("jade_01", input).await;

        assert!(
            matches!(result, Err(message) if message.contains("模様タグ") && message.contains("missing_pattern_tag"))
        );
    }

    #[tokio::test]
    async fn stone_listing_filter_options_include_unknown_stone_shape() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let mut listing = data
                .stone_listings
                .get("jade_01")
                .cloned()
                .expect("jade_01 listing should exist");
            listing.key = "jade_freeform_01".to_owned();
            listing.facets.stone_shape = "freeform".to_owned();
            data.stone_listings.insert(listing.key.clone(), listing);
            data.refresh_stone_listing_ids();
        }

        let options = state.stone_listing_filter_options().await;

        assert!(
            options
                .stone_shape_options
                .iter()
                .any(|option| option.value == "freeform" && option.label == "freeform")
        );
    }

    #[tokio::test]
    async fn create_stone_listing_rejects_missing_material() {
        let state = mock_server_state();
        let mut input = valid_stone_listing_create_input();
        input.material_key = "missing_material".to_owned();

        let result = state.create_stone_listing(input).await;

        assert!(matches!(result, Err(message) if message.contains("有効な材質")));
    }

    #[tokio::test]
    async fn update_stone_listing_allows_existing_inactive_material() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            data.materials.remove("jade");
            data.refresh_material_ids();
        }

        let mut input = valid_stone_listing_patch_input();
        input.title_ja = "翡翠の一点物 01 更新".to_owned();

        let result = state.update_stone_listing("jade_01", input).await;

        assert!(result.is_ok());

        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_01")
            .expect("listing should still exist after update");
        assert_eq!(
            listing.title_i18n.get("ja").map(String::as_str),
            Some("翡翠の一点物 01 更新")
        );
    }

    #[test]
    fn stone_shape_helpers_preserve_unknown_values() {
        assert_eq!(normalize_stone_shape("freeform"), "freeform");
        assert_eq!(
            normalize_stone_shape_optional("freeform"),
            Some("freeform".to_owned())
        );
        assert_eq!(stone_shape_label("freeform"), "freeform");
    }

    #[tokio::test]
    async fn update_stone_listing_rejects_switching_to_inactive_material() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let material = data
                .materials
                .get_mut("lapis_lazuli")
                .expect("lapis_lazuli material should exist");
            material.is_active = false;
        }

        let mut input = valid_stone_listing_patch_input();
        input.material_key = "lapis_lazuli".to_owned();

        let result = state.update_stone_listing("jade_01", input).await;

        assert!(matches!(result, Err(message) if message.contains("有効な材質")));
    }

    #[tokio::test]
    async fn delete_stone_listing_rejects_order_reference() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            if let Some(order) = data.orders.get_mut("ord_1003") {
                order.listing_key = "jade_01".to_owned();
            }
        }

        let result = state.delete_stone_listing("jade_01").await;

        assert!(
            matches!(result, Err(message) if message.contains("注文") && message.contains("参照"))
        );
    }

    #[tokio::test]
    async fn delete_facet_tag_rejects_referenced_listing() {
        let state = mock_server_state();

        let result = state.delete_facet_tag("color:deep_green").await;

        assert!(
            matches!(result, Err(message) if message.contains("一点物") && message.contains("参照"))
        );
    }

    #[tokio::test]
    async fn create_facet_tag_rejects_alias_collision() {
        let state = mock_server_state();
        let result = state
            .create_facet_tag(FacetTagCreateInput {
                key: "emerald".to_owned(),
                facet_type: "color".to_owned(),
                label_ja: "翠".to_owned(),
                label_en: "Emerald".to_owned(),
                aliases: vec!["dark_green".to_owned()],
                sort_order: 90,
                is_active: true,
            })
            .await;

        assert!(
            matches!(result, Err(message) if message.contains("別名") && message.contains("dark_green") && message.contains("color:deep_green"))
        );
    }

    #[tokio::test]
    async fn facet_tag_field_errors_are_classified_by_field() {
        let (key_error, aliases_error) =
            facet_tag_field_errors_from_message("タグキーは 64 文字以内で入力してください。");
        assert_eq!(key_error, "タグキーは 64 文字以内で入力してください。");
        assert!(aliases_error.is_empty());

        let (key_error, aliases_error) = facet_tag_field_errors_from_message(
            "色タグの別名 `dark_green` は既に color:deep_green (濃緑) と重複しています。",
        );
        assert!(key_error.is_empty());
        assert_eq!(
            aliases_error,
            "色タグの別名 `dark_green` は既に color:deep_green (濃緑) と重複しています。"
        );
    }

    #[tokio::test]
    async fn update_facet_tag_rejects_alias_collision() {
        let state = mock_server_state();
        let result = state
            .update_facet_tag(
                "color:soft_pink",
                FacetTagPatchInput {
                    label_ja: "淡桃".to_owned(),
                    label_en: "Soft Pink".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    aliases: vec!["forest_green".to_owned()],
                    sort_order: 20,
                    is_active: true,
                },
            )
            .await;

        assert!(
            matches!(result, Err(message) if message.contains("別名") && message.contains("forest_green") && message.contains("color:deep_green"))
        );
    }

    #[tokio::test]
    async fn stone_listing_tag_options_use_facet_tags_master() {
        let state = mock_server_state();
        let options = state.stone_listing_tag_options().await;

        assert!(
            options
                .color_tag_options
                .iter()
                .any(|option| option.value == "deep_green" && option.label == "濃緑")
        );
        assert!(
            options
                .pattern_tag_options
                .iter()
                .any(|option| option.value == "cloud" && option.label == "雲状")
        );
    }

    #[tokio::test]
    async fn update_order_status_valid_transition() {
        let state = mock_server_state();

        let result = state
            .update_order_status("ord_1006", "manufacturing", "test.admin")
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("order should exist");
        assert_eq!(detail.status_label, "製造中");
        assert_eq!(detail.payment_status_label, "支払い済み");
        assert_eq!(detail.fulfillment_status_label, "製造中");
    }

    #[tokio::test]
    async fn app_order_status_transition_preserves_app_metadata() {
        let state = mock_server_state();

        let result = state
            .update_order_status("ord_1006", "manufacturing", "test.admin")
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");
        assert_eq!(detail.status_label, "製造中");
        assert_eq!(detail.ai_generation_id, "seal_request_006");
        assert_eq!(detail.ai_variant_id, "seal_variant_006");
        assert!(detail.has_ai_seal_metadata);
        assert!(detail.has_seal_preview_image);
        assert!(detail.has_customer_confirmation);

        let data = state.data.read().await;
        let order = data
            .orders
            .get("ord_1006")
            .expect("app order should remain in mock state");
        assert!(order.events.iter().any(|event| {
            event.kind == "status_changed"
                && event.actor_id == "test.admin"
                && event.before_status == "paid"
                && event.after_status == "manufacturing"
        }));
    }

    #[tokio::test]
    async fn app_order_shipping_update_uses_existing_operation() {
        let state = mock_server_state();

        state
            .update_order_status("ord_1006", "manufacturing", "test.admin")
            .await
            .expect("app order should enter manufacturing before shipping");

        let result = state
            .update_shipping("ord_1006", "DHL", "APP-TRACK-001", "shipped", "test.admin")
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_order_detail("ord_1006", "", "")
            .await
            .expect("app order should exist");
        assert_eq!(detail.status_label, "出荷済み");
        assert_eq!(detail.payment_status_label, "支払い済み");
        assert_eq!(detail.fulfillment_status_label, "出荷済み");
        assert_eq!(detail.carrier, "DHL");
        assert_eq!(detail.tracking_no, "APP-TRACK-001");
        assert!(detail.has_tracking_no);
        assert_eq!(detail.ai_variant_id, "seal_variant_006");
        assert!(detail.has_ai_seal_metadata);
        assert!(detail.has_customer_confirmation);

        let data = state.data.read().await;
        let order = data
            .orders
            .get("ord_1006")
            .expect("app order should remain in mock state");
        assert!(order.events.iter().any(|event| {
            event.kind == "shipment_registered"
                && event.actor_id == "test.admin"
                && event.note == "DHL / APP-TRACK-001"
        }));
        assert!(order.events.iter().any(|event| {
            event.kind == "status_changed"
                && event.actor_id == "test.admin"
                && event.before_status == "manufacturing"
                && event.after_status == "shipped"
        }));
    }

    #[test]
    fn stone_listing_status_tracks_order_resolution() {
        assert_eq!(
            stone_listing_status_after_order_status("paid"),
            Some("sold")
        );
        assert_eq!(
            stone_listing_status_after_order_status("canceled"),
            Some("published")
        );
        assert_eq!(stone_listing_status_after_order_status("refunded"), None);
    }

    #[test]
    fn canceled_order_keeps_archived_listings_terminal() {
        assert!(stone_listing_should_restore_after_canceled_order(
            "reserved", false,
        ));
        assert!(stone_listing_should_restore_after_canceled_order(
            " published ",
            true,
        ));
        assert!(!stone_listing_should_restore_after_canceled_order(
            "published",
            false,
        ));
        assert!(!stone_listing_should_restore_after_canceled_order(
            "archived", true,
        ));
    }

    #[test]
    fn m5_t02_admin_localized_merge_preserves_unknown_locale_keys() {
        let mut values = HashMap::from([
            ("ja".to_owned(), "旧日本語".to_owned()),
            ("en".to_owned(), "Old English".to_owned()),
            ("fr".to_owned(), "Français".to_owned()),
            ("zh".to_owned(), "简体中文".to_owned()),
            ("zhtw".to_owned(), "繁體中文".to_owned()),
        ]);

        merge_admin_localized_map_values(
            &mut values,
            &[("ja", " 新日本語 "), ("en", " New English ")],
        );

        assert_eq!(values.get("ja").map(String::as_str), Some("新日本語"));
        assert_eq!(values.get("en").map(String::as_str), Some("New English"));
        assert_eq!(values.get("fr").map(String::as_str), Some("Français"));
        assert_eq!(values.get("zh").map(String::as_str), Some("简体中文"));
        assert_eq!(values.get("zhtw").map(String::as_str), Some("繁體中文"));
    }

    #[test]
    fn m5_t02_optional_localized_merge_only_removes_edited_blank_keys() {
        let mut values = HashMap::from([
            ("ja".to_owned(), "旧日本語".to_owned()),
            ("en".to_owned(), "Old English".to_owned()),
            ("fr".to_owned(), "Français".to_owned()),
            ("zh".to_owned(), "简体中文".to_owned()),
            ("zhtw".to_owned(), "繁體中文".to_owned()),
        ]);

        merge_optional_admin_localized_map_values(
            &mut values,
            &[("ja", " "), ("en", " New English ")],
        );

        assert!(!values.contains_key("ja"));
        assert_eq!(values.get("en").map(String::as_str), Some("New English"));
        assert_eq!(values.get("fr").map(String::as_str), Some("Français"));
        assert_eq!(values.get("zh").map(String::as_str), Some("简体中文"));
        assert_eq!(values.get("zhtw").map(String::as_str), Some("繁體中文"));
    }

    #[test]
    fn m5_t02_localized_update_masks_target_editable_subkeys_only() {
        let mut paths = vec!["version".to_owned()];

        append_admin_localized_update_mask_paths(&mut paths, "label_i18n");

        assert_eq!(
            paths,
            vec![
                "version".to_owned(),
                "label_i18n.ja".to_owned(),
                "label_i18n.en".to_owned(),
            ]
        );
        assert!(!paths.iter().any(|path| path == "label_i18n"));
    }

    #[test]
    fn m5_t02_primary_photo_alt_merge_preserves_unknown_locales_and_extra_photos() {
        let existing = vec![
            MaterialPhoto {
                asset_id: "primary".to_owned(),
                storage_path: "stone_listings/jade_01/primary.webp".to_owned(),
                alt_i18n: HashMap::from([
                    ("ja".to_owned(), "旧日本語".to_owned()),
                    ("en".to_owned(), "Old English".to_owned()),
                    ("fr".to_owned(), "Français".to_owned()),
                    ("zh".to_owned(), "简体中文".to_owned()),
                    ("zhtw".to_owned(), "繁體中文".to_owned()),
                ]),
                sort_order: 0,
                is_primary: true,
                width: 1200,
                height: 800,
            },
            MaterialPhoto {
                asset_id: "secondary".to_owned(),
                storage_path: "stone_listings/jade_01/secondary.webp".to_owned(),
                alt_i18n: HashMap::from([("fr".to_owned(), "Secondaire".to_owned())]),
                sort_order: 1,
                is_primary: false,
                width: 900,
                height: 600,
            },
        ];

        let photos = merge_primary_stone_listing_photo(
            &existing,
            "jade_01",
            "stone_listings/jade_01/primary-updated.webp",
            " ",
            " New English ",
        );

        assert_eq!(photos.len(), 2);
        let primary = photos
            .iter()
            .find(|photo| photo.asset_id == "primary")
            .expect("primary photo should remain");
        assert_eq!(
            primary.storage_path,
            "stone_listings/jade_01/primary-updated.webp"
        );
        assert!(!primary.alt_i18n.contains_key("ja"));
        assert_eq!(
            primary.alt_i18n.get("en").map(String::as_str),
            Some("New English")
        );
        assert_eq!(
            primary.alt_i18n.get("fr").map(String::as_str),
            Some("Français")
        );
        assert_eq!(
            primary.alt_i18n.get("zh").map(String::as_str),
            Some("简体中文")
        );
        assert_eq!(
            primary.alt_i18n.get("zhtw").map(String::as_str),
            Some("繁體中文")
        );

        let secondary = photos
            .iter()
            .find(|photo| photo.asset_id == "secondary")
            .expect("secondary photo should remain");
        assert!(!secondary.is_primary);
        assert_eq!(
            secondary.alt_i18n.get("fr").map(String::as_str),
            Some("Secondaire")
        );
    }

    fn insert_preservation_locales(values: &mut HashMap<String, String>, prefix: &str) {
        values.insert("fr".to_owned(), format!("{prefix} français"));
        values.insert("zh".to_owned(), format!("{prefix} 简体中文"));
        values.insert("zhtw".to_owned(), format!("{prefix} 繁體中文"));
    }

    fn assert_preservation_locales(values: &HashMap<String, String>, prefix: &str) {
        assert_eq!(
            values.get("fr").map(String::as_str),
            Some(format!("{prefix} français").as_str())
        );
        assert_eq!(
            values.get("zh").map(String::as_str),
            Some(format!("{prefix} 简体中文").as_str())
        );
        assert_eq!(
            values.get("zhtw").map(String::as_str),
            Some(format!("{prefix} 繁體中文").as_str())
        );
    }

    #[tokio::test]
    async fn m5_t03_update_material_preserves_fr_zh_and_zhtw_values() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let material = data
                .materials
                .get_mut("jade")
                .expect("jade material should exist");
            insert_preservation_locales(&mut material.label_i18n, "material label");
            insert_preservation_locales(&mut material.description_i18n, "material description");
        }

        let result = state
            .update_material(
                "jade",
                MaterialPatchInput {
                    label_ja: "翡翠 更新".to_owned(),
                    label_en: "Jade Updated".to_owned(),
                    description_ja: "更新された説明です。".to_owned(),
                    description_en: "Updated description.".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    description_i18n_updates: HashMap::new(),
                    is_active: true,
                },
            )
            .await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let material = data
            .materials
            .get("jade")
            .expect("jade material should remain");
        assert_eq!(
            material.label_i18n.get("ja").map(String::as_str),
            Some("翡翠 更新")
        );
        assert_eq!(
            material.label_i18n.get("en").map(String::as_str),
            Some("Jade Updated")
        );
        assert_preservation_locales(&material.label_i18n, "material label");
        assert_preservation_locales(&material.description_i18n, "material description");
    }

    #[tokio::test]
    async fn m5_t03_update_stone_listing_preserves_fr_zh_and_zhtw_values() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let listing = data
                .stone_listings
                .get_mut("jade_01")
                .expect("jade_01 listing should exist");
            insert_preservation_locales(&mut listing.title_i18n, "listing title");
            insert_preservation_locales(&mut listing.description_i18n, "listing description");
            insert_preservation_locales(&mut listing.story_i18n, "listing story");
            let primary = listing
                .photos
                .iter_mut()
                .find(|photo| photo.is_primary)
                .expect("primary photo should exist");
            insert_preservation_locales(&mut primary.alt_i18n, "listing photo alt");
        }

        let mut input = valid_stone_listing_patch_input();
        input.title_ja = "翡翠の一点物 01 更新".to_owned();
        input.title_en = "Updated One-of-a-kind Jade 01".to_owned();
        input.description_ja = "更新された一点物説明です。".to_owned();
        input.description_en = "Updated listing description.".to_owned();
        input.story_ja = "更新されたストーリーです。".to_owned();
        input.story_en = "Updated story.".to_owned();
        input.photo_alt_ja = "更新された翡翠写真".to_owned();
        input.photo_alt_en = "Updated jade photo".to_owned();

        let result = state.update_stone_listing("jade_01", input).await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let listing = data
            .stone_listings
            .get("jade_01")
            .expect("jade_01 listing should remain");
        assert_eq!(
            listing.title_i18n.get("ja").map(String::as_str),
            Some("翡翠の一点物 01 更新")
        );
        assert_eq!(
            listing.title_i18n.get("en").map(String::as_str),
            Some("Updated One-of-a-kind Jade 01")
        );
        assert_preservation_locales(&listing.title_i18n, "listing title");
        assert_preservation_locales(&listing.description_i18n, "listing description");
        assert_preservation_locales(&listing.story_i18n, "listing story");
        let primary = listing
            .photos
            .iter()
            .find(|photo| photo.is_primary)
            .expect("primary photo should remain");
        assert_eq!(
            primary.alt_i18n.get("ja").map(String::as_str),
            Some("更新された翡翠写真")
        );
        assert_eq!(
            primary.alt_i18n.get("en").map(String::as_str),
            Some("Updated jade photo")
        );
        assert_preservation_locales(&primary.alt_i18n, "listing photo alt");
    }

    #[tokio::test]
    async fn m5_t03_update_country_preserves_fr_zh_and_zhtw_values() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let country = data.countries.get_mut("JP").expect("JP should exist");
            insert_preservation_locales(&mut country.label_i18n, "country label");
        }

        let result = state
            .update_country(
                "JP",
                CountryPatchInput {
                    label_ja: "日本国内".to_owned(),
                    label_en: "Japan Domestic".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    shipping_fee_usd: 900,
                    shipping_fee_jpy: 1200,
                    sort_order: 15,
                    is_active: true,
                },
            )
            .await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let country = data.countries.get("JP").expect("JP should remain");
        assert_eq!(
            country.label_i18n.get("ja").map(String::as_str),
            Some("日本国内")
        );
        assert_eq!(
            country.label_i18n.get("en").map(String::as_str),
            Some("Japan Domestic")
        );
        assert_preservation_locales(&country.label_i18n, "country label");
    }

    #[tokio::test]
    async fn m5_t03_update_facet_tag_preserves_fr_zh_and_zhtw_values() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let tag = data
                .facet_tags
                .get_mut("color:soft_pink")
                .expect("soft pink tag should exist");
            insert_preservation_locales(&mut tag.label_i18n, "facet label");
        }

        let result = state
            .update_facet_tag(
                "color:soft_pink",
                FacetTagPatchInput {
                    label_ja: "淡桃 更新".to_owned(),
                    label_en: "Soft Pink Updated".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    aliases: vec!["light_pink".to_owned()],
                    sort_order: 25,
                    is_active: true,
                },
            )
            .await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let tag = data
            .facet_tags
            .get("color:soft_pink")
            .expect("soft pink tag should remain");
        assert_eq!(
            tag.label_i18n.get("ja").map(String::as_str),
            Some("淡桃 更新")
        );
        assert_eq!(
            tag.label_i18n.get("en").map(String::as_str),
            Some("Soft Pink Updated")
        );
        assert_preservation_locales(&tag.label_i18n, "facet label");
    }

    #[test]
    fn m5_t04_registry_editor_languages_exclude_primary_admin_locales() {
        let languages = admin_localized_editor_languages();

        assert_eq!(languages.len(), 66);
        assert!(!languages.iter().any(|language| language.route_code == "ja"));
        assert!(!languages.iter().any(|language| language.route_code == "en"));
        assert!(languages.iter().any(|language| language.route_code == "fr"));
        assert!(languages.iter().any(|language| language.route_code == "zh"));
        assert!(
            languages
                .iter()
                .any(|language| language.route_code == "zhtw")
        );
        assert!(
            languages
                .iter()
                .any(|language| language.route_code == "ar" && language.text_direction == "rtl")
        );
    }

    #[test]
    fn m5_t04_parse_localized_form_updates_accepts_registry_routes_only() {
        let form = HashMap::from([
            ("label_i18n__fr".to_owned(), " Français ".to_owned()),
            ("label_i18n__zh".to_owned(), " 简体中文 ".to_owned()),
            ("label_i18n__ja".to_owned(), "日本語".to_owned()),
            ("label_i18n__xx".to_owned(), "Unknown".to_owned()),
            ("label_i18n__zhtw".to_owned(), " ".to_owned()),
        ]);

        let updates = parse_localized_form_updates(&form, "label_i18n");

        assert_eq!(updates.len(), 2);
        assert_eq!(updates.get("fr").map(String::as_str), Some("Français"));
        assert_eq!(updates.get("zh").map(String::as_str), Some("简体中文"));
        assert!(!updates.contains_key("ja"));
        assert!(!updates.contains_key("xx"));
        assert!(!updates.contains_key("zhtw"));
    }

    #[tokio::test]
    async fn m5_t04_material_detail_renders_collapsed_localized_editor() {
        let state = mock_server_state();

        {
            let mut data = state.data.write().await;
            let material = data
                .materials
                .get_mut("jade")
                .expect("jade material should exist");
            material
                .label_i18n
                .insert("fr".to_owned(), "Jade français".to_owned());
        }

        let detail = state
            .get_material_detail("jade", "", "")
            .await
            .expect("jade material detail should render");
        let html = render_material_detail(&detail).expect("material detail should render");

        assert!(html.contains("<details"));
        assert!(!html.contains("<details open"));
        assert!(html.contains("追加ロケール"));
        assert!(html.contains("name=\"label_i18n__fr\""));
        assert!(html.contains("value=\"Jade français\""));
        assert!(html.contains("name=\"description_i18n__zhtw\""));
    }

    #[tokio::test]
    async fn m5_t04_update_material_saves_extra_localized_editor_values() {
        let state = mock_server_state();

        let result = state
            .update_material(
                "jade",
                MaterialPatchInput {
                    label_ja: "翡翠".to_owned(),
                    label_en: "Jade".to_owned(),
                    description_ja: "説明です。".to_owned(),
                    description_en: "Description.".to_owned(),
                    label_i18n_updates: HashMap::from([
                        ("fr".to_owned(), "Jade français".to_owned()),
                        ("zh".to_owned(), "翡翠简体".to_owned()),
                    ]),
                    description_i18n_updates: HashMap::from([(
                        "zhtw".to_owned(),
                        "繁體説明".to_owned(),
                    )]),
                    is_active: true,
                },
            )
            .await;

        assert!(result.is_ok());
        let data = state.data.read().await;
        let material = data
            .materials
            .get("jade")
            .expect("jade material should remain");
        assert_eq!(
            material.label_i18n.get("fr").map(String::as_str),
            Some("Jade français")
        );
        assert_eq!(
            material.label_i18n.get("zh").map(String::as_str),
            Some("翡翠简体")
        );
        assert_eq!(
            material.description_i18n.get("zhtw").map(String::as_str),
            Some("繁體説明")
        );
    }

    #[tokio::test]
    async fn create_material_accepts_master_fields_only() {
        let form = HashMap::from([
            ("key".to_owned(), "rose_quartz_variant".to_owned()),
            ("label_ja".to_owned(), "ローズクオーツ".to_owned()),
            ("label_en".to_owned(), "Rose Quartz".to_owned()),
            (
                "description_ja".to_owned(),
                "やわらかな色合いで、親しみやすい印象の石材".to_owned(),
            ),
            (
                "description_en".to_owned(),
                "A soft-toned stone with a warm, approachable presence".to_owned(),
            ),
            ("is_active".to_owned(), "1".to_owned()),
        ]);

        let state = mock_app_state();
        let response = handle_material_create(State(state.clone()), Ok(Form(form.clone()))).await;
        assert_eq!(response.status(), StatusCode::CREATED);

        let body = to_bytes(response.into_body(), usize::MAX)
            .await
            .expect("response body should be readable");
        let html = String::from_utf8(body.to_vec()).expect("response body should be utf-8");
        assert!(html.contains("材質「rose_quartz_variant」を作成しました。"));

        let detail = state
            .server
            .get_material_detail("rose_quartz_variant", "", "")
            .await;
        assert!(
            detail.is_some(),
            "material should be inserted into mock state"
        );
    }

    #[tokio::test]
    async fn mock_material_master_includes_requested_materials() {
        let state = mock_server_state();
        let materials = state.list_materials().await;
        let labels_by_key = materials
            .iter()
            .map(|material| (material.key.as_str(), material.label_ja.as_str()))
            .collect::<HashMap<_, _>>();

        for (key, label_ja) in [
            ("wood", "木材"),
            ("qingtian_stone", "青田石"),
            ("shoushan_stone", "寿山石"),
            ("balin_stone", "巴林石"),
            ("yili_stone", "伊犁石"),
            ("laos_stone", "ラオス石"),
            ("xixia_stone", "西峡石"),
            ("frozen_stone", "凍石"),
        ] {
            assert_eq!(labels_by_key.get(key).copied(), Some(label_ja));
        }
    }

    #[tokio::test]
    async fn update_shipping_invalid_transition_does_not_mutate() {
        let state = mock_server_state();

        let before = state
            .get_order_detail("ord_1007", "", "")
            .await
            .expect("order should exist");

        let result = state
            .update_shipping("ord_1007", "DHL", "DHL-999", "delivered", "test.admin")
            .await;
        assert!(result.is_err());

        let after = state
            .get_order_detail("ord_1007", "", "")
            .await
            .expect("order should exist");

        assert_eq!(before.carrier, after.carrier);
        assert_eq!(before.tracking_no, after.tracking_no);
    }

    #[tokio::test]
    async fn update_country_updates_shipping_fee() {
        let state = mock_server_state();

        let result = state
            .update_country(
                "JP",
                CountryPatchInput {
                    label_ja: "日本国内".to_owned(),
                    label_en: "Japan Domestic".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    shipping_fee_usd: 900,
                    shipping_fee_jpy: 1200,
                    sort_order: 15,
                    is_active: true,
                },
            )
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_country_detail("JP", "", "")
            .await
            .expect("country should exist");
        assert_eq!(detail.label_ja, "日本国内");
        assert_eq!(detail.shipping_fee_usd, 900);
        assert_eq!(detail.shipping_fee_jpy, 1200);
        assert_eq!(detail.sort_order, 15);
    }

    #[tokio::test]
    async fn update_country_rejects_negative_shipping_fee() {
        let state = mock_server_state();

        let result = state
            .update_country(
                "JP",
                CountryPatchInput {
                    label_ja: "日本".to_owned(),
                    label_en: "Japan".to_owned(),
                    label_i18n_updates: HashMap::new(),
                    shipping_fee_usd: -1,
                    shipping_fee_jpy: 800,
                    sort_order: 10,
                    is_active: true,
                },
            )
            .await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn update_font_updates_family_and_sort_order() {
        let state = mock_server_state();

        let result = state
            .update_font(
                "zen_maru_gothic",
                FontPatchInput {
                    label: "Zen 丸".to_owned(),
                    font_family: "'Zen Maru Gothic', 'Noto Sans JP', sans-serif".to_owned(),
                    kanji_style: "taiwanese".to_owned(),
                    sort_order: 15,
                    is_active: true,
                },
            )
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_font_detail("zen_maru_gothic", "", "")
            .await
            .expect("font should exist");
        assert_eq!(detail.label, "Zen 丸");
        assert_eq!(
            detail.font_family,
            "'Zen Maru Gothic', 'Noto Sans JP', sans-serif"
        );
        assert_eq!(
            detail.font_stylesheet_url,
            "https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic&display=swap"
        );
        assert_eq!(detail.kanji_style, "taiwanese");
        assert_eq!(detail.sort_order, 15);
    }

    #[tokio::test]
    async fn create_font_adds_font() {
        let state = mock_server_state();

        let result = state
            .create_font(FontCreateInput {
                key: "kiwi_maru".to_owned(),
                label: "Kiwi Maru".to_owned(),
                font_family: "'Kiwi Maru', sans-serif".to_owned(),
                kanji_style: "japanese".to_owned(),
                sort_order: 40,
                is_active: true,
            })
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_font_detail("kiwi_maru", "", "")
            .await
            .expect("font should exist");
        assert_eq!(detail.key, "kiwi_maru");
        assert_eq!(detail.label, "Kiwi Maru");
        assert_eq!(detail.kanji_style, "japanese");
        assert_eq!(
            detail.font_stylesheet_url,
            "https://fonts.googleapis.com/css2?family=Kiwi+Maru&display=swap"
        );
    }

    #[tokio::test]
    async fn create_font_rejects_duplicate_key() {
        let state = mock_server_state();

        let result = state
            .create_font(FontCreateInput {
                key: "zen_maru_gothic".to_owned(),
                label: "Zen Maru Gothic".to_owned(),
                font_family: "'Zen Maru Gothic', sans-serif".to_owned(),
                kanji_style: "japanese".to_owned(),
                sort_order: 10,
                is_active: true,
            })
            .await;
        assert!(result.is_err());
    }

    #[test]
    fn build_google_fonts_stylesheet_url_uses_primary_font_name() {
        let url =
            build_google_fonts_stylesheet_url("'Zen Maru Gothic', 'Noto Sans JP', sans-serif")
                .expect("url should be generated");
        assert_eq!(
            url,
            "https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic&display=swap"
        );
    }

    #[test]
    fn build_google_fonts_stylesheet_url_rejects_generic_family() {
        let result = build_google_fonts_stylesheet_url("sans-serif, serif");
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn create_country_adds_country() {
        let state = mock_server_state();

        let result = state
            .create_country(CountryCreateInput {
                code: "kr".to_owned(),
                label_ja: "韓国".to_owned(),
                label_en: "Korea".to_owned(),
                shipping_fee_usd: 1700,
                shipping_fee_jpy: 2400,
                sort_order: 70,
                is_active: true,
            })
            .await;
        assert!(result.is_ok());

        let detail = state
            .get_country_detail("KR", "", "")
            .await
            .expect("country should exist");
        assert_eq!(detail.code, "KR");
        assert_eq!(detail.shipping_fee_usd, 1700);
        assert_eq!(detail.shipping_fee_jpy, 2400);
    }

    #[tokio::test]
    async fn create_country_rejects_duplicate_code() {
        let state = mock_server_state();

        let result = state
            .create_country(CountryCreateInput {
                code: "JP".to_owned(),
                label_ja: "日本".to_owned(),
                label_en: "Japan".to_owned(),
                shipping_fee_usd: 600,
                shipping_fee_jpy: 600,
                sort_order: 10,
                is_active: true,
            })
            .await;
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn delete_country_removes_country() {
        let state = mock_server_state();

        let result = state.delete_country("SG").await;
        assert!(result.is_ok());
        assert!(state.get_country_detail("SG", "", "").await.is_none());
    }

    #[test]
    fn format_usd_with_separator() {
        assert_eq!(format_usd(0), "USD 0.00");
        assert_eq!(format_usd(1200), "USD 12.00");
        assert_eq!(format_usd(1234567), "USD 12,345.67");
    }

    #[test]
    fn format_jpy_with_separator() {
        assert_eq!(format_jpy(0), "0円");
        assert_eq!(format_jpy(1200), "1,200円");
        assert_eq!(format_jpy(1234567), "1,234,567円");
    }

    #[test]
    fn resolve_order_currency_prefers_locale_when_mismatched() {
        let mut data = BTreeMap::new();
        data.insert("locale".to_owned(), fs_string("en"));
        data.insert("currency".to_owned(), fs_string("JPY"));

        let pricing = btree_from_pairs(vec![("currency", fs_string("JPY"))]);
        let payment = BTreeMap::new();

        let resolved = resolve_order_currency(&data, &pricing, &payment, "en");
        assert_eq!(resolved, "USD");
    }

    #[test]
    fn resolve_order_total_reads_pricing_total_only() {
        let data = BTreeMap::new();

        let pricing = btree_from_pairs(vec![("total", fs_int(11600))]);
        let resolved = resolve_order_total(&data, &pricing);
        assert_eq!(resolved, 11600);
    }

    #[test]
    fn resolve_order_listing_fields_uses_legacy_material_snapshot() {
        let data = btree_from_pairs(vec![("material_label_ja", fs_string("翡翠"))]);
        let listing = BTreeMap::new();
        let material = btree_from_pairs(vec![("key", fs_string("jade"))]);

        let (listing_key, listing_label_ja) =
            FirestoreAdminSource::resolve_order_listing_fields(&data, &listing, &material);

        assert_eq!(listing_key, "jade");
        assert_eq!(listing_label_ja, "翡翠");
    }

    #[test]
    fn normalize_storage_bucket_name_accepts_gs_uri() {
        assert_eq!(
            normalize_storage_bucket_name("gs://hanko-field.firebasestorage.app"),
            "hanko-field.firebasestorage.app"
        );
        assert_eq!(
            normalize_storage_bucket_name("hanko-field.firebasestorage.app/"),
            "hanko-field.firebasestorage.app"
        );
    }

    #[test]
    fn build_storage_media_url_uses_public_gcs_url() {
        assert_eq!(
            build_storage_media_url(
                "gs://hanko-field-dev/",
                "materials/rose_quartz/mat_rose_quartz_01.webp",
            ),
            "https://storage.googleapis.com/hanko-field-dev/materials/rose_quartz/mat_rose_quartz_01.webp"
        );
    }

    #[test]
    fn validate_storage_bucket_name_rejects_path_segments() {
        let result = validate_storage_bucket_name("hanko-field.firebasestorage.app/path");
        assert!(result.is_err());
    }

    #[tokio::test]
    async fn redirect_to_admin_login_for_htmx_uses_lowercase_header() {
        let response = redirect_to_admin_login(true);

        assert_eq!(response.status(), StatusCode::UNAUTHORIZED);
        assert_eq!(
            response
                .headers()
                .get("hx-redirect")
                .and_then(|value| value.to_str().ok()),
            Some("/admin-login")
        );
    }
}
