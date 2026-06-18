use std::{
    collections::{BTreeMap, HashMap, HashSet},
    fmt,
    io::Cursor,
    net::SocketAddr,
    str::FromStr,
    sync::{Arc, OnceLock},
    time::Duration as StdDuration,
};

use anyhow::{Context, Result, anyhow, bail};
use axum::{
    Router,
    body::Bytes,
    extract::{DefaultBodyLimit, Path, Query, State},
    http::{HeaderMap, StatusCode},
    response::{IntoResponse, Response},
    routing::{get, post},
};
use chrono::{DateTime, Duration, FixedOffset, SecondsFormat, Utc};
use firebase_sdk_rust::firebase_firestore::{
    CommitRequest, CreateDocumentOptions, Document, FirebaseFirestoreClient,
    FirebaseFirestoreError, GetDocumentOptions, PatchDocumentOptions, RunQueryRequest,
};
use gcp_auth::{CustomServiceAccount, TokenProvider, provider};
use hmac::{Hmac, Mac};
use image::{DynamicImage, ImageFormat, Rgba, RgbaImage, imageops::FilterType};
use regex::Regex;
use serde::Deserialize;
use serde_json::{Value as JsonValue, json};
use sha2::{Digest, Sha256};
use tokio::net::TcpListener;
use uuid::Uuid;

mod language_registry;
mod seal_fonts;
mod seal_renderer;

const DATASTORE_SCOPE: &str = "https://www.googleapis.com/auth/datastore";
const MAX_REQUEST_BODY_BYTES: usize = 1 << 20;
const DEFAULT_PORT: &str = "3050";
const DEFAULT_LOCALE: &str = "ja";
const DEFAULT_CURRENCY: &str = "USD";
const DEFAULT_STRIPE_CHECKOUT_SUCCESS_URL: &str =
    "http://127.0.0.1:3052/payment/success?session_id={CHECKOUT_SESSION_ID}";
const DEFAULT_STRIPE_CHECKOUT_CANCEL_URL: &str = "http://127.0.0.1:3052/payment/failure";
const DEFAULT_STRIPE_APP_CHECKOUT_SUCCESS_URL: &str =
    "hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}";
const DEFAULT_STRIPE_APP_CHECKOUT_CANCEL_URL: &str = "hankofield://checkout/cancel";
const CHECKOUT_COPY_EN_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/checkout/en.json"
));
const CHECKOUT_COPY_JA_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/checkout/ja.json"
));
const CHECKOUT_COPY_ZH_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/checkout/zh.json"
));
const CHECKOUT_COPY_ZHTW_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/checkout/zhtw.json"
));
const CHECKOUT_COPY_AR_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/checkout/ar.json"
));
const STRIPE_CHECKOUT_SESSIONS_URL: &str = "https://api.stripe.com/v1/checkout/sessions";
const STRIPE_CHECKOUT_API_VERSION: &str = "2025-07-30.basil";
const STRIPE_WEBHOOK_TOLERANCE_SECONDS: i64 = 5 * 60;
const STORAGE_SCOPE: &str = "https://www.googleapis.com/auth/devstorage.read_write";
const DEFAULT_GEMINI_MODEL: &str = "gemini-2.5-flash-lite";
const DEFAULT_GEMINI_THINKING_BUDGET: i32 = 1024;
const DEFAULT_KANJI_CANDIDATE_COUNT: usize = 6;
const MAX_KANJI_CANDIDATE_COUNT: usize = 10;
const DEFAULT_SEAL_DESIGN_VARIANT_COUNT: usize = 3;
const MAX_SEAL_GENERATION_ATTEMPTS: usize = 3;
const SEAL_DESIGN_IMAGE_SIZE: usize = 1024;
const SEAL_DESIGN_FRAME_INSET: u32 = 96;
const SEAL_DESIGN_FRAME_STROKE_WIDTH: u32 = 34;
const SEAL_DESIGN_RED: [u8; 3] = [0x9D, 0x1F, 0x22];
const SEAL_DESIGN_BACKGROUND: [u8; 3] = [0xFF, 0xFF, 0xFF];
const SEAL_DESIGN_PAINT_STRENGTH_THRESHOLD: f32 = 0.08;
const SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD: f32 = 0.22;

#[derive(Debug, Clone)]
struct AppConfig {
    addr: String,
    firestore_project_id: String,
    storage_assets_bucket: String,
    stripe_api_key: String,
    stripe_webhook_secret: String,
    stripe_checkout_success_url: String,
    stripe_checkout_cancel_url: String,
    stripe_app_checkout_success_url: String,
    stripe_app_checkout_cancel_url: String,
    gemini_api_key: String,
    gemini_model: String,
    gemini_base_url: String,
    credentials_file: Option<String>,
}

#[derive(Clone)]
struct AppState {
    store: Arc<FirestoreStore>,
    storage_assets_bucket: String,
    stripe_webhook_secret: String,
    stripe_api_key: String,
    stripe_checkout: StripeCheckoutConfig,
    gemini: GeminiClientConfig,
    http_client: reqwest::Client,
}

#[derive(Debug, Clone)]
struct StripeCheckoutConfig {
    success_url: String,
    cancel_url: String,
    app_success_url: String,
    app_cancel_url: String,
}

#[derive(Debug, Clone)]
struct GeminiClientConfig {
    api_key: String,
    model: String,
    base_url: String,
}

#[derive(Clone)]
struct FirestoreStore {
    database: String,
    parent: String,
    token_provider: Arc<dyn TokenProvider>,
    jst: FixedOffset,
}

#[derive(Debug, Clone)]
struct PublicConfig {
    supported_locales: Vec<String>,
    default_locale: String,
    default_currency: String,
    currency_by_locale: HashMap<String, String>,
}

#[derive(Debug, Clone)]
struct Font {
    key: String,
    label: String,
    font_family: String,
    kanji_style: String,
    version: i64,
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
struct Material {
    key: String,
    label_i18n: HashMap<String, String>,
    description_i18n: HashMap<String, String>,
    shape: String,
    photos: Vec<MaterialPhoto>,
    price_by_currency: HashMap<String, i64>,
    version: i64,
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
    sort_order: i64,
    version: i64,
}

#[derive(Debug, Clone)]
struct Country {
    code: String,
    label_i18n: HashMap<String, String>,
    shipping_fee_by_currency: HashMap<String, i64>,
    version: i64,
}

#[derive(Debug, Clone)]
struct FacetTag {
    facet_type: String,
    key: String,
    label_i18n: HashMap<String, String>,
    aliases: Vec<String>,
}

#[derive(Debug, Clone, Default)]
struct FacetTagLabels {
    labels_by_type: HashMap<String, HashMap<String, String>>,
}

impl FacetTagLabels {
    fn insert(&mut self, facet_type: &str, key: &str, label: &str, aliases: &[String]) {
        let Some(facet_type) = normalize_facet_tag_type(facet_type) else {
            return;
        };
        let key = normalize_facet_value(key);
        let label = label.trim();
        if key.is_empty() || label.is_empty() {
            return;
        }

        let labels = self
            .labels_by_type
            .entry(facet_type.to_owned())
            .or_default();
        labels.insert(key.clone(), label.to_owned());
        for alias in aliases {
            let alias = normalize_facet_value(alias);
            if !alias.is_empty() {
                labels.insert(alias, label.to_owned());
            }
        }
    }

    fn resolve_or_raw(&self, facet_type: &str, value: &str) -> String {
        let Some(facet_type) = normalize_facet_tag_type(facet_type) else {
            return String::new();
        };
        let normalized = normalize_facet_value(value);
        if normalized.is_empty() {
            return String::new();
        }

        self.labels_by_type
            .get(facet_type)
            .and_then(|labels| labels.get(&normalized))
            .cloned()
            .unwrap_or(normalized)
    }

    fn resolve_list(&self, facet_type: &str, values: &[String]) -> Vec<String> {
        let mut labels = Vec::new();
        let mut seen = HashSet::new();

        for value in values {
            let label = self.resolve_or_raw(facet_type, value);
            if label.is_empty() {
                continue;
            }

            if seen.insert(normalize_facet_value(&label)) {
                labels.push(label);
            }
        }

        labels
    }
}

#[derive(Debug, Clone)]
struct MaterialFilterOption {
    value: String,
    label: String,
}

#[derive(Debug, Clone)]
struct CreateOrderResult {
    order_id: String,
    order_no: String,
    status: String,
    payment_status: String,
    fulfillment_status: String,
    total: i64,
    currency: String,
    idempotent_replay: bool,
}

#[derive(Debug, Clone)]
struct OrderStatusResult {
    order_id: String,
    order_no: String,
    status: String,
    payment_status: String,
    checkout_session_id: String,
    payment_intent_id: String,
    fulfillment_status: String,
    fulfillment_carrier: String,
    fulfillment_tracking_no: String,
    production_status: String,
    shipping_status: String,
    total: i64,
    currency: String,
    updated_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct LookupOrderRequest {
    order_no: String,
    email: String,
}

#[derive(Debug, Clone)]
struct LookupOrderInput {
    order_no: String,
    email: String,
}

#[derive(Debug, Clone)]
struct OrderLookupResult {
    status: OrderStatusResult,
    created_at: Option<DateTime<Utc>>,
    seal_confirmed_text: String,
    seal_preview_image_url: String,
    listing_id: String,
    listing_title: String,
    shipped_at: Option<DateTime<Utc>>,
}

#[derive(Debug, Clone)]
struct ProcessStripeWebhookResult {
    processed: bool,
    already_processed: bool,
}

#[derive(Debug, Clone)]
struct CreateOrderInput {
    channel: String,
    locale: String,
    idempotency_key: String,
    terms_agreed: bool,
    seal: SealInput,
    listing_id: Option<String>,
    shipping: ShippingInput,
    contact: ContactInput,
    customer_confirmation: Option<CustomerConfirmationInput>,
    order_note: Option<String>,
}

#[derive(Debug, Clone)]
struct SealInput {
    line1: String,
    line2: String,
    shape: String,
    font_key: String,
    ai_generation_id: Option<String>,
    ai_variant_id: Option<String>,
    preview_image: Option<SealPreviewImageInput>,
    style: Option<SealStyleInput>,
}

#[derive(Debug, Clone)]
struct SealPreviewImageInput {
    storage_path: String,
    download_url: Option<String>,
    width: Option<i64>,
    height: Option<i64>,
}

#[derive(Debug, Clone)]
struct SealStyleInput {
    name: String,
    stroke_weight: String,
    balance: String,
    prompt_summary: Option<String>,
}

#[derive(Debug, Clone)]
struct CustomerConfirmationInput {
    kanji_and_design: bool,
    custom_made_policy: bool,
    confirmed_at: DateTime<Utc>,
    confirmed_seal_text: String,
}

#[derive(Debug, Clone)]
struct ShippingInput {
    country_code: String,
    recipient_name: String,
    phone: String,
    postal_code: String,
    state: String,
    city: String,
    address_line1: String,
    address_line2: String,
}

#[derive(Debug, Clone)]
struct ContactInput {
    email: String,
    preferred_locale: String,
}

#[derive(Debug, Clone)]
struct OrderCheckoutContext {
    order_id: String,
    order_locale: String,
    status: String,
    payment_status: String,
    listing_key: String,
    listing_label: String,
    seal_shape: String,
    shipping_country_code: String,
    shipping_recipient_name: String,
    shipping_phone: String,
    shipping_postal_code: String,
    shipping_state: String,
    shipping_city: String,
    shipping_address_line1: String,
    shipping_address_line2: String,
    total: i64,
    currency: String,
    contact_email: String,
}

#[derive(Debug, Deserialize)]
struct CheckoutCopy {
    product_name_template: String,
    product_description_template: String,
    shape_labels: HashMap<String, String>,
}

#[derive(Debug, Clone)]
struct StripeWebhookEvent {
    provider_event_id: String,
    event_type: String,
    checkout_session_id: String,
    checkout_session_payment_status: String,
    checkout_session_status: String,
    payment_intent_id: String,
    order_id: String,
}

#[derive(Debug, Clone)]
struct StripeCheckoutSessionSnapshot {
    session_id: String,
    payment_status: String,
    session_status: String,
    payment_intent_id: String,
    order_id: String,
}

#[derive(Debug)]
struct StripeOrderWebhookMutation {
    event_fields: BTreeMap<String, JsonValue>,
    listing_status: Option<&'static str>,
}

type HmacSha256 = Hmac<Sha256>;

#[derive(Debug, thiserror::Error)]
enum StripeWebhookError {
    #[error("stripe signature timestamp is missing")]
    MissingTimestamp,
    #[error("stripe signature timestamp is invalid")]
    InvalidTimestamp,
    #[error("stripe signature timestamp is outside tolerance")]
    TimestampOutsideTolerance,
    #[error("stripe signature is missing")]
    MissingSignature,
    #[error("stripe signature is invalid")]
    InvalidSignature,
    #[error(transparent)]
    Json(#[from] serde_json::Error),
}

#[derive(Debug, Deserialize)]
struct QueryLocale {
    locale: Option<String>,
}

#[derive(Debug, Deserialize, Default)]
struct QueryStoneListings {
    locale: Option<String>,
    material_key: Option<String>,
    color_family: Option<String>,
    pattern_primary: Option<String>,
    stone_shape: Option<String>,
    status: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct GenerateKanjiCandidatesRequest {
    real_name: String,
    reason_language: Option<String>,
    gender: Option<String>,
    kanji_style: Option<String>,
    count: Option<usize>,
}

#[derive(Debug, Clone)]
struct GenerateKanjiCandidatesInput {
    real_name: String,
    reason_language: String,
    reason_language_requested: String,
    reason_language_route_code: String,
    reason_language_fallback_reason: Option<&'static str>,
    gender: CandidateGender,
    kanji_style: KanjiStyle,
    count: usize,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum CandidateGender {
    Unspecified,
    Male,
    Female,
}

impl CandidateGender {
    fn as_str(self) -> &'static str {
        match self {
            Self::Unspecified => "unspecified",
            Self::Male => "male",
            Self::Female => "female",
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Unspecified => {
                "Gender preference: unspecified. Do not force masculine/feminine bias."
            }
            Self::Male => {
                "Gender preference: masculine. Prefer names with masculine impression while staying natural."
            }
            Self::Female => {
                "Gender preference: feminine. Prefer names with feminine impression while staying natural."
            }
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum KanjiStyle {
    Japanese,
    Chinese,
    Taiwanese,
}

impl KanjiStyle {
    fn as_str(self) -> &'static str {
        match self {
            Self::Japanese => "japanese",
            Self::Chinese => "chinese",
            Self::Taiwanese => "taiwanese",
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Japanese => {
                "Style preference: Japanese style. Follow Japanese naming conventions and aesthetics. reading must be lowercase romaji."
            }
            Self::Chinese => {
                "Style preference: Chinese style. Follow modern Chinese naming conventions and aesthetics. reading must be lowercase Hanyu Pinyin without tone marks."
            }
            Self::Taiwanese => {
                "Style preference: Taiwanese style. Prefer Traditional Chinese naming conventions common in Taiwan. reading must be lowercase Hanyu Pinyin without tone marks."
            }
        }
    }
}

#[derive(Debug, Clone)]
struct KanjiNameCandidate {
    kanji: String,
    reading: String,
    meaning: String,
    impression: Vec<String>,
    reason: String,
    character_count: usize,
    stroke_complexity: String,
    engraving_suitability: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct GenerateSealDesignsRequest {
    input_name: String,
    kanji: String,
    shape: String,
    style: String,
    stroke_weight: String,
    balance: String,
    variant_count: Option<usize>,
    attempt_number: Option<usize>,
    previous_recipes: Option<Vec<SealDesignRecipeDto>>,
    generation_rules: Option<SealGenerationRulesRequest>,
}

#[derive(Debug, Deserialize, Default)]
#[serde(deny_unknown_fields)]
struct SealGenerationRulesRequest {
    max_characters: Option<usize>,
    avoid_complex_characters: Option<bool>,
    engraving_friendly: Option<bool>,
    avoid_thin_lines: Option<bool>,
    avoid_decorative_details: Option<bool>,
    plain_background: Option<bool>,
}

#[derive(Debug, Clone)]
struct GenerateSealDesignsInput {
    input_name: String,
    kanji: String,
    shape: SealShape,
    style: SealStyleName,
    stroke_weight: SealStrokeWeight,
    balance: SealBalance,
    variant_count: usize,
    attempt_number: usize,
    previous_recipes: Vec<SealDesignRecipe>,
    generation_rules: SealGenerationRules,
}

#[derive(Debug, Clone)]
struct SealGenerationRules {
    max_characters: usize,
    avoid_complex_characters: bool,
    engraving_friendly: bool,
    avoid_thin_lines: bool,
    avoid_decorative_details: bool,
    plain_background: bool,
}

impl Default for SealGenerationRules {
    fn default() -> Self {
        Self {
            max_characters: 2,
            avoid_complex_characters: true,
            engraving_friendly: true,
            avoid_thin_lines: true,
            avoid_decorative_details: true,
            plain_background: true,
        }
    }
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct SealDesignRecipeVariantsDto {
    variants: Vec<SealDesignRecipeVariantDto>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct SealDesignRecipeVariantDto {
    label: String,
    recipe: SealDesignRecipeDto,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
struct SealDesignRecipeDto {
    font_profile: String,
    impression: String,
    weight: String,
    spacing: String,
    texture: String,
    frame: String,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct SealDesignRecipeVariant {
    label: String,
    recipe: SealDesignRecipe,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
struct SealDesignRecipe {
    font_profile: SealRecipeFontProfile,
    impression: SealRecipeImpression,
    weight: SealRecipeWeight,
    spacing: SealRecipeSpacing,
    texture: SealRecipeTexture,
    frame: SealRecipeFrame,
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeFontProfile {
    FormalSerif,
    SoftSans,
    BoldBrush,
    ClassicSeal,
}

impl SealRecipeFontProfile {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::FormalSerif => "formal_serif",
            Self::SoftSans => "soft_sans",
            Self::BoldBrush => "bold_brush",
            Self::ClassicSeal => "classic_seal",
        }
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeImpression {
    Traditional,
    Elegant,
    Soft,
    Bold,
}

impl SealRecipeImpression {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::Traditional => "traditional",
            Self::Elegant => "elegant",
            Self::Soft => "soft",
            Self::Bold => "bold",
        }
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeWeight {
    Standard,
    Bold,
}

impl SealRecipeWeight {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::Standard => "standard",
            Self::Bold => "bold",
        }
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeSpacing {
    Airy,
    Balanced,
    Dense,
}

impl SealRecipeSpacing {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::Airy => "airy",
            Self::Balanced => "balanced",
            Self::Dense => "dense",
        }
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeTexture {
    None,
    SubtleInk,
    SoftBleed,
}

impl SealRecipeTexture {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::None => "none",
            Self::SubtleInk => "subtle_ink",
            Self::SoftBleed => "soft_bleed",
        }
    }
}

#[allow(dead_code)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealRecipeFrame {
    SquareStandard,
    RoundStandard,
}

impl SealRecipeFrame {
    #[allow(dead_code)]
    fn as_str(self) -> &'static str {
        match self {
            Self::SquareStandard => "square_standard",
            Self::RoundStandard => "round_standard",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealShape {
    Square,
    Round,
}

impl SealShape {
    fn as_str(self) -> &'static str {
        match self {
            Self::Square => "square",
            Self::Round => "round",
        }
    }

    fn recipe_frame(self) -> SealRecipeFrame {
        match self {
            Self::Square => SealRecipeFrame::SquareStandard,
            Self::Round => SealRecipeFrame::RoundStandard,
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Square => "Use a square outer seal frame with balanced inner margins.",
            Self::Round => "Use a round outer seal frame with balanced inner margins.",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealStyleName {
    Traditional,
    Elegant,
    Soft,
    Bold,
}

impl SealStyleName {
    fn as_str(self) -> &'static str {
        match self {
            Self::Traditional => "traditional",
            Self::Elegant => "elegant",
            Self::Soft => "soft",
            Self::Bold => "bold",
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Traditional => "Style: traditional hanko composition with dignified spacing.",
            Self::Elegant => "Style: elegant composition with refined, calm proportions.",
            Self::Soft => "Style: soft composition with gentle curves and friendly balance.",
            Self::Bold => "Style: bold composition with strong strokes and high legibility.",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealStrokeWeight {
    Standard,
    Bold,
}

impl SealStrokeWeight {
    fn as_str(self) -> &'static str {
        match self {
            Self::Standard => "standard",
            Self::Bold => "bold",
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Standard => "Stroke weight: standard, with engraving-safe line thickness.",
            Self::Bold => "Stroke weight: bold, with strong engraving-safe line thickness.",
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealBalance {
    Airy,
    Balanced,
    Dense,
}

impl SealBalance {
    fn as_str(self) -> &'static str {
        match self {
            Self::Airy => "airy",
            Self::Balanced => "balanced",
            Self::Dense => "dense",
        }
    }

    fn prompt_instruction(self) -> &'static str {
        match self {
            Self::Airy => "Balance: airy spacing with readable, uncluttered strokes.",
            Self::Balanced => "Balance: balanced spacing and consistent visual weight.",
            Self::Dense => "Balance: dense composition while preserving readability.",
        }
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct SealDesignVariant {
    id: String,
    storage_path: String,
    download_url: String,
    label: String,
    recipe: SealDesignRecipe,
    width: usize,
    height: usize,
}

#[derive(Debug, Clone, PartialEq, Eq)]
struct GeneratedSealDesignImage {
    content_type: String,
    bytes: Vec<u8>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderRequest {
    channel: String,
    locale: String,
    idempotency_key: String,
    terms_agreed: bool,
    seal: CreateOrderSealRequest,
    listing_id: Option<String>,
    shipping: CreateOrderShippingRequest,
    contact: CreateOrderContactRequest,
    customer_confirmation: Option<CreateOrderCustomerConfirmationRequest>,
    order_note: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderSealRequest {
    line1: String,
    line2: String,
    shape: String,
    font_key: String,
    ai_generation_id: Option<String>,
    ai_variant_id: Option<String>,
    preview_image: Option<CreateOrderSealPreviewImageRequest>,
    style: Option<CreateOrderSealStyleRequest>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderSealPreviewImageRequest {
    storage_path: String,
    download_url: Option<String>,
    width: Option<i64>,
    height: Option<i64>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderSealStyleRequest {
    name: String,
    stroke_weight: String,
    balance: String,
    prompt_summary: Option<String>,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderCustomerConfirmationRequest {
    kanji_and_design: bool,
    custom_made_policy: bool,
    confirmed_at: String,
    confirmed_seal_text: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderShippingRequest {
    country_code: String,
    recipient_name: String,
    phone: String,
    postal_code: String,
    state: String,
    city: String,
    address_line1: String,
    address_line2: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateOrderContactRequest {
    email: String,
    preferred_locale: String,
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CreateStripeCheckoutSessionRequest {
    order_id: String,
    customer_email: Option<String>,
    return_to_app: Option<bool>,
}

#[derive(Debug, Clone)]
struct CreateStripeCheckoutSessionInput {
    order_id: String,
    customer_email: String,
    return_to_app: bool,
}

#[derive(Debug, Clone)]
struct CreateStripeCheckoutSessionResult {
    session_id: String,
    checkout_url: String,
    payment_intent_id: String,
}

#[derive(Debug, thiserror::Error)]
enum StoreError {
    #[error("unsupported locale")]
    UnsupportedLocale,
    #[error("invalid reference")]
    InvalidReference,
    #[error("inactive reference")]
    InactiveReference,
    #[error("material shape mismatch")]
    MaterialShapeMismatch,
    #[error("idempotency key conflict")]
    IdempotencyConflict,
    #[error("internal: {0}")]
    Internal(#[from] anyhow::Error),
}

#[tokio::main]
async fn main() {
    if let Err(err) = run().await {
        eprintln!("failed to start api server: {err:#}");
        std::process::exit(1);
    }
}

async fn run() -> Result<()> {
    let cfg = load_config().context("failed to load config")?;

    let token_provider: Arc<dyn TokenProvider> = if let Some(credentials_file) =
        cfg.credentials_file.as_deref()
    {
        Arc::new(
            CustomServiceAccount::from_file(credentials_file)
                .with_context(|| format!("failed to read credentials file: {credentials_file}"))?,
        )
    } else {
        provider()
            .await
            .context("failed to initialize default GCP auth provider")?
    };

    let database = firestore_database_path(&cfg.firestore_project_id);
    let store = Arc::new(FirestoreStore {
        parent: firestore_documents_parent(&database),
        database,
        token_provider,
        jst: FixedOffset::east_opt(9 * 60 * 60).expect("valid JST offset"),
    });

    let http_client = reqwest::Client::builder()
        .timeout(StdDuration::from_secs(20))
        .build()
        .context("failed to initialize http client")?;

    let state = AppState {
        store,
        storage_assets_bucket: cfg.storage_assets_bucket,
        stripe_webhook_secret: cfg.stripe_webhook_secret,
        stripe_api_key: cfg.stripe_api_key.trim().to_owned(),
        stripe_checkout: StripeCheckoutConfig {
            success_url: cfg.stripe_checkout_success_url,
            cancel_url: cfg.stripe_checkout_cancel_url,
            app_success_url: cfg.stripe_app_checkout_success_url,
            app_cancel_url: cfg.stripe_app_checkout_cancel_url,
        },
        gemini: GeminiClientConfig {
            api_key: cfg.gemini_api_key,
            model: cfg.gemini_model,
            base_url: cfg.gemini_base_url,
        },
        http_client,
    };

    let app = Router::new()
        .route("/healthz", get(handle_healthz))
        .route("/v1/config/public", get(handle_public_config))
        .route("/v1/catalog", get(handle_catalog))
        .route("/v1/stone-listings", get(handle_stone_listings))
        .route(
            "/v1/stone-listings/{listing_id}",
            get(handle_stone_listing_detail),
        )
        .route(
            "/v1/kanji-candidates",
            post(handle_generate_kanji_candidates),
        )
        .route(
            "/v1/seal-designs/generate",
            post(handle_generate_seal_designs),
        )
        .route("/v1/orders", post(handle_create_order))
        .route("/v1/orders/lookup", post(handle_lookup_order))
        .route("/v1/orders/{order_id}/status", get(handle_get_order_status))
        .route(
            "/v1/payments/stripe/checkout-session",
            post(handle_create_stripe_checkout_session),
        )
        .route("/v1/payments/stripe/webhook", post(handle_stripe_webhook))
        .layer(DefaultBodyLimit::max(MAX_REQUEST_BODY_BYTES))
        .with_state(state);

    let addr = SocketAddr::from_str(&format!("0.0.0.0{}", cfg.addr))
        .with_context(|| format!("invalid listen addr: {}", cfg.addr))?;
    let listener = TcpListener::bind(addr)
        .await
        .with_context(|| format!("failed to bind on {addr}"))?;

    eprintln!("hanko api listening on http://localhost{}", cfg.addr);

    axum::serve(listener, app)
        .with_graceful_shutdown(async {
            let _ = tokio::signal::ctrl_c().await;
        })
        .await
        .context("api server exited unexpectedly")
}

fn firestore_database_path(project_id: &str) -> String {
    format!("projects/{}/databases/(default)", project_id.trim())
}

fn firestore_documents_parent(database: &str) -> String {
    format!("{}/documents", database.trim_end_matches('/'))
}

fn load_config() -> Result<AppConfig> {
    let port = first_non_empty(&[
        std::env::var("API_SERVER_PORT").ok(),
        std::env::var("PORT").ok(),
    ])
    .unwrap_or_else(|| DEFAULT_PORT.to_owned());

    let firestore_project_id = first_non_empty(&[
        std::env::var("API_FIRESTORE_PROJECT_ID").ok(),
        std::env::var("FIRESTORE_PROJECT_ID").ok(),
        std::env::var("API_FIREBASE_PROJECT_ID").ok(),
        std::env::var("FIREBASE_PROJECT_ID").ok(),
        std::env::var("GOOGLE_CLOUD_PROJECT").ok(),
    ])
    .ok_or_else(|| {
        anyhow!(
            "missing Firestore project id: set API_FIRESTORE_PROJECT_ID or FIRESTORE_PROJECT_ID"
        )
    })?;

    let storage_assets_bucket = first_non_empty(&[std::env::var("API_STORAGE_ASSETS_BUCKET").ok()])
        .unwrap_or_else(|| "hanko-field-dev".to_owned());

    let stripe_api_key =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_API_KEY").ok()]).unwrap_or_default();

    let stripe_webhook_secret =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_WEBHOOK_SECRET").ok()]).unwrap_or_default();

    let stripe_checkout_success_url =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_CHECKOUT_SUCCESS_URL").ok()])
            .unwrap_or_else(|| DEFAULT_STRIPE_CHECKOUT_SUCCESS_URL.to_owned());

    let stripe_checkout_cancel_url =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_CHECKOUT_CANCEL_URL").ok()])
            .unwrap_or_else(|| DEFAULT_STRIPE_CHECKOUT_CANCEL_URL.to_owned());

    let stripe_app_checkout_success_url =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_APP_CHECKOUT_SUCCESS_URL").ok()])
            .unwrap_or_else(|| DEFAULT_STRIPE_APP_CHECKOUT_SUCCESS_URL.to_owned());

    let stripe_app_checkout_cancel_url =
        first_non_empty(&[std::env::var("API_PSP_STRIPE_APP_CHECKOUT_CANCEL_URL").ok()])
            .unwrap_or_else(|| DEFAULT_STRIPE_APP_CHECKOUT_CANCEL_URL.to_owned());

    let gemini_api_key = first_non_empty(&[
        std::env::var("API_GEMINI_API_KEY").ok(),
        std::env::var("GEMINI_API_KEY").ok(),
    ])
    .unwrap_or_default();

    let gemini_model = first_non_empty(&[std::env::var("API_GEMINI_MODEL").ok()])
        .unwrap_or_else(|| DEFAULT_GEMINI_MODEL.to_owned());

    let gemini_base_url = first_non_empty(&[std::env::var("API_GEMINI_BASE_URL").ok()])
        .unwrap_or_else(|| "https://generativelanguage.googleapis.com".to_owned());

    let credentials_file = first_non_empty(&[
        std::env::var("API_FIREBASE_CREDENTIALS_FILE").ok(),
        std::env::var("GOOGLE_APPLICATION_CREDENTIALS").ok(),
    ]);

    Ok(AppConfig {
        addr: format!(":{}", port.trim()),
        firestore_project_id,
        storage_assets_bucket,
        stripe_api_key,
        stripe_webhook_secret,
        stripe_checkout_success_url,
        stripe_checkout_cancel_url,
        stripe_app_checkout_success_url,
        stripe_app_checkout_cancel_url,
        gemini_api_key,
        gemini_model,
        gemini_base_url,
        credentials_file,
    })
}

fn first_non_empty(values: &[Option<String>]) -> Option<String> {
    values
        .iter()
        .filter_map(|value| value.as_deref())
        .map(str::trim)
        .find(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

async fn handle_healthz() -> Response {
    json_response(StatusCode::OK, json!({ "ok": true }))
}

async fn handle_public_config(State(state): State<AppState>) -> Response {
    match state.store.get_public_config().await {
        Ok(cfg) => json_response(
            StatusCode::OK,
            json!({
                "supported_locales": cfg.supported_locales,
                "default_locale": cfg.default_locale,
                "default_currency": cfg.default_currency,
                "currency_by_locale": cfg.currency_by_locale,
            }),
        ),
        Err(err) => {
            eprintln!("failed to load public config: {err:#}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
    }
}

async fn handle_catalog(
    State(state): State<AppState>,
    Query(query): Query<QueryLocale>,
) -> Response {
    let cfg = match state.store.get_public_config().await {
        Ok(cfg) => cfg,
        Err(err) => {
            eprintln!("failed to load public config: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let Some(requested_locale) = requested_locale_route_code(query.locale, &cfg.default_locale)
    else {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    };
    let pricing_currency = resolve_pricing_currency(&cfg, &requested_locale);

    if !cfg
        .supported_locales
        .iter()
        .any(|locale| locale == &requested_locale)
    {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    }

    let fonts = match state.store.list_active_fonts().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load fonts: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let materials = match state.store.list_active_materials().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load materials: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let facet_tags = match state.store.list_active_facet_tags().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load facet tags: {err:#}");
            Vec::new()
        }
    };
    let facet_tag_labels =
        build_facet_tag_labels(&facet_tags, &requested_locale, &cfg.default_locale);

    let stone_listings = match state.store.list_active_stone_listings().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load stone listings: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };
    let material_labels =
        material_labels_for_locale(&materials, &requested_locale, &cfg.default_locale);
    let material_filters = build_material_filters(&stone_listings, &facet_tag_labels);
    let stone_listing_resp = stone_listings
        .iter()
        .cloned()
        .map(|listing| {
            stone_listing_response(
                &state.storage_assets_bucket,
                &requested_locale,
                &cfg.default_locale,
                &pricing_currency,
                &facet_tag_labels,
                &material_labels,
                listing,
            )
        })
        .collect::<Vec<_>>();

    let countries = match state.store.list_active_countries().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load countries: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let font_resp = fonts
        .into_iter()
        .map(|font| {
            json!({
                "key": font.key,
                "label": font.label,
                "font_family": font.font_family,
                "kanji_style": font.kanji_style,
                "version": font.version,
            })
        })
        .collect::<Vec<_>>();

    let material_resp = materials
        .into_iter()
        .map(|material| {
            let resolved_price = material_price_for_currency(&material, &pricing_currency);
            let photos = material
                .photos
                .into_iter()
                .map(|photo| {
                    json!({
                        "asset_id": photo.asset_id,
                        "asset_url": make_asset_url(&state.storage_assets_bucket, &photo.storage_path),
                        "storage_path": photo.storage_path,
                        "alt": resolve_localized(&photo.alt_i18n, &requested_locale, &cfg.default_locale),
                        "sort_order": photo.sort_order,
                        "is_primary": photo.is_primary,
                        "width": photo.width,
                        "height": photo.height,
                    })
                })
                .collect::<Vec<_>>();

            json!({
                "key": material.key,
                "label": resolve_localized(&material.label_i18n, &requested_locale, &cfg.default_locale),
                "description": resolve_localized(&material.description_i18n, &requested_locale, &cfg.default_locale),
                "shape": material.shape,
                "price": resolved_price,
                "price_by_currency": material.price_by_currency,
                "version": material.version,
                "photos": photos,
            })
        })
        .collect::<Vec<_>>();

    let country_resp = countries
        .into_iter()
        .map(|country| {
            json!({
                "code": country.code,
                "label": resolve_localized(&country.label_i18n, &requested_locale, &cfg.default_locale),
                "shipping_fee": country_shipping_fee_for_currency(&country, &pricing_currency),
                "shipping_fee_by_currency": country.shipping_fee_by_currency,
                "version": country.version,
            })
        })
        .collect::<Vec<_>>();

    json_response(
        StatusCode::OK,
        json!({
            "locale": requested_locale,
            "supported_locales": cfg.supported_locales,
            "default_locale": cfg.default_locale,
            "default_currency": cfg.default_currency,
            "currency_by_locale": cfg.currency_by_locale,
            "currency": pricing_currency,
            "fonts": font_resp,
            "materials": material_resp,
            "material_filters": material_filters_response(&material_filters),
            "stone_listings": stone_listing_resp,
            "countries": country_resp,
        }),
    )
}

async fn handle_stone_listings(
    State(state): State<AppState>,
    Query(query): Query<QueryStoneListings>,
) -> Response {
    let cfg = match state.store.get_public_config().await {
        Ok(cfg) => cfg,
        Err(err) => {
            eprintln!("failed to load public config: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let Some(requested_locale) = requested_locale_route_code(query.locale, &cfg.default_locale)
    else {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    };
    let pricing_currency = resolve_pricing_currency(&cfg, &requested_locale);

    if !cfg
        .supported_locales
        .iter()
        .any(|locale| locale == &requested_locale)
    {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    }

    let stone_listings = match state.store.list_active_stone_listings().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load stone listings: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };
    let facet_tags = match state.store.list_active_facet_tags().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load facet tags: {err:#}");
            Vec::new()
        }
    };
    let facet_tag_labels =
        build_facet_tag_labels(&facet_tags, &requested_locale, &cfg.default_locale);
    let materials = match state.store.list_active_materials().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load material labels: {err:#}");
            Vec::new()
        }
    };
    let material_labels =
        material_labels_for_locale(&materials, &requested_locale, &cfg.default_locale);

    let requested_material_key = query.material_key.unwrap_or_default().trim().to_owned();
    let requested_color_family = query.color_family.unwrap_or_default().trim().to_lowercase();
    let requested_pattern_primary = query
        .pattern_primary
        .unwrap_or_default()
        .trim()
        .to_lowercase();
    let requested_stone_shape = query.stone_shape.unwrap_or_default().trim().to_lowercase();
    let requested_status = query.status.unwrap_or_default().trim().to_lowercase();

    let stone_listings = stone_listings
        .into_iter()
        .filter(|listing| {
            if !requested_material_key.is_empty() && listing.material_key != requested_material_key
            {
                return false;
            }
            if !requested_color_family.is_empty()
                && listing.facets.color_family.to_lowercase() != requested_color_family
            {
                return false;
            }
            if !requested_pattern_primary.is_empty()
                && listing.facets.pattern_primary.to_lowercase() != requested_pattern_primary
            {
                return false;
            }
            if !requested_stone_shape.is_empty()
                && listing.facets.stone_shape.to_lowercase() != requested_stone_shape
            {
                return false;
            }
            if !requested_status.is_empty() && listing.status.to_lowercase() != requested_status {
                return false;
            }
            true
        })
        .map(|listing| {
            stone_listing_response(
                &state.storage_assets_bucket,
                &requested_locale,
                &cfg.default_locale,
                &pricing_currency,
                &facet_tag_labels,
                &material_labels,
                listing,
            )
        })
        .collect::<Vec<_>>();

    json_response(
        StatusCode::OK,
        json!({
            "locale": requested_locale,
            "currency": pricing_currency,
            "stone_listings": stone_listings,
        }),
    )
}

async fn handle_stone_listing_detail(
    State(state): State<AppState>,
    Path(listing_id): Path<String>,
    Query(query): Query<QueryLocale>,
) -> Response {
    let cfg = match state.store.get_public_config().await {
        Ok(cfg) => cfg,
        Err(err) => {
            eprintln!("failed to load public config: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let Some(requested_locale) = requested_locale_route_code(query.locale, &cfg.default_locale)
    else {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    };
    let pricing_currency = resolve_pricing_currency(&cfg, &requested_locale);

    if !cfg
        .supported_locales
        .iter()
        .any(|locale| locale == &requested_locale)
    {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_locale",
            "unsupported locale",
        );
    }

    let listing = match state.store.get_app_visible_stone_listing(&listing_id).await {
        Ok(Some(listing)) => listing,
        Ok(None) => {
            return error_response(
                StatusCode::NOT_FOUND,
                "stone_listing_not_found",
                "stone listing not found",
            );
        }
        Err(err) => {
            eprintln!("failed to load stone listing {listing_id}: {err:#}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    };

    let facet_tags = match state.store.list_active_facet_tags().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load facet tags: {err:#}");
            Vec::new()
        }
    };
    let facet_tag_labels =
        build_facet_tag_labels(&facet_tags, &requested_locale, &cfg.default_locale);
    let materials = match state.store.list_active_materials().await {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load material labels: {err:#}");
            Vec::new()
        }
    };
    let material_labels =
        material_labels_for_locale(&materials, &requested_locale, &cfg.default_locale);

    json_response(
        StatusCode::OK,
        stone_listing_response(
            &state.storage_assets_bucket,
            &requested_locale,
            &cfg.default_locale,
            &pricing_currency,
            &facet_tag_labels,
            &material_labels,
            listing,
        ),
    )
}

async fn handle_generate_kanji_candidates(State(state): State<AppState>, body: Bytes) -> Response {
    if state.gemini.api_key.trim().is_empty() {
        return error_response(
            StatusCode::SERVICE_UNAVAILABLE,
            "gemini_not_configured",
            "gemini api key is not configured",
        );
    }

    let request = match serde_json::from_slice::<GenerateKanjiCandidatesRequest>(&body) {
        Ok(value) => value,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                &format!("invalid JSON: {err}"),
            );
        }
    };

    let input = match validate_generate_kanji_candidates_request(request) {
        Ok(value) => value,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "validation_error",
                &err.to_string(),
            );
        }
    };

    let candidates = match generate_kanji_candidates_with_gemini(&state, &input).await {
        Ok(value) => value,
        Err(err) => {
            eprintln!("failed to generate kanji candidates with gemini: {err:#}");
            return error_response(
                StatusCode::BAD_GATEWAY,
                "gemini_generation_failed",
                "failed to generate kanji candidates",
            );
        }
    };

    if candidates.is_empty() {
        return error_response(
            StatusCode::BAD_GATEWAY,
            "gemini_generation_failed",
            "gemini returned no valid candidates",
        );
    }

    json_response(
        StatusCode::OK,
        json!({
            "real_name": input.real_name,
            "reason_language": input.reason_language,
            "reason_language_requested": input.reason_language_requested,
            "reason_language_route_code": input.reason_language_route_code,
            "reason_language_fallback": input.reason_language_fallback_reason.map(|reason| {
                json!({
                    "used": true,
                    "reason": reason,
                    "fallback_to": input.reason_language,
                })
            }),
            "gender": input.gender.as_str(),
            "kanji_style": input.kanji_style.as_str(),
            "candidates": candidates.into_iter().map(|candidate| {
                json!({
                    "kanji": candidate.kanji,
                    "reading": candidate.reading,
                    "meaning": candidate.meaning,
                    "impression": candidate.impression,
                    "reason": candidate.reason,
                    "character_count": candidate.character_count,
                    "stroke_complexity": candidate.stroke_complexity,
                    "engraving_suitability": candidate.engraving_suitability,
                })
            }).collect::<Vec<_>>(),
        }),
    )
}

async fn handle_generate_seal_designs(State(state): State<AppState>, body: Bytes) -> Response {
    if state.gemini.api_key.trim().is_empty() {
        return error_response(
            StatusCode::SERVICE_UNAVAILABLE,
            "gemini_not_configured",
            "gemini api key is not configured",
        );
    }

    let request = match serde_json::from_slice::<GenerateSealDesignsRequest>(&body) {
        Ok(value) => value,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                &format!("invalid JSON: {err}"),
            );
        }
    };

    let input = match validate_generate_seal_designs_request(request) {
        Ok(value) => value,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "validation_error",
                &err.to_string(),
            );
        }
    };

    let recipe_variants = match generate_seal_design_recipes_with_gemini(&state, &input).await {
        Ok(value) => value,
        Err(err) => {
            log_seal_design_failure(SealDesignFailurePhase::RecipeGeneration, format!("{err:#}"));
            return seal_design_failure_response(SealDesignFailurePhase::RecipeGeneration);
        }
    };

    if recipe_variants.len() != input.variant_count {
        log_seal_design_failure(
            SealDesignFailurePhase::RecipeValidation,
            format!(
                "expected_variant_count={} actual_variant_count={}",
                input.variant_count,
                recipe_variants.len()
            ),
        );
        return seal_design_failure_response(SealDesignFailurePhase::RecipeValidation);
    }
    let recipe_variants = ensure_fresh_seal_design_recipes(&input, recipe_variants);

    let request_id = format!("seal_request_{}", Uuid::new_v4().simple());
    let variants =
        build_seal_design_variants(&state.storage_assets_bucket, &request_id, &recipe_variants);

    let images = match generate_seal_design_images_with_renderer(&input, &variants) {
        Ok(value) => value,
        Err(err) => {
            log_seal_design_failure(SealDesignFailurePhase::Rendering, format!("{err:#}"));
            return seal_design_failure_response(SealDesignFailurePhase::Rendering);
        }
    };

    if let Err(err) = upload_seal_design_images_to_storage(&state, &variants, &images).await {
        log_seal_design_failure(SealDesignFailurePhase::StorageUpload, format!("{err:#}"));
        return seal_design_failure_response(SealDesignFailurePhase::StorageUpload);
    }

    json_response(
        StatusCode::OK,
        json!({
            "request_id": request_id,
            "variants": variants.into_iter().map(|variant| {
                json!({
                    "id": variant.id,
                    "storage_path": variant.storage_path,
                    "download_url": variant.download_url,
                    "label": variant.label,
                    "recipe": seal_design_recipe_to_json(variant.recipe),
                    "width": variant.width,
                    "height": variant.height,
                })
            }).collect::<Vec<_>>(),
        }),
    )
}

async fn handle_create_order(State(state): State<AppState>, body: Bytes) -> Response {
    let request = match serde_json::from_slice::<CreateOrderRequest>(&body) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                &format!("invalid JSON: {err}"),
            );
        }
    };

    let input = match validate_create_order_request(request) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "validation_error",
                &err.to_string(),
            );
        }
    };

    match state.store.create_order(input).await {
        Ok(result) => {
            let status = if result.idempotent_replay {
                StatusCode::OK
            } else {
                StatusCode::CREATED
            };
            json_response(
                status,
                json!({
                    "order_id": result.order_id,
                    "order_no": result.order_no,
                    "status": result.status,
                    "payment_status": result.payment_status,
                    "fulfillment_status": result.fulfillment_status,
                    "pricing": {
                        "total": result.total,
                        "currency": result.currency,
                    },
                    "idempotent_replay": result.idempotent_replay,
                }),
            )
        }
        Err(StoreError::UnsupportedLocale) => error_response(
            StatusCode::BAD_REQUEST,
            "unsupported_locale",
            "unsupported locale",
        ),
        Err(StoreError::InvalidReference) => error_response(
            StatusCode::BAD_REQUEST,
            "invalid_reference",
            "invalid font/material/listing/country",
        ),
        Err(StoreError::InactiveReference) => error_response(
            StatusCode::BAD_REQUEST,
            "inactive_reference",
            "inactive font/material/listing/country",
        ),
        Err(StoreError::MaterialShapeMismatch) => error_response(
            StatusCode::BAD_REQUEST,
            "material_shape_mismatch",
            "selected material does not support seal shape",
        ),
        Err(StoreError::IdempotencyConflict) => error_response(
            StatusCode::CONFLICT,
            "idempotency_conflict",
            "idempotency key is already used with different payload",
        ),
        Err(StoreError::Internal(err)) => {
            eprintln!("failed to create order: {err:#}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
    }
}

async fn handle_get_order_status(
    State(state): State<AppState>,
    Path(order_id): Path<String>,
) -> Response {
    let order_id = order_id.trim();
    if order_id.is_empty() {
        return error_response(
            StatusCode::BAD_REQUEST,
            "validation_error",
            "order_id is required",
        );
    }

    match state.store.get_order_status(order_id).await {
        Ok(Some(result)) => {
            let result = reconcile_order_status_from_stripe_checkout(&state, result).await;
            json_response(StatusCode::OK, order_status_response_json(&result))
        }
        Ok(None) => error_response(StatusCode::NOT_FOUND, "order_not_found", "order not found"),
        Err(StoreError::Internal(err)) => {
            eprintln!("failed to load order status: {err:#}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
        Err(err) => {
            eprintln!("failed to load order status: {err}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
    }
}

async fn reconcile_order_status_from_stripe_checkout(
    state: &AppState,
    current: OrderStatusResult,
) -> OrderStatusResult {
    if !order_status_needs_stripe_checkout_reconciliation(&current) {
        return current;
    }

    let checkout_session_id = current.checkout_session_id.trim();
    if checkout_session_id.is_empty() || state.stripe_api_key.trim().is_empty() {
        return current;
    }

    let snapshot = match retrieve_stripe_checkout_session(state, checkout_session_id).await {
        Ok(snapshot) => snapshot,
        Err(err) => {
            eprintln!(
                "failed to reconcile order status from stripe checkout session order_id={} session_id={}: {err:#}",
                current.order_id, checkout_session_id
            );
            return current;
        }
    };

    if !snapshot.order_id.trim().is_empty() && snapshot.order_id.trim() != current.order_id {
        eprintln!(
            "stripe checkout session order_id mismatch status_order_id={} session_order_id={} session_id={}",
            current.order_id, snapshot.order_id, snapshot.session_id
        );
        return current;
    }

    match state
        .store
        .reconcile_order_from_stripe_checkout_session(&current.order_id, &snapshot)
        .await
    {
        Ok(Some(result)) => result,
        Ok(None) => current,
        Err(StoreError::Internal(err)) => {
            eprintln!(
                "failed to persist stripe checkout reconciliation order_id={} session_id={}: {err:#}",
                current.order_id, snapshot.session_id
            );
            current
        }
        Err(err) => {
            eprintln!(
                "failed to persist stripe checkout reconciliation order_id={} session_id={}: {err}",
                current.order_id, snapshot.session_id
            );
            current
        }
    }
}

fn order_status_needs_stripe_checkout_reconciliation(status: &OrderStatusResult) -> bool {
    let order_status = status.status.trim().to_ascii_lowercase();
    let payment_status = status.payment_status.trim().to_ascii_lowercase();
    if order_status == "paid" || payment_status == "paid" {
        return false;
    }

    !status.checkout_session_id.trim().is_empty()
        && matches!(
            order_status.as_str(),
            "pending_payment" | "created" | "pending" | ""
        )
        && matches!(payment_status.as_str(), "unpaid" | "pending" | "")
}

async fn handle_lookup_order(State(state): State<AppState>, body: Bytes) -> Response {
    let request = match serde_json::from_slice::<LookupOrderRequest>(&body) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                &format!("invalid JSON: {err}"),
            );
        }
    };

    let input = match validate_lookup_order_request(request) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "validation_error",
                &err.to_string(),
            );
        }
    };

    match state.store.lookup_order(input).await {
        Ok(Some(result)) => json_response(StatusCode::OK, order_lookup_response_json(&result)),
        Ok(None) => error_response(StatusCode::NOT_FOUND, "order_not_found", "order not found"),
        Err(StoreError::Internal(err)) => {
            eprintln!("failed to lookup order: {err:#}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
        Err(err) => {
            eprintln!("failed to lookup order: {err}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
    }
}

async fn handle_create_stripe_checkout_session(
    State(state): State<AppState>,
    body: Bytes,
) -> Response {
    if state.stripe_api_key.trim().is_empty() {
        return error_response(
            StatusCode::SERVICE_UNAVAILABLE,
            "stripe_not_configured",
            "stripe client is not configured",
        );
    }
    if state.stripe_checkout.success_url.trim().is_empty()
        || state.stripe_checkout.cancel_url.trim().is_empty()
        || state.stripe_checkout.app_success_url.trim().is_empty()
        || state.stripe_checkout.app_cancel_url.trim().is_empty()
    {
        return error_response(
            StatusCode::SERVICE_UNAVAILABLE,
            "stripe_not_configured",
            "stripe checkout urls are not configured",
        );
    }

    let request = match serde_json::from_slice::<CreateStripeCheckoutSessionRequest>(&body) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "invalid_json",
                &format!("invalid JSON: {err}"),
            );
        }
    };

    let input = match validate_create_stripe_checkout_session_request(request) {
        Ok(v) => v,
        Err(err) => {
            return error_response(
                StatusCode::BAD_REQUEST,
                "validation_error",
                &err.to_string(),
            );
        }
    };

    let Some(order) = (match state
        .store
        .get_order_checkout_context(&input.order_id)
        .await
    {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to load order for stripe checkout session: {err}");
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
    }) else {
        return error_response(StatusCode::NOT_FOUND, "order_not_found", "order not found");
    };

    if order.status != "pending_payment" || order.payment_status != "unpaid" {
        return error_response(
            StatusCode::CONFLICT,
            "order_not_payable",
            "order is not payable",
        );
    }
    if order.total <= 0 {
        return error_response(
            StatusCode::BAD_REQUEST,
            "invalid_order_total",
            "order total must be greater than zero",
        );
    }

    let customer_email = if input.customer_email.is_empty() {
        order.contact_email.clone()
    } else {
        input.customer_email.clone()
    };

    let session = match create_stripe_checkout_session(
        &state,
        &order,
        &customer_email,
        input.return_to_app,
    )
    .await
    {
        Ok(v) => v,
        Err(err) => {
            eprintln!("failed to create stripe checkout session: {err:#}");
            if let Err(rollback_err) = state
                .store
                .cancel_pending_order_and_release_listing(
                    &order.order_id,
                    order.listing_key.as_str(),
                    "Stripe checkout session creation failed",
                )
                .await
            {
                eprintln!(
                    "failed to roll back reserved listing after stripe checkout failure: {rollback_err:#}"
                );
                return error_response(
                    StatusCode::INTERNAL_SERVER_ERROR,
                    "internal",
                    "internal server error",
                );
            }
            return error_response(
                StatusCode::BAD_GATEWAY,
                "stripe_checkout_failed",
                "failed to create stripe checkout session",
            );
        }
    };

    if state
        .store
        .set_order_checkout_session(&order.order_id, &session.session_id)
        .await
        .is_err()
    {
        if let Err(rollback_err) = state
            .store
            .cancel_pending_order_and_release_listing(
                &order.order_id,
                order.listing_key.as_str(),
                "Stripe checkout session persistence failed",
            )
            .await
        {
            eprintln!(
                "failed to roll back reserved listing after stripe checkout persistence failure: {rollback_err:#}"
            );
            return error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            );
        }
        return error_response(
            StatusCode::INTERNAL_SERVER_ERROR,
            "internal",
            "internal server error",
        );
    }

    json_response(
        StatusCode::CREATED,
        json!({
            "order_id": order.order_id,
            "session_id": session.session_id,
            "checkout_url": session.checkout_url,
            "payment_intent_id": session.payment_intent_id,
        }),
    )
}

async fn handle_stripe_webhook(
    State(state): State<AppState>,
    headers: HeaderMap,
    body: Bytes,
) -> Response {
    let signature = headers
        .get("Stripe-Signature")
        .and_then(|value| value.to_str().ok())
        .unwrap_or_default();

    let payload = if state.stripe_webhook_secret.trim().is_empty() {
        match serde_json::from_slice::<JsonValue>(&body) {
            Ok(event) => event,
            Err(err) => {
                return error_response(
                    StatusCode::BAD_REQUEST,
                    "invalid_payload",
                    &err.to_string(),
                );
            }
        }
    } else {
        match construct_stripe_webhook_event(&body, signature, &state.stripe_webhook_secret) {
            Ok(event) => event,
            Err(StripeWebhookError::Json(err)) => {
                return error_response(
                    StatusCode::BAD_REQUEST,
                    "invalid_payload",
                    &err.to_string(),
                );
            }
            Err(err) => {
                return error_response(
                    StatusCode::UNAUTHORIZED,
                    "invalid_signature",
                    &err.to_string(),
                );
            }
        }
    };

    let event = match parse_stripe_event(payload) {
        Ok(event) => event,
        Err(err) => {
            return error_response(StatusCode::BAD_REQUEST, "invalid_payload", &err.to_string());
        }
    };

    match state.store.process_stripe_webhook(event).await {
        Ok(result) => json_response(
            StatusCode::OK,
            json!({
                "ok": true,
                "processed": result.processed,
                "already_processed": result.already_processed,
            }),
        ),
        Err(StoreError::Internal(err)) => {
            eprintln!("failed to process stripe webhook: {err:#}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
        Err(other) => {
            eprintln!("failed to process stripe webhook: {other}");
            error_response(
                StatusCode::INTERNAL_SERVER_ERROR,
                "internal",
                "internal server error",
            )
        }
    }
}

fn json_response(status: StatusCode, payload: JsonValue) -> Response {
    (status, axum::Json(payload)).into_response()
}

fn error_response(status: StatusCode, code: &str, message: &str) -> Response {
    json_response(
        status,
        json!({
            "error": {
                "code": code,
                "message": message,
            }
        }),
    )
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
enum SealDesignFailurePhase {
    RecipeGeneration,
    RecipeValidation,
    Rendering,
    StorageUpload,
}

impl SealDesignFailurePhase {
    fn log_name(self) -> &'static str {
        match self {
            Self::RecipeGeneration => "recipe_generation",
            Self::RecipeValidation => "recipe_validation",
            Self::Rendering => "rendering",
            Self::StorageUpload => "storage_upload",
        }
    }

    fn error_code(self) -> &'static str {
        match self {
            Self::RecipeGeneration | Self::RecipeValidation => "gemini_generation_failed",
            Self::Rendering | Self::StorageUpload => "seal_generation_failed",
        }
    }

    fn public_message(self) -> &'static str {
        match self {
            Self::RecipeGeneration => "failed to generate seal design recipes",
            Self::RecipeValidation => "gemini returned an unexpected number of seal design recipes",
            Self::Rendering => "failed to render seal design images",
            Self::StorageUpload => "failed to save seal design images",
        }
    }
}

fn seal_design_failure_response(phase: SealDesignFailurePhase) -> Response {
    error_response(
        StatusCode::BAD_GATEWAY,
        phase.error_code(),
        phase.public_message(),
    )
}

fn log_seal_design_failure(phase: SealDesignFailurePhase, detail: impl fmt::Display) {
    eprintln!(
        "seal design generation failed phase={} code={} detail={}",
        phase.log_name(),
        phase.error_code(),
        detail
    );
}

fn order_status_response_json(result: &OrderStatusResult) -> JsonValue {
    json!({
        "order_id": result.order_id,
        "order_no": result.order_no,
        "status": result.status,
        "order_status": result.status,
        "payment": {
            "status": result.payment_status,
            "checkout_session_id": nullable_string_json(&result.checkout_session_id),
            "payment_intent_id": nullable_string_json(&result.payment_intent_id),
        },
        "payment_status": result.payment_status,
        "fulfillment": {
            "status": result.fulfillment_status,
            "carrier": nullable_string_json(&result.fulfillment_carrier),
            "tracking_no": nullable_string_json(&result.fulfillment_tracking_no),
        },
        "fulfillment_status": result.fulfillment_status,
        "production_status": result.production_status,
        "shipping_status": result.shipping_status,
        "tracking_number": nullable_string_json(&result.fulfillment_tracking_no),
        "pricing": {
            "total": result.total,
            "currency": result.currency,
        },
        "updated_at": timestamp_json(result.updated_at.as_ref()),
    })
}

fn order_lookup_response_json(result: &OrderLookupResult) -> JsonValue {
    json!({
        "order_id": result.status.order_id,
        "order_no": result.status.order_no,
        "created_at": timestamp_json(result.created_at.as_ref()),
        "status": result.status.status,
        "order_status": result.status.status,
        "payment": {
            "status": result.status.payment_status,
            "checkout_session_id": nullable_string_json(&result.status.checkout_session_id),
            "payment_intent_id": nullable_string_json(&result.status.payment_intent_id),
        },
        "payment_status": result.status.payment_status,
        "fulfillment": {
            "status": result.status.fulfillment_status,
            "carrier": nullable_string_json(&result.status.fulfillment_carrier),
            "tracking_no": nullable_string_json(&result.status.fulfillment_tracking_no),
            "shipped_at": timestamp_json(result.shipped_at.as_ref()),
        },
        "fulfillment_status": result.status.fulfillment_status,
        "production_status": result.status.production_status,
        "shipping_status": result.status.shipping_status,
        "tracking_number": nullable_string_json(&result.status.fulfillment_tracking_no),
        "pricing": {
            "total": result.status.total,
            "currency": result.status.currency,
        },
        "seal": {
            "confirmed_seal_text": result.seal_confirmed_text,
            "preview_image_url": nullable_string_json(&result.seal_preview_image_url),
        },
        "listing": {
            "id": nullable_string_json(&result.listing_id),
            "title": nullable_string_json(&result.listing_title),
        },
        "updated_at": timestamp_json(result.status.updated_at.as_ref()),
    })
}

fn timestamp_json(value: Option<&DateTime<Utc>>) -> JsonValue {
    value
        .map(|value| JsonValue::String(value.to_rfc3339_opts(SecondsFormat::Secs, true)))
        .unwrap_or(JsonValue::Null)
}

fn nullable_string_json(value: &str) -> JsonValue {
    let value = value.trim();
    if value.is_empty() {
        JsonValue::Null
    } else {
        JsonValue::String(value.to_owned())
    }
}

fn order_lookup_query_request(order_no: &str) -> RunQueryRequest {
    RunQueryRequest {
        structured_query: Some(json!({
            "from": [
                { "collectionId": "orders" }
            ],
            "where": {
                "fieldFilter": {
                    "field": { "fieldPath": "order_no" },
                    "op": "EQUAL",
                    "value": { "stringValue": order_no }
                }
            }
        })),
        ..RunQueryRequest::default()
    }
}

fn order_status_result_from_fields(
    order_id: &str,
    fields: &BTreeMap<String, JsonValue>,
) -> OrderStatusResult {
    let payment = read_map_field(fields, "payment");
    let fulfillment = read_map_field(fields, "fulfillment");
    let production = read_map_field(fields, "production");
    let shipping = read_map_field(fields, "shipping");
    let pricing = read_map_field(fields, "pricing");

    OrderStatusResult {
        order_id: order_id.to_owned(),
        order_no: read_string_field(fields, "order_no"),
        status: read_string_field(fields, "status"),
        payment_status: read_string_field(&payment, "status"),
        checkout_session_id: read_string_field(&payment, "checkout_session_id"),
        payment_intent_id: first_non_empty(&[
            Some(read_string_field(&payment, "intent_id")),
            Some(read_string_field(&payment, "payment_intent_id")),
        ])
        .unwrap_or_default(),
        fulfillment_status: read_string_field(&fulfillment, "status"),
        fulfillment_carrier: read_string_field(&fulfillment, "carrier"),
        fulfillment_tracking_no: first_non_empty(&[
            Some(read_string_field(&fulfillment, "tracking_no")),
            Some(read_string_field(&fulfillment, "tracking_number")),
        ])
        .unwrap_or_default(),
        production_status: first_non_empty(&[
            Some(read_string_field(&production, "status")),
            Some(read_string_field(fields, "production_status")),
        ])
        .unwrap_or_else(|| "not_started".to_owned()),
        shipping_status: first_non_empty(&[
            Some(read_string_field(&shipping, "status")),
            Some(read_string_field(fields, "shipping_status")),
        ])
        .unwrap_or_else(|| "not_shipped".to_owned()),
        total: pricing_total(&pricing),
        currency: pricing_currency(&pricing),
        updated_at: read_timestamp_field(fields, "updated_at"),
    }
}

fn order_lookup_result_from_document(
    document: &Document,
    requested_email: &str,
) -> Option<OrderLookupResult> {
    let contact = read_map_field(&document.fields, "contact");
    let stored_email = read_string_field(&contact, "email");
    if !order_lookup_email_matches(&stored_email, requested_email) {
        return None;
    }

    let order_id = document_id(document)?;
    let status = order_status_result_from_fields(&order_id, &document.fields);
    let seal = read_map_field(&document.fields, "seal");
    let preview_image = read_map_field(&seal, "preview_image");
    let customer_confirmation = read_map_field(&document.fields, "customer_confirmation");
    let listing = read_map_field(&document.fields, "listing");
    let fulfillment = read_map_field(&document.fields, "fulfillment");
    let order_locale = first_non_empty(&[
        Some(read_string_field(&document.fields, "locale")),
        Some(DEFAULT_LOCALE.to_owned()),
    ])
    .unwrap_or_else(|| DEFAULT_LOCALE.to_owned());
    let seal_text = first_non_empty(&[
        Some(read_string_field(
            &customer_confirmation,
            "confirmed_seal_text",
        )),
        Some(format!(
            "{}{}",
            read_string_field(&seal, "line1"),
            read_string_field(&seal, "line2")
        )),
    ])
    .unwrap_or_default();
    let listing_title = first_non_empty(&[
        Some(resolve_localized(
            &read_string_map_field(&listing, "title_i18n"),
            &order_locale,
            DEFAULT_LOCALE,
        )),
        Some(read_string_field(&listing, "title")),
        Some(read_string_field(&listing, "listing_code")),
    ])
    .unwrap_or_default();

    Some(OrderLookupResult {
        status,
        created_at: read_timestamp_field(&document.fields, "created_at"),
        seal_confirmed_text: seal_text,
        seal_preview_image_url: read_string_field(&preview_image, "download_url"),
        listing_id: first_non_empty(&[
            Some(read_string_field(&listing, "key")),
            Some(read_string_field(&listing, "id")),
        ])
        .unwrap_or_default(),
        listing_title,
        shipped_at: read_timestamp_field(&fulfillment, "shipped_at"),
    })
}

fn order_lookup_email_matches(stored_email: &str, requested_email: &str) -> bool {
    let stored_email = stored_email.trim();
    !stored_email.is_empty() && stored_email.eq_ignore_ascii_case(requested_email.trim())
}

fn firestore_client_from_access_token(access_token: &str) -> Result<FirebaseFirestoreClient> {
    Ok(FirebaseFirestoreClient::new(access_token.to_owned()))
}

impl FirestoreStore {
    async fn firestore_client(&self) -> Result<FirebaseFirestoreClient> {
        let access_token = self
            .token_provider
            .token(&[DATASTORE_SCOPE])
            .await
            .context("failed to acquire firestore access token")?;
        firestore_client_from_access_token(access_token.as_str())
    }

    async fn get_public_config(&self) -> Result<PublicConfig> {
        let client = self.firestore_client().await?;
        let name = format!("{}/app_config/public", self.parent);

        match client
            .get_document(&name, &GetDocumentOptions::default())
            .await
        {
            Ok(document) => {
                let supported = read_string_array_from_map(&document.fields, "supported_locales");
                let default_locale = read_string_field(&document.fields, "default_locale");
                let default_currency = first_non_empty(&[
                    Some(read_string_field(&document.fields, "default_currency")),
                    Some(read_string_field(&document.fields, "currency")),
                ])
                .unwrap_or_default();
                let currency_by_locale =
                    read_string_map_field(&document.fields, "currency_by_locale");
                Ok(normalize_public_config(PublicConfig {
                    supported_locales: supported,
                    default_locale,
                    default_currency,
                    currency_by_locale,
                }))
            }
            Err(err) if is_not_found(&err) => Ok(default_public_config()),
            Err(err) => Err(anyhow!(err)),
        }
    }

    async fn list_active_fonts(&self) -> Result<Vec<Font>> {
        let documents = self.query_active_documents("fonts").await?;
        let mut fonts = Vec::with_capacity(documents.len());

        for document in documents {
            let key =
                document_id(&document).ok_or_else(|| anyhow!("fonts document is missing name"))?;
            let font_family = read_string_field(&document.fields, "font_family");
            if font_family.is_empty() {
                bail!("fonts/{key} is missing font_family");
            }
            let label = resolve_font_label_field(&document.fields, &key);
            let kanji_style = read_string_field(&document.fields, "kanji_style");
            let kanji_style = normalize_catalog_kanji_style(&kanji_style).to_owned();

            fonts.push(Font {
                key,
                label,
                font_family,
                kanji_style,
                version: read_int_field(&document.fields, "version").unwrap_or(1),
            });
        }

        Ok(fonts)
    }

    async fn list_active_materials(&self) -> Result<Vec<Material>> {
        let documents = self.query_active_documents("materials").await?;
        let mut materials = Vec::with_capacity(documents.len());

        for document in documents {
            let key = document_id(&document)
                .ok_or_else(|| anyhow!("materials document is missing name"))?;
            let price_by_currency = material_price_by_currency_from_fields(&document.fields);
            if price_by_currency.is_empty() {
                bail!("materials/{key} is missing price_by_currency");
            }

            let photos = read_material_photos(&document.fields);

            materials.push(Material {
                key,
                label_i18n: read_string_map_field(&document.fields, "label_i18n"),
                description_i18n: read_string_map_field(&document.fields, "description_i18n"),
                shape: read_material_shape(&document.fields),
                photos,
                price_by_currency,
                version: read_int_field(&document.fields, "version").unwrap_or(1),
            });
        }

        Ok(materials)
    }

    async fn list_active_stone_listings(&self) -> Result<Vec<StoneListing>> {
        let client = self.firestore_client().await?;
        let documents = self
            .run_documents_query(&client, "stone_listings", false, true)
            .await?;
        let mut listings = Vec::with_capacity(documents.len());

        for document in documents {
            let key = document_id(&document)
                .ok_or_else(|| anyhow!("stone_listings document is missing name"))?;
            let is_active = read_bool_field(&document.fields, "is_active").unwrap_or(true);
            let status = read_string_field(&document.fields, "status");
            if !stone_listing_is_app_visible(is_active, &status) {
                continue;
            }
            let price_by_currency = stone_listing_price_by_currency_from_fields(&document.fields);
            if price_by_currency.is_empty() {
                eprintln!("warning: stone_listings/{key} is missing price data; skipping");
                continue;
            }

            listings.push(stone_listing_from_fields(&key, &document.fields));
        }

        Ok(listings)
    }

    async fn get_app_visible_stone_listing(&self, key: &str) -> Result<Option<StoneListing>> {
        let key = key.trim();
        if key.is_empty() || key.contains('/') {
            return Ok(None);
        }

        let client = self.firestore_client().await?;
        let doc_name = format!("{}/stone_listings/{}", self.parent, key);
        let doc = match client
            .get_document(&doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(None),
            Err(err) => return Err(anyhow!(err)),
        };

        let listing = stone_listing_from_fields(key, &doc.fields);
        if !stone_listing_is_app_visible(listing.is_active, &listing.status) {
            return Ok(None);
        }
        if listing.price_by_currency.is_empty() {
            bail!("stone_listings/{key} is missing price data");
        }

        Ok(Some(listing))
    }

    async fn list_active_countries(&self) -> Result<Vec<Country>> {
        let documents = self.query_active_documents("countries").await?;
        let mut countries = Vec::with_capacity(documents.len());

        for document in documents {
            let code = document_id(&document)
                .ok_or_else(|| anyhow!("countries document is missing name"))?;
            let shipping_fee_by_currency =
                country_shipping_fee_by_currency_from_fields(&document.fields);
            if shipping_fee_by_currency.is_empty() {
                bail!("countries/{code} is missing shipping_fee_by_currency");
            }

            countries.push(Country {
                code,
                label_i18n: read_string_map_field(&document.fields, "label_i18n"),
                shipping_fee_by_currency,
                version: read_int_field(&document.fields, "version").unwrap_or(1),
            });
        }

        Ok(countries)
    }

    async fn list_active_facet_tags(&self) -> Result<Vec<FacetTag>> {
        let client = self.firestore_client().await?;
        let documents = self
            .run_documents_query(&client, "facet_tags", true, false)
            .await?;
        let mut tags = Vec::with_capacity(documents.len());

        for document in documents {
            let document_id = document_id(&document).unwrap_or_default();
            let facet_type = first_non_empty(&[
                Some(read_string_field(&document.fields, "facet_type")),
                document_id
                    .split_once(':')
                    .map(|(facet_type, _)| facet_type.to_owned()),
            ])
            .and_then(|raw| normalize_facet_tag_type(&raw).map(ToOwned::to_owned))
            .unwrap_or_default();
            let key = first_non_empty(&[
                Some(read_string_field(&document.fields, "key")),
                document_id
                    .split_once(':')
                    .map(|(_, key)| key.to_owned())
                    .or_else(|| {
                        if document_id.is_empty() {
                            None
                        } else {
                            Some(document_id.clone())
                        }
                    }),
            ])
            .map(|raw| normalize_facet_value(&raw))
            .unwrap_or_default();

            if facet_type.is_empty() || key.is_empty() {
                continue;
            }

            tags.push(FacetTag {
                facet_type,
                key,
                label_i18n: read_string_map_field(&document.fields, "label_i18n"),
                aliases: read_string_array_field(&document.fields, "aliases"),
            });
        }

        Ok(tags)
    }

    async fn get_orderable_stone_listing_document(
        &self,
        client: &FirebaseFirestoreClient,
        key: &str,
    ) -> Result<Document, StoreError> {
        let doc_name = format!("{}/stone_listings/{}", self.parent, key);
        let doc = client
            .get_document(&doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| {
                if is_not_found(&err) {
                    StoreError::InvalidReference
                } else {
                    StoreError::Internal(anyhow!(err))
                }
            })?;
        let is_active = read_bool_field(&doc.fields, "is_active").unwrap_or(true);
        let status = read_string_field(&doc.fields, "status");
        if !stone_listing_is_orderable(is_active, &status) {
            return Err(StoreError::InactiveReference);
        }
        if stone_listing_price_by_currency_from_fields(&doc.fields).is_empty() {
            return Err(StoreError::Internal(anyhow!(
                "stone_listings/{key} is missing price data"
            )));
        }

        Ok(doc)
    }

    async fn query_active_documents(&self, collection: &str) -> Result<Vec<Document>> {
        let client = self.firestore_client().await?;
        let documents = self
            .run_documents_query(&client, collection, true, false)
            .await?;
        if documents.is_empty() {
            bail!("no active {collection} found in firestore");
        }

        Ok(documents)
    }

    async fn run_documents_query(
        &self,
        client: &FirebaseFirestoreClient,
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
            .run_query(&self.parent, &query)
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
                    let left_published_at = read_timestamp_field(&left.fields, "published_at")
                        .unwrap_or(DateTime::<Utc>::UNIX_EPOCH);
                    let right_published_at = read_timestamp_field(&right.fields, "published_at")
                        .unwrap_or(DateTime::<Utc>::UNIX_EPOCH);
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

    async fn create_order(&self, input: CreateOrderInput) -> Result<CreateOrderResult, StoreError> {
        let normalized = normalize_create_order_input(input);
        let request_hash =
            hash_order_request(&normalized).map_err(|err| StoreError::Internal(err.into()))?;
        let idempotency_id = format!("{}:{}", normalized.channel, normalized.idempotency_key);

        let cfg = self
            .get_public_config()
            .await
            .map_err(StoreError::Internal)?;
        if !contains(&cfg.supported_locales, &normalized.locale)
            || !contains(&cfg.supported_locales, &normalized.contact.preferred_locale)
        {
            return Err(StoreError::UnsupportedLocale);
        }
        let pricing_currency = resolve_pricing_currency(&cfg, &normalized.locale);

        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;

        let idempotency_doc_name = format!("{}/idempotency_keys/{}", self.parent, idempotency_id);
        if let Some(result) = self
            .try_idempotent_replay(&client, &idempotency_doc_name, &request_hash)
            .await?
        {
            return Ok(result);
        }

        let font = self
            .get_active_font(&client, &normalized.seal.font_key)
            .await?;
        let listing_id = normalized
            .listing_id
            .as_deref()
            .ok_or(StoreError::InvalidReference)?;
        let listing_doc = self
            .get_orderable_stone_listing_document(&client, listing_id)
            .await?;
        let listing_snapshot = stone_listing_from_fields(listing_id, &listing_doc.fields);
        let country = self
            .get_active_country(&client, &normalized.shipping.country_code)
            .await?;
        let shape_supports = stone_listing_shape_matches(&listing_snapshot, &normalized.seal.shape);
        if !shape_supports {
            return Err(StoreError::MaterialShapeMismatch);
        }

        let now = Utc::now();
        let subtotal = stone_listing_price_for_currency(&listing_snapshot, &pricing_currency);
        let shipping = country_shipping_fee_for_currency(&country, &pricing_currency);
        let tax = 0_i64;
        let discount = 0_i64;
        let total = (subtotal + shipping + tax - discount).max(0);

        let random = Uuid::new_v4();
        let order_suffix =
            u16::from_be_bytes([random.as_bytes()[0], random.as_bytes()[1]]) % 10_000;
        let order_no = format!(
            "HF-{}-{:04}",
            now.with_timezone(&self.jst).format("%Y%m%d"),
            order_suffix
        );

        let order_id = Uuid::new_v4().to_string();
        let order_doc = Document {
            name: None,
            fields: build_order_fields(
                &normalized,
                &font,
                Some(&listing_snapshot),
                &country,
                &order_no,
                subtotal,
                shipping,
                tax,
                discount,
                total,
                &pricing_currency,
                now,
            ),
            ..Document::default()
        };
        let event_doc = Document {
            name: None,
            fields: btree_from_pairs(vec![
                ("type", fs_string("order_created")),
                ("actor_type", fs_string("system")),
                (
                    "payload",
                    fs_map(btree_from_pairs(vec![
                        ("channel", fs_string(normalized.channel.clone())),
                        ("total", fs_int(total)),
                        ("currency", fs_string(pricing_currency.clone())),
                    ])),
                ),
                ("created_at", fs_timestamp(now)),
            ]),
            ..Document::default()
        };
        let idempotency_doc = Document {
            name: None,
            fields: btree_from_pairs(vec![
                ("channel", fs_string(normalized.channel.clone())),
                (
                    "idempotency_key",
                    fs_string(normalized.idempotency_key.clone()),
                ),
                ("request_hash", fs_string(request_hash.clone())),
                ("order_id", fs_string(order_id.clone())),
                ("created_at", fs_timestamp(now)),
                ("expire_at", fs_timestamp(now + Duration::days(30))),
            ]),
            ..Document::default()
        };
        let listing_doc_name = format!("{}/stone_listings/{}", self.parent, listing_id);
        let listing_update_time = listing_doc.update_time.clone().ok_or_else(|| {
            StoreError::Internal(anyhow!(
                "stone_listings/{listing_id} is missing update_time"
            ))
        })?;

        let commit_result = client
            .commit(
                &self.database,
                &CommitRequest {
                    writes: vec![
                        reserve_stone_listing_write(
                            &listing_doc_name,
                            listing_snapshot.version + 1,
                            &listing_update_time,
                            now,
                        ),
                        json!({
                            "update": {
                                "name": format!("{}/orders/{}", self.parent, order_id),
                                "fields": order_doc.fields
                            },
                            "currentDocument": {
                                "exists": false
                            }
                        }),
                        json!({
                            "update": {
                                "name": format!("{}/orders/{}/events/{}", self.parent, order_id, Uuid::new_v4()),
                                "fields": event_doc.fields
                            },
                            "currentDocument": {
                                "exists": false
                            }
                        }),
                        json!({
                            "update": {
                                "name": format!("{}/idempotency_keys/{}", self.parent, idempotency_id),
                                "fields": idempotency_doc.fields
                            },
                            "currentDocument": {
                                "exists": false
                            }
                        }),
                    ],
                    transaction: None,
                },
            )
            .await;

        if let Err(err) = commit_result {
            if create_order_commit_conflicted(&err) {
                if let Some(result) = self
                    .try_idempotent_replay(&client, &idempotency_doc_name, &request_hash)
                    .await?
                {
                    return Ok(result);
                }
                return Err(StoreError::InactiveReference);
            }
            return Err(StoreError::Internal(anyhow!(err)));
        }

        return Ok(CreateOrderResult {
            order_id,
            order_no,
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            fulfillment_status: "pending".to_owned(),
            total,
            currency: pricing_currency,
            idempotent_replay: false,
        });
    }

    async fn get_order_checkout_context(
        &self,
        order_id: &str,
    ) -> Result<Option<OrderCheckoutContext>, StoreError> {
        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);

        let order_doc = match client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(None),
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        let payment = read_map_field(&order_doc.fields, "payment");
        let pricing = read_map_field(&order_doc.fields, "pricing");
        let contact = read_map_field(&order_doc.fields, "contact");
        let seal = read_map_field(&order_doc.fields, "seal");
        let listing = read_map_field(&order_doc.fields, "listing");
        let material = read_map_field(&order_doc.fields, "material");
        let shipping = read_map_field(&order_doc.fields, "shipping");
        let order_locale = read_string_field(&order_doc.fields, "locale");
        let resolved_order_locale = if order_locale.trim().is_empty() {
            DEFAULT_LOCALE.to_owned()
        } else {
            order_locale.trim().to_lowercase()
        };
        let (listing_key, listing_label) = Self::resolve_order_listing_fields(
            &order_doc.fields,
            &listing,
            &material,
            &resolved_order_locale,
            DEFAULT_LOCALE,
        );
        let seal_shape = read_string_field(&seal, "shape");

        Ok(Some(OrderCheckoutContext {
            order_id: order_id.to_owned(),
            order_locale: resolved_order_locale,
            status: read_string_field(&order_doc.fields, "status"),
            payment_status: read_string_field(&payment, "status"),
            listing_key,
            listing_label,
            seal_shape,
            shipping_country_code: read_string_field(&shipping, "country_code"),
            shipping_recipient_name: read_string_field(&shipping, "recipient_name"),
            shipping_phone: read_string_field(&shipping, "phone"),
            shipping_postal_code: read_string_field(&shipping, "postal_code"),
            shipping_state: read_string_field(&shipping, "state"),
            shipping_city: read_string_field(&shipping, "city"),
            shipping_address_line1: read_string_field(&shipping, "address_line1"),
            shipping_address_line2: read_string_field(&shipping, "address_line2"),
            total: pricing_total(&pricing),
            currency: pricing_currency(&pricing),
            contact_email: read_string_field(&contact, "email"),
        }))
    }

    async fn get_order_status(
        &self,
        order_id: &str,
    ) -> Result<Option<OrderStatusResult>, StoreError> {
        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);

        let order_doc = match client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(None),
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        Ok(Some(order_status_result_from_fields(
            order_id,
            &order_doc.fields,
        )))
    }

    async fn lookup_order(
        &self,
        input: LookupOrderInput,
    ) -> Result<Option<OrderLookupResult>, StoreError> {
        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let query = order_lookup_query_request(&input.order_no);
        let rows = client
            .run_query(&self.parent, &query)
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        for document in rows.into_iter().filter_map(|row| row.document) {
            if let Some(result) = order_lookup_result_from_document(&document, &input.email) {
                return Ok(Some(result));
            }
        }

        Ok(None)
    }

    fn resolve_order_listing_fields(
        data: &BTreeMap<String, JsonValue>,
        listing: &BTreeMap<String, JsonValue>,
        material: &BTreeMap<String, JsonValue>,
        requested_locale: &str,
        default_locale: &str,
    ) -> (String, String) {
        let listing_key = first_non_empty(&[
            Some(read_string_field(listing, "key")),
            Some(read_string_field(data, "listing_key")),
            Some(read_string_field(material, "key")),
        ])
        .unwrap_or_default();
        let listing_label = first_non_empty(&[
            Some(resolve_localized(
                &read_string_map_field(listing, "title_i18n"),
                requested_locale,
                default_locale,
            )),
            Some(resolve_localized(
                &read_string_map_field(material, "label_i18n"),
                requested_locale,
                default_locale,
            )),
            Some(read_string_field(data, "material_label_ja")),
            Some(read_string_field(listing, "listing_code")),
            Some(read_string_field(data, "listing_key")),
            Some(read_string_field(material, "key")),
        ])
        .unwrap_or_default();

        (listing_key, listing_label)
    }

    async fn try_set_order_checkout_session(
        client: &FirebaseFirestoreClient,
        order_doc_name: &str,
        session_id: &str,
    ) -> Result<(), StoreError> {
        let mut order_doc = client
            .get_document(order_doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        let mut payment = read_map_field(&order_doc.fields, "payment");
        payment.insert("provider".to_owned(), fs_string("stripe"));
        payment.insert(
            "checkout_session_id".to_owned(),
            fs_string(session_id.to_owned()),
        );
        order_doc
            .fields
            .insert("payment".to_owned(), fs_map(payment));
        order_doc
            .fields
            .insert("updated_at".to_owned(), fs_timestamp(Utc::now()));

        client
            .patch_document(order_doc_name, &order_doc, &PatchDocumentOptions::default())
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        Ok(())
    }

    async fn order_checkout_session_is_persisted(
        client: &FirebaseFirestoreClient,
        order_doc_name: &str,
        session_id: &str,
    ) -> Result<bool, StoreError> {
        let order_doc = match client
            .get_document(order_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(false),
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        let payment = read_map_field(&order_doc.fields, "payment");
        Ok(read_string_field(&payment, "checkout_session_id") == session_id)
    }

    async fn set_order_checkout_session(
        &self,
        order_id: &str,
        session_id: &str,
    ) -> Result<(), StoreError> {
        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);

        if let Err(err) =
            Self::try_set_order_checkout_session(&client, &order_doc_name, session_id).await
        {
            eprintln!("failed to persist stripe checkout session id: {err}");
        } else {
            return Ok(());
        }

        // Firestore can surface an error after the write has already been stored.
        // Read the order back before treating the failure as definitive.
        match Self::order_checkout_session_is_persisted(&client, &order_doc_name, session_id).await
        {
            Ok(true) => return Ok(()),
            Ok(false) => {}
            Err(err) => {
                eprintln!("failed to verify stripe checkout session persistence: {err}");
            }
        }

        if let Err(retry_err) =
            Self::try_set_order_checkout_session(&client, &order_doc_name, session_id).await
        {
            eprintln!("failed to retry persisting stripe checkout session id: {retry_err}");
        } else {
            return Ok(());
        }

        match Self::order_checkout_session_is_persisted(&client, &order_doc_name, session_id).await
        {
            Ok(true) => return Ok(()),
            Ok(false) | Err(_) => {}
        }

        Err(StoreError::Internal(anyhow!(
            "failed to persist stripe checkout session id"
        )))
    }

    async fn cancel_pending_order_and_release_listing(
        &self,
        order_id: &str,
        listing_key: &str,
        rollback_reason: &str,
    ) -> Result<(), StoreError> {
        let listing_key = listing_key.trim();

        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);
        let listing_doc_name = format!("{}/stone_listings/{}", self.parent, listing_key);
        let now = Utc::now();

        let order_doc = client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;
        let order_update_time = order_doc.update_time.clone().ok_or_else(|| {
            StoreError::Internal(anyhow!("orders/{order_id} is missing update_time"))
        })?;
        let before_status = read_string_field(&order_doc.fields, "status");
        let mut payment = read_map_field(&order_doc.fields, "payment");
        payment.insert("status".to_owned(), fs_string("failed"));

        let mut writes = vec![json!({
            "update": {
                "name": order_doc_name,
                "fields": {
                    "payment": fs_map(payment),
                    "status": fs_string("canceled"),
                    "status_updated_at": fs_timestamp(now),
                    "updated_at": fs_timestamp(now),
                }
            },
            "updateMask": {
                "fieldPaths": ["payment", "status", "status_updated_at", "updated_at"]
            },
            "currentDocument": {
                "updateTime": order_update_time
            }
        })];

        let event_doc = Document {
            name: None,
            fields: btree_from_pairs(vec![
                ("type", fs_string("status_changed")),
                ("actor_type", fs_string("system")),
                ("actor_id", fs_string("stripe.checkout_session")),
                ("before_status", fs_string(before_status)),
                ("after_status", fs_string("canceled")),
                ("note", fs_string(rollback_reason.trim().to_owned())),
                ("created_at", fs_timestamp(now)),
            ]),
            ..Document::default()
        };
        writes.push(json!({
            "update": {
                "name": format!("{}/orders/{}/events/{}", self.parent, order_id, Uuid::new_v4()),
                "fields": event_doc.fields
            },
            "currentDocument": {
                "exists": false
            }
        }));

        if !listing_key.is_empty() {
            let listing_doc = client
                .get_document(&listing_doc_name, &GetDocumentOptions::default())
                .await
                .map_err(|err| StoreError::Internal(anyhow!(err)))?;
            let target_listing_status =
                stone_listing_status_after_order_status("canceled").unwrap_or("published");
            let current_listing_status = read_string_field(&listing_doc.fields, "status");
            let current_published_at = read_timestamp_field(&listing_doc.fields, "published_at");
            if stone_listing_should_restore_after_canceled_order(
                &current_listing_status,
                current_published_at.is_none(),
            ) {
                let published_at = current_published_at.unwrap_or_else(|| now.clone());
                let listing_update_time = listing_doc.update_time.clone().ok_or_else(|| {
                    StoreError::Internal(anyhow!(
                        "stone_listings/{listing_key} is missing update_time"
                    ))
                })?;
                let listing_version =
                    read_int_field(&listing_doc.fields, "version").unwrap_or_default();
                writes.push(json!({
                    "update": {
                        "name": listing_doc_name,
                        "fields": {
                            "status": fs_string(target_listing_status),
                            "published_at": fs_timestamp(published_at),
                            "version": fs_int(listing_version + 1),
                            "updated_at": fs_timestamp(now),
                        }
                    },
                    "updateMask": {
                        "fieldPaths": ["status", "published_at", "version", "updated_at"]
                    },
                    "currentDocument": {
                        "updateTime": listing_update_time
                    }
                }));
            }
        }

        client
            .commit(
                &self.database,
                &CommitRequest {
                    writes,
                    transaction: None,
                },
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        Ok(())
    }

    async fn update_stone_listing_status(
        &self,
        listing_key: &str,
        next_status: &str,
    ) -> Result<(), StoreError> {
        let listing_key = listing_key.trim();
        if listing_key.is_empty() {
            return Ok(());
        }

        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let listing_doc_name = format!("{}/stone_listings/{}", self.parent, listing_key);
        let mut listing_doc = client
            .get_document(&listing_doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        let current_status = read_string_field(&listing_doc.fields, "status");
        let current_published_at = read_timestamp_field(&listing_doc.fields, "published_at");
        let should_update = if next_status.eq_ignore_ascii_case("published") {
            stone_listing_should_restore_after_canceled_order(
                &current_status,
                current_published_at.is_none(),
            )
        } else {
            !current_status.trim().eq_ignore_ascii_case(next_status)
        };
        if !should_update {
            return Ok(());
        }

        let now = Utc::now();
        let listing_version = read_int_field(&listing_doc.fields, "version").unwrap_or_default();
        listing_doc
            .fields
            .insert("status".to_owned(), fs_string(next_status));
        if next_status.eq_ignore_ascii_case("published") {
            let published_at = current_published_at.unwrap_or_else(|| now.clone());
            listing_doc
                .fields
                .insert("published_at".to_owned(), fs_timestamp(published_at));
        }
        listing_doc
            .fields
            .insert("version".to_owned(), fs_int(listing_version + 1));
        listing_doc
            .fields
            .insert("updated_at".to_owned(), fs_timestamp(now));

        client
            .patch_document(
                &listing_doc_name,
                &listing_doc,
                &PatchDocumentOptions::default(),
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        Ok(())
    }

    async fn reconcile_order_from_stripe_checkout_session(
        &self,
        order_id: &str,
        snapshot: &StripeCheckoutSessionSnapshot,
    ) -> Result<Option<OrderStatusResult>, StoreError> {
        let Some(event) = stripe_event_from_checkout_session_snapshot(snapshot, order_id) else {
            return Ok(None);
        };

        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;
        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);
        let mut order_doc = match client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(None),
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        let payment = read_map_field(&order_doc.fields, "payment");
        let stored_session_id = read_string_field(&payment, "checkout_session_id");
        if stored_session_id.trim() != snapshot.session_id {
            eprintln!(
                "stripe checkout reconciliation skipped because session id did not match order_id={} stored_session_id={} session_id={}",
                order_id, stored_session_id, snapshot.session_id
            );
            return Ok(None);
        }

        let now = Utc::now();
        let mutation = apply_stripe_webhook_to_order_fields(&mut order_doc.fields, &event, now);

        client
            .patch_document(
                &order_doc_name,
                &order_doc,
                &PatchDocumentOptions::default(),
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        if let Some(listing_status) = mutation.listing_status {
            let listing = read_map_field(&order_doc.fields, "listing");
            let listing_key = read_string_field(&listing, "key");
            if !listing_key.is_empty() {
                self.update_stone_listing_status(&listing_key, listing_status)
                    .await?;
            }
        }

        client
            .create_document(
                &format!("{}/orders/{}", self.parent, order_id),
                "events",
                &Document {
                    name: None,
                    fields: mutation.event_fields,
                    ..Document::default()
                },
                &CreateDocumentOptions::default(),
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        Ok(Some(order_status_result_from_fields(
            order_id,
            &order_doc.fields,
        )))
    }

    async fn try_idempotent_replay(
        &self,
        client: &FirebaseFirestoreClient,
        idempotency_doc_name: &str,
        request_hash: &str,
    ) -> Result<Option<CreateOrderResult>, StoreError> {
        let idempotency_doc = match client
            .get_document(idempotency_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => return Ok(None),
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        let existing_hash = read_string_field(&idempotency_doc.fields, "request_hash");
        if existing_hash != request_hash {
            return Err(StoreError::IdempotencyConflict);
        }

        let order_id = read_string_field(&idempotency_doc.fields, "order_id");
        if order_id.is_empty() {
            return Err(StoreError::Internal(anyhow!(
                "idempotency key exists but order_id is empty"
            )));
        }

        let order_doc_name = format!("{}/orders/{}", self.parent, order_id);
        let order_doc = client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        let status = read_string_field(&order_doc.fields, "status");
        let order_no = read_string_field(&order_doc.fields, "order_no");

        let pricing = read_map_field(&order_doc.fields, "pricing");
        let payment = read_map_field(&order_doc.fields, "payment");
        let fulfillment = read_map_field(&order_doc.fields, "fulfillment");

        Ok(Some(CreateOrderResult {
            order_id,
            order_no,
            status,
            payment_status: read_string_field(&payment, "status"),
            fulfillment_status: read_string_field(&fulfillment, "status"),
            total: pricing_total(&pricing),
            currency: pricing_currency(&pricing),
            idempotent_replay: true,
        }))
    }

    async fn get_active_font(
        &self,
        client: &FirebaseFirestoreClient,
        key: &str,
    ) -> Result<Font, StoreError> {
        let doc_name = format!("{}/fonts/{}", self.parent, key);
        let doc = client
            .get_document(&doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| {
                if is_not_found(&err) {
                    StoreError::InvalidReference
                } else {
                    StoreError::Internal(anyhow!(err))
                }
            })?;
        if !read_bool_field(&doc.fields, "is_active").unwrap_or(false) {
            return Err(StoreError::InactiveReference);
        }

        Ok(Font {
            key: key.to_owned(),
            label: resolve_font_label_field(&doc.fields, key),
            font_family: read_string_field(&doc.fields, "font_family"),
            kanji_style: normalize_catalog_kanji_style(&read_string_field(
                &doc.fields,
                "kanji_style",
            ))
            .to_owned(),
            version: read_int_field(&doc.fields, "version").unwrap_or(1),
        })
    }

    async fn get_active_country(
        &self,
        client: &FirebaseFirestoreClient,
        code: &str,
    ) -> Result<Country, StoreError> {
        let doc_name = format!("{}/countries/{}", self.parent, code);
        let doc = client
            .get_document(&doc_name, &GetDocumentOptions::default())
            .await
            .map_err(|err| {
                if is_not_found(&err) {
                    StoreError::InvalidReference
                } else {
                    StoreError::Internal(anyhow!(err))
                }
            })?;
        if !read_bool_field(&doc.fields, "is_active").unwrap_or(false) {
            return Err(StoreError::InactiveReference);
        }

        Ok(Country {
            code: code.to_owned(),
            label_i18n: read_string_map_field(&doc.fields, "label_i18n"),
            shipping_fee_by_currency: country_shipping_fee_by_currency_from_fields(&doc.fields),
            version: read_int_field(&doc.fields, "version").unwrap_or(1),
        })
    }

    async fn process_stripe_webhook(
        &self,
        event: StripeWebhookEvent,
    ) -> Result<ProcessStripeWebhookResult, StoreError> {
        let normalized = normalize_webhook_event(event);
        if normalized.provider_event_id.is_empty() {
            return Err(StoreError::Internal(anyhow!(
                "provider event id is required"
            )));
        }

        let client = self
            .firestore_client()
            .await
            .map_err(StoreError::Internal)?;

        let webhook_doc_name = format!(
            "{}/payment_webhook_events/{}",
            self.parent, normalized.provider_event_id
        );

        if let Ok(existing) = client
            .get_document(&webhook_doc_name, &GetDocumentOptions::default())
            .await
        {
            if read_bool_field(&existing.fields, "processed").unwrap_or(false) {
                return Ok(ProcessStripeWebhookResult {
                    processed: false,
                    already_processed: true,
                });
            }
        }

        let now = Utc::now();

        let mut webhook_fields = btree_from_pairs(vec![
            ("provider", fs_string("stripe")),
            ("event_type", fs_string(normalized.event_type.clone())),
            ("order_id", fs_string(normalized.order_id.clone())),
            ("processed", fs_bool(false)),
            ("created_at", fs_timestamp(now)),
            ("expire_at", fs_timestamp(now + Duration::days(90))),
        ]);

        upsert_named_document(
            &client,
            &self.parent,
            "payment_webhook_events",
            &normalized.provider_event_id,
            webhook_fields.clone(),
        )
        .await
        .map_err(|err| StoreError::Internal(err.into()))?;

        if normalized.order_id.is_empty() {
            webhook_fields.insert("processed".to_owned(), fs_bool(true));
            upsert_named_document(
                &client,
                &self.parent,
                "payment_webhook_events",
                &normalized.provider_event_id,
                webhook_fields,
            )
            .await
            .map_err(|err| StoreError::Internal(err.into()))?;

            return Ok(ProcessStripeWebhookResult {
                processed: true,
                already_processed: false,
            });
        }

        let order_doc_name = format!("{}/orders/{}", self.parent, normalized.order_id);
        let mut order_doc = match client
            .get_document(&order_doc_name, &GetDocumentOptions::default())
            .await
        {
            Ok(doc) => doc,
            Err(err) if is_not_found(&err) => {
                webhook_fields.insert("processed".to_owned(), fs_bool(true));
                upsert_named_document(
                    &client,
                    &self.parent,
                    "payment_webhook_events",
                    &normalized.provider_event_id,
                    webhook_fields,
                )
                .await
                .map_err(|e| StoreError::Internal(e.into()))?;
                return Ok(ProcessStripeWebhookResult {
                    processed: true,
                    already_processed: false,
                });
            }
            Err(err) => return Err(StoreError::Internal(anyhow!(err))),
        };

        let mutation =
            apply_stripe_webhook_to_order_fields(&mut order_doc.fields, &normalized, now);

        client
            .patch_document(
                &order_doc_name,
                &order_doc,
                &PatchDocumentOptions::default(),
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        if let Some(listing_status) = mutation.listing_status {
            let listing = read_map_field(&order_doc.fields, "listing");
            let listing_key = read_string_field(&listing, "key");
            if !listing_key.is_empty() {
                self.update_stone_listing_status(&listing_key, listing_status)
                    .await?;
            }
        }

        client
            .create_document(
                &format!("{}/orders/{}", self.parent, normalized.order_id),
                "events",
                &Document {
                    name: None,
                    fields: mutation.event_fields,
                    ..Document::default()
                },
                &CreateDocumentOptions::default(),
            )
            .await
            .map_err(|err| StoreError::Internal(anyhow!(err)))?;

        webhook_fields.insert("processed".to_owned(), fs_bool(true));
        upsert_named_document(
            &client,
            &self.parent,
            "payment_webhook_events",
            &normalized.provider_event_id,
            webhook_fields,
        )
        .await
        .map_err(|err| StoreError::Internal(err.into()))?;

        Ok(ProcessStripeWebhookResult {
            processed: true,
            already_processed: false,
        })
    }
}

fn build_order_fields(
    input: &CreateOrderInput,
    font: &Font,
    listing: Option<&StoneListing>,
    country: &Country,
    order_no: &str,
    subtotal: i64,
    shipping: i64,
    tax: i64,
    discount: i64,
    total: i64,
    currency: &str,
    now: DateTime<Utc>,
) -> BTreeMap<String, JsonValue> {
    let pricing_currency =
        normalize_currency_code(currency).unwrap_or_else(|| DEFAULT_CURRENCY.to_owned());

    let mut seal_fields = btree_from_pairs(vec![
        ("line1", fs_string(input.seal.line1.clone())),
        ("line2", fs_string(input.seal.line2.clone())),
        ("shape", fs_string(input.seal.shape.clone())),
        ("font_key", fs_string(font.key.clone())),
        ("font_label", fs_string(font.label.clone())),
        ("font_version", fs_int(font.version)),
    ]);

    if let Some(ai_generation_id) = &input.seal.ai_generation_id {
        seal_fields.insert(
            "ai_generation_id".to_owned(),
            fs_string(ai_generation_id.clone()),
        );
    }
    if let Some(ai_variant_id) = &input.seal.ai_variant_id {
        seal_fields.insert("ai_variant_id".to_owned(), fs_string(ai_variant_id.clone()));
    }
    if let Some(preview_image) = &input.seal.preview_image {
        let mut preview_fields = btree_from_pairs(vec![(
            "storage_path",
            fs_string(preview_image.storage_path.clone()),
        )]);
        if let Some(download_url) = &preview_image.download_url {
            preview_fields.insert("download_url".to_owned(), fs_string(download_url.clone()));
        }
        if let Some(width) = preview_image.width {
            preview_fields.insert("width".to_owned(), fs_int(width));
        }
        if let Some(height) = preview_image.height {
            preview_fields.insert("height".to_owned(), fs_int(height));
        }
        seal_fields.insert("preview_image".to_owned(), fs_map(preview_fields));
    }
    if let Some(style) = &input.seal.style {
        let mut style_fields = btree_from_pairs(vec![
            ("name", fs_string(style.name.clone())),
            ("stroke_weight", fs_string(style.stroke_weight.clone())),
            ("balance", fs_string(style.balance.clone())),
        ]);
        if let Some(prompt_summary) = &style.prompt_summary {
            style_fields.insert(
                "prompt_summary".to_owned(),
                fs_string(prompt_summary.clone()),
            );
        }
        seal_fields.insert("style".to_owned(), fs_map(style_fields));
    }

    let mut fields = btree_from_pairs(vec![
        ("order_no", fs_string(order_no)),
        ("channel", fs_string(input.channel.clone())),
        ("locale", fs_string(input.locale.clone())),
        ("status", fs_string("pending_payment")),
        ("status_updated_at", fs_timestamp(now)),
        ("seal", fs_map(seal_fields)),
        (
            "listing",
            fs_map(stone_listing_snapshot_fields(listing, subtotal)),
        ),
        (
            "shipping",
            fs_map(btree_from_pairs(vec![
                ("country_code", fs_string(country.code.clone())),
                ("country_label_i18n", fs_string_map(&country.label_i18n)),
                ("country_version", fs_int(country.version)),
                ("fee", fs_int(shipping)),
                (
                    "recipient_name",
                    fs_string(input.shipping.recipient_name.clone()),
                ),
                ("phone", fs_string(input.shipping.phone.clone())),
                ("postal_code", fs_string(input.shipping.postal_code.clone())),
                ("state", fs_string(input.shipping.state.clone())),
                ("city", fs_string(input.shipping.city.clone())),
                (
                    "address_line1",
                    fs_string(input.shipping.address_line1.clone()),
                ),
                (
                    "address_line2",
                    fs_string(input.shipping.address_line2.clone()),
                ),
            ])),
        ),
        (
            "contact",
            fs_map(btree_from_pairs(vec![
                ("email", fs_string(input.contact.email.clone())),
                (
                    "preferred_locale",
                    fs_string(input.contact.preferred_locale.clone()),
                ),
            ])),
        ),
        (
            "pricing",
            fs_map(btree_from_pairs(vec![
                ("subtotal", fs_int(subtotal)),
                ("shipping", fs_int(shipping)),
                ("tax", fs_int(tax)),
                ("discount", fs_int(discount)),
                ("total", fs_int(total)),
                ("currency", fs_string(pricing_currency)),
            ])),
        ),
        (
            "payment",
            fs_map(btree_from_pairs(vec![
                ("provider", fs_string("stripe")),
                ("status", fs_string("unpaid")),
            ])),
        ),
        (
            "fulfillment",
            fs_map(btree_from_pairs(vec![("status", fs_string("pending"))])),
        ),
        ("idempotency_key", fs_string(input.idempotency_key.clone())),
        ("terms_agreed", fs_bool(input.terms_agreed)),
        ("created_at", fs_timestamp(now)),
        ("updated_at", fs_timestamp(now)),
    ]);

    if let Some(customer_confirmation) = &input.customer_confirmation {
        fields.insert(
            "customer_confirmation".to_owned(),
            fs_map(btree_from_pairs(vec![
                (
                    "kanji_and_design",
                    fs_bool(customer_confirmation.kanji_and_design),
                ),
                (
                    "custom_made_policy",
                    fs_bool(customer_confirmation.custom_made_policy),
                ),
                ("confirmed_at", fs_timestamp(now)),
                (
                    "confirmed_seal_text",
                    fs_string(customer_confirmation.confirmed_seal_text.clone()),
                ),
            ])),
        );
    }

    if let Some(order_note) = &input.order_note {
        fields.insert("order_note".to_owned(), fs_string(order_note.clone()));
    }

    fields
}

async fn upsert_named_document(
    client: &FirebaseFirestoreClient,
    parent: &str,
    collection: &str,
    doc_id: &str,
    fields: BTreeMap<String, JsonValue>,
) -> Result<Document> {
    let name = format!("{parent}/{collection}/{doc_id}");
    let document = Document {
        name: Some(name.clone()),
        fields,
        ..Document::default()
    };

    match client
        .patch_document(&name, &document, &PatchDocumentOptions::default())
        .await
    {
        Ok(doc) => Ok(doc),
        Err(err) if is_not_found(&err) => client
            .create_document(
                parent,
                collection,
                &document,
                &CreateDocumentOptions {
                    document_id: Some(doc_id.to_owned()),
                    ..CreateDocumentOptions::default()
                },
            )
            .await
            .map_err(anyhow::Error::from),
        Err(err) => Err(anyhow!(err)),
    }
}

fn default_public_config() -> PublicConfig {
    public_config_from_registry()
        .expect("checked-in language registry should generate public config")
}

fn public_config_from_registry() -> Result<PublicConfig> {
    let cfg = language_registry::public_config_from_registry()?;
    Ok(PublicConfig {
        supported_locales: cfg.supported_locales,
        default_locale: cfg.default_locale,
        default_currency: cfg.default_currency,
        currency_by_locale: cfg.currency_by_locale,
    })
}

fn normalize_public_config(cfg: PublicConfig) -> PublicConfig {
    let registry_config = default_public_config();
    let mut normalized = Vec::with_capacity(cfg.supported_locales.len());
    let mut seen = HashSet::with_capacity(cfg.supported_locales.len());
    for locale in cfg.supported_locales {
        let value = normalize_locale_route_code(&locale).unwrap_or_default();
        if value.is_empty() || seen.contains(&value) {
            continue;
        }
        seen.insert(value.clone());
        normalized.push(value);
    }

    if normalized.is_empty() {
        normalized = registry_config.supported_locales.clone();
    }

    let mut default_locale = normalize_locale_route_code(&cfg.default_locale).unwrap_or_default();
    if default_locale.is_empty() || !contains(&normalized, &default_locale) {
        default_locale = if contains(&normalized, &registry_config.default_locale) {
            registry_config.default_locale.clone()
        } else {
            normalized
                .first()
                .cloned()
                .unwrap_or_else(|| DEFAULT_LOCALE.to_owned())
        };
    }

    let default_currency = language_registry::normalize_currency_code(&cfg.default_currency)
        .unwrap_or_else(|| registry_config.default_currency.clone());

    let mut currency_by_locale = HashMap::new();
    for (locale, currency) in cfg.currency_by_locale {
        let locale = normalize_locale_route_code(&locale).unwrap_or_default();
        if locale.is_empty() || !contains(&normalized, &locale) {
            continue;
        }
        let Some(currency) = language_registry::normalize_currency_code(&currency) else {
            continue;
        };
        currency_by_locale.insert(locale, currency);
    }
    for locale in &normalized {
        let fallback_currency = registry_config
            .currency_by_locale
            .get(locale)
            .cloned()
            .unwrap_or_else(|| default_currency.clone());
        currency_by_locale
            .entry(locale.clone())
            .or_insert(fallback_currency);
    }

    PublicConfig {
        supported_locales: normalized,
        default_locale,
        default_currency,
        currency_by_locale,
    }
}

fn normalize_currency_code(raw: &str) -> Option<String> {
    language_registry::normalize_currency_code(raw)
}

fn normalize_locale_route_code(raw: &str) -> Option<String> {
    language_registry::route_code_for_locale(raw)
}

fn requested_locale_route_code(raw: Option<String>, default_locale: &str) -> Option<String> {
    raw.as_deref()
        .map(normalize_locale_route_code)
        .unwrap_or_else(|| Some(default_locale.to_owned()))
}

fn resolve_currency_for_locale(cfg: &PublicConfig, locale: &str) -> String {
    if let Some(value) = lookup_locale(&cfg.currency_by_locale, locale)
        && let Some(currency) = normalize_currency_code(&value)
    {
        return currency;
    }
    if let Some(value) = lookup_locale(&cfg.currency_by_locale, &cfg.default_locale)
        && let Some(currency) = normalize_currency_code(&value)
    {
        return currency;
    }
    cfg.default_currency.clone()
}

fn resolve_pricing_currency(cfg: &PublicConfig, locale: &str) -> String {
    resolve_currency_for_locale(cfg, locale)
}

fn material_price_for_currency(material: &Material, currency: &str) -> i64 {
    resolve_amount_for_currency(&material.price_by_currency, currency)
}

fn stone_listing_price_for_currency(listing: &StoneListing, currency: &str) -> i64 {
    resolve_amount_for_currency(&listing.price_by_currency, currency)
}

fn stone_listing_is_published(status: &str) -> bool {
    status.trim().eq_ignore_ascii_case("published")
}

fn stone_listing_is_orderable(is_active: bool, status: &str) -> bool {
    is_active && stone_listing_is_published(status)
}

fn stone_listing_is_app_visible(is_active: bool, status: &str) -> bool {
    if !is_active {
        return false;
    }
    matches!(
        status.trim().to_ascii_lowercase().as_str(),
        "published" | "reserved" | "sold"
    )
}

fn country_shipping_fee_for_currency(country: &Country, currency: &str) -> i64 {
    resolve_amount_for_currency(&country.shipping_fee_by_currency, currency)
}

fn stone_listing_shape_matches(listing: &StoneListing, shape: &str) -> bool {
    let normalized_shape = shape.trim().to_lowercase();
    listing.facets.stone_shape.trim().to_lowercase() == normalized_shape
}

fn resolve_amount_for_currency(values: &HashMap<String, i64>, currency: &str) -> i64 {
    if let Some(amount) =
        normalize_currency_code(currency).and_then(|code| values.get(&code).copied())
    {
        return amount.max(0);
    }

    if let Some(amount) = values.get("USD").copied() {
        return amount.max(0);
    }
    if let Some(amount) = values.get("JPY").copied() {
        return amount.max(0);
    }

    let mut fallback_keys = values.keys().cloned().collect::<Vec<_>>();
    fallback_keys.sort();
    for key in fallback_keys {
        if let Some(amount) = values.get(&key).copied() {
            return amount.max(0);
        }
    }

    0
}

fn material_price_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    normalize_currency_amount_map(read_int_map_field(data, "price_by_currency"))
}

fn stone_listing_price_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    normalize_currency_amount_map(read_int_map_field(data, "price_by_currency"))
}

fn country_shipping_fee_by_currency_from_fields(
    data: &BTreeMap<String, JsonValue>,
) -> HashMap<String, i64> {
    normalize_currency_amount_map(read_int_map_field(data, "shipping_fee_by_currency"))
}

fn normalize_currency_amount_map(values: HashMap<String, i64>) -> HashMap<String, i64> {
    let mut normalized = HashMap::new();
    for (key, amount) in values {
        if let Some(currency) = normalize_currency_code(&key) {
            normalized.insert(currency, amount.max(0));
        }
    }
    normalized
}

fn normalize_create_order_input(input: CreateOrderInput) -> CreateOrderInput {
    let listing_id = input
        .listing_id
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned);
    let order_note = normalize_optional_string(input.order_note);

    CreateOrderInput {
        channel: input.channel.trim().to_lowercase(),
        locale: normalize_locale_route_code(&input.locale)
            .unwrap_or_else(|| input.locale.trim().to_lowercase()),
        idempotency_key: input.idempotency_key.trim().to_owned(),
        terms_agreed: input.terms_agreed,
        seal: SealInput {
            line1: input.seal.line1.trim().to_owned(),
            line2: input.seal.line2.trim().to_owned(),
            shape: input.seal.shape.trim().to_lowercase(),
            font_key: input.seal.font_key.trim().to_owned(),
            ai_generation_id: normalize_optional_string(input.seal.ai_generation_id),
            ai_variant_id: normalize_optional_string(input.seal.ai_variant_id),
            preview_image: input
                .seal
                .preview_image
                .map(|preview_image| SealPreviewImageInput {
                    storage_path: preview_image.storage_path.trim().to_owned(),
                    download_url: normalize_optional_string(preview_image.download_url),
                    width: preview_image.width,
                    height: preview_image.height,
                }),
            style: input.seal.style.map(|style| SealStyleInput {
                name: style.name.trim().to_lowercase(),
                stroke_weight: style.stroke_weight.trim().to_lowercase(),
                balance: style.balance.trim().to_lowercase(),
                prompt_summary: normalize_optional_string(style.prompt_summary),
            }),
        },
        listing_id,
        shipping: ShippingInput {
            country_code: input.shipping.country_code.trim().to_uppercase(),
            recipient_name: input.shipping.recipient_name.trim().to_owned(),
            phone: input.shipping.phone.trim().to_owned(),
            postal_code: input.shipping.postal_code.trim().to_owned(),
            state: input.shipping.state.trim().to_owned(),
            city: input.shipping.city.trim().to_owned(),
            address_line1: input.shipping.address_line1.trim().to_owned(),
            address_line2: input.shipping.address_line2.trim().to_owned(),
        },
        contact: ContactInput {
            email: input.contact.email.trim().to_owned(),
            preferred_locale: normalize_locale_route_code(&input.contact.preferred_locale)
                .unwrap_or_else(|| input.contact.preferred_locale.trim().to_lowercase()),
        },
        customer_confirmation: input.customer_confirmation.map(|customer_confirmation| {
            CustomerConfirmationInput {
                kanji_and_design: customer_confirmation.kanji_and_design,
                custom_made_policy: customer_confirmation.custom_made_policy,
                confirmed_at: customer_confirmation.confirmed_at,
                confirmed_seal_text: customer_confirmation.confirmed_seal_text.trim().to_owned(),
            }
        }),
        order_note,
    }
}

fn normalize_optional_string(value: Option<String>) -> Option<String> {
    value
        .as_deref()
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

fn hash_order_request(input: &CreateOrderInput) -> Result<String> {
    let mut seal = serde_json::Map::new();
    seal.insert("line1".to_owned(), json!(input.seal.line1.clone()));
    seal.insert("line2".to_owned(), json!(input.seal.line2.clone()));
    seal.insert("shape".to_owned(), json!(input.seal.shape.clone()));
    seal.insert("font_key".to_owned(), json!(input.seal.font_key.clone()));
    if let Some(ai_generation_id) = &input.seal.ai_generation_id {
        seal.insert("ai_generation_id".to_owned(), json!(ai_generation_id));
    }
    if let Some(ai_variant_id) = &input.seal.ai_variant_id {
        seal.insert("ai_variant_id".to_owned(), json!(ai_variant_id));
    }
    if let Some(preview_image) = &input.seal.preview_image {
        seal.insert(
            "preview_image".to_owned(),
            json!({
                "storage_path": preview_image.storage_path,
                "download_url": preview_image.download_url,
                "width": preview_image.width,
                "height": preview_image.height,
            }),
        );
    }
    if let Some(style) = &input.seal.style {
        seal.insert(
            "style".to_owned(),
            json!({
                "name": style.name,
                "stroke_weight": style.stroke_weight,
                "balance": style.balance,
                "prompt_summary": style.prompt_summary,
            }),
        );
    }

    let mut payload = serde_json::Map::new();
    payload.insert("channel".to_owned(), json!(input.channel.clone()));
    payload.insert("locale".to_owned(), json!(input.locale.clone()));
    payload.insert(
        "idempotency_key".to_owned(),
        json!(input.idempotency_key.clone()),
    );
    payload.insert("terms_agreed".to_owned(), json!(input.terms_agreed));
    payload.insert("seal".to_owned(), JsonValue::Object(seal));
    payload.insert("listing_id".to_owned(), json!(input.listing_id.clone()));
    payload.insert(
        "shipping".to_owned(),
        json!({
            "country_code": input.shipping.country_code,
            "recipient_name": input.shipping.recipient_name,
            "phone": input.shipping.phone,
            "postal_code": input.shipping.postal_code,
            "state": input.shipping.state,
            "city": input.shipping.city,
            "address_line1": input.shipping.address_line1,
            "address_line2": input.shipping.address_line2,
        }),
    );
    payload.insert(
        "contact".to_owned(),
        json!({
            "email": input.contact.email,
            "preferred_locale": input.contact.preferred_locale,
        }),
    );
    if let Some(customer_confirmation) = &input.customer_confirmation {
        payload.insert(
            "customer_confirmation".to_owned(),
            json!({
                "kanji_and_design": customer_confirmation.kanji_and_design,
                "custom_made_policy": customer_confirmation.custom_made_policy,
                "confirmed_at": customer_confirmation
                    .confirmed_at
                    .to_rfc3339_opts(SecondsFormat::Secs, true),
                "confirmed_seal_text": customer_confirmation.confirmed_seal_text,
            }),
        );
    }
    if let Some(order_note) = &input.order_note {
        payload.insert("order_note".to_owned(), json!(order_note));
    }

    let payload = serde_json::to_vec(&JsonValue::Object(payload))?;

    let mut hasher = Sha256::new();
    hasher.update(payload);
    Ok(hex::encode(hasher.finalize()))
}

fn validate_generate_seal_designs_request(
    request: GenerateSealDesignsRequest,
) -> Result<GenerateSealDesignsInput> {
    let input_name = request.input_name.trim().to_owned();
    if input_name.is_empty() {
        bail!("input_name is required");
    }
    if input_name.chars().count() > 120 {
        bail!("input_name must be 120 characters or fewer");
    }

    let rules = validate_seal_generation_rules(request.generation_rules.unwrap_or_default())?;

    let kanji = request.kanji.trim().to_owned();
    if kanji.is_empty() {
        bail!("kanji is required");
    }
    if kanji.chars().any(char::is_whitespace) {
        bail!("kanji must not contain whitespace");
    }
    let kanji_count = kanji.chars().count();
    if kanji_count > rules.max_characters {
        bail!("kanji must be {} characters or fewer", rules.max_characters);
    }
    if !kanji.chars().all(is_cjk_han_character) {
        bail!("kanji must contain only CJK Han characters");
    }

    let shape = parse_seal_shape(&request.shape)?;
    let style = parse_seal_style_name(&request.style)?;
    let stroke_weight = parse_seal_stroke_weight(&request.stroke_weight)?;
    let balance = parse_seal_balance(&request.balance)?;

    let variant_count = request
        .variant_count
        .unwrap_or(DEFAULT_SEAL_DESIGN_VARIANT_COUNT);
    if variant_count != DEFAULT_SEAL_DESIGN_VARIANT_COUNT {
        bail!(
            "variant_count must be {}",
            DEFAULT_SEAL_DESIGN_VARIANT_COUNT
        );
    }

    let attempt_number = request.attempt_number.unwrap_or(1);
    if attempt_number == 0 || attempt_number > MAX_SEAL_GENERATION_ATTEMPTS {
        bail!(
            "attempt_number must be between 1 and {}",
            MAX_SEAL_GENERATION_ATTEMPTS
        );
    }

    let previous_recipe_limit =
        DEFAULT_SEAL_DESIGN_VARIANT_COUNT * (MAX_SEAL_GENERATION_ATTEMPTS - 1);
    let previous_recipe_dtos = request.previous_recipes.unwrap_or_default();
    if previous_recipe_dtos.len() > previous_recipe_limit {
        bail!(
            "previous_recipes must contain {} recipes or fewer",
            previous_recipe_limit
        );
    }
    let previous_recipes = previous_recipe_dtos
        .into_iter()
        .map(validate_seal_design_recipe)
        .collect::<Result<Vec<_>>>()?;

    Ok(GenerateSealDesignsInput {
        input_name,
        kanji,
        shape,
        style,
        stroke_weight,
        balance,
        variant_count,
        attempt_number,
        previous_recipes,
        generation_rules: rules,
    })
}

fn validate_seal_generation_rules(
    request: SealGenerationRulesRequest,
) -> Result<SealGenerationRules> {
    let defaults = SealGenerationRules::default();
    let rules = SealGenerationRules {
        max_characters: request.max_characters.unwrap_or(defaults.max_characters),
        avoid_complex_characters: request
            .avoid_complex_characters
            .unwrap_or(defaults.avoid_complex_characters),
        engraving_friendly: request
            .engraving_friendly
            .unwrap_or(defaults.engraving_friendly),
        avoid_thin_lines: request
            .avoid_thin_lines
            .unwrap_or(defaults.avoid_thin_lines),
        avoid_decorative_details: request
            .avoid_decorative_details
            .unwrap_or(defaults.avoid_decorative_details),
        plain_background: request
            .plain_background
            .unwrap_or(defaults.plain_background),
    };

    if rules.max_characters == 0 || rules.max_characters > 2 {
        bail!("generation_rules.max_characters must be in range 1-2");
    }
    if !rules.avoid_complex_characters {
        bail!("generation_rules.avoid_complex_characters must be true");
    }
    if !rules.engraving_friendly {
        bail!("generation_rules.engraving_friendly must be true");
    }
    if !rules.avoid_thin_lines {
        bail!("generation_rules.avoid_thin_lines must be true");
    }
    if !rules.avoid_decorative_details {
        bail!("generation_rules.avoid_decorative_details must be true");
    }
    if !rules.plain_background {
        bail!("generation_rules.plain_background must be true");
    }

    Ok(rules)
}

fn parse_seal_shape(raw: &str) -> Result<SealShape> {
    match raw.trim().to_lowercase().as_str() {
        "square" => Ok(SealShape::Square),
        "round" => Ok(SealShape::Round),
        _ => bail!("shape must be one of square, round"),
    }
}

fn parse_seal_style_name(raw: &str) -> Result<SealStyleName> {
    match raw.trim().to_lowercase().as_str() {
        "traditional" => Ok(SealStyleName::Traditional),
        "elegant" => Ok(SealStyleName::Elegant),
        "soft" => Ok(SealStyleName::Soft),
        "bold" => Ok(SealStyleName::Bold),
        _ => bail!("style must be one of traditional, elegant, soft, bold"),
    }
}

fn parse_seal_stroke_weight(raw: &str) -> Result<SealStrokeWeight> {
    match raw.trim().to_lowercase().as_str() {
        "standard" => Ok(SealStrokeWeight::Standard),
        "bold" => Ok(SealStrokeWeight::Bold),
        _ => bail!("stroke_weight must be one of standard, bold"),
    }
}

fn parse_seal_balance(raw: &str) -> Result<SealBalance> {
    match raw.trim().to_lowercase().as_str() {
        "airy" => Ok(SealBalance::Airy),
        "balanced" => Ok(SealBalance::Balanced),
        "dense" => Ok(SealBalance::Dense),
        _ => bail!("balance must be one of airy, balanced, dense"),
    }
}

#[allow(dead_code)]
fn validate_seal_design_recipe_variant(
    dto: SealDesignRecipeVariantDto,
) -> Result<SealDesignRecipeVariant> {
    let label = dto.label.trim().to_owned();
    if label.is_empty() {
        bail!("label is required");
    }
    if label.chars().count() > 80 {
        bail!("label must be 80 characters or fewer");
    }

    Ok(SealDesignRecipeVariant {
        label,
        recipe: validate_seal_design_recipe(dto.recipe)?,
    })
}

#[allow(dead_code)]
fn validate_seal_design_recipe(dto: SealDesignRecipeDto) -> Result<SealDesignRecipe> {
    Ok(SealDesignRecipe {
        font_profile: parse_seal_recipe_font_profile(&dto.font_profile)?,
        impression: parse_seal_recipe_impression(&dto.impression)?,
        weight: parse_seal_recipe_weight(&dto.weight)?,
        spacing: parse_seal_recipe_spacing(&dto.spacing)?,
        texture: parse_seal_recipe_texture(&dto.texture)?,
        frame: parse_seal_recipe_frame(&dto.frame)?,
    })
}

#[allow(dead_code)]
fn parse_seal_recipe_font_profile(raw: &str) -> Result<SealRecipeFontProfile> {
    match raw.trim().to_lowercase().as_str() {
        "formal_serif" => Ok(SealRecipeFontProfile::FormalSerif),
        "soft_sans" => Ok(SealRecipeFontProfile::SoftSans),
        "bold_brush" => Ok(SealRecipeFontProfile::BoldBrush),
        "classic_seal" => Ok(SealRecipeFontProfile::ClassicSeal),
        _ => bail!("font_profile must be one of formal_serif, soft_sans, bold_brush, classic_seal"),
    }
}

#[allow(dead_code)]
fn parse_seal_recipe_impression(raw: &str) -> Result<SealRecipeImpression> {
    match raw.trim().to_lowercase().as_str() {
        "traditional" => Ok(SealRecipeImpression::Traditional),
        "elegant" => Ok(SealRecipeImpression::Elegant),
        "soft" => Ok(SealRecipeImpression::Soft),
        "bold" => Ok(SealRecipeImpression::Bold),
        _ => bail!("impression must be one of traditional, elegant, soft, bold"),
    }
}

#[allow(dead_code)]
fn parse_seal_recipe_weight(raw: &str) -> Result<SealRecipeWeight> {
    match raw.trim().to_lowercase().as_str() {
        "standard" => Ok(SealRecipeWeight::Standard),
        "bold" => Ok(SealRecipeWeight::Bold),
        _ => bail!("weight must be one of standard, bold"),
    }
}

#[allow(dead_code)]
fn parse_seal_recipe_spacing(raw: &str) -> Result<SealRecipeSpacing> {
    match raw.trim().to_lowercase().as_str() {
        "airy" => Ok(SealRecipeSpacing::Airy),
        "balanced" => Ok(SealRecipeSpacing::Balanced),
        "dense" => Ok(SealRecipeSpacing::Dense),
        _ => bail!("spacing must be one of airy, balanced, dense"),
    }
}

#[allow(dead_code)]
fn parse_seal_recipe_texture(raw: &str) -> Result<SealRecipeTexture> {
    match raw.trim().to_lowercase().as_str() {
        "none" => Ok(SealRecipeTexture::None),
        "subtle_ink" => Ok(SealRecipeTexture::SubtleInk),
        "soft_bleed" => Ok(SealRecipeTexture::SoftBleed),
        _ => bail!("texture must be one of none, subtle_ink, soft_bleed"),
    }
}

#[allow(dead_code)]
fn parse_seal_recipe_frame(raw: &str) -> Result<SealRecipeFrame> {
    match raw.trim().to_lowercase().as_str() {
        "square_standard" => Ok(SealRecipeFrame::SquareStandard),
        "round_standard" => Ok(SealRecipeFrame::RoundStandard),
        _ => bail!("frame must be one of square_standard, round_standard"),
    }
}

fn validate_generate_kanji_candidates_request(
    request: GenerateKanjiCandidatesRequest,
) -> Result<GenerateKanjiCandidatesInput> {
    let real_name = request.real_name.trim().to_owned();
    if real_name.is_empty() {
        bail!("real_name is required");
    }
    if real_name.chars().count() > 120 {
        bail!("real_name must be 120 characters or fewer");
    }

    let reason_language_raw = request.reason_language.as_deref().unwrap_or("en").trim();
    if reason_language_raw.is_empty() {
        bail!("reason_language is required");
    }
    if reason_language_raw.chars().count() > 32 {
        bail!("reason_language must be 32 characters or fewer");
    }
    let reason_language_resolution =
        language_registry::reason_language_for_locale(Some(reason_language_raw));

    let gender = parse_candidate_gender(request.gender.as_deref())?;
    let kanji_style = parse_kanji_style(request.kanji_style.as_deref())?;

    let count = request.count.unwrap_or(DEFAULT_KANJI_CANDIDATE_COUNT);
    if count == 0 || count > MAX_KANJI_CANDIDATE_COUNT {
        bail!("count must be in range 1-{}", MAX_KANJI_CANDIDATE_COUNT);
    }

    Ok(GenerateKanjiCandidatesInput {
        real_name,
        reason_language: reason_language_resolution.prompt_language,
        reason_language_requested: reason_language_resolution.requested_locale,
        reason_language_route_code: reason_language_resolution.route_code,
        reason_language_fallback_reason: reason_language_resolution.fallback_reason,
        gender,
        kanji_style,
        count,
    })
}

fn parse_candidate_gender(raw: Option<&str>) -> Result<CandidateGender> {
    let normalized = raw.unwrap_or("unspecified").trim().to_lowercase();
    if normalized.is_empty() || normalized == "unspecified" || normalized == "none" {
        return Ok(CandidateGender::Unspecified);
    }
    if normalized == "male" {
        return Ok(CandidateGender::Male);
    }
    if normalized == "female" {
        return Ok(CandidateGender::Female);
    }
    bail!("gender must be one of unspecified, male, female")
}

fn parse_kanji_style(raw: Option<&str>) -> Result<KanjiStyle> {
    let normalized = raw.unwrap_or("japanese").trim().to_lowercase();
    if normalized.is_empty()
        || normalized == "japanese"
        || normalized == "japan"
        || normalized == "jp"
    {
        return Ok(KanjiStyle::Japanese);
    }
    if normalized == "chinese" || normalized == "china" || normalized == "cn" {
        return Ok(KanjiStyle::Chinese);
    }
    if normalized == "taiwanese" || normalized == "taiwan" || normalized == "tw" {
        return Ok(KanjiStyle::Taiwanese);
    }
    bail!("kanji_style must be one of japanese, chinese, taiwanese")
}

fn stone_listing_from_fields(key: &str, data: &BTreeMap<String, JsonValue>) -> StoneListing {
    let facets_data = read_map_field(data, "facets");
    let title_i18n = read_string_map_field(data, "title_i18n");
    let description_i18n = read_string_map_field(data, "description_i18n");
    let story_i18n = read_string_map_field(data, "story_i18n");

    StoneListing {
        key: key.to_owned(),
        listing_code: first_non_empty(&[
            Some(read_string_field(data, "listing_code")),
            Some(key.to_owned()),
        ])
        .unwrap_or_else(|| key.to_owned()),
        material_key: read_string_field(data, "material_key"),
        size: read_string_field(data, "size"),
        title_i18n,
        description_i18n,
        story_i18n,
        facets: StoneListingFacets {
            color_family: read_string_field(&facets_data, "color_family"),
            color_tags: read_string_array_field(&facets_data, "color_tags"),
            pattern_primary: read_string_field(&facets_data, "pattern_primary"),
            pattern_tags: read_string_array_field(&facets_data, "pattern_tags"),
            stone_shape: read_string_field(&facets_data, "stone_shape"),
            translucency: read_string_field(&facets_data, "translucency"),
        },
        photos: read_material_photos(data),
        price_by_currency: stone_listing_price_by_currency_from_fields(data),
        status: first_non_empty(&[
            Some(read_string_field(data, "status")),
            Some("published".to_owned()),
        ])
        .unwrap_or_else(|| "published".to_owned()),
        is_active: read_bool_field(data, "is_active").unwrap_or(true),
        sort_order: read_int_field(data, "sort_order").unwrap_or_default(),
        version: read_int_field(data, "version").unwrap_or(1),
    }
}

fn stone_listing_response(
    storage_assets_bucket: &str,
    requested_locale: &str,
    default_locale: &str,
    pricing_currency: &str,
    facet_tag_labels: &FacetTagLabels,
    material_labels: &HashMap<String, String>,
    listing: StoneListing,
) -> JsonValue {
    let price = stone_listing_price_for_currency(&listing, pricing_currency);
    let price_response = stone_listing_price_response(price, pricing_currency);
    let title = resolve_localized(&listing.title_i18n, requested_locale, default_locale);
    let description =
        resolve_localized(&listing.description_i18n, requested_locale, default_locale);
    let story = resolve_localized(&listing.story_i18n, requested_locale, default_locale);
    let material_label = material_labels
        .get(&listing.material_key)
        .cloned()
        .unwrap_or_else(|| listing.material_key.clone());
    let color_tag_labels = facet_tag_labels.resolve_list("color", &listing.facets.color_tags);
    let pattern_tag_labels = facet_tag_labels.resolve_list("pattern", &listing.facets.pattern_tags);
    let is_orderable = stone_listing_is_orderable(listing.is_active, &listing.status);
    let photos = listing
        .photos
        .into_iter()
        .map(|photo| {
            json!({
                "asset_id": photo.asset_id,
                "asset_url": make_asset_url(storage_assets_bucket, &photo.storage_path),
                "storage_path": photo.storage_path,
                "alt": resolve_localized(&photo.alt_i18n, requested_locale, default_locale),
                "sort_order": photo.sort_order,
                "is_primary": photo.is_primary,
                "width": photo.width,
                "height": photo.height,
            })
        })
        .collect::<Vec<_>>();

    json!({
        "id": listing.key.clone(),
        "key": listing.key,
        "code": listing.listing_code.clone(),
        "listing_code": listing.listing_code,
        "material_key": listing.material_key.clone(),
        "material_label": material_label,
        "size": listing.size,
        "title": title,
        "description": description,
        "story": story,
        "facets": {
            "color_family": listing.facets.color_family,
            "color_tags": listing.facets.color_tags,
            "pattern_primary": listing.facets.pattern_primary,
            "pattern_tags": listing.facets.pattern_tags,
            "stone_shape": listing.facets.stone_shape,
            "translucency": listing.facets.translucency,
        },
        "price": price_response,
        "price_amount": price,
        "currency": pricing_currency,
        "price_by_currency": listing.price_by_currency,
        "color_tag_labels": color_tag_labels,
        "pattern_tag_labels": pattern_tag_labels,
        "status": listing.status,
        "is_active": listing.is_active,
        "is_orderable": is_orderable,
        "sort_order": listing.sort_order,
        "version": listing.version,
        "photos": photos,
    })
}

fn stone_listing_price_response(amount: i64, currency: &str) -> JsonValue {
    let currency = normalize_currency_code(currency).unwrap_or_else(|| DEFAULT_CURRENCY.to_owned());
    let amount = amount.max(0);
    let display = format_price_display(&currency, amount);
    json!({
        "amount": amount,
        "currency": currency,
        "display": display,
    })
}

fn format_price_display(currency: &str, amount: i64) -> String {
    let normalized = currency.trim().to_ascii_uppercase();
    let amount = amount.max(0);
    match normalized.as_str() {
        "JPY" => format!("JPY {}", format_with_grouping(amount)),
        "USD" => {
            let dollars = amount / 100;
            let cents = amount % 100;
            format!("USD {}.{cents:02}", format_with_grouping(dollars))
        }
        _ => format!("{normalized} {}", format_with_grouping(amount)),
    }
}

fn format_with_grouping(value: i64) -> String {
    let digits = value.abs().to_string();
    let mut output = String::new();
    for (index, ch) in digits.chars().enumerate() {
        if index > 0 && (digits.len() - index) % 3 == 0 {
            output.push(',');
        }
        output.push(ch);
    }
    output
}

fn stone_listing_snapshot_fields(
    listing: Option<&StoneListing>,
    unit_price: i64,
) -> BTreeMap<String, JsonValue> {
    let Some(listing) = listing else {
        return BTreeMap::new();
    };

    let mut fields = btree_from_pairs(vec![
        ("key", fs_string(listing.key.clone())),
        ("listing_code", fs_string(listing.listing_code.clone())),
        ("material_key", fs_string(listing.material_key.clone())),
        ("size", fs_string(listing.size.clone())),
        ("title_i18n", fs_string_map(&listing.title_i18n)),
        ("description_i18n", fs_string_map(&listing.description_i18n)),
        ("story_i18n", fs_string_map(&listing.story_i18n)),
        ("unit_price", fs_int(unit_price)),
        ("version", fs_int(listing.version)),
    ]);

    fields.insert(
        "facets".to_owned(),
        fs_map(btree_from_pairs(vec![
            (
                "color_family",
                fs_string(listing.facets.color_family.clone()),
            ),
            ("color_tags", fs_string_array(&listing.facets.color_tags)),
            (
                "pattern_primary",
                fs_string(listing.facets.pattern_primary.clone()),
            ),
            (
                "pattern_tags",
                fs_string_array(&listing.facets.pattern_tags),
            ),
            ("stone_shape", fs_string(listing.facets.stone_shape.clone())),
            (
                "translucency",
                fs_string(listing.facets.translucency.clone()),
            ),
        ])),
    );
    fields.insert("photos".to_owned(), fs_material_photos(&listing.photos));
    fields.insert(
        "price_by_currency".to_owned(),
        fs_int_map(&listing.price_by_currency),
    );
    fields.insert("status".to_owned(), fs_string(listing.status.clone()));
    fields.insert("is_active".to_owned(), fs_bool(listing.is_active));
    fields.insert("sort_order".to_owned(), fs_int(listing.sort_order));

    fields
}

async fn generate_seal_design_recipes_with_gemini(
    state: &AppState,
    input: &GenerateSealDesignsInput,
) -> Result<Vec<SealDesignRecipeVariant>> {
    let request_body = build_seal_designs_request_body(input);

    let endpoint = format!(
        "{}/v1beta/models/{}:generateContent",
        state.gemini.base_url.trim_end_matches('/'),
        state.gemini.model.trim()
    );

    let response = state
        .http_client
        .post(endpoint)
        .query(&[("key", state.gemini.api_key.as_str())])
        .json(&request_body)
        .send()
        .await
        .context("failed to call gemini generateContent")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response
            .text()
            .await
            .unwrap_or_else(|_| "<unable to read response body>".to_owned());
        bail!("gemini request failed status={} body={}", status, body);
    }

    let payload = response
        .json::<JsonValue>()
        .await
        .context("failed to parse gemini response payload")?;

    let text = extract_gemini_response_text(&payload)
        .ok_or_else(|| anyhow!("gemini response did not include text"))?;

    parse_seal_design_recipes_from_gemini_text(&text, input.variant_count)
}

fn build_seal_designs_request_body(input: &GenerateSealDesignsInput) -> JsonValue {
    json!({
        "contents": [
            {
                "role": "user",
                "parts": [
                    {
                        "text": build_seal_designs_prompt(input),
                    }
                ]
            }
        ],
        "generationConfig": {
            "temperature": 0.35,
            "responseMimeType": "application/json",
            "responseJsonSchema": build_seal_designs_response_schema(input),
            "thinkingConfig": {
                "thinkingBudget": DEFAULT_GEMINI_THINKING_BUDGET,
            }
        }
    })
}

fn build_seal_designs_response_schema(input: &GenerateSealDesignsInput) -> JsonValue {
    let variant_count = input.variant_count;
    let required_frame = input.shape.recipe_frame().as_str();

    json!({
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "variants": {
                "type": "array",
                "minItems": variant_count,
                "maxItems": variant_count,
                "items": {
                    "type": "object",
                    "additionalProperties": false,
                    "properties": {
                        "label": {
                            "type": "string",
                            "minLength": 1,
                            "maxLength": 80,
                            "description": "A concise display label for one generated seal design variant.",
                        },
                        "recipe": {
                            "type": "object",
                            "additionalProperties": false,
                            "properties": {
                                "font_profile": {
                                    "type": "string",
                                    "enum": ["formal_serif", "soft_sans", "bold_brush", "classic_seal"],
                                },
                                "impression": {
                                    "type": "string",
                                    "enum": ["traditional", "elegant", "soft", "bold"],
                                },
                                "weight": {
                                    "type": "string",
                                    "enum": ["standard", "bold"],
                                },
                                "spacing": {
                                    "type": "string",
                                    "enum": ["airy", "balanced", "dense"],
                                },
                                "texture": {
                                    "type": "string",
                                    "enum": ["none", "subtle_ink", "soft_bleed"],
                                },
                                "frame": {
                                    "type": "string",
                                    "enum": [required_frame],
                                },
                            },
                            "required": [
                                "font_profile",
                                "impression",
                                "weight",
                                "spacing",
                                "texture",
                                "frame"
                            ],
                        },
                    },
                    "required": ["label", "recipe"],
                },
            },
        },
        "required": ["variants"],
    })
}

fn seal_previous_recipes_prompt(input: &GenerateSealDesignsInput) -> String {
    if input.previous_recipes.is_empty() {
        if input.attempt_number <= 1 {
            return String::new();
        }
        return format!(
            "Regeneration attempt: {}. Favor a fresh visual direction even though no previous recipe metadata was provided.",
            input.attempt_number
        );
    }

    let previous_signatures = input
        .previous_recipes
        .iter()
        .map(|recipe| {
            format!(
                "{}+{}+{}+{}+{}+{}",
                recipe.font_profile.as_str(),
                recipe.impression.as_str(),
                recipe.weight.as_str(),
                recipe.spacing.as_str(),
                recipe.texture.as_str(),
                recipe.frame.as_str()
            )
        })
        .collect::<Vec<_>>()
        .join("; ");
    let previous_visual_signatures = input
        .previous_recipes
        .iter()
        .map(|recipe| seal_recipe_visual_signature(*recipe))
        .collect::<Vec<_>>()
        .join("; ");
    let previous_font_profiles = input
        .previous_recipes
        .iter()
        .map(|recipe| recipe.font_profile.as_str())
        .collect::<Vec<_>>()
        .join(", ");

    format!(
        "Regeneration attempt: {}.\n\
Avoid these previously shown full recipe combinations: {}.\n\
Do not reuse these previous visual render signatures: {}.\n\
Prefer different font_profile values from these previous candidates when possible: {}.\n\
Every new variant must change at least two of font_profile, impression, weight, spacing, or texture compared with every previous recipe.",
        input.attempt_number,
        previous_signatures,
        previous_visual_signatures,
        previous_font_profiles
    )
}

fn ensure_fresh_seal_design_recipes(
    input: &GenerateSealDesignsInput,
    recipe_variants: Vec<SealDesignRecipeVariant>,
) -> Vec<SealDesignRecipeVariant> {
    let frame = input.shape.recipe_frame();
    let mut used_visual_signatures = input
        .previous_recipes
        .iter()
        .map(|recipe| seal_recipe_visual_signature(*recipe))
        .collect::<HashSet<_>>();

    recipe_variants
        .into_iter()
        .map(|mut variant| {
            let current_signature = seal_recipe_visual_signature(variant.recipe);
            if used_visual_signatures.contains(&current_signature) {
                if let Some((font_profile, spacing)) =
                    next_unused_seal_recipe_visual(frame, &used_visual_signatures)
                {
                    variant.recipe.font_profile = font_profile;
                    variant.recipe.spacing = spacing;
                    variant.recipe.frame = frame;
                }
            }

            used_visual_signatures.insert(seal_recipe_visual_signature(variant.recipe));
            variant
        })
        .collect()
}

fn next_unused_seal_recipe_visual(
    frame: SealRecipeFrame,
    used_visual_signatures: &HashSet<String>,
) -> Option<(SealRecipeFontProfile, SealRecipeSpacing)> {
    for font_profile in [
        SealRecipeFontProfile::FormalSerif,
        SealRecipeFontProfile::SoftSans,
        SealRecipeFontProfile::BoldBrush,
        SealRecipeFontProfile::ClassicSeal,
    ] {
        for spacing in [
            SealRecipeSpacing::Airy,
            SealRecipeSpacing::Balanced,
            SealRecipeSpacing::Dense,
        ] {
            let signature = seal_recipe_visual_signature(SealDesignRecipe {
                font_profile,
                impression: SealRecipeImpression::Traditional,
                weight: SealRecipeWeight::Standard,
                spacing,
                texture: SealRecipeTexture::None,
                frame,
            });
            if !used_visual_signatures.contains(&signature) {
                return Some((font_profile, spacing));
            }
        }
    }

    None
}

fn seal_recipe_visual_signature(recipe: SealDesignRecipe) -> String {
    format!(
        "{}+{}+{}",
        recipe.font_profile.as_str(),
        recipe.spacing.as_str(),
        recipe.frame.as_str()
    )
}

fn build_seal_designs_prompt(input: &GenerateSealDesignsInput) -> String {
    format!(
        "Generate exactly {} unique hanko seal design recipe variants.\n\
Input name: \"{}\"\n\
Selected kanji, for context only and never to replace: \"{}\"\n\
Shape: {}\n\
Style: {}\n\
Stroke weight: {}\n\
Balance: {}\n\
{}\n\
{}\n\
{}\n\
{}\n\
{}\n\
Rules:\n\
- Return structured JSON recipes only. Do not generate, describe, embed, or request final seal images, image bytes, base64, SVG, glyph outlines, storage paths, URLs, colors, or backgrounds; the API renderer owns the final fixed size, red ink color (#9D1F22), pure white background, and no background pattern behavior.\n\
- Do not change, replace, simplify, transliterate, or invent kanji glyphs; the API renderer will draw the selected Unicode kanji with real fonts.\n\
- Each recipe must use only these enum values:\n\
  font_profile: formal_serif, soft_sans, bold_brush, classic_seal\n\
  impression: traditional, elegant, soft, bold\n\
  weight: standard, bold\n\
  spacing: airy, balanced, dense\n\
  texture: none, subtle_ink, soft_bleed\n\
  frame: {} only\n\
- The recipe must fit {} or fewer CJK Han characters and remain engraving-friendly.\n\
- Return exactly {} recipe variants and no extra keys.\n\
- Return only JSON in this exact shape: {{\"variants\":[{{\"label\":\"Formal balanced\",\"recipe\":{{\"font_profile\":\"formal_serif\",\"impression\":\"traditional\",\"weight\":\"standard\",\"spacing\":\"balanced\",\"texture\":\"none\",\"frame\":\"{}\"}}}}]}}",
        input.variant_count,
        input.input_name,
        input.kanji,
        input.shape.as_str(),
        input.style.as_str(),
        input.stroke_weight.as_str(),
        input.balance.as_str(),
        input.shape.prompt_instruction(),
        input.style.prompt_instruction(),
        input.stroke_weight.prompt_instruction(),
        input.balance.prompt_instruction(),
        seal_previous_recipes_prompt(input),
        input.shape.recipe_frame().as_str(),
        input.generation_rules.max_characters,
        input.variant_count,
        input.shape.recipe_frame().as_str()
    )
}

fn build_seal_design_variants(
    storage_assets_bucket: &str,
    request_id: &str,
    recipe_variants: &[SealDesignRecipeVariant],
) -> Vec<SealDesignVariant> {
    recipe_variants
        .iter()
        .enumerate()
        .map(|(index, recipe_variant)| {
            let id = format!("seal_variant_{:03}", index + 1);
            let storage_path = format!("seal_designs/{request_id}/{id}.png");
            SealDesignVariant {
                id,
                download_url: make_asset_url(storage_assets_bucket, &storage_path),
                storage_path,
                label: recipe_variant.label.clone(),
                recipe: recipe_variant.recipe,
                width: SEAL_DESIGN_IMAGE_SIZE,
                height: SEAL_DESIGN_IMAGE_SIZE,
            }
        })
        .collect()
}

fn seal_design_recipe_to_json(recipe: SealDesignRecipe) -> JsonValue {
    json!({
        "font_profile": recipe.font_profile.as_str(),
        "impression": recipe.impression.as_str(),
        "weight": recipe.weight.as_str(),
        "spacing": recipe.spacing.as_str(),
        "texture": recipe.texture.as_str(),
        "frame": recipe.frame.as_str(),
    })
}

fn generate_seal_design_images_with_renderer(
    input: &GenerateSealDesignsInput,
    variants: &[SealDesignVariant],
) -> Result<Vec<GeneratedSealDesignImage>> {
    let mut images = Vec::with_capacity(variants.len());
    for variant in variants {
        let image = render_seal_design_image(input, variant)
            .with_context(|| format!("failed to render image for {}", variant.id))?;
        images.push(image);
    }
    Ok(images)
}

fn render_seal_design_image(
    input: &GenerateSealDesignsInput,
    variant: &SealDesignVariant,
) -> Result<GeneratedSealDesignImage> {
    let shape = seal_shape_for_recipe_frame(variant.recipe.frame);
    if shape != input.shape {
        bail!(
            "recipe frame {} does not match requested shape {}",
            variant.recipe.frame.as_str(),
            input.shape.as_str()
        );
    }

    let rendered = seal_renderer::render_fixed_rule_seal_png_with_spacing(
        &input.kanji,
        variant.recipe.font_profile,
        shape,
        variant.recipe.spacing,
    )?;

    if rendered.width as usize != SEAL_DESIGN_IMAGE_SIZE
        || rendered.height as usize != SEAL_DESIGN_IMAGE_SIZE
    {
        bail!(
            "rendered seal image size {}x{} did not match expected {}x{}",
            rendered.width,
            rendered.height,
            SEAL_DESIGN_IMAGE_SIZE,
            SEAL_DESIGN_IMAGE_SIZE
        );
    }

    let image = GeneratedSealDesignImage {
        content_type: rendered.content_type.to_owned(),
        bytes: rendered.bytes,
    };
    validate_generated_seal_image(&image)?;
    Ok(image)
}

fn seal_shape_for_recipe_frame(frame: SealRecipeFrame) -> SealShape {
    match frame {
        SealRecipeFrame::SquareStandard => SealShape::Square,
        SealRecipeFrame::RoundStandard => SealShape::Round,
    }
}

fn validate_generated_seal_image(image: &GeneratedSealDesignImage) -> Result<()> {
    let content_type = image.content_type.trim().to_ascii_lowercase();
    if content_type != "image/png" {
        bail!("generated seal image must be image/png");
    }

    const PNG_SIGNATURE: &[u8; 8] = b"\x89PNG\r\n\x1a\n";
    if image.bytes.len() < PNG_SIGNATURE.len() || !image.bytes.starts_with(PNG_SIGNATURE) {
        bail!("generated seal image payload is not a png");
    }

    Ok(())
}

#[derive(Debug, Clone, Copy)]
struct ArtworkBounds {
    left: u32,
    top: u32,
    right: u32,
    bottom: u32,
}

impl ArtworkBounds {
    fn width(self) -> u32 {
        self.right - self.left + 1
    }

    fn height(self) -> u32 {
        self.bottom - self.top + 1
    }
}

fn normalize_generated_seal_image(
    image: GeneratedSealDesignImage,
    shape: SealShape,
) -> Result<GeneratedSealDesignImage> {
    validate_generated_seal_image(&image)?;

    let source = image::load_from_memory_with_format(&image.bytes, ImageFormat::Png)
        .context("failed to decode generated seal png")?
        .to_rgba8();
    let bounds = find_artwork_bounds(&source)
        .ok_or_else(|| anyhow!("generated seal image has no artwork"))?;
    let content_bounds = pad_artwork_bounds(
        find_kanji_artwork_bounds(&source, bounds).unwrap_or(bounds),
        source.width(),
        source.height(),
    );

    let target_size = SEAL_DESIGN_IMAGE_SIZE as u32;
    let target_extent = target_size
        .saturating_sub((SEAL_DESIGN_FRAME_INSET + SEAL_DESIGN_FRAME_STROKE_WIDTH + 82) * 2)
        .max(1);
    let scale = target_extent as f32 / content_bounds.width().max(content_bounds.height()) as f32;
    let target_width = ((content_bounds.width() as f32 * scale).round() as u32).max(1);
    let target_height = ((content_bounds.height() as f32 * scale).round() as u32).max(1);
    let target_left = (target_size - target_width) / 2;
    let target_top = (target_size - target_height) / 2;

    let crop = image::imageops::crop_imm(
        &source,
        content_bounds.left,
        content_bounds.top,
        content_bounds.width(),
        content_bounds.height(),
    )
    .to_image();
    let resized = image::imageops::resize(&crop, target_width, target_height, FilterType::Lanczos3);
    let mut canvas = RgbaImage::from_pixel(
        target_size,
        target_size,
        Rgba([
            SEAL_DESIGN_BACKGROUND[0],
            SEAL_DESIGN_BACKGROUND[1],
            SEAL_DESIGN_BACKGROUND[2],
            255,
        ]),
    );

    for (x, y, pixel) in resized.enumerate_pixels() {
        let strength = seal_artwork_strength(*pixel);
        if strength >= SEAL_DESIGN_PAINT_STRENGTH_THRESHOLD {
            paint_seal_red_pixel(&mut canvas, target_left + x, target_top + y, strength);
        }
    }
    draw_canonical_seal_frame(&mut canvas, shape);

    let mut bytes = Vec::new();
    DynamicImage::ImageRgba8(canvas)
        .write_to(&mut Cursor::new(&mut bytes), ImageFormat::Png)
        .context("failed to encode normalized seal png")?;

    Ok(GeneratedSealDesignImage {
        content_type: "image/png".to_owned(),
        bytes,
    })
}

fn find_artwork_bounds(image: &RgbaImage) -> Option<ArtworkBounds> {
    find_artwork_bounds_in_region(
        image,
        ArtworkBounds {
            left: 0,
            top: 0,
            right: image.width().saturating_sub(1),
            bottom: image.height().saturating_sub(1),
        },
        SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD,
    )
}

fn find_artwork_bounds_in_region(
    image: &RgbaImage,
    region: ArtworkBounds,
    min_strength: f32,
) -> Option<ArtworkBounds> {
    let mut bounds: Option<ArtworkBounds> = None;
    for y in region.top..=region.bottom {
        for x in region.left..=region.right {
            if seal_artwork_strength(*image.get_pixel(x, y)) < min_strength {
                continue;
            }
            bounds = Some(match bounds {
                Some(current) => ArtworkBounds {
                    left: current.left.min(x),
                    top: current.top.min(y),
                    right: current.right.max(x),
                    bottom: current.bottom.max(y),
                },
                None => ArtworkBounds {
                    left: x,
                    top: y,
                    right: x,
                    bottom: y,
                },
            });
        }
    }
    bounds
}

fn pad_artwork_bounds(bounds: ArtworkBounds, image_width: u32, image_height: u32) -> ArtworkBounds {
    let padding_x = ((bounds.width() as f32) * 0.04).round() as u32;
    let padding_y = ((bounds.height() as f32) * 0.04).round() as u32;
    ArtworkBounds {
        left: bounds.left.saturating_sub(padding_x.max(2)),
        top: bounds.top.saturating_sub(padding_y.max(2)),
        right: (bounds.right + padding_x.max(2)).min(image_width.saturating_sub(1)),
        bottom: (bounds.bottom + padding_y.max(2)).min(image_height.saturating_sub(1)),
    }
}

#[derive(Debug)]
struct FrameLineMask {
    rows: Vec<bool>,
    columns: Vec<bool>,
    has_frame_lines: bool,
}

fn find_kanji_artwork_bounds(image: &RgbaImage, bounds: ArtworkBounds) -> Option<ArtworkBounds> {
    let frame_mask = build_frame_line_mask(image, bounds);
    if !frame_mask.has_frame_lines {
        return Some(bounds);
    }

    let mut kanji_bounds: Option<ArtworkBounds> = None;
    for y in bounds.top..=bounds.bottom {
        let row_index = (y - bounds.top) as usize;
        for x in bounds.left..=bounds.right {
            let column_index = (x - bounds.left) as usize;
            if frame_mask.rows[row_index]
                || frame_mask.columns[column_index]
                || is_outer_frame_band(bounds, x, y)
                || seal_artwork_strength(*image.get_pixel(x, y))
                    < SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD
            {
                continue;
            }

            kanji_bounds = Some(match kanji_bounds {
                Some(current) => ArtworkBounds {
                    left: current.left.min(x),
                    top: current.top.min(y),
                    right: current.right.max(x),
                    bottom: current.bottom.max(y),
                },
                None => ArtworkBounds {
                    left: x,
                    top: y,
                    right: x,
                    bottom: y,
                },
            });
        }
    }

    kanji_bounds
}

fn build_frame_line_mask(image: &RgbaImage, bounds: ArtworkBounds) -> FrameLineMask {
    let width = bounds.width();
    let height = bounds.height();
    let row_threshold = ((width as f32) * 0.52).round() as u32;
    let column_threshold = ((height as f32) * 0.52).round() as u32;
    let edge_band_x = ((width as f32) * 0.18).round() as u32;
    let edge_band_y = ((height as f32) * 0.18).round() as u32;

    let mut rows = Vec::with_capacity(height as usize);
    let mut columns = Vec::with_capacity(width as usize);
    let mut dense_edge_rows = 0;
    let mut dense_edge_columns = 0;
    let mut dense_rows = 0;
    let mut dense_columns = 0;

    for y in bounds.top..=bounds.bottom {
        let artwork_count = (bounds.left..=bounds.right)
            .filter(|x| {
                seal_artwork_strength(*image.get_pixel(*x, y))
                    >= SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD
            })
            .count() as u32;
        let is_dense = artwork_count >= row_threshold.max(1);
        if is_dense {
            dense_rows += 1;
            if y < bounds.top + edge_band_y || y > bounds.bottom.saturating_sub(edge_band_y) {
                dense_edge_rows += 1;
            }
        }
        rows.push(is_dense);
    }

    for x in bounds.left..=bounds.right {
        let artwork_count = (bounds.top..=bounds.bottom)
            .filter(|y| {
                seal_artwork_strength(*image.get_pixel(x, *y))
                    >= SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD
            })
            .count() as u32;
        let is_dense = artwork_count >= column_threshold.max(1);
        if is_dense {
            dense_columns += 1;
            if x < bounds.left + edge_band_x || x > bounds.right.saturating_sub(edge_band_x) {
                dense_edge_columns += 1;
            }
        }
        columns.push(is_dense);
    }

    FrameLineMask {
        rows,
        columns,
        has_frame_lines: dense_edge_rows > 0
            || dense_edge_columns > 0
            || (dense_rows >= 2 && dense_columns >= 2),
    }
}

fn is_outer_frame_band(bounds: ArtworkBounds, x: u32, y: u32) -> bool {
    let horizontal_padding = ((bounds.width() as f32) * 0.08).round() as u32;
    let vertical_padding = ((bounds.height() as f32) * 0.08).round() as u32;
    x < bounds.left + horizontal_padding
        || x > bounds.right.saturating_sub(horizontal_padding)
        || y < bounds.top + vertical_padding
        || y > bounds.bottom.saturating_sub(vertical_padding)
}

fn seal_artwork_strength(pixel: Rgba<u8>) -> f32 {
    let [red, green, blue, alpha] = pixel.0;
    if alpha < 16 {
        return 0.0;
    }

    let luma = (0.299 * red as f32) + (0.587 * green as f32) + (0.114 * blue as f32);
    if luma > 205.0 {
        return 0.0;
    }

    let darkness = ((205.0 - luma) / 145.0).clamp(0.0, 1.0);
    let strength = darkness * (alpha as f32 / 255.0);
    if strength < SEAL_DESIGN_PAINT_STRENGTH_THRESHOLD {
        0.0
    } else {
        strength
    }
}

fn paint_seal_red_pixel(canvas: &mut RgbaImage, x: u32, y: u32, strength: f32) {
    if x >= canvas.width() || y >= canvas.height() {
        return;
    }
    let strength = strength.clamp(0.0, 1.0);
    let pixel = canvas.get_pixel_mut(x, y);
    let [base_red, base_green, base_blue, _] = pixel.0;
    *pixel = Rgba([
        blend_channel(SEAL_DESIGN_RED[0], base_red, strength),
        blend_channel(SEAL_DESIGN_RED[1], base_green, strength),
        blend_channel(SEAL_DESIGN_RED[2], base_blue, strength),
        255,
    ]);
}

fn blend_channel(foreground: u8, background: u8, strength: f32) -> u8 {
    ((foreground as f32 * strength) + (background as f32 * (1.0 - strength))).round() as u8
}

fn draw_canonical_seal_frame(canvas: &mut RgbaImage, shape: SealShape) {
    match shape {
        SealShape::Square => draw_canonical_square_frame(canvas),
        SealShape::Round => draw_canonical_round_frame(canvas),
    }
}

fn draw_canonical_square_frame(canvas: &mut RgbaImage) {
    let size = canvas.width();
    let left = SEAL_DESIGN_FRAME_INSET;
    let top = SEAL_DESIGN_FRAME_INSET;
    let right = size - SEAL_DESIGN_FRAME_INSET;
    let bottom = size - SEAL_DESIGN_FRAME_INSET;
    let stroke = SEAL_DESIGN_FRAME_STROKE_WIDTH;

    for y in top..bottom {
        for x in left..right {
            let in_stroke = x < left + stroke
                || x >= right - stroke
                || y < top + stroke
                || y >= bottom - stroke;
            if in_stroke {
                paint_seal_red_pixel(canvas, x, y, 1.0);
            }
        }
    }
}

fn draw_canonical_round_frame(canvas: &mut RgbaImage) {
    let size = canvas.width() as f32;
    let center = (size - 1.0) / 2.0;
    let stroke = SEAL_DESIGN_FRAME_STROKE_WIDTH as f32;
    let radius = ((canvas.width() - (SEAL_DESIGN_FRAME_INSET * 2)) as f32 / 2.0) - (stroke / 2.0);
    let inner = radius - (stroke / 2.0);
    let outer = radius + (stroke / 2.0);

    for y in 0..canvas.height() {
        for x in 0..canvas.width() {
            let dx = x as f32 - center;
            let dy = y as f32 - center;
            let distance = (dx * dx + dy * dy).sqrt();
            if distance >= inner && distance <= outer {
                paint_seal_red_pixel(canvas, x, y, 1.0);
            }
        }
    }
}

async fn upload_seal_design_images_to_storage(
    state: &AppState,
    variants: &[SealDesignVariant],
    images: &[GeneratedSealDesignImage],
) -> Result<()> {
    validate_seal_design_upload_inputs(variants, images)?;

    for (variant, image) in variants.iter().zip(images.iter()) {
        upload_storage_object(
            state,
            &state.storage_assets_bucket,
            &variant.storage_path,
            &image.content_type,
            &image.bytes,
        )
        .await
        .with_context(|| format!("failed to upload {}", variant.storage_path))?;
    }

    Ok(())
}

fn validate_seal_design_upload_inputs(
    variants: &[SealDesignVariant],
    images: &[GeneratedSealDesignImage],
) -> Result<()> {
    if variants.len() != images.len() {
        bail!(
            "seal design image count {} did not match variant count {}",
            images.len(),
            variants.len()
        );
    }
    Ok(())
}

async fn upload_storage_object(
    state: &AppState,
    bucket: &str,
    storage_path: &str,
    content_type: &str,
    bytes: &[u8],
) -> Result<()> {
    let endpoint = storage_upload_endpoint(bucket)?;
    let storage_path = normalize_storage_object_name(storage_path)?;
    let token = state
        .store
        .token_provider
        .token(&[STORAGE_SCOPE])
        .await
        .context("failed to obtain storage access token")?;

    let response = state
        .http_client
        .post(endpoint)
        .query(&[("uploadType", "media"), ("name", storage_path.as_str())])
        .bearer_auth(token.as_str())
        .header("Content-Type", content_type)
        .body(bytes.to_vec())
        .send()
        .await
        .context("failed to call storage upload API")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response
            .text()
            .await
            .unwrap_or_else(|_| "<unable to read response body>".to_owned());
        bail!("storage upload failed status={} body={}", status, body);
    }

    Ok(())
}

fn storage_upload_endpoint(bucket: &str) -> Result<String> {
    let bucket = bucket.trim().trim_matches('/');
    if bucket.is_empty() {
        bail!("storage bucket is required");
    }
    if bucket.contains('/') {
        bail!("storage bucket must not contain slash");
    }

    Ok(format!(
        "https://storage.googleapis.com/upload/storage/v1/b/{bucket}/o"
    ))
}

fn normalize_storage_object_name(storage_path: &str) -> Result<String> {
    let storage_path = storage_path.trim().trim_start_matches('/');
    if storage_path.is_empty() {
        bail!("storage object name is required");
    }
    if storage_path.contains("..") {
        bail!("storage object name must not contain parent traversal");
    }
    Ok(storage_path.to_owned())
}

async fn generate_kanji_candidates_with_gemini(
    state: &AppState,
    input: &GenerateKanjiCandidatesInput,
) -> Result<Vec<KanjiNameCandidate>> {
    let request_body = build_kanji_candidates_request_body(input);

    let endpoint = format!(
        "{}/v1beta/models/{}:generateContent",
        state.gemini.base_url.trim_end_matches('/'),
        state.gemini.model.trim()
    );

    let response = state
        .http_client
        .post(endpoint)
        .query(&[("key", state.gemini.api_key.as_str())])
        .json(&request_body)
        .send()
        .await
        .context("failed to call gemini generateContent")?;

    if !response.status().is_success() {
        let status = response.status();
        let body = response
            .text()
            .await
            .unwrap_or_else(|_| "<unable to read response body>".to_owned());
        bail!("gemini request failed status={} body={}", status, body);
    }

    let payload = response
        .json::<JsonValue>()
        .await
        .context("failed to parse gemini response payload")?;

    let text = extract_gemini_response_text(&payload)
        .ok_or_else(|| anyhow!("gemini response did not include text"))?;

    parse_kanji_candidates_from_gemini_text(&text, input.count)
}

fn build_kanji_candidates_request_body(input: &GenerateKanjiCandidatesInput) -> JsonValue {
    json!({
        "contents": [
            {
                "role": "user",
                "parts": [
                    {
                        "text": build_kanji_candidates_prompt(input),
                    }
                ]
            }
        ],
        "generationConfig": {
            "temperature": 0.4,
            "responseMimeType": "application/json",
            "responseJsonSchema": build_kanji_candidates_response_schema(input.count),
            "thinkingConfig": {
                "thinkingBudget": DEFAULT_GEMINI_THINKING_BUDGET,
            }
        }
    })
}

fn build_kanji_candidates_response_schema(max_count: usize) -> JsonValue {
    json!({
        "type": "object",
        "additionalProperties": false,
        "properties": {
            "candidates": {
                "type": "array",
                "minItems": 1,
                "maxItems": max_count,
                "items": {
                    "type": "object",
                    "additionalProperties": false,
                    "properties": {
                        "kanji": {
                            "type": "string",
                            "description": "1-2 CJK Han characters suitable for a seal.",
                        },
                        "reading": {
                            "type": "string",
                            "description": "Lowercase romaji for Japanese style or lowercase Hanyu Pinyin without tone marks for Chinese/Taiwanese styles.",
                        },
                        "meaning": {
                            "type": "string",
                            "description": "Concise meaning of the kanji candidate, written in the requested reason language.",
                        },
                        "impression": {
                            "type": "array",
                            "minItems": 2,
                            "maxItems": 4,
                            "items": {
                                "type": "string",
                            },
                            "description": "Short impression words for the candidate, written in the requested reason language.",
                        },
                        "reason": {
                            "type": "string",
                            "description": "Why this Kanji name was chosen.",
                        },
                        "character_count": {
                            "type": "integer",
                            "minimum": 1,
                            "maximum": 2,
                            "description": "Number of kanji characters.",
                        },
                        "stroke_complexity": {
                            "type": "string",
                            "enum": ["low", "medium", "high"],
                            "description": "Overall visual stroke complexity for engraving.",
                        },
                        "engraving_suitability": {
                            "type": "string",
                            "enum": ["high", "medium", "low"],
                            "description": "Suitability for small gemstone seal engraving.",
                        },
                    },
                    "required": [
                        "kanji",
                        "reading",
                        "meaning",
                        "impression",
                        "reason",
                        "character_count",
                        "stroke_complexity",
                        "engraving_suitability"
                    ],
                },
            },
        },
        "required": ["candidates"],
    })
}

fn build_kanji_candidates_prompt(input: &GenerateKanjiCandidatesInput) -> String {
    let gender_instruction = input.gender.prompt_instruction();
    let style_instruction = input.kanji_style.prompt_instruction();
    let reason_language = reason_language_label(&input.reason_language);
    format!(
        "Generate {} unique Kanji name candidates for a hanko seal.\n\
Input name: \"{}\"\n\
{}\n\
{}\n\
Think through the name fit internally before answering. Consider balance, seal suitability, stroke shape, and pronunciation, but do not reveal your chain of thought.\n\
For each candidate, return these fields:\n\
- kanji: 1-2 CJK Han characters suitable for a seal (no spaces)\n\
- reading: lowercase romaji for japanese style, lowercase Hanyu Pinyin without tone marks for chinese/taiwanese styles\n\
- meaning: concise literal or symbolic meaning, written in {}\n\
- impression: 2-4 short impression words, written in {}\n\
- reason: why this Kanji name was chosen, written in {}\n\
- character_count: number of kanji characters, 1 or 2\n\
- stroke_complexity: one of low, medium, high\n\
- engraving_suitability: one of high, medium, low\n\
Return only JSON (no markdown, no explanation) in this exact shape:\n\
{{\"candidates\":[{{\"kanji\":\"\",\"reading\":\"\",\"meaning\":\"\",\"impression\":[\"\",\"\"],\"reason\":\"\",\"character_count\":2,\"stroke_complexity\":\"medium\",\"engraving_suitability\":\"high\"}}]}}",
        input.count,
        input.real_name,
        gender_instruction,
        style_instruction,
        reason_language,
        reason_language,
        reason_language
    )
}

fn reason_language_label(raw: &str) -> &'static str {
    match raw.trim().to_lowercase().as_str() {
        "ja" | "ja-jp" | "japanese" => "Japanese",
        "en" | "en-us" | "english" => "English",
        _ => "English",
    }
}

fn extract_gemini_response_text(payload: &JsonValue) -> Option<String> {
    let candidates = payload.get("candidates")?.as_array()?;
    for candidate in candidates {
        let Some(parts) = candidate
            .get("content")
            .and_then(|content| content.get("parts"))
            .and_then(JsonValue::as_array)
        else {
            continue;
        };
        for part in parts {
            if let Some(text) = part.get("text").and_then(JsonValue::as_str) {
                let trimmed = text.trim();
                if !trimmed.is_empty() {
                    return Some(trimmed.to_owned());
                }
            }
        }
    }
    None
}

fn parse_seal_design_recipes_from_gemini_text(
    raw_text: &str,
    expected_count: usize,
) -> Result<Vec<SealDesignRecipeVariant>> {
    let parsed = parse_json_value_loose(raw_text)?;
    let dto = serde_json::from_value::<SealDesignRecipeVariantsDto>(parsed)
        .context("gemini recipe response JSON must match schema")?;
    if dto.variants.len() != expected_count {
        bail!(
            "gemini returned {} seal design recipes, expected {}",
            dto.variants.len(),
            expected_count
        );
    }

    dto.variants
        .into_iter()
        .map(validate_seal_design_recipe_variant)
        .collect()
}

fn parse_kanji_candidates_from_gemini_text(
    raw_text: &str,
    max_count: usize,
) -> Result<Vec<KanjiNameCandidate>> {
    let parsed = parse_json_value_loose(raw_text)?;
    let array = parsed
        .get("candidates")
        .and_then(JsonValue::as_array)
        .ok_or_else(|| anyhow!("gemini response JSON must contain candidates array"))?;

    let mut candidates = Vec::with_capacity(array.len());
    let mut seen = HashSet::new();
    for entry in array {
        let Some(candidate) = normalize_kanji_candidate(entry) else {
            continue;
        };
        if !seen.insert(candidate.kanji.clone()) {
            continue;
        }
        candidates.push(candidate);
        if candidates.len() >= max_count {
            break;
        }
    }

    Ok(candidates)
}

fn parse_json_value_loose(raw_text: &str) -> Result<JsonValue> {
    let trimmed = raw_text.trim();
    if trimmed.is_empty() {
        bail!("gemini response text is empty");
    }

    if let Ok(value) = serde_json::from_str::<JsonValue>(trimmed) {
        return Ok(value);
    }

    let content = trimmed
        .trim_start_matches("```json")
        .trim_start_matches("```")
        .trim_end_matches("```")
        .trim();
    if let Ok(value) = serde_json::from_str::<JsonValue>(content) {
        return Ok(value);
    }

    if let (Some(start), Some(end)) = (trimmed.find('{'), trimmed.rfind('}'))
        && start < end
    {
        let slice = &trimmed[start..=end];
        if let Ok(value) = serde_json::from_str::<JsonValue>(slice) {
            return Ok(value);
        }
    }

    bail!("failed to parse json from gemini response text")
}

fn normalize_kanji_candidate(value: &JsonValue) -> Option<KanjiNameCandidate> {
    let kanji = read_json_string(value, &["kanji", "name_kanji"]);
    if kanji.is_empty() {
        return None;
    }
    let kanji_count = kanji.chars().count();
    if kanji_count > 2
        || kanji.chars().any(char::is_whitespace)
        || !kanji.chars().all(is_cjk_han_character)
    {
        return None;
    }

    let reading = read_json_string(value, &["reading", "reading_romaji", "romaji"]);
    let meaning = read_json_string(value, &["meaning"]);
    let impression = read_json_string_list(value, &["impression", "impressions"]);
    let reason = read_json_string(value, &["reason"]);
    if reading.is_empty() || reason.is_empty() {
        return None;
    }
    let character_count = match read_json_usize(value, &["character_count"]) {
        Some(count) if (1..=2).contains(&count) => count,
        Some(_) => return None,
        None => kanji_count,
    };
    if character_count != kanji_count {
        return None;
    }
    let stroke_complexity = normalize_ai_quality_level(
        &read_json_string(value, &["stroke_complexity"]),
        &["low", "medium", "high"],
    )?;
    let engraving_suitability = normalize_ai_quality_level(
        &read_json_string(value, &["engraving_suitability"]),
        &["high", "medium", "low"],
    )?;

    Some(KanjiNameCandidate {
        kanji,
        reading,
        meaning,
        impression,
        reason,
        character_count,
        stroke_complexity,
        engraving_suitability,
    })
}

fn is_cjk_han_character(ch: char) -> bool {
    matches!(
        ch as u32,
        0x3400..=0x4dbf | 0x4e00..=0x9fff | 0xf900..=0xfaff
    )
}

fn normalize_ai_quality_level(raw: &str, allowed: &[&str]) -> Option<String> {
    let normalized = raw.trim().to_ascii_lowercase();
    if allowed.contains(&normalized.as_str()) {
        Some(normalized)
    } else {
        None
    }
}

fn read_json_string(value: &JsonValue, keys: &[&str]) -> String {
    for key in keys {
        if let Some(text) = value.get(*key).and_then(JsonValue::as_str) {
            let trimmed = text.trim();
            if !trimmed.is_empty() {
                return trimmed.to_owned();
            }
        }
    }
    String::new()
}

fn read_json_string_list(value: &JsonValue, keys: &[&str]) -> Vec<String> {
    for key in keys {
        if let Some(array) = value.get(*key).and_then(JsonValue::as_array) {
            let items = array
                .iter()
                .filter_map(JsonValue::as_str)
                .map(str::trim)
                .filter(|text| !text.is_empty())
                .map(str::to_owned)
                .collect::<Vec<_>>();
            if !items.is_empty() {
                return items;
            }
        }

        if let Some(text) = value.get(*key).and_then(JsonValue::as_str) {
            let trimmed = text.trim();
            if !trimmed.is_empty() {
                return vec![trimmed.to_owned()];
            }
        }
    }
    Vec::new()
}

fn read_json_usize(value: &JsonValue, keys: &[&str]) -> Option<usize> {
    for key in keys {
        if let Some(number) = value.get(*key).and_then(JsonValue::as_u64) {
            if let Ok(value) = usize::try_from(number) {
                return Some(value);
            }
        }
    }
    None
}

fn validate_create_stripe_checkout_session_request(
    request: CreateStripeCheckoutSessionRequest,
) -> Result<CreateStripeCheckoutSessionInput> {
    let order_id = request.order_id.trim().to_owned();
    if order_id.is_empty() {
        bail!("order_id is required");
    }

    let customer_email = request.customer_email.unwrap_or_default().trim().to_owned();
    if !customer_email.is_empty() && !is_valid_email(&customer_email) {
        bail!("customer_email must be valid");
    }

    Ok(CreateStripeCheckoutSessionInput {
        order_id,
        customer_email,
        return_to_app: request.return_to_app.unwrap_or(false),
    })
}

fn validate_lookup_order_request(request: LookupOrderRequest) -> Result<LookupOrderInput> {
    let order_no = request.order_no.trim().to_ascii_uppercase();
    if order_no.is_empty() {
        bail!("order_no is required");
    }
    if order_no.chars().count() > 64 {
        bail!("order_no must be 64 characters or fewer");
    }

    let email = request.email.trim().to_owned();
    if email.is_empty() {
        bail!("email is required");
    }
    if !is_valid_email(&email) {
        bail!("email must be valid");
    }
    if email.chars().count() > 254 {
        bail!("email must be 254 characters or fewer");
    }

    Ok(LookupOrderInput { order_no, email })
}

async fn create_stripe_checkout_session(
    state: &AppState,
    order: &OrderCheckoutContext,
    customer_email: &str,
    return_to_app: bool,
) -> Result<CreateStripeCheckoutSessionResult> {
    let stripe_api_key = state.stripe_api_key.trim();
    if stripe_api_key.is_empty() {
        bail!("stripe client is not configured");
    }

    let form = build_stripe_checkout_session_form(
        &state.stripe_checkout,
        order,
        customer_email,
        return_to_app,
    );

    let response = state
        .http_client
        .post(STRIPE_CHECKOUT_SESSIONS_URL)
        .bearer_auth(stripe_api_key)
        .header("Stripe-Version", STRIPE_CHECKOUT_API_VERSION)
        .header(
            "Idempotency-Key",
            format!("checkout_session_{}", order.order_id),
        )
        .form(&form)
        .send()
        .await
        .context("failed to request stripe checkout session")?;

    let status = response.status();
    let response_body = response
        .text()
        .await
        .unwrap_or_else(|_| "<unable to read response body>".to_owned());
    if !status.is_success() {
        bail!(
            "stripe checkout session request failed status={} body={}",
            status,
            response_body
        );
    }

    let payload: JsonValue = serde_json::from_str(&response_body)
        .context("failed to parse stripe checkout session response")?;

    let session_id = payload
        .get("id")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_owned();
    if session_id.is_empty() {
        bail!("stripe checkout session response is missing id");
    }
    let checkout_url = payload
        .get("url")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_owned();
    if checkout_url.is_empty() {
        bail!("stripe checkout session response is missing url");
    }

    let payment_intent_id = payload
        .get("payment_intent")
        .and_then(stripe_payment_intent_id)
        .unwrap_or_default();

    Ok(CreateStripeCheckoutSessionResult {
        session_id,
        checkout_url,
        payment_intent_id,
    })
}

async fn retrieve_stripe_checkout_session(
    state: &AppState,
    session_id: &str,
) -> Result<StripeCheckoutSessionSnapshot> {
    let session_id = session_id.trim();
    validate_stripe_checkout_session_id(session_id)?;

    let stripe_api_key = state.stripe_api_key.trim();
    if stripe_api_key.is_empty() {
        bail!("stripe client is not configured");
    }

    let response = state
        .http_client
        .get(format!("{STRIPE_CHECKOUT_SESSIONS_URL}/{session_id}"))
        .bearer_auth(stripe_api_key)
        .header("Stripe-Version", STRIPE_CHECKOUT_API_VERSION)
        .query(&[("expand[]", "payment_intent")])
        .send()
        .await
        .context("failed to retrieve stripe checkout session")?;

    let status = response.status();
    let response_body = response
        .text()
        .await
        .unwrap_or_else(|_| "<unable to read response body>".to_owned());
    if !status.is_success() {
        bail!(
            "stripe checkout session retrieve failed status={} body={}",
            status,
            response_body
        );
    }

    let payload: JsonValue = serde_json::from_str(&response_body)
        .context("failed to parse stripe checkout session retrieve response")?;
    parse_stripe_checkout_session_snapshot(&payload)
}

fn validate_stripe_checkout_session_id(session_id: &str) -> Result<()> {
    if session_id.is_empty() {
        bail!("stripe checkout session id is required");
    }
    if !session_id.starts_with("cs_")
        || !session_id
            .chars()
            .all(|ch| ch.is_ascii_alphanumeric() || ch == '_' || ch == '-')
    {
        bail!("stripe checkout session id is invalid");
    }
    Ok(())
}

fn parse_stripe_checkout_session_snapshot(
    payload: &JsonValue,
) -> Result<StripeCheckoutSessionSnapshot> {
    let session_id = payload
        .get("id")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_owned();
    validate_stripe_checkout_session_id(&session_id)?;

    let payment_status = payload
        .get("payment_status")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_ascii_lowercase();
    let session_status = payload
        .get("status")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_ascii_lowercase();
    let payment_intent_id = payload
        .get("payment_intent")
        .and_then(stripe_payment_intent_id)
        .unwrap_or_default();
    let order_id = stripe_order_id_from_object(payload);

    Ok(StripeCheckoutSessionSnapshot {
        session_id,
        payment_status,
        session_status,
        payment_intent_id,
        order_id,
    })
}

fn build_stripe_checkout_session_form(
    stripe_checkout: &StripeCheckoutConfig,
    order: &OrderCheckoutContext,
    customer_email: &str,
    return_to_app: bool,
) -> Vec<(String, String)> {
    let product_name = build_checkout_product_name(order);
    let product_description = build_checkout_product_description(order);
    let return_locale = checkout_return_locale(order);
    let checkout_currency = stripe_checkout_currency(&order.currency);
    let success_base_url = if return_to_app {
        &stripe_checkout.app_success_url
    } else {
        &stripe_checkout.success_url
    };
    let cancel_base_url = if return_to_app {
        &stripe_checkout.app_cancel_url
    } else {
        &stripe_checkout.cancel_url
    };

    let mut success_params = vec![
        ("checkout", "success"),
        ("order_id", order.order_id.as_str()),
        ("lang", return_locale.as_str()),
    ];
    let mut cancel_params = vec![
        ("checkout", "cancel"),
        ("order_id", order.order_id.as_str()),
        ("lang", return_locale.as_str()),
    ];
    if return_to_app {
        success_params.push(("return_to", "app"));
        cancel_params.push(("return_to", "app"));
    }
    let success_url = append_query_params(success_base_url, &success_params);
    let cancel_url = append_query_params(cancel_base_url, &cancel_params);

    let mut form = vec![
        ("mode".to_owned(), "payment".to_owned()),
        ("success_url".to_owned(), success_url),
        ("cancel_url".to_owned(), cancel_url),
        ("line_items[0][quantity]".to_owned(), "1".to_owned()),
        (
            "line_items[0][price_data][currency]".to_owned(),
            checkout_currency,
        ),
        (
            "line_items[0][price_data][unit_amount]".to_owned(),
            order.total.to_string(),
        ),
        (
            "line_items[0][price_data][product_data][name]".to_owned(),
            product_name,
        ),
        (
            "line_items[0][price_data][product_data][description]".to_owned(),
            product_description,
        ),
        ("metadata[order_id]".to_owned(), order.order_id.clone()),
        (
            "payment_intent_data[metadata][order_id]".to_owned(),
            order.order_id.clone(),
        ),
        ("expand[0]".to_owned(), "payment_intent".to_owned()),
    ];

    if return_to_app {
        form.push(("origin_context".to_owned(), "mobile_app".to_owned()));
    }
    if !customer_email.trim().is_empty() {
        let customer_email = customer_email.trim().to_owned();
        form.push(("customer_email".to_owned(), customer_email.clone()));
        form.push((
            "payment_intent_data[receipt_email]".to_owned(),
            customer_email,
        ));
    }
    if let Some(shipping) = build_payment_intent_shipping(order) {
        push_stripe_shipping_form_fields(&mut form, &shipping);
    }

    form
}

fn push_stripe_shipping_form_fields(form: &mut Vec<(String, String)>, shipping: &JsonValue) {
    let Some(shipping) = shipping.as_object() else {
        return;
    };

    if let Some(name) = shipping
        .get("name")
        .and_then(JsonValue::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        form.push((
            "payment_intent_data[shipping][name]".to_owned(),
            name.to_owned(),
        ));
    }

    if let Some(phone) = shipping
        .get("phone")
        .and_then(JsonValue::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
    {
        form.push((
            "payment_intent_data[shipping][phone]".to_owned(),
            phone.to_owned(),
        ));
    }

    let Some(address) = shipping.get("address").and_then(JsonValue::as_object) else {
        return;
    };
    for field in ["country", "postal_code", "state", "city", "line1", "line2"] {
        if let Some(value) = address
            .get(field)
            .and_then(JsonValue::as_str)
            .map(str::trim)
            .filter(|value| !value.is_empty())
        {
            form.push((
                format!("payment_intent_data[shipping][address][{field}]"),
                value.to_owned(),
            ));
        }
    }
}

fn append_query_params(base_url: &str, params: &[(&str, &str)]) -> String {
    let mut url = base_url.trim().to_owned();
    let mut first_param = !url.contains('?') || url.ends_with('?') || url.ends_with('&');

    for (key, value) in params {
        let value = value.trim();
        if value.is_empty() {
            continue;
        }

        if first_param {
            first_param = false;
            if !url.ends_with('?') && !url.ends_with('&') {
                url.push(if url.contains('?') { '&' } else { '?' });
            }
        } else {
            url.push('&');
        }

        url.push_str(key);
        url.push('=');
        url.push_str(value);
    }

    url
}

fn build_checkout_product_name(order: &OrderCheckoutContext) -> String {
    let copy = checkout_copy_for_locale(&order.order_locale);
    let listing_label = display_or_dash(order.listing_label.trim());
    let shape_label = display_or_dash(checkout_shape_label(copy, &order.seal_shape));

    render_checkout_product_name_template(&copy.product_name_template, listing_label, shape_label)
}

fn build_checkout_product_description(order: &OrderCheckoutContext) -> String {
    checkout_copy_for_locale(&order.order_locale)
        .product_description_template
        .clone()
}

fn checkout_return_locale(order: &OrderCheckoutContext) -> String {
    normalize_locale_route_code(&order.order_locale).unwrap_or_else(|| DEFAULT_LOCALE.to_owned())
}

fn checkout_copy_for_locale(locale: &str) -> &'static CheckoutCopy {
    let route_code = checkout_copy_route_code(locale);
    let copies = checkout_copies();

    copies
        .get(route_code)
        .or_else(|| copies.get("en"))
        .expect("checkout copy map must contain en")
}

fn checkout_copies() -> &'static HashMap<&'static str, CheckoutCopy> {
    static CHECKOUT_COPIES: OnceLock<HashMap<&'static str, CheckoutCopy>> = OnceLock::new();

    CHECKOUT_COPIES.get_or_init(|| {
        [
            ("en", CHECKOUT_COPY_EN_JSON),
            ("ja", CHECKOUT_COPY_JA_JSON),
            ("zh", CHECKOUT_COPY_ZH_JSON),
            ("zhtw", CHECKOUT_COPY_ZHTW_JSON),
            ("ar", CHECKOUT_COPY_AR_JSON),
        ]
        .into_iter()
        .map(|(route_code, source)| {
            let copy = serde_json::from_str::<CheckoutCopy>(source)
                .unwrap_or_else(|error| panic!("invalid checkout copy for {route_code}: {error}"));
            (route_code, copy)
        })
        .collect()
    })
}

fn checkout_copy_route_code(locale: &str) -> &'static str {
    let value = locale.trim().to_lowercase().replace('_', "-");
    let language = value.split('-').next().unwrap_or_default();

    match value.as_str() {
        "zhtw" | "zh-hant" | "zh-tw" | "zh-hk" | "zh-mo" => "zhtw",
        "zh" | "zh-hans" | "zh-cn" | "zh-sg" => "zh",
        "ar" => "ar",
        _ if language == "ja" => "ja",
        _ if language == "zh" => "zh",
        _ if language == "ar" => "ar",
        _ => "en",
    }
}

fn checkout_shape_label<'a>(copy: &'a CheckoutCopy, shape: &str) -> &'a str {
    let shape_key = shape.trim().to_lowercase();
    copy.shape_labels
        .get(shape_key.as_str())
        .map(String::as_str)
        .unwrap_or_default()
}

fn render_checkout_product_name_template(
    template: &str,
    listing_label: &str,
    shape_label: &str,
) -> String {
    template
        .replace("{listing_label}", listing_label)
        .replace("{shape_label}", shape_label)
}

fn build_payment_intent_shipping(order: &OrderCheckoutContext) -> Option<JsonValue> {
    let country = order.shipping_country_code.trim().to_uppercase();
    let recipient_name = order.shipping_recipient_name.trim();
    let postal_code = order.shipping_postal_code.trim();
    let city = order.shipping_city.trim();
    let line1 = order.shipping_address_line1.trim();

    // Stripe shipping requires core address fields.
    if country.is_empty()
        || recipient_name.is_empty()
        || postal_code.is_empty()
        || city.is_empty()
        || line1.is_empty()
    {
        return None;
    }

    let mut address = serde_json::Map::new();
    address.insert("country".to_owned(), JsonValue::String(country));
    address.insert(
        "postal_code".to_owned(),
        JsonValue::String(postal_code.to_owned()),
    );
    address.insert("city".to_owned(), JsonValue::String(city.to_owned()));
    address.insert("line1".to_owned(), JsonValue::String(line1.to_owned()));

    let state = order.shipping_state.trim();
    if !state.is_empty() {
        address.insert("state".to_owned(), JsonValue::String(state.to_owned()));
    }

    let line2 = order.shipping_address_line2.trim();
    if !line2.is_empty() {
        address.insert("line2".to_owned(), JsonValue::String(line2.to_owned()));
    }

    let mut shipping = serde_json::Map::new();
    shipping.insert(
        "name".to_owned(),
        JsonValue::String(recipient_name.to_owned()),
    );
    shipping.insert("address".to_owned(), JsonValue::Object(address));

    let phone = order.shipping_phone.trim();
    if !phone.is_empty() {
        shipping.insert("phone".to_owned(), JsonValue::String(phone.to_owned()));
    }

    Some(JsonValue::Object(shipping))
}

fn display_or_dash(value: &str) -> &str {
    if value.is_empty() { "-" } else { value }
}

fn pricing_currency(pricing: &BTreeMap<String, JsonValue>) -> String {
    normalize_currency_code(&read_string_field(pricing, "currency"))
        .unwrap_or_else(|| DEFAULT_CURRENCY.to_owned())
}

fn pricing_total(pricing: &BTreeMap<String, JsonValue>) -> i64 {
    read_int_field(pricing, "total").unwrap_or_default()
}

fn normalize_catalog_kanji_style(raw: &str) -> &'static str {
    let normalized = raw.trim().to_lowercase();
    match normalized.as_str() {
        "chinese" | "china" | "cn" => "chinese",
        "taiwanese" | "taiwan" | "tw" => "taiwanese",
        _ => "japanese",
    }
}

fn stripe_checkout_currency(currency: &str) -> String {
    normalize_currency_code(currency)
        .unwrap_or_else(|| DEFAULT_CURRENCY.to_owned())
        .to_lowercase()
}

fn stripe_payment_intent_id(value: &JsonValue) -> Option<String> {
    if let Some(id) = value.as_str() {
        let trimmed = id.trim();
        if !trimmed.is_empty() {
            return Some(trimmed.to_owned());
        }
    }

    value
        .as_object()
        .and_then(|object| object.get("id"))
        .and_then(JsonValue::as_str)
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

fn stripe_order_id_from_object(object: &JsonValue) -> String {
    if let Some(id) = object.get("order_id").and_then(JsonValue::as_str) {
        let trimmed = id.trim();
        if !trimmed.is_empty() {
            return trimmed.to_owned();
        }
    }

    if let Some(metadata) = object.get("metadata").and_then(JsonValue::as_object) {
        for key in ["order_id", "orderId", "orderID"] {
            if let Some(value) = metadata.get(key).and_then(JsonValue::as_str) {
                let trimmed = value.trim();
                if !trimmed.is_empty() {
                    return trimmed.to_owned();
                }
            }
        }
    }

    String::new()
}

fn validate_create_order_request(request: CreateOrderRequest) -> Result<CreateOrderInput> {
    let idempotency_key_pattern =
        Regex::new(r"^[A-Za-z0-9_-]{8,128}$").expect("idempotency key regex must compile");

    let channel = request.channel.trim().to_lowercase();
    if channel != "app" && channel != "web" {
        bail!("channel must be one of app or web");
    }

    let locale = normalize_locale_route_code(&request.locale)
        .ok_or_else(|| anyhow!("locale must map to a supported route code"))?;

    let idempotency_key = request.idempotency_key.trim().to_owned();
    if !idempotency_key_pattern.is_match(&idempotency_key) {
        bail!("idempotency_key must match ^[A-Za-z0-9_-]{{8,128}}$");
    }

    if !request.terms_agreed {
        bail!("terms_agreed must be true");
    }

    let line1 = request.seal.line1.trim().to_owned();
    validate_seal_line("seal.line1", &line1, 1, 2)?;

    let line2 = request.seal.line2.trim().to_owned();
    validate_seal_line("seal.line2", &line2, 0, 2)?;

    let shape = request.seal.shape.trim().to_lowercase();
    if shape != "square" && shape != "round" {
        bail!("seal.shape must be one of square or round");
    }

    let font_key = request.seal.font_key.trim().to_owned();
    if font_key.is_empty() {
        bail!("seal.font_key is required");
    }

    let ai_generation_id = normalize_optional_string(request.seal.ai_generation_id);
    if channel == "app" && ai_generation_id.is_none() {
        bail!("seal.ai_generation_id is required for app channel");
    }

    let ai_variant_id = normalize_optional_string(request.seal.ai_variant_id);
    if channel == "app" && ai_variant_id.is_none() {
        bail!("seal.ai_variant_id is required for app channel");
    }

    let preview_image =
        validate_create_order_seal_preview_image(request.seal.preview_image, channel.as_str())?;
    let style = validate_create_order_seal_style(request.seal.style, channel.as_str())?;
    let customer_confirmation = validate_create_order_customer_confirmation(
        request.customer_confirmation,
        channel.as_str(),
        &format!("{line1}{line2}"),
    )?;

    let order_note = normalize_optional_string(request.order_note);
    if let Some(order_note) = &order_note
        && order_note.chars().count() > 1000
    {
        bail!("order_note must be 1000 characters or fewer");
    }

    let listing_id = request.listing_id.unwrap_or_default().trim().to_owned();
    let listing_id = if listing_id.is_empty() {
        None
    } else {
        Some(listing_id)
    };

    if listing_id.is_none() {
        bail!("listing_id is required");
    }

    let country_code = request.shipping.country_code.trim().to_uppercase();
    if country_code.chars().count() != 2 {
        bail!("shipping.country_code must be ISO alpha-2");
    }

    require_non_empty("shipping.recipient_name", &request.shipping.recipient_name)?;
    require_non_empty("shipping.phone", &request.shipping.phone)?;
    require_non_empty("shipping.postal_code", &request.shipping.postal_code)?;
    require_non_empty("shipping.state", &request.shipping.state)?;
    require_non_empty("shipping.city", &request.shipping.city)?;
    require_non_empty("shipping.address_line1", &request.shipping.address_line1)?;

    let email = request.contact.email.trim().to_owned();
    if email.is_empty() {
        bail!("contact.email is required");
    }
    if !is_valid_email(&email) {
        bail!("contact.email must be valid");
    }

    let preferred_locale = normalize_locale_route_code(&request.contact.preferred_locale)
        .ok_or_else(|| anyhow!("contact.preferred_locale must map to a supported route code"))?;

    Ok(CreateOrderInput {
        channel,
        locale,
        idempotency_key,
        terms_agreed: request.terms_agreed,
        seal: SealInput {
            line1,
            line2,
            shape,
            font_key,
            ai_generation_id,
            ai_variant_id,
            preview_image,
            style,
        },
        listing_id,
        shipping: ShippingInput {
            country_code,
            recipient_name: request.shipping.recipient_name.trim().to_owned(),
            phone: request.shipping.phone.trim().to_owned(),
            postal_code: request.shipping.postal_code.trim().to_owned(),
            state: request.shipping.state.trim().to_owned(),
            city: request.shipping.city.trim().to_owned(),
            address_line1: request.shipping.address_line1.trim().to_owned(),
            address_line2: request.shipping.address_line2.trim().to_owned(),
        },
        contact: ContactInput {
            email,
            preferred_locale,
        },
        customer_confirmation,
        order_note,
    })
}

fn validate_create_order_seal_preview_image(
    request: Option<CreateOrderSealPreviewImageRequest>,
    channel: &str,
) -> Result<Option<SealPreviewImageInput>> {
    let Some(request) = request else {
        if channel == "app" {
            bail!("seal.preview_image is required for app channel");
        }
        return Ok(None);
    };

    let storage_path = request.storage_path.trim().to_owned();
    validate_seal_preview_storage_path(&storage_path)?;

    let download_url = normalize_optional_string(request.download_url);
    if let Some(download_url) = &download_url
        && download_url.chars().count() > 2048
    {
        bail!("seal.preview_image.download_url must be 2048 characters or fewer");
    }

    validate_optional_positive_dimension("seal.preview_image.width", request.width)?;
    validate_optional_positive_dimension("seal.preview_image.height", request.height)?;

    Ok(Some(SealPreviewImageInput {
        storage_path,
        download_url,
        width: request.width,
        height: request.height,
    }))
}

fn validate_create_order_seal_style(
    request: Option<CreateOrderSealStyleRequest>,
    channel: &str,
) -> Result<Option<SealStyleInput>> {
    let Some(request) = request else {
        if channel == "app" {
            bail!("seal.style is required for app channel");
        }
        return Ok(None);
    };

    let name = request.name.trim().to_lowercase();
    if !matches!(name.as_str(), "traditional" | "elegant" | "soft" | "bold") {
        bail!("seal.style.name must be one of traditional, elegant, soft, or bold");
    }

    let stroke_weight = request.stroke_weight.trim().to_lowercase();
    if !matches!(stroke_weight.as_str(), "standard" | "bold") {
        bail!("seal.style.stroke_weight must be one of standard or bold");
    }

    let balance = request.balance.trim().to_lowercase();
    if !matches!(balance.as_str(), "airy" | "balanced" | "dense") {
        bail!("seal.style.balance must be one of airy, balanced, or dense");
    }

    let prompt_summary = normalize_optional_string(request.prompt_summary);
    if let Some(prompt_summary) = &prompt_summary
        && prompt_summary.chars().count() > 400
    {
        bail!("seal.style.prompt_summary must be 400 characters or fewer");
    }

    Ok(Some(SealStyleInput {
        name,
        stroke_weight,
        balance,
        prompt_summary,
    }))
}

fn validate_create_order_customer_confirmation(
    request: Option<CreateOrderCustomerConfirmationRequest>,
    channel: &str,
    expected_seal_text: &str,
) -> Result<Option<CustomerConfirmationInput>> {
    let Some(request) = request else {
        if channel == "app" {
            bail!("customer_confirmation is required for app channel");
        }
        return Ok(None);
    };

    if !request.kanji_and_design {
        bail!("customer_confirmation.kanji_and_design must be true");
    }
    if !request.custom_made_policy {
        bail!("customer_confirmation.custom_made_policy must be true");
    }

    let confirmed_at = DateTime::parse_from_rfc3339(request.confirmed_at.trim())
        .map_err(|_| anyhow!("customer_confirmation.confirmed_at must be ISO 8601 timestamp"))?
        .with_timezone(&Utc);

    let confirmed_seal_text = request.confirmed_seal_text.trim().to_owned();
    if confirmed_seal_text != expected_seal_text {
        bail!("customer_confirmation.confirmed_seal_text must match seal.line1 + seal.line2");
    }

    Ok(Some(CustomerConfirmationInput {
        kanji_and_design: request.kanji_and_design,
        custom_made_policy: request.custom_made_policy,
        confirmed_at,
        confirmed_seal_text,
    }))
}

fn validate_seal_preview_storage_path(storage_path: &str) -> Result<()> {
    const ALLOWED_PREFIX: &str = "seal_designs/";

    if storage_path.is_empty() {
        bail!("seal.preview_image.storage_path is required");
    }
    if !storage_path.starts_with(ALLOWED_PREFIX) {
        bail!("seal.preview_image.storage_path must be under seal_designs/");
    }
    if storage_path.len() == ALLOWED_PREFIX.len() {
        bail!("seal.preview_image.storage_path must include an object name");
    }
    if storage_path.starts_with('/')
        || storage_path.contains('\\')
        || storage_path.contains("://")
        || storage_path.contains('?')
        || storage_path.contains('#')
        || storage_path.split('/').any(|segment| segment == "..")
    {
        bail!("seal.preview_image.storage_path must be a relative Firebase Storage path");
    }
    Ok(())
}

fn validate_optional_positive_dimension(field_name: &str, value: Option<i64>) -> Result<()> {
    if let Some(value) = value
        && !(1..=4096).contains(&value)
    {
        bail!("{field_name} must be between 1 and 4096");
    }
    Ok(())
}

fn require_non_empty(field_name: &str, raw: &str) -> Result<()> {
    if raw.trim().is_empty() {
        bail!("{field_name} is required");
    }
    Ok(())
}

fn validate_seal_line(field_name: &str, value: &str, min_len: usize, max_len: usize) -> Result<()> {
    let length = value.chars().count();
    if length < min_len || length > max_len {
        bail!("{field_name} must be {min_len}-{max_len} characters");
    }
    if value.chars().any(char::is_whitespace) {
        bail!("{field_name} must not contain whitespace");
    }
    Ok(())
}

fn is_valid_email(email: &str) -> bool {
    let trimmed = email.trim();
    let mut parts = trimmed.split('@');
    let local = parts.next().unwrap_or_default();
    let domain = parts.next().unwrap_or_default();
    if local.is_empty() || domain.is_empty() || parts.next().is_some() {
        return false;
    }
    if domain.starts_with('.') || domain.ends_with('.') || !domain.contains('.') {
        return false;
    }
    true
}

fn resolve_localized(
    values: &HashMap<String, String>,
    requested_locale: &str,
    default_locale: &str,
) -> String {
    if let Some(value) = lookup_locale(values, requested_locale) {
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
                Some(key.to_owned())
            }
        })
        .collect::<Vec<_>>();
    keys.sort();

    keys.first()
        .and_then(|key| values.get(key))
        .map(|value| value.trim().to_owned())
        .unwrap_or_default()
}

fn resolve_font_label_field(data: &BTreeMap<String, JsonValue>, fallback: &str) -> String {
    let label = read_string_field(data, "label");
    if !label.is_empty() {
        return label;
    }

    let localized = resolve_localized(&read_string_map_field(data, "label_i18n"), "ja", "ja");
    if !localized.is_empty() {
        return localized;
    }

    fallback.trim().to_owned()
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

fn construct_stripe_webhook_event(
    payload: &[u8],
    signature_header: &str,
    secret: &str,
) -> std::result::Result<JsonValue, StripeWebhookError> {
    let mut timestamp = None;
    let mut signatures = Vec::new();

    for part in signature_header.split(',') {
        let Some((key, value)) = part.split_once('=') else {
            continue;
        };
        match key.trim() {
            "t" => {
                timestamp = Some(
                    value
                        .trim()
                        .parse::<i64>()
                        .map_err(|_| StripeWebhookError::InvalidTimestamp)?,
                );
            }
            "v1" => signatures.push(value.trim()),
            _ => {}
        }
    }

    let timestamp = timestamp.ok_or(StripeWebhookError::MissingTimestamp)?;
    if signatures.is_empty() {
        return Err(StripeWebhookError::MissingSignature);
    }

    let now = Utc::now().timestamp();
    if (now - timestamp).abs() > STRIPE_WEBHOOK_TOLERANCE_SECONDS {
        return Err(StripeWebhookError::TimestampOutsideTolerance);
    }

    let mut signed_payload = timestamp.to_string().into_bytes();
    signed_payload.push(b'.');
    signed_payload.extend_from_slice(payload);

    let mut verified = false;
    for signature in signatures {
        let Ok(signature_bytes) = hex::decode(signature) else {
            continue;
        };
        let mut mac =
            HmacSha256::new_from_slice(secret.as_bytes()).expect("HMAC accepts any key length");
        mac.update(&signed_payload);
        if mac.verify_slice(&signature_bytes).is_ok() {
            verified = true;
            break;
        }
    }
    if !verified {
        return Err(StripeWebhookError::InvalidSignature);
    }

    serde_json::from_slice(payload).map_err(StripeWebhookError::Json)
}

fn parse_stripe_event(event: JsonValue) -> Result<StripeWebhookEvent> {
    let provider_event_id = event
        .get("id")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_owned();
    let event_type = event
        .get("type")
        .and_then(JsonValue::as_str)
        .unwrap_or_default()
        .trim()
        .to_owned();
    if provider_event_id.is_empty() || event_type.is_empty() {
        bail!("stripe event must include id and type");
    }

    let mut checkout_session_id = String::new();
    let mut checkout_session_payment_status = String::new();
    let mut checkout_session_status = String::new();
    let mut payment_intent_id = String::new();
    let mut order_id = String::new();

    if let Some(object) = event
        .get("data")
        .and_then(|data| data.get("object"))
        .and_then(JsonValue::as_object)
    {
        if let Some(id) = object.get("id").and_then(JsonValue::as_str)
            && id.trim().starts_with("pi_")
        {
            payment_intent_id = id.trim().to_owned();
        }
        if let Some(id) = object.get("id").and_then(JsonValue::as_str)
            && id.trim().starts_with("cs_")
        {
            checkout_session_id = id.trim().to_owned();
        }
        if let Some(id) = object
            .get("payment_intent")
            .and_then(stripe_payment_intent_id)
        {
            payment_intent_id = id;
        }
        if let Some(status) = object.get("payment_status").and_then(JsonValue::as_str) {
            checkout_session_payment_status = status.trim().to_ascii_lowercase();
        }
        if let Some(status) = object.get("status").and_then(JsonValue::as_str) {
            checkout_session_status = status.trim().to_ascii_lowercase();
        }

        order_id = stripe_order_id_from_object(&JsonValue::Object(object.clone()));
    }

    Ok(StripeWebhookEvent {
        provider_event_id,
        event_type,
        checkout_session_id,
        checkout_session_payment_status,
        checkout_session_status,
        payment_intent_id,
        order_id,
    })
}

fn stripe_event_from_checkout_session_snapshot(
    snapshot: &StripeCheckoutSessionSnapshot,
    fallback_order_id: &str,
) -> Option<StripeWebhookEvent> {
    let event_type = if stripe_checkout_payment_is_paid(&snapshot.payment_status) {
        "checkout.session.completed"
    } else if snapshot
        .session_status
        .trim()
        .eq_ignore_ascii_case("expired")
    {
        "checkout.session.expired"
    } else {
        return None;
    };

    Some(StripeWebhookEvent {
        provider_event_id: format!(
            "checkout_session_reconcile_{}_{}_{}",
            snapshot.session_id, snapshot.payment_status, snapshot.session_status
        ),
        event_type: event_type.to_owned(),
        checkout_session_id: snapshot.session_id.clone(),
        checkout_session_payment_status: snapshot.payment_status.clone(),
        checkout_session_status: snapshot.session_status.clone(),
        payment_intent_id: snapshot.payment_intent_id.clone(),
        order_id: if snapshot.order_id.trim().is_empty() {
            fallback_order_id.trim().to_owned()
        } else {
            snapshot.order_id.clone()
        },
    })
}

fn normalize_webhook_event(event: StripeWebhookEvent) -> StripeWebhookEvent {
    StripeWebhookEvent {
        provider_event_id: event.provider_event_id.trim().to_owned(),
        event_type: event.event_type.trim().to_owned(),
        checkout_session_id: event.checkout_session_id.trim().to_owned(),
        checkout_session_payment_status: event
            .checkout_session_payment_status
            .trim()
            .to_ascii_lowercase(),
        checkout_session_status: event.checkout_session_status.trim().to_ascii_lowercase(),
        payment_intent_id: event.payment_intent_id.trim().to_owned(),
        order_id: event.order_id.trim().to_owned(),
    }
}

fn apply_stripe_webhook_to_order_fields(
    order_fields: &mut BTreeMap<String, JsonValue>,
    normalized: &StripeWebhookEvent,
    now: DateTime<Utc>,
) -> StripeOrderWebhookMutation {
    let order_status = read_string_field(order_fields, "status");
    let (payment_status, next_status, audit_event_type) = stripe_transition(normalized);

    let mut payment = read_map_field(order_fields, "payment");
    payment.insert(
        "last_event_id".to_owned(),
        fs_string(normalized.provider_event_id.clone()),
    );
    if !normalized.checkout_session_id.is_empty() {
        payment.insert(
            "checkout_session_id".to_owned(),
            fs_string(normalized.checkout_session_id.clone()),
        );
    }
    if !normalized.payment_intent_id.is_empty() {
        payment.insert(
            "intent_id".to_owned(),
            fs_string(normalized.payment_intent_id.clone()),
        );
    }
    if !payment_status.is_empty() {
        payment.insert("status".to_owned(), fs_string(payment_status));
    }
    order_fields.insert("payment".to_owned(), fs_map(payment));
    order_fields.insert("updated_at".to_owned(), fs_timestamp(now));

    let mut after_status = order_status.clone();
    if !next_status.is_empty()
        && next_status != order_status
        && can_transition(&order_status, next_status)
    {
        order_fields.insert("status".to_owned(), fs_string(next_status));
        order_fields.insert("status_updated_at".to_owned(), fs_timestamp(now));
        after_status = next_status.to_owned();
    }

    let mut payload = btree_from_pairs(vec![
        (
            "provider_event_id",
            fs_string(normalized.provider_event_id.clone()),
        ),
        ("event_type", fs_string(normalized.event_type.clone())),
    ]);
    if !normalized.payment_intent_id.is_empty() {
        payload.insert(
            "payment_intent_id".to_owned(),
            fs_string(normalized.payment_intent_id.clone()),
        );
    }
    if !normalized.checkout_session_id.is_empty() {
        payload.insert(
            "checkout_session_id".to_owned(),
            fs_string(normalized.checkout_session_id.clone()),
        );
    }
    if !normalized.checkout_session_payment_status.is_empty() {
        payload.insert(
            "checkout_session_payment_status".to_owned(),
            fs_string(normalized.checkout_session_payment_status.clone()),
        );
    }
    if !normalized.checkout_session_status.is_empty() {
        payload.insert(
            "checkout_session_status".to_owned(),
            fs_string(normalized.checkout_session_status.clone()),
        );
    }

    let mut event_fields = btree_from_pairs(vec![
        ("type", fs_string(audit_event_type)),
        ("actor_type", fs_string("webhook")),
        ("actor_id", fs_string("stripe")),
        ("payload", fs_map(payload)),
        ("created_at", fs_timestamp(now)),
    ]);
    if after_status != order_status {
        event_fields.insert("before_status".to_owned(), fs_string(order_status));
        event_fields.insert("after_status".to_owned(), fs_string(&after_status));
    }

    StripeOrderWebhookMutation {
        event_fields,
        listing_status: stone_listing_status_after_order_status(&after_status),
    }
}

fn stripe_transition(event: &StripeWebhookEvent) -> (&'static str, &'static str, &'static str) {
    match event.event_type.as_str() {
        "payment_intent.succeeded" => ("paid", "paid", "payment_paid"),
        "checkout.session.completed" | "checkout.session.async_payment_succeeded"
            if stripe_checkout_payment_is_paid(&event.checkout_session_payment_status) =>
        {
            ("paid", "paid", "payment_paid")
        }
        "payment_intent.payment_failed"
        | "payment_intent.canceled"
        | "checkout.session.async_payment_failed"
        | "checkout.session.expired" => ("failed", "canceled", "payment_failed"),
        "charge.refunded" => ("refunded", "refunded", "payment_refunded"),
        _ => ("", "", "payment_event_recorded"),
    }
}

fn stripe_checkout_payment_is_paid(payment_status: &str) -> bool {
    payment_status.trim().eq_ignore_ascii_case("paid")
}

fn stone_listing_status_after_order_status(status: &str) -> Option<&'static str> {
    match status {
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

fn can_transition(current: &str, next: &str) -> bool {
    match current {
        "pending_payment" => matches!(next, "paid" | "canceled"),
        "paid" => matches!(next, "manufacturing" | "refunded"),
        "manufacturing" => matches!(next, "shipped" | "refunded"),
        "shipped" => matches!(next, "delivered" | "refunded"),
        _ => false,
    }
}

fn make_asset_url(bucket: &str, storage_path: &str) -> String {
    let trimmed_path = storage_path.trim().trim_start_matches('/');
    let trimmed_bucket = bucket.trim().trim_matches('/');
    if trimmed_path.is_empty() {
        return String::new();
    }
    if trimmed_bucket.is_empty() {
        return format!("/{trimmed_path}");
    }
    format!("https://storage.googleapis.com/{trimmed_bucket}/{trimmed_path}")
}

fn contains(values: &[String], value: &str) -> bool {
    values.iter().any(|item| item == value)
}

fn normalize_facet_tag_type(raw: &str) -> Option<&'static str> {
    match raw.trim().to_ascii_lowercase().as_str() {
        "color" | "colors" => Some("color"),
        "pattern" | "patterns" => Some("pattern"),
        _ => None,
    }
}

fn normalize_facet_value(raw: &str) -> String {
    raw.trim().to_ascii_lowercase()
}

fn build_facet_tag_labels(
    tags: &[FacetTag],
    requested_locale: &str,
    default_locale: &str,
) -> FacetTagLabels {
    let mut labels = FacetTagLabels::default();

    for tag in tags {
        let label = resolve_localized(&tag.label_i18n, requested_locale, default_locale);
        let label = if label.is_empty() {
            tag.key.clone()
        } else {
            label
        };
        labels.insert(&tag.facet_type, &tag.key, &label, &tag.aliases);
    }

    labels
}

fn build_material_filters(
    listings: &[StoneListing],
    facet_tag_labels: &FacetTagLabels,
) -> HashMap<&'static str, Vec<MaterialFilterOption>> {
    HashMap::from([
        (
            "color_options",
            collect_material_filter_options(
                listings,
                "color",
                |listing| listing.facets.color_family.as_str(),
                facet_tag_labels,
            ),
        ),
        (
            "pattern_options",
            collect_material_filter_options(
                listings,
                "pattern",
                |listing| listing.facets.pattern_primary.as_str(),
                facet_tag_labels,
            ),
        ),
    ])
}

fn collect_material_filter_options(
    listings: &[StoneListing],
    facet_type: &str,
    value_fn: impl Fn(&StoneListing) -> &str,
    facet_tag_labels: &FacetTagLabels,
) -> Vec<MaterialFilterOption> {
    let mut seen = HashSet::new();
    let mut options = Vec::new();

    for listing in listings {
        let value = normalize_facet_value(value_fn(listing));
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

fn material_labels_for_locale(
    materials: &[Material],
    requested_locale: &str,
    default_locale: &str,
) -> HashMap<String, String> {
    let mut labels = HashMap::new();

    for material in materials {
        let label = resolve_localized(&material.label_i18n, requested_locale, default_locale);
        if label.is_empty() {
            continue;
        }
        labels.insert(material.key.clone(), label);
    }

    labels
}

fn material_filters_response(
    filters: &HashMap<&'static str, Vec<MaterialFilterOption>>,
) -> JsonValue {
    let empty = Vec::new();
    let color_options = filters
        .get("color_options")
        .unwrap_or(&empty)
        .iter()
        .map(material_filter_option_response)
        .collect::<Vec<_>>();
    let pattern_options = filters
        .get("pattern_options")
        .unwrap_or(&empty)
        .iter()
        .map(material_filter_option_response)
        .collect::<Vec<_>>();

    json!({
        "color_options": color_options,
        "pattern_options": pattern_options,
    })
}

fn material_filter_option_response(option: &MaterialFilterOption) -> JsonValue {
    json!({
        "value": option.value.as_str(),
        "label": option.label.as_str(),
    })
}

fn is_not_found(error: &FirebaseFirestoreError) -> bool {
    matches!(
        error,
        FirebaseFirestoreError::UnexpectedStatus { status, .. } if status.as_u16() == 404
    )
}

fn is_conflict(error: &FirebaseFirestoreError) -> bool {
    matches!(
        error,
        FirebaseFirestoreError::UnexpectedStatus { status, .. } if status.as_u16() == 409
    )
}

fn is_precondition_failed(error: &FirebaseFirestoreError) -> bool {
    matches!(
        error,
        FirebaseFirestoreError::UnexpectedStatus { status, .. } if status.as_u16() == 412
    )
}

fn create_order_commit_conflicted(error: &FirebaseFirestoreError) -> bool {
    is_conflict(error) || is_precondition_failed(error)
}

fn reserve_stone_listing_write(
    listing_doc_name: &str,
    next_version: i64,
    listing_update_time: &str,
    now: DateTime<Utc>,
) -> JsonValue {
    json!({
        "update": {
            "name": listing_doc_name,
            "fields": {
                "status": fs_string("reserved"),
                "version": fs_int(next_version),
                "updated_at": fs_timestamp(now),
            }
        },
        "updateMask": {
            "fieldPaths": ["status", "version", "updated_at"]
        },
        "currentDocument": {
            "updateTime": listing_update_time
        }
    })
}

fn document_id(document: &Document) -> Option<String> {
    document
        .name
        .as_deref()
        .and_then(|name| name.rsplit('/').next())
        .map(ToOwned::to_owned)
}

fn normalize_material_shape(raw: &str) -> &'static str {
    match raw.trim().to_ascii_lowercase().as_str() {
        "round" => "round",
        _ => "square",
    }
}

fn read_material_shape(data: &BTreeMap<String, JsonValue>) -> String {
    normalize_material_shape(&read_string_field(data, "shape")).to_owned()
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
                width: read_int_field(&fields, "width").unwrap_or_default(),
                height: read_int_field(&fields, "height").unwrap_or_default(),
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
    for (currency, amount_value) in fields {
        let mut container = BTreeMap::new();
        container.insert("amount".to_owned(), amount_value.clone());
        if let Some(amount) = read_int_field(&container, "amount") {
            result.insert(currency.trim().to_owned(), amount);
        }
    }

    result
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

fn read_string_array_from_map(data: &BTreeMap<String, JsonValue>, key: &str) -> Vec<String> {
    read_array_field(data, key)
        .into_iter()
        .filter_map(|value| {
            value
                .get("stringValue")
                .and_then(JsonValue::as_str)
                .map(str::trim)
                .filter(|value| !value.is_empty())
                .map(ToOwned::to_owned)
        })
        .collect::<Vec<_>>()
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
        .unwrap_or_default()
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

fn fs_string(value: impl Into<String>) -> JsonValue {
    json!({ "stringValue": value.into() })
}

fn fs_bool(value: bool) -> JsonValue {
    json!({ "booleanValue": value })
}

fn fs_int(value: i64) -> JsonValue {
    json!({ "integerValue": value.to_string() })
}

fn fs_int_map(values: &HashMap<String, i64>) -> JsonValue {
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
    fs_array(values.iter().cloned().map(fs_string).collect::<Vec<_>>())
}

fn fs_material_photos(photos: &[MaterialPhoto]) -> JsonValue {
    fs_array(
        photos
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
            .collect::<Vec<_>>(),
    )
}

fn btree_from_pairs(pairs: Vec<(&str, JsonValue)>) -> BTreeMap<String, JsonValue> {
    pairs
        .into_iter()
        .map(|(key, value)| (key.to_owned(), value))
        .collect::<BTreeMap<_, _>>()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn resolve_localized_uses_fallback_order() {
        let mut values = HashMap::new();
        values.insert("ja".to_owned(), "翡翠".to_owned());
        values.insert("en".to_owned(), "Jade".to_owned());

        assert_eq!(resolve_localized(&values, "en", "ja"), "Jade");
        assert_eq!(resolve_localized(&values, "fr", "en"), "Jade");
        assert_eq!(resolve_localized(&values, "fr", "ko"), "翡翠");
    }

    #[test]
    fn m4_t07_resolve_localized_falls_back_for_missing_requested_value() {
        let values = HashMap::from([
            ("en".to_owned(), "Jade".to_owned()),
            ("ja".to_owned(), " ".to_owned()),
            ("zhtw".to_owned(), "青田石".to_owned()),
        ]);

        assert_eq!(resolve_localized(&values, "fr", "zhtw"), "青田石");
        assert_eq!(resolve_localized(&values, "ja", "en"), "Jade");
    }

    #[test]
    fn m4_t07_requested_locale_accepts_supported_route_and_bcp47_values() {
        assert_eq!(
            requested_locale_route_code(Some("zh-Hant".to_owned()), "en").as_deref(),
            Some("zhtw")
        );
        assert_eq!(
            requested_locale_route_code(Some("en-US".to_owned()), "ja").as_deref(),
            Some("en")
        );
        assert_eq!(
            requested_locale_route_code(None, "ja").as_deref(),
            Some("ja")
        );
    }

    #[test]
    fn m4_t07_requested_locale_rejects_unsupported_values() {
        assert_eq!(
            requested_locale_route_code(Some("xx".to_owned()), "en"),
            None
        );
        assert_eq!(
            requested_locale_route_code(Some("../ja".to_owned()), "en"),
            None
        );
    }

    #[test]
    fn default_public_config_sets_ja_to_jpy() {
        let cfg = default_public_config();
        assert_eq!(cfg.supported_locales, vec!["ar", "en", "ja", "zh", "zhtw"]);
        assert_eq!(cfg.default_locale, "ja");
        assert_eq!(cfg.default_currency, "USD");
        assert_eq!(
            cfg.currency_by_locale.get("ja").map(String::as_str),
            Some("JPY")
        );
        assert_eq!(
            cfg.currency_by_locale.get("ar").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("zh").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("zhtw").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("en").map(String::as_str),
            Some("USD")
        );
    }

    #[test]
    fn normalize_public_config_uses_registry_defaults_for_missing_values() {
        let cfg = normalize_public_config(PublicConfig {
            supported_locales: Vec::new(),
            default_locale: String::new(),
            default_currency: String::new(),
            currency_by_locale: HashMap::new(),
        });

        assert_eq!(cfg.supported_locales, vec!["ar", "en", "ja", "zh", "zhtw"]);
        assert_eq!(cfg.default_locale, "ja");
        assert_eq!(cfg.default_currency, "USD");
        assert_eq!(
            cfg.currency_by_locale.get("en").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("ja").map(String::as_str),
            Some("JPY")
        );
        assert_eq!(
            cfg.currency_by_locale.get("ar").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("zh").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            cfg.currency_by_locale.get("zhtw").map(String::as_str),
            Some("USD")
        );
    }

    #[test]
    fn normalize_public_config_maps_bcp47_locales_to_route_codes() {
        let cfg = normalize_public_config(PublicConfig {
            supported_locales: vec!["EN-US".to_owned(), "zh-Hant".to_owned()],
            default_locale: "zh_Hant".to_owned(),
            default_currency: "usd".to_owned(),
            currency_by_locale: HashMap::from([
                ("en-US".to_owned(), "usd".to_owned()),
                ("zh-Hant".to_owned(), "usd".to_owned()),
            ]),
        });

        assert_eq!(cfg.supported_locales, vec!["en", "zhtw"]);
        assert_eq!(cfg.default_locale, "zhtw");
        assert_eq!(
            cfg.currency_by_locale.get("zhtw").map(String::as_str),
            Some("USD")
        );
    }

    #[test]
    fn stone_listing_status_helper_requires_published() {
        assert!(stone_listing_is_published("published"));
        assert!(stone_listing_is_published(" Published "));
        assert!(!stone_listing_is_published("draft"));
    }

    #[test]
    fn stone_listing_orderable_helper_requires_active_and_published() {
        assert!(stone_listing_is_orderable(true, "published"));
        assert!(stone_listing_is_orderable(true, " Published "));
        assert!(!stone_listing_is_orderable(true, "draft"));
        assert!(!stone_listing_is_orderable(false, "published"));
    }

    #[test]
    fn m13_t06_reserved_and_sold_listings_are_not_orderable() {
        assert!(!stone_listing_is_orderable(true, "reserved"));
        assert!(!stone_listing_is_orderable(true, "sold"));
    }

    #[test]
    fn m13_t06_reserve_listing_write_requires_update_time_precondition() {
        let now = DateTime::parse_from_rfc3339("2026-05-21T12:34:56Z")
            .unwrap()
            .with_timezone(&Utc);
        let listing_doc_name =
            "projects/demo/databases/(default)/documents/stone_listings/stone_listing_001";

        let write = reserve_stone_listing_write(listing_doc_name, 8, "2026-05-21T11:14:00Z", now);

        assert_eq!(write["update"]["name"], listing_doc_name);
        assert_eq!(
            write["update"]["fields"]["status"]["stringValue"],
            "reserved"
        );
        assert_eq!(write["update"]["fields"]["version"]["integerValue"], "8");
        assert_eq!(write["update"]["fields"]["updated_at"], fs_timestamp(now));
        assert_eq!(
            write["updateMask"]["fieldPaths"],
            json!(["status", "version", "updated_at"])
        );
        assert_eq!(
            write["currentDocument"]["updateTime"],
            "2026-05-21T11:14:00Z"
        );
    }

    #[test]
    fn m13_t06_create_order_conflict_statuses_are_inventory_races() {
        let conflict = FirebaseFirestoreError::UnexpectedStatus {
            status: StatusCode::CONFLICT,
            body: "already exists".to_owned(),
        };
        let precondition_failed = FirebaseFirestoreError::UnexpectedStatus {
            status: StatusCode::PRECONDITION_FAILED,
            body: "stale update_time".to_owned(),
        };
        let server_error = FirebaseFirestoreError::UnexpectedStatus {
            status: StatusCode::INTERNAL_SERVER_ERROR,
            body: "internal".to_owned(),
        };

        assert!(create_order_commit_conflicted(&conflict));
        assert!(create_order_commit_conflicted(&precondition_failed));
        assert!(!create_order_commit_conflicted(&server_error));
    }

    #[test]
    fn stone_listing_app_visibility_keeps_sold_out_details_private_drafts_hidden() {
        assert!(stone_listing_is_app_visible(true, "published"));
        assert!(stone_listing_is_app_visible(true, "reserved"));
        assert!(stone_listing_is_app_visible(true, "sold"));
        assert!(!stone_listing_is_app_visible(true, "draft"));
        assert!(!stone_listing_is_app_visible(true, "archived"));
        assert!(!stone_listing_is_app_visible(false, "published"));
    }

    #[test]
    fn stone_listing_status_follows_paid_and_canceled_orders() {
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
    fn canceled_order_only_reopens_reserved_or_backfilled_published_listings() {
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
        assert!(!stone_listing_should_restore_after_canceled_order(
            "draft", true
        ));
    }

    #[test]
    fn firestore_paths_keep_commit_database_separate_from_document_parent() {
        let database = firestore_database_path("hanko-field");

        assert_eq!(database, "projects/hanko-field/databases/(default)");
        assert_eq!(
            firestore_documents_parent(&database),
            "projects/hanko-field/databases/(default)/documents"
        );
    }

    #[test]
    fn resolve_currency_for_locale_uses_locale_map_and_default() {
        let mut currency_by_locale = HashMap::new();
        currency_by_locale.insert("ja".to_owned(), "JPY".to_owned());
        currency_by_locale.insert("en".to_owned(), "USD".to_owned());

        let cfg = PublicConfig {
            supported_locales: vec!["ja".to_owned(), "en".to_owned()],
            default_locale: "ja".to_owned(),
            default_currency: "USD".to_owned(),
            currency_by_locale,
        };

        assert_eq!(resolve_currency_for_locale(&cfg, "ja"), "JPY");
        assert_eq!(resolve_currency_for_locale(&cfg, "ja-jp"), "JPY");
        assert_eq!(resolve_currency_for_locale(&cfg, "fr"), "JPY");
    }

    #[test]
    fn resolve_pricing_currency_uses_locale_currency() {
        let mut currency_by_locale = HashMap::new();
        currency_by_locale.insert("ja".to_owned(), "JPY".to_owned());
        currency_by_locale.insert("en".to_owned(), "USD".to_owned());

        let cfg = PublicConfig {
            supported_locales: vec!["ja".to_owned(), "en".to_owned()],
            default_locale: "ja".to_owned(),
            default_currency: "USD".to_owned(),
            currency_by_locale,
        };

        assert_eq!(resolve_pricing_currency(&cfg, "ja"), "JPY");
        assert_eq!(resolve_pricing_currency(&cfg, "en"), "USD");
        assert_eq!(resolve_pricing_currency(&cfg, "ja-jp"), "JPY");
    }

    #[test]
    fn stripe_checkout_currency_normalizes_iso_code() {
        assert_eq!(stripe_checkout_currency("usd"), "usd");
        assert_eq!(stripe_checkout_currency(" JpY "), "jpy");
        assert_eq!(stripe_checkout_currency(""), "usd");
    }

    #[test]
    fn material_price_for_currency_uses_currency_specific_field() {
        let material = Material {
            key: "jade".to_owned(),
            label_i18n: HashMap::new(),
            description_i18n: HashMap::new(),
            shape: "square".to_owned(),
            photos: Vec::new(),
            price_by_currency: HashMap::from([
                ("USD".to_owned(), 88500),
                ("JPY".to_owned(), 150000),
            ]),
            version: 1,
        };

        assert_eq!(material_price_for_currency(&material, "USD"), 88500);
        assert_eq!(material_price_for_currency(&material, "JPY"), 150000);
    }

    #[test]
    fn stone_listing_price_map_uses_currency_fields() {
        let fields = btree_from_pairs(vec![(
            "price_by_currency",
            fs_map(btree_from_pairs(vec![
                ("USD", fs_int(88500)),
                ("JPY", fs_int(150000)),
            ])),
        )]);

        assert_eq!(
            stone_listing_price_by_currency_from_fields(&fields),
            HashMap::from([("USD".to_owned(), 88500), ("JPY".to_owned(), 150000),])
        );
    }

    #[test]
    fn stone_listing_response_includes_app_aliases_labels_and_price_object() {
        let mut facet_tag_labels = FacetTagLabels::default();
        facet_tag_labels.insert("color", "pink", "Pink", &[]);
        facet_tag_labels.insert("pattern", "plain", "Plain", &[]);
        let material_labels = HashMap::from([("rose_quartz".to_owned(), "Rose Quartz".to_owned())]);

        let response = stone_listing_response(
            "assets.example.test",
            "en",
            "ja",
            "JPY",
            &facet_tag_labels,
            &material_labels,
            StoneListing {
                key: "stone_listing_001".to_owned(),
                listing_code: "RQZ-0001".to_owned(),
                material_key: "rose_quartz".to_owned(),
                size: "24x24x60 mm".to_owned(),
                title_i18n: HashMap::from([
                    ("ja".to_owned(), "ソフトピンクローズクォーツ印材".to_owned()),
                    (
                        "en".to_owned(),
                        "Soft Pink Rose Quartz Seal Stone".to_owned(),
                    ),
                ]),
                description_i18n: HashMap::from([(
                    "en".to_owned(),
                    "A soft pink rose quartz seal stone.".to_owned(),
                )]),
                story_i18n: HashMap::from([("en".to_owned(), "A one-of-a-kind piece.".to_owned())]),
                facets: StoneListingFacets {
                    color_family: "pink".to_owned(),
                    color_tags: vec!["pink".to_owned()],
                    pattern_primary: "plain".to_owned(),
                    pattern_tags: vec!["plain".to_owned()],
                    stone_shape: "square".to_owned(),
                    translucency: "semi_translucent".to_owned(),
                },
                photos: vec![MaterialPhoto {
                    asset_id: "asset_001".to_owned(),
                    storage_path: "stone_listings/rose_quartz/main.webp".to_owned(),
                    alt_i18n: HashMap::from([("en".to_owned(), "Rose quartz photo".to_owned())]),
                    sort_order: 1,
                    is_primary: true,
                    width: 1200,
                    height: 900,
                }],
                price_by_currency: HashMap::from([
                    ("JPY".to_owned(), 18000),
                    ("USD".to_owned(), 12000),
                ]),
                status: "published".to_owned(),
                is_active: true,
                sort_order: 10,
                version: 2,
            },
        );

        assert_eq!(response["id"], json!("stone_listing_001"));
        assert_eq!(response["key"], json!("stone_listing_001"));
        assert_eq!(response["code"], json!("RQZ-0001"));
        assert_eq!(response["listing_code"], json!("RQZ-0001"));
        assert_eq!(response["material_label"], json!("Rose Quartz"));
        assert_eq!(response["title"], json!("Soft Pink Rose Quartz Seal Stone"));
        assert_eq!(response["price"]["amount"], json!(18000));
        assert_eq!(response["price"]["currency"], json!("JPY"));
        assert_eq!(response["price"]["display"], json!("JPY 18,000"));
        assert_eq!(response["price_amount"], json!(18000));
        assert_eq!(response["is_orderable"], json!(true));
        assert_eq!(response["color_tag_labels"], json!(["Pink"]));
        assert_eq!(response["pattern_tag_labels"], json!(["Plain"]));
        assert_eq!(
            response["photos"][0]["asset_url"],
            json!(
                "https://storage.googleapis.com/assets.example.test/stone_listings/rose_quartz/main.webp"
            )
        );
    }

    #[test]
    fn country_shipping_fee_for_currency_uses_currency_specific_field() {
        let country = Country {
            code: "JP".to_owned(),
            label_i18n: HashMap::new(),
            shipping_fee_by_currency: HashMap::from([
                ("USD".to_owned(), 600),
                ("JPY".to_owned(), 800),
            ]),
            version: 1,
        };

        assert_eq!(country_shipping_fee_for_currency(&country, "USD"), 600);
        assert_eq!(country_shipping_fee_for_currency(&country, "JPY"), 800);
    }

    #[test]
    fn pricing_total_reads_neutral_key_only() {
        let pricing_neutral = btree_from_pairs(vec![("total", fs_int(1234))]);
        assert_eq!(pricing_total(&pricing_neutral), 1234);
    }

    #[test]
    fn resolve_order_listing_fields_uses_legacy_material_snapshot() {
        let data = btree_from_pairs(vec![("material_label_ja", fs_string("翡翠"))]);
        let listing = BTreeMap::new();
        let material = btree_from_pairs(vec![("key", fs_string("jade"))]);

        let (listing_key, listing_label) = FirestoreStore::resolve_order_listing_fields(
            &data,
            &listing,
            &material,
            "ja",
            DEFAULT_LOCALE,
        );

        assert_eq!(listing_key, "jade");
        assert_eq!(listing_label, "翡翠");
    }

    fn order_checkout_context_fixture() -> OrderCheckoutContext {
        OrderCheckoutContext {
            order_id: "order_1".to_owned(),
            order_locale: "en".to_owned(),
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            listing_key: "stone_listing_001".to_owned(),
            listing_label: "Rose Quartz".to_owned(),
            seal_shape: "round".to_owned(),
            shipping_country_code: "US".to_owned(),
            shipping_recipient_name: "Michael Smith".to_owned(),
            shipping_phone: "+1-000-000-0000".to_owned(),
            shipping_postal_code: "10001".to_owned(),
            shipping_state: "NY".to_owned(),
            shipping_city: "New York".to_owned(),
            shipping_address_line1: "123 Example Street".to_owned(),
            shipping_address_line2: "Apt 1".to_owned(),
            total: 18600,
            currency: "JPY".to_owned(),
            contact_email: "customer@example.com".to_owned(),
        }
    }

    fn stripe_form_value<'a>(form: &'a [(String, String)], key: &str) -> &'a str {
        form.iter()
            .find(|(form_key, _)| form_key == key)
            .map(|(_, value)| value.as_str())
            .unwrap_or_else(|| panic!("missing Stripe form key {key}"))
    }

    fn stripe_checkout_config_fixture() -> StripeCheckoutConfig {
        StripeCheckoutConfig {
            success_url: DEFAULT_STRIPE_CHECKOUT_SUCCESS_URL.to_owned(),
            cancel_url: DEFAULT_STRIPE_CHECKOUT_CANCEL_URL.to_owned(),
            app_success_url: DEFAULT_STRIPE_APP_CHECKOUT_SUCCESS_URL.to_owned(),
            app_cancel_url: DEFAULT_STRIPE_APP_CHECKOUT_CANCEL_URL.to_owned(),
        }
    }

    #[test]
    fn m13_t05_checkout_session_form_carries_return_urls_and_order_metadata() {
        let order = order_checkout_context_fixture();
        let checkout = stripe_checkout_config_fixture();

        let form =
            build_stripe_checkout_session_form(&checkout, &order, " customer@example.com ", false);

        assert_eq!(stripe_form_value(&form, "mode"), "payment");
        assert_eq!(
            stripe_form_value(&form, "success_url"),
            "http://127.0.0.1:3052/payment/success?session_id={CHECKOUT_SESSION_ID}&checkout=success&order_id=order_1&lang=en"
        );
        assert_eq!(
            stripe_form_value(&form, "cancel_url"),
            "http://127.0.0.1:3052/payment/failure?checkout=cancel&order_id=order_1&lang=en"
        );
        assert_eq!(stripe_form_value(&form, "metadata[order_id]"), "order_1");
        assert_eq!(
            stripe_form_value(&form, "payment_intent_data[metadata][order_id]"),
            "order_1"
        );
        assert_eq!(
            stripe_form_value(&form, "customer_email"),
            "customer@example.com"
        );
        assert_eq!(
            stripe_form_value(&form, "payment_intent_data[receipt_email]"),
            "customer@example.com"
        );
        assert_eq!(
            stripe_form_value(&form, "line_items[0][price_data][currency]"),
            "jpy"
        );
        assert_eq!(
            stripe_form_value(&form, "line_items[0][price_data][unit_amount]"),
            "18600"
        );
        assert_eq!(
            stripe_form_value(&form, "line_items[0][price_data][product_data][name]"),
            "Stone seal (Rose Quartz; circle)"
        );
        assert_eq!(
            stripe_form_value(
                &form,
                "line_items[0][price_data][product_data][description]"
            ),
            "Custom stone seal order"
        );
        assert_eq!(
            stripe_form_value(&form, "payment_intent_data[shipping][address][country]"),
            "US"
        );
    }

    #[test]
    fn checkout_session_form_uses_app_custom_scheme_for_app_returns() {
        let order = order_checkout_context_fixture();
        let checkout = stripe_checkout_config_fixture();

        let form =
            build_stripe_checkout_session_form(&checkout, &order, "customer@example.com", true);

        assert_eq!(
            stripe_form_value(&form, "success_url"),
            "hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}&checkout=success&order_id=order_1&lang=en&return_to=app"
        );
        assert_eq!(
            stripe_form_value(&form, "cancel_url"),
            "hankofield://checkout/cancel?checkout=cancel&order_id=order_1&lang=en&return_to=app"
        );
        assert_eq!(stripe_form_value(&form, "origin_context"), "mobile_app");
    }

    #[test]
    fn build_checkout_product_name_uses_japanese_format_for_ja_locale() {
        let order = checkout_product_name_context("ja", "翡翠", "square");

        assert_eq!(build_checkout_product_name(&order), "宝石印鑑 (翡翠、角)");
    }

    #[test]
    fn build_checkout_product_name_uses_english_format_for_non_ja_locale() {
        let order = checkout_product_name_context("en", "Jade", "round");

        assert_eq!(
            build_checkout_product_name(&order),
            "Stone seal (Jade; circle)"
        );
    }

    #[test]
    fn build_checkout_product_name_uses_simplified_chinese_copy_for_zh_locale() {
        let order = checkout_product_name_context("zh-CN", "青田石", "round");

        assert_eq!(
            build_checkout_product_name(&order),
            "宝石印章（青田石，圆）"
        );
    }

    #[test]
    fn build_checkout_product_name_uses_traditional_chinese_copy_for_zhtw_locale() {
        let order = checkout_product_name_context("zh_Hant", "青田石", "square");

        assert_eq!(
            build_checkout_product_name(&order),
            "寶石印章（青田石，方）"
        );
    }

    #[test]
    fn build_checkout_product_name_uses_arabic_copy_for_ar_locale() {
        let order = checkout_product_name_context("ar-EG", "حجر اليشم", "square");

        assert_eq!(
            build_checkout_product_name(&order),
            "ختم حجر كريم (حجر اليشم؛ مربع)"
        );
        assert_eq!(
            build_checkout_product_description(&order),
            "طلب ختم حجر كريم مخصص"
        );
    }

    #[test]
    fn stripe_checkout_return_urls_preserve_normalized_route_code() {
        let mut order = order_checkout_context_fixture();
        let checkout = stripe_checkout_config_fixture();

        order.order_locale = "zh-Hans".to_owned();
        let simplified_form =
            build_stripe_checkout_session_form(&checkout, &order, "customer@example.com", true);

        assert_eq!(
            stripe_form_value(&simplified_form, "success_url"),
            "hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}&checkout=success&order_id=order_1&lang=zh&return_to=app"
        );
        assert_eq!(
            stripe_form_value(&simplified_form, "cancel_url"),
            "hankofield://checkout/cancel?checkout=cancel&order_id=order_1&lang=zh&return_to=app"
        );

        order.order_locale = "zh-Hant".to_owned();
        let form =
            build_stripe_checkout_session_form(&checkout, &order, "customer@example.com", true);

        assert_eq!(
            stripe_form_value(&form, "success_url"),
            "hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}&checkout=success&order_id=order_1&lang=zhtw&return_to=app"
        );
        assert_eq!(
            stripe_form_value(&form, "cancel_url"),
            "hankofield://checkout/cancel?checkout=cancel&order_id=order_1&lang=zhtw&return_to=app"
        );
    }

    #[test]
    fn m4_t07_checkout_language_persistence_uses_normalized_order_route_code() {
        let mut order = order_checkout_context_fixture();
        order.order_locale = "zh_TW".to_owned();
        let checkout = stripe_checkout_config_fixture();

        let form =
            build_stripe_checkout_session_form(&checkout, &order, "customer@example.com", true);

        assert!(
            stripe_form_value(&form, "success_url").contains("lang=zhtw"),
            "success_url should persist normalized checkout language"
        );
        assert!(
            stripe_form_value(&form, "cancel_url").contains("lang=zhtw"),
            "cancel_url should persist normalized checkout language"
        );
    }

    #[test]
    fn pilot_checkout_return_urls_preserve_arabic_route_code() {
        let mut order = order_checkout_context_fixture();
        order.order_locale = "ar-EG".to_owned();
        let checkout = stripe_checkout_config_fixture();

        let form =
            build_stripe_checkout_session_form(&checkout, &order, "customer@example.com", true);

        assert_eq!(
            stripe_form_value(&form, "success_url"),
            "hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}&checkout=success&order_id=order_1&lang=ar&return_to=app"
        );
        assert_eq!(
            stripe_form_value(&form, "cancel_url"),
            "hankofield://checkout/cancel?checkout=cancel&order_id=order_1&lang=ar&return_to=app"
        );
        assert_eq!(
            stripe_form_value(&form, "line_items[0][price_data][product_data][name]"),
            "ختم حجر كريم (Rose Quartz؛ دائري)"
        );
    }

    #[test]
    fn checkout_copy_templates_have_required_placeholders() {
        for (route_code, copy) in checkout_copies() {
            assert!(
                copy.product_name_template.contains("{listing_label}"),
                "{route_code} checkout template must contain listing_label placeholder"
            );
            assert!(
                copy.product_name_template.contains("{shape_label}"),
                "{route_code} checkout template must contain shape_label placeholder"
            );
            assert!(
                !copy.product_description_template.trim().is_empty(),
                "{route_code} checkout description template must not be empty"
            );
            assert!(
                copy.shape_labels.contains_key("round"),
                "{route_code} checkout shape labels must include round"
            );
            assert!(
                copy.shape_labels.contains_key("square"),
                "{route_code} checkout shape labels must include square"
            );
        }
    }

    fn checkout_product_name_context(
        order_locale: &str,
        listing_label: &str,
        seal_shape: &str,
    ) -> OrderCheckoutContext {
        OrderCheckoutContext {
            order_id: "order_1".to_owned(),
            order_locale: order_locale.to_owned(),
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            listing_key: String::new(),
            listing_label: listing_label.to_owned(),
            seal_shape: seal_shape.to_owned(),
            shipping_country_code: "JP".to_owned(),
            shipping_recipient_name: "田中 太郎".to_owned(),
            shipping_phone: "09000001111".to_owned(),
            shipping_postal_code: "1000001".to_owned(),
            shipping_state: "東京都".to_owned(),
            shipping_city: "千代田区".to_owned(),
            shipping_address_line1: "1-1-1".to_owned(),
            shipping_address_line2: "テストビル101".to_owned(),
            total: 12345,
            currency: DEFAULT_CURRENCY.to_owned(),
            contact_email: "buyer@example.com".to_owned(),
        }
    }

    #[test]
    fn build_payment_intent_shipping_includes_address_and_phone() {
        let order = OrderCheckoutContext {
            order_id: "order_1".to_owned(),
            order_locale: "ja".to_owned(),
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            listing_key: String::new(),
            listing_label: "翡翠".to_owned(),
            seal_shape: "round".to_owned(),
            shipping_country_code: "jp".to_owned(),
            shipping_recipient_name: "田中 太郎".to_owned(),
            shipping_phone: "09000001111".to_owned(),
            shipping_postal_code: "1000001".to_owned(),
            shipping_state: "東京都".to_owned(),
            shipping_city: "千代田区".to_owned(),
            shipping_address_line1: "1-1-1".to_owned(),
            shipping_address_line2: "テストビル101".to_owned(),
            total: 12345,
            currency: DEFAULT_CURRENCY.to_owned(),
            contact_email: "buyer@example.com".to_owned(),
        };

        let shipping = build_payment_intent_shipping(&order).expect("shipping must be present");
        let shipping_obj = shipping.as_object().expect("shipping must be object");
        let address = shipping_obj
            .get("address")
            .and_then(JsonValue::as_object)
            .expect("address must be object");

        assert_eq!(
            shipping_obj.get("name").and_then(JsonValue::as_str),
            Some("田中 太郎")
        );
        assert_eq!(
            shipping_obj.get("phone").and_then(JsonValue::as_str),
            Some("09000001111")
        );
        assert_eq!(
            address.get("country").and_then(JsonValue::as_str),
            Some("JP")
        );
        assert_eq!(
            address.get("line1").and_then(JsonValue::as_str),
            Some("1-1-1")
        );
        assert_eq!(
            address.get("line2").and_then(JsonValue::as_str),
            Some("テストビル101")
        );
    }

    #[test]
    fn build_payment_intent_shipping_returns_none_when_address_is_missing() {
        let order = OrderCheckoutContext {
            order_id: "order_1".to_owned(),
            order_locale: "ja".to_owned(),
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            listing_key: String::new(),
            listing_label: "翡翠".to_owned(),
            seal_shape: "round".to_owned(),
            shipping_country_code: "JP".to_owned(),
            shipping_recipient_name: "田中 太郎".to_owned(),
            shipping_phone: "09000001111".to_owned(),
            shipping_postal_code: "".to_owned(),
            shipping_state: "東京都".to_owned(),
            shipping_city: "千代田区".to_owned(),
            shipping_address_line1: "1-1-1".to_owned(),
            shipping_address_line2: "".to_owned(),
            total: 12345,
            currency: DEFAULT_CURRENCY.to_owned(),
            contact_email: "buyer@example.com".to_owned(),
        };

        assert!(build_payment_intent_shipping(&order).is_none());
    }

    #[test]
    fn append_query_params_preserves_existing_query_string() {
        let url = append_query_params(
            "https://example.com/payment/success?session_id={CHECKOUT_SESSION_ID}",
            &[("order_id", "order_1"), ("lang", "ja")],
        );

        assert_eq!(
            url,
            "https://example.com/payment/success?session_id={CHECKOUT_SESSION_ID}&order_id=order_1&lang=ja"
        );
    }

    #[test]
    fn append_query_params_adds_query_string_when_missing() {
        let url = append_query_params(
            "https://example.com/payment/failure",
            &[("order_id", "order_1"), ("lang", "en")],
        );

        assert_eq!(
            url,
            "https://example.com/payment/failure?order_id=order_1&lang=en"
        );
    }

    fn valid_app_create_order_request() -> CreateOrderRequest {
        CreateOrderRequest {
            channel: "app".to_owned(),
            locale: "en".to_owned(),
            idempotency_key: "demo_key_123".to_owned(),
            terms_agreed: true,
            seal: CreateOrderSealRequest {
                line1: "美".to_owned(),
                line2: "空".to_owned(),
                shape: "square".to_owned(),
                font_key: "ai_generated_seal".to_owned(),
                ai_generation_id: Some("seal_request_001".to_owned()),
                ai_variant_id: Some("seal_variant_001".to_owned()),
                preview_image: Some(CreateOrderSealPreviewImageRequest {
                    storage_path: "seal_designs/seal_request_001/seal_variant_001.png".to_owned(),
                    download_url: Some(
                        "https://firebasestorage.googleapis.com/v0/b/example/o/seal.png".to_owned(),
                    ),
                    width: Some(1024),
                    height: Some(1024),
                }),
                style: Some(CreateOrderSealStyleRequest {
                    name: "elegant".to_owned(),
                    stroke_weight: "standard".to_owned(),
                    balance: "balanced".to_owned(),
                    prompt_summary: Some("Elegant, standard stroke, balanced spacing.".to_owned()),
                }),
            },
            listing_id: Some("stone_listing_001".to_owned()),
            shipping: CreateOrderShippingRequest {
                country_code: "us".to_owned(),
                recipient_name: "Michael Smith".to_owned(),
                phone: "+1-000-000-0000".to_owned(),
                postal_code: "10001".to_owned(),
                state: "NY".to_owned(),
                city: "New York".to_owned(),
                address_line1: "123 Example Street".to_owned(),
                address_line2: "Apt 1".to_owned(),
            },
            contact: CreateOrderContactRequest {
                email: "customer@example.com".to_owned(),
                preferred_locale: "en".to_owned(),
            },
            customer_confirmation: Some(CreateOrderCustomerConfirmationRequest {
                kanji_and_design: true,
                custom_made_policy: true,
                confirmed_at: "2026-05-21T11:00:00Z".to_owned(),
                confirmed_seal_text: "美空".to_owned(),
            }),
            order_note: Some("Optional note".to_owned()),
        }
    }

    fn assert_create_order_error_contains(request: CreateOrderRequest, expected: &str) {
        let error = validate_create_order_request(request).expect_err("request should be invalid");
        let message = error.to_string();
        assert!(
            message.contains(expected),
            "expected error to contain {expected:?}, got {message:?}"
        );
    }

    #[test]
    fn validate_create_order_request_accepts_valid_payload() {
        let request = CreateOrderRequest {
            channel: "web".to_owned(),
            locale: "ja".to_owned(),
            idempotency_key: "demo_key_123".to_owned(),
            terms_agreed: true,
            seal: CreateOrderSealRequest {
                line1: "田中".to_owned(),
                line2: "太郎".to_owned(),
                shape: "square".to_owned(),
                font_key: "zen_maru_gothic".to_owned(),
                ai_generation_id: None,
                ai_variant_id: None,
                preview_image: None,
                style: None,
            },
            listing_id: Some("rose_quartz_01".to_owned()),
            shipping: CreateOrderShippingRequest {
                country_code: "jp".to_owned(),
                recipient_name: "田中 太郎".to_owned(),
                phone: "09000001111".to_owned(),
                postal_code: "1000001".to_owned(),
                state: "東京都".to_owned(),
                city: "千代田区".to_owned(),
                address_line1: "1-1-1".to_owned(),
                address_line2: "".to_owned(),
            },
            contact: CreateOrderContactRequest {
                email: "taro@example.com".to_owned(),
                preferred_locale: "ja".to_owned(),
            },
            customer_confirmation: None,
            order_note: None,
        };

        let input = validate_create_order_request(request).expect("request must be valid");
        assert_eq!(input.shipping.country_code, "JP");
    }

    #[test]
    fn validate_create_order_request_normalizes_bcp47_locale_to_route_code() {
        let mut request = valid_app_create_order_request();
        request.locale = "zh-Hant".to_owned();
        request.contact.preferred_locale = "zh_TW".to_owned();

        let input = validate_create_order_request(request).expect("request must be valid");

        assert_eq!(input.locale, "zhtw");
        assert_eq!(input.contact.preferred_locale, "zhtw");
    }

    #[test]
    fn m4_t07_create_order_request_rejects_unsupported_locale_values() {
        let mut request = valid_app_create_order_request();
        request.locale = "xx".to_owned();
        assert_create_order_error_contains(request, "locale must map to a supported route code");

        let mut request = valid_app_create_order_request();
        request.contact.preferred_locale = "../ja".to_owned();
        assert_create_order_error_contains(
            request,
            "contact.preferred_locale must map to a supported route code",
        );
    }

    #[test]
    fn create_order_request_deserializes_legacy_web_payload_without_app_fields() {
        let request = serde_json::from_value::<CreateOrderRequest>(json!({
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
                "country_code": "jp",
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
        }))
        .expect("legacy web payload should deserialize");

        let input = validate_create_order_request(request).expect("legacy web payload is valid");

        assert_eq!(input.channel, "web");
        assert!(input.seal.ai_generation_id.is_none());
        assert!(input.seal.ai_variant_id.is_none());
        assert!(input.seal.preview_image.is_none());
        assert!(input.seal.style.is_none());
        assert!(input.customer_confirmation.is_none());
        assert!(input.order_note.is_none());
    }

    #[test]
    fn m13_t03_accepts_legacy_web_payload_without_app_order_fields() {
        let request = serde_json::from_value::<CreateOrderRequest>(json!({
            "channel": "web",
            "locale": "ja",
            "idempotency_key": "legacy_web_key_123",
            "terms_agreed": true,
            "seal": {
                "line1": "佐藤",
                "line2": "",
                "shape": "round",
                "font_key": "zen_maru_gothic"
            },
            "listing_id": "rose_quartz_01",
            "shipping": {
                "country_code": "jp",
                "recipient_name": "佐藤 花子",
                "phone": "09000001111",
                "postal_code": "1000001",
                "state": "東京都",
                "city": "千代田区",
                "address_line1": "1-1-1",
                "address_line2": ""
            },
            "contact": {
                "email": "hanako@example.com",
                "preferred_locale": "ja"
            }
        }))
        .expect("legacy web payload should deserialize without app fields");

        let input =
            validate_create_order_request(request).expect("legacy web payload must stay valid");

        assert_eq!(input.channel, "web");
        assert_eq!(input.seal.line1, "佐藤");
        assert_eq!(input.seal.line2, "");
        assert_eq!(input.seal.shape, "round");
        assert!(input.seal.ai_generation_id.is_none());
        assert!(input.seal.ai_variant_id.is_none());
        assert!(input.seal.preview_image.is_none());
        assert!(input.seal.style.is_none());
        assert!(input.customer_confirmation.is_none());
    }

    #[test]
    fn validate_create_order_request_accepts_app_payload_with_ai_metadata() {
        let input =
            validate_create_order_request(valid_app_create_order_request()).expect("request valid");

        assert_eq!(input.channel, "app");
        assert_eq!(input.seal.font_key, "ai_generated_seal");
        assert_eq!(
            input.seal.ai_generation_id.as_deref(),
            Some("seal_request_001")
        );
        assert_eq!(
            input.seal.ai_variant_id.as_deref(),
            Some("seal_variant_001")
        );
        let preview_image = input.seal.preview_image.as_ref().expect("preview image");
        assert_eq!(
            preview_image.storage_path,
            "seal_designs/seal_request_001/seal_variant_001.png"
        );
        let style = input.seal.style.as_ref().expect("style");
        assert_eq!(style.name, "elegant");
        assert_eq!(style.stroke_weight, "standard");
        assert_eq!(style.balance, "balanced");
        assert_eq!(
            input
                .customer_confirmation
                .as_ref()
                .map(|value| value.confirmed_seal_text.as_str()),
            Some("美空")
        );
        assert_eq!(input.order_note.as_deref(), Some("Optional note"));
    }

    #[test]
    fn m13_t03_rejects_app_payload_missing_each_app_required_field() {
        let mut missing_ai_generation_id = valid_app_create_order_request();
        missing_ai_generation_id.seal.ai_generation_id = None;
        assert_create_order_error_contains(
            missing_ai_generation_id,
            "seal.ai_generation_id is required for app channel",
        );

        let mut missing_ai_variant_id = valid_app_create_order_request();
        missing_ai_variant_id.seal.ai_variant_id = None;
        assert_create_order_error_contains(
            missing_ai_variant_id,
            "seal.ai_variant_id is required for app channel",
        );

        let mut missing_preview_image = valid_app_create_order_request();
        missing_preview_image.seal.preview_image = None;
        assert_create_order_error_contains(
            missing_preview_image,
            "seal.preview_image is required for app channel",
        );

        let mut missing_style = valid_app_create_order_request();
        missing_style.seal.style = None;
        assert_create_order_error_contains(missing_style, "seal.style is required for app channel");

        let mut missing_customer_confirmation = valid_app_create_order_request();
        missing_customer_confirmation.customer_confirmation = None;
        assert_create_order_error_contains(
            missing_customer_confirmation,
            "customer_confirmation is required for app channel",
        );
    }

    #[test]
    fn validate_create_order_request_rejects_app_payload_without_customer_confirmation() {
        let mut request = valid_app_create_order_request();
        request.customer_confirmation = None;

        assert!(validate_create_order_request(request).is_err());
    }

    #[test]
    fn validate_create_order_request_rejects_app_payload_without_preview_storage_path_prefix() {
        let mut request = valid_app_create_order_request();
        request
            .seal
            .preview_image
            .as_mut()
            .expect("preview image")
            .storage_path = "https://example.com/seal.png".to_owned();

        assert!(validate_create_order_request(request).is_err());
    }

    #[test]
    fn validate_create_order_request_rejects_app_payload_with_fine_stroke() {
        let mut request = valid_app_create_order_request();
        request.seal.style.as_mut().expect("style").stroke_weight = "fine".to_owned();

        assert!(validate_create_order_request(request).is_err());
    }

    #[test]
    fn build_order_fields_saves_app_ai_metadata_and_customer_confirmation() {
        let input =
            validate_create_order_request(valid_app_create_order_request()).expect("request valid");
        let font = Font {
            key: "ai_generated_seal".to_owned(),
            label: "AI generated seal preview".to_owned(),
            font_family: "system".to_owned(),
            kanji_style: "japanese".to_owned(),
            version: 1,
        };
        let country = Country {
            code: "US".to_owned(),
            label_i18n: HashMap::from([("en".to_owned(), "United States".to_owned())]),
            shipping_fee_by_currency: HashMap::from([("USD".to_owned(), 600)]),
            version: 1,
        };
        let now = DateTime::parse_from_rfc3339("2026-05-21T11:30:00Z")
            .expect("timestamp")
            .with_timezone(&Utc);

        let fields = build_order_fields(
            &input,
            &font,
            None,
            &country,
            "HF-20260521-0001",
            12000,
            600,
            0,
            0,
            12600,
            "USD",
            now,
        );

        let seal = read_map_field(&fields, "seal");
        assert_eq!(
            read_string_field(&seal, "ai_generation_id"),
            "seal_request_001"
        );
        assert_eq!(
            read_string_field(&seal, "ai_variant_id"),
            "seal_variant_001"
        );
        let preview_image = read_map_field(&seal, "preview_image");
        assert_eq!(
            read_string_field(&preview_image, "storage_path"),
            "seal_designs/seal_request_001/seal_variant_001.png"
        );
        let style = read_map_field(&seal, "style");
        assert_eq!(read_string_field(&style, "name"), "elegant");
        assert_eq!(read_string_field(&style, "stroke_weight"), "standard");
        assert_eq!(read_string_field(&style, "balance"), "balanced");
        let customer_confirmation = read_map_field(&fields, "customer_confirmation");
        assert_eq!(
            read_bool_field(&customer_confirmation, "kanji_and_design"),
            Some(true)
        );
        assert_eq!(
            read_bool_field(&customer_confirmation, "custom_made_policy"),
            Some(true)
        );
        assert_eq!(
            read_timestamp_field(&customer_confirmation, "confirmed_at"),
            Some(now)
        );
        assert_eq!(
            read_string_field(&customer_confirmation, "confirmed_seal_text"),
            "美空"
        );
        assert_eq!(read_string_field(&fields, "order_note"), "Optional note");
    }

    #[test]
    fn validate_create_order_request_rejects_missing_listing_id() {
        let request = CreateOrderRequest {
            channel: "web".to_owned(),
            locale: "ja".to_owned(),
            idempotency_key: "demo_key_123".to_owned(),
            terms_agreed: true,
            seal: CreateOrderSealRequest {
                line1: "田中".to_owned(),
                line2: "太郎".to_owned(),
                shape: "square".to_owned(),
                font_key: "zen_maru_gothic".to_owned(),
                ai_generation_id: None,
                ai_variant_id: None,
                preview_image: None,
                style: None,
            },
            listing_id: None,
            shipping: CreateOrderShippingRequest {
                country_code: "JP".to_owned(),
                recipient_name: "田中 太郎".to_owned(),
                phone: "09000001111".to_owned(),
                postal_code: "1000001".to_owned(),
                state: "東京都".to_owned(),
                city: "千代田区".to_owned(),
                address_line1: "1-1-1".to_owned(),
                address_line2: "".to_owned(),
            },
            contact: CreateOrderContactRequest {
                email: "taro@example.com".to_owned(),
                preferred_locale: "ja".to_owned(),
            },
            customer_confirmation: None,
            order_note: None,
        };

        assert!(validate_create_order_request(request).is_err());
    }

    #[test]
    fn validate_create_order_request_rejects_seal_whitespace() {
        let request = CreateOrderRequest {
            channel: "web".to_owned(),
            locale: "ja".to_owned(),
            idempotency_key: "demo_key_123".to_owned(),
            terms_agreed: true,
            seal: CreateOrderSealRequest {
                line1: "田 中".to_owned(),
                line2: "".to_owned(),
                shape: "square".to_owned(),
                font_key: "zen_maru_gothic".to_owned(),
                ai_generation_id: None,
                ai_variant_id: None,
                preview_image: None,
                style: None,
            },
            listing_id: Some("rose_quartz_01".to_owned()),
            shipping: CreateOrderShippingRequest {
                country_code: "JP".to_owned(),
                recipient_name: "田中 太郎".to_owned(),
                phone: "09000001111".to_owned(),
                postal_code: "1000001".to_owned(),
                state: "東京都".to_owned(),
                city: "千代田区".to_owned(),
                address_line1: "1-1-1".to_owned(),
                address_line2: "".to_owned(),
            },
            contact: CreateOrderContactRequest {
                email: "taro@example.com".to_owned(),
                preferred_locale: "ja".to_owned(),
            },
            customer_confirmation: None,
            order_note: None,
        };

        assert!(validate_create_order_request(request).is_err());
    }

    #[test]
    fn validate_create_stripe_checkout_session_request_accepts_valid_payload() {
        let request = CreateStripeCheckoutSessionRequest {
            order_id: "order_1".to_owned(),
            customer_email: Some("buyer@example.com".to_owned()),
            return_to_app: Some(true),
        };

        let input = validate_create_stripe_checkout_session_request(request)
            .expect("request must be valid");
        assert_eq!(input.order_id, "order_1");
        assert_eq!(input.customer_email, "buyer@example.com");
        assert!(input.return_to_app);
    }

    #[test]
    fn validate_create_stripe_checkout_session_request_rejects_invalid_email() {
        let request = CreateStripeCheckoutSessionRequest {
            order_id: "order_1".to_owned(),
            customer_email: Some("invalid".to_owned()),
            return_to_app: None,
        };

        assert!(validate_create_stripe_checkout_session_request(request).is_err());
    }

    #[test]
    fn validate_lookup_order_request_normalizes_order_no_and_requires_email() {
        let input = validate_lookup_order_request(LookupOrderRequest {
            order_no: " hf-20260521-0001 ".to_owned(),
            email: " customer@example.com ".to_owned(),
        })
        .expect("lookup request must be valid");

        assert_eq!(input.order_no, "HF-20260521-0001");
        assert_eq!(input.email, "customer@example.com");
        assert!(
            validate_lookup_order_request(LookupOrderRequest {
                order_no: "HF-20260521-0001".to_owned(),
                email: "invalid".to_owned(),
            })
            .is_err()
        );
    }

    #[test]
    fn order_lookup_query_request_filters_by_order_no() {
        let query = order_lookup_query_request("HF-20260521-0001");
        let structured_query = query
            .structured_query
            .as_ref()
            .expect("structured query must be present");

        assert_eq!(structured_query["from"][0]["collectionId"], "orders");
        assert_eq!(
            structured_query["where"]["fieldFilter"]["field"]["fieldPath"],
            "order_no"
        );
        assert_eq!(
            structured_query["where"]["fieldFilter"]["value"]["stringValue"],
            "HF-20260521-0001"
        );
    }

    fn lookup_order_document() -> Document {
        let created_at = DateTime::parse_from_rfc3339("2026-05-21T11:00:00Z")
            .expect("created timestamp")
            .with_timezone(&Utc);
        let updated_at = DateTime::parse_from_rfc3339("2026-05-21T11:15:00Z")
            .expect("updated timestamp")
            .with_timezone(&Utc);
        let shipped_at = DateTime::parse_from_rfc3339("2026-05-22T03:00:00Z")
            .expect("shipped timestamp")
            .with_timezone(&Utc);

        Document {
            name: Some("projects/demo/databases/(default)/documents/orders/order_001".to_owned()),
            fields: btree_from_pairs(vec![
                ("order_no", fs_string("HF-20260521-0001")),
                ("locale", fs_string("en")),
                ("status", fs_string("paid")),
                (
                    "contact",
                    fs_map(btree_from_pairs(vec![(
                        "email",
                        fs_string("customer@example.com"),
                    )])),
                ),
                (
                    "payment",
                    fs_map(btree_from_pairs(vec![
                        ("status", fs_string("paid")),
                        ("checkout_session_id", fs_string("cs_test_xxx")),
                        ("intent_id", fs_string("pi_xxx")),
                    ])),
                ),
                (
                    "fulfillment",
                    fs_map(btree_from_pairs(vec![
                        ("status", fs_string("shipped")),
                        ("carrier", fs_string("Yamato")),
                        ("tracking_no", fs_string("1234567890")),
                        ("shipped_at", fs_timestamp(shipped_at)),
                    ])),
                ),
                (
                    "pricing",
                    fs_map(btree_from_pairs(vec![
                        ("total", fs_int(18600)),
                        ("currency", fs_string("JPY")),
                    ])),
                ),
                (
                    "seal",
                    fs_map(btree_from_pairs(vec![
                        ("line1", fs_string("美")),
                        ("line2", fs_string("空")),
                        (
                            "preview_image",
                            fs_map(btree_from_pairs(vec![(
                                "download_url",
                                fs_string("https://example.test/seal.png"),
                            )])),
                        ),
                    ])),
                ),
                (
                    "customer_confirmation",
                    fs_map(btree_from_pairs(vec![(
                        "confirmed_seal_text",
                        fs_string("美空"),
                    )])),
                ),
                (
                    "listing",
                    fs_map(btree_from_pairs(vec![
                        ("key", fs_string("stone_listing_001")),
                        (
                            "title_i18n",
                            fs_map(btree_from_pairs(vec![
                                ("ja", fs_string("ソフトピンクローズクォーツ印材")),
                                ("en", fs_string("Soft Pink Rose Quartz Seal Stone")),
                            ])),
                        ),
                    ])),
                ),
                ("created_at", fs_timestamp(created_at)),
                ("updated_at", fs_timestamp(updated_at)),
            ]),
            ..Document::default()
        }
    }

    fn order_status_result_fixture(
        order_id: &str,
        order_no: &str,
        status: &str,
        payment_status: &str,
    ) -> OrderStatusResult {
        OrderStatusResult {
            order_id: order_id.to_owned(),
            order_no: order_no.to_owned(),
            status: status.to_owned(),
            payment_status: payment_status.to_owned(),
            checkout_session_id: "cs_test_xxx".to_owned(),
            payment_intent_id: "pi_xxx".to_owned(),
            fulfillment_status: "pending".to_owned(),
            fulfillment_carrier: String::new(),
            fulfillment_tracking_no: String::new(),
            production_status: "not_started".to_owned(),
            shipping_status: "not_shipped".to_owned(),
            total: 18600,
            currency: "JPY".to_owned(),
            updated_at: None,
        }
    }

    #[test]
    fn order_lookup_result_matches_email_and_builds_limited_response() {
        let document = lookup_order_document();

        assert!(order_lookup_result_from_document(&document, "other@example.com").is_none());

        let result = order_lookup_result_from_document(&document, "CUSTOMER@example.com")
            .expect("matching email should return lookup result");
        let response = order_lookup_response_json(&result);

        assert_eq!(response["order_id"], "order_001");
        assert_eq!(response["order_no"], "HF-20260521-0001");
        assert_eq!(response["created_at"], "2026-05-21T11:00:00Z");
        assert_eq!(response["status"], "paid");
        assert_eq!(response["payment_status"], "paid");
        assert_eq!(response["payment"]["checkout_session_id"], "cs_test_xxx");
        assert_eq!(response["fulfillment_status"], "shipped");
        assert_eq!(response["fulfillment"]["carrier"], "Yamato");
        assert_eq!(response["fulfillment"]["tracking_no"], "1234567890");
        assert_eq!(
            response["fulfillment"]["shipped_at"],
            "2026-05-22T03:00:00Z"
        );
        assert_eq!(response["tracking_number"], "1234567890");
        assert_eq!(response["seal"]["confirmed_seal_text"], "美空");
        assert_eq!(
            response["seal"]["preview_image_url"],
            "https://example.test/seal.png"
        );
        assert_eq!(response["listing"]["id"], "stone_listing_001");
        assert_eq!(
            response["listing"]["title"],
            "Soft Pink Rose Quartz Seal Stone"
        );
        assert_eq!(response["pricing"]["total"], 18600);
        assert_eq!(response["updated_at"], "2026-05-21T11:15:00Z");
    }

    #[test]
    fn m13_t04_order_lookup_covers_statuses_and_email_boundaries() {
        let document = lookup_order_document();

        assert!(order_lookup_result_from_document(&document, "other@example.com").is_none());

        let mut missing_id_document = lookup_order_document();
        missing_id_document.name = None;
        assert!(
            order_lookup_result_from_document(&missing_id_document, "customer@example.com")
                .is_none()
        );

        for (status, payment_status) in [
            ("paid", "paid"),
            ("pending_payment", "unpaid"),
            ("canceled", "failed"),
        ] {
            let result = OrderLookupResult {
                status: order_status_result_fixture(
                    "order_001",
                    "HF-20260521-0001",
                    status,
                    payment_status,
                ),
                created_at: None,
                seal_confirmed_text: "美空".to_owned(),
                seal_preview_image_url: String::new(),
                listing_id: "stone_listing_001".to_owned(),
                listing_title: "Soft Pink Rose Quartz Seal Stone".to_owned(),
                shipped_at: None,
            };
            let response = order_lookup_response_json(&result);

            assert_eq!(response["status"], status);
            assert_eq!(response["order_status"], status);
            assert_eq!(response["payment_status"], payment_status);
            assert_eq!(response["payment"]["status"], payment_status);
            assert_eq!(response["seal"]["confirmed_seal_text"], "美空");
            assert_eq!(response["listing"]["id"], "stone_listing_001");
        }
    }

    #[test]
    fn order_status_response_includes_nested_and_flat_fields() {
        let updated_at = DateTime::parse_from_rfc3339("2026-05-21T11:15:00Z")
            .expect("fixture timestamp")
            .with_timezone(&Utc);
        let response = order_status_response_json(&OrderStatusResult {
            order_id: "order_001".to_owned(),
            order_no: "HF-20260521-0001".to_owned(),
            status: "paid".to_owned(),
            payment_status: "paid".to_owned(),
            checkout_session_id: "cs_test_xxx".to_owned(),
            payment_intent_id: "pi_xxx".to_owned(),
            fulfillment_status: "pending".to_owned(),
            fulfillment_carrier: String::new(),
            fulfillment_tracking_no: String::new(),
            production_status: "not_started".to_owned(),
            shipping_status: "not_shipped".to_owned(),
            total: 18600,
            currency: "JPY".to_owned(),
            updated_at: Some(updated_at),
        });

        assert_eq!(response["order_id"], "order_001");
        assert_eq!(response["order_no"], "HF-20260521-0001");
        assert_eq!(response["status"], "paid");
        assert_eq!(response["order_status"], "paid");
        assert_eq!(response["payment"]["status"], "paid");
        assert_eq!(response["payment_status"], "paid");
        assert_eq!(response["payment"]["checkout_session_id"], "cs_test_xxx");
        assert_eq!(response["payment"]["payment_intent_id"], "pi_xxx");
        assert_eq!(response["fulfillment"]["status"], "pending");
        assert!(response["fulfillment"]["carrier"].is_null());
        assert!(response["fulfillment"]["tracking_no"].is_null());
        assert_eq!(response["pricing"]["total"], 18600);
        assert_eq!(response["pricing"]["currency"], "JPY");
        assert_eq!(response["production_status"], "not_started");
        assert_eq!(response["shipping_status"], "not_shipped");
        assert!(response["tracking_number"].is_null());
        assert_eq!(response["updated_at"], "2026-05-21T11:15:00Z");
    }

    #[test]
    fn order_status_response_serializes_pending_without_optional_payment_refs() {
        let response = order_status_response_json(&OrderStatusResult {
            order_id: "order_pending".to_owned(),
            order_no: "HF-20260521-0002".to_owned(),
            status: "pending_payment".to_owned(),
            payment_status: "unpaid".to_owned(),
            checkout_session_id: String::new(),
            payment_intent_id: String::new(),
            fulfillment_status: "pending".to_owned(),
            fulfillment_carrier: String::new(),
            fulfillment_tracking_no: String::new(),
            production_status: "not_started".to_owned(),
            shipping_status: "not_shipped".to_owned(),
            total: 18600,
            currency: "JPY".to_owned(),
            updated_at: None,
        });

        assert_eq!(response["order_id"], "order_pending");
        assert_eq!(response["order_status"], "pending_payment");
        assert_eq!(response["payment"]["status"], "unpaid");
        assert!(response["payment"]["checkout_session_id"].is_null());
        assert!(response["payment"]["payment_intent_id"].is_null());
        assert_eq!(response["fulfillment"]["status"], "pending");
        assert!(response["updated_at"].is_null());
    }

    #[test]
    fn m13_t04_order_status_response_covers_paid_pending_and_failed() {
        for (status, payment_status) in [
            ("paid", "paid"),
            ("pending_payment", "unpaid"),
            ("canceled", "failed"),
        ] {
            let response = order_status_response_json(&order_status_result_fixture(
                "order_001",
                "HF-20260521-0001",
                status,
                payment_status,
            ));

            assert_eq!(response["status"], status);
            assert_eq!(response["order_status"], status);
            assert_eq!(response["payment"]["status"], payment_status);
            assert_eq!(response["payment_status"], payment_status);
            assert_eq!(response["fulfillment"]["status"], "pending");
            assert_eq!(response["pricing"]["total"], 18600);
        }
    }

    #[tokio::test]
    async fn m13_t04_order_not_found_response_uses_public_error_shape() {
        let response = error_response(StatusCode::NOT_FOUND, "order_not_found", "order not found");

        assert_eq!(response.status(), StatusCode::NOT_FOUND);

        let body = axum::body::to_bytes(response.into_body(), MAX_REQUEST_BODY_BYTES)
            .await
            .expect("response body");
        let payload =
            serde_json::from_slice::<JsonValue>(&body).expect("response body should be JSON");

        assert_eq!(payload["error"]["code"], "order_not_found");
        assert_eq!(payload["error"]["message"], "order not found");
    }

    fn stripe_webhook_order_fields(
        status: &str,
        payment_status: &str,
    ) -> BTreeMap<String, JsonValue> {
        btree_from_pairs(vec![
            ("status", fs_string(status)),
            (
                "payment",
                fs_map(btree_from_pairs(vec![(
                    "status",
                    fs_string(payment_status),
                )])),
            ),
            (
                "listing",
                fs_map(btree_from_pairs(vec![(
                    "key",
                    fs_string("stone_listing_001"),
                )])),
            ),
        ])
    }

    fn stripe_webhook_event_fixture(event_type: &str) -> StripeWebhookEvent {
        let is_checkout_completed = event_type == "checkout.session.completed";
        let is_checkout_expired = event_type == "checkout.session.expired";
        StripeWebhookEvent {
            provider_event_id: format!("evt_{}", event_type.replace('.', "_")),
            event_type: event_type.to_owned(),
            checkout_session_id: if is_checkout_completed || is_checkout_expired {
                "cs_test_001".to_owned()
            } else {
                String::new()
            },
            checkout_session_payment_status: if is_checkout_completed {
                "paid".to_owned()
            } else {
                String::new()
            },
            checkout_session_status: if is_checkout_completed {
                "complete".to_owned()
            } else if is_checkout_expired {
                "expired".to_owned()
            } else {
                String::new()
            },
            payment_intent_id: "pi_test_001".to_owned(),
            order_id: "order_001".to_owned(),
        }
    }

    #[test]
    fn m13_t05_webhook_mutation_reflects_success_cancel_failure_and_refund() {
        let now = DateTime::parse_from_rfc3339("2026-05-21T11:15:00Z")
            .expect("fixture timestamp")
            .with_timezone(&Utc);

        for (
            event_type,
            current_status,
            current_payment_status,
            expected_status,
            expected_payment_status,
            expected_event_type,
            expected_listing_status,
        ) in [
            (
                "payment_intent.succeeded",
                "pending_payment",
                "unpaid",
                "paid",
                "paid",
                "payment_paid",
                Some("sold"),
            ),
            (
                "checkout.session.completed",
                "pending_payment",
                "unpaid",
                "paid",
                "paid",
                "payment_paid",
                Some("sold"),
            ),
            (
                "payment_intent.canceled",
                "pending_payment",
                "unpaid",
                "canceled",
                "failed",
                "payment_failed",
                Some("published"),
            ),
            (
                "payment_intent.payment_failed",
                "pending_payment",
                "unpaid",
                "canceled",
                "failed",
                "payment_failed",
                Some("published"),
            ),
            (
                "checkout.session.expired",
                "pending_payment",
                "unpaid",
                "canceled",
                "failed",
                "payment_failed",
                Some("published"),
            ),
            (
                "charge.refunded",
                "paid",
                "paid",
                "refunded",
                "refunded",
                "payment_refunded",
                None,
            ),
        ] {
            let mut fields = stripe_webhook_order_fields(current_status, current_payment_status);
            let event = stripe_webhook_event_fixture(event_type);

            let mutation = apply_stripe_webhook_to_order_fields(&mut fields, &event, now);

            let payment = read_map_field(&fields, "payment");
            assert_eq!(read_string_field(&fields, "status"), expected_status);
            assert_eq!(
                read_string_field(&payment, "status"),
                expected_payment_status
            );
            assert_eq!(read_string_field(&payment, "intent_id"), "pi_test_001");
            assert_eq!(
                read_string_field(&payment, "last_event_id"),
                format!("evt_{}", event_type.replace('.', "_"))
            );
            assert_eq!(read_timestamp_field(&fields, "updated_at"), Some(now));
            assert_eq!(
                read_timestamp_field(&fields, "status_updated_at"),
                Some(now)
            );

            assert_eq!(
                read_string_field(&mutation.event_fields, "type"),
                expected_event_type
            );
            assert_eq!(
                read_string_field(&mutation.event_fields, "before_status"),
                current_status
            );
            assert_eq!(
                read_string_field(&mutation.event_fields, "after_status"),
                expected_status
            );
            let payload = read_map_field(&mutation.event_fields, "payload");
            assert_eq!(read_string_field(&payload, "event_type"), event_type);
            assert_eq!(
                read_string_field(&payload, "payment_intent_id"),
                "pi_test_001"
            );
            assert_eq!(mutation.listing_status, expected_listing_status);
        }
    }

    #[test]
    fn m13_t05_parse_checkout_session_completed_keeps_order_metadata_for_webhook() {
        let payload = json!({
            "id": "evt_checkout_completed",
            "type": "checkout.session.completed",
            "data": {
                "object": {
                    "id": "cs_test_001",
                    "payment_intent": "pi_test_001",
                    "payment_status": "paid",
                    "status": "complete",
                    "metadata": {
                        "order_id": "order_001"
                    }
                }
            }
        });

        let event = parse_stripe_event(payload).expect("checkout session event must parse");

        assert_eq!(event.provider_event_id, "evt_checkout_completed");
        assert_eq!(event.event_type, "checkout.session.completed");
        assert_eq!(event.checkout_session_id, "cs_test_001");
        assert_eq!(event.checkout_session_payment_status, "paid");
        assert_eq!(event.checkout_session_status, "complete");
        assert_eq!(event.payment_intent_id, "pi_test_001");
        assert_eq!(event.order_id, "order_001");
        assert_eq!(stripe_transition(&event), ("paid", "paid", "payment_paid"));
    }

    #[test]
    fn checkout_session_reconciliation_only_targets_unpaid_pending_orders() {
        let mut status = order_status_result_fixture(
            "order_001",
            "HF-20260602-0001",
            "pending_payment",
            "unpaid",
        );
        status.checkout_session_id = "cs_test_001".to_owned();
        assert!(order_status_needs_stripe_checkout_reconciliation(&status));

        status.payment_status = "paid".to_owned();
        assert!(!order_status_needs_stripe_checkout_reconciliation(&status));

        status.payment_status = "unpaid".to_owned();
        status.status = "paid".to_owned();
        assert!(!order_status_needs_stripe_checkout_reconciliation(&status));

        status.status = "pending_payment".to_owned();
        status.checkout_session_id.clear();
        assert!(!order_status_needs_stripe_checkout_reconciliation(&status));
    }

    #[test]
    fn checkout_session_snapshot_maps_paid_and_expired_states_to_order_events() {
        let paid = StripeCheckoutSessionSnapshot {
            session_id: "cs_test_001".to_owned(),
            payment_status: "paid".to_owned(),
            session_status: "complete".to_owned(),
            payment_intent_id: "pi_test_001".to_owned(),
            order_id: "order_001".to_owned(),
        };
        let paid_event = stripe_event_from_checkout_session_snapshot(&paid, "fallback_order")
            .expect("paid checkout session should produce a payment event");
        assert_eq!(paid_event.event_type, "checkout.session.completed");
        assert_eq!(
            stripe_transition(&paid_event),
            ("paid", "paid", "payment_paid")
        );

        let expired = StripeCheckoutSessionSnapshot {
            session_id: "cs_test_002".to_owned(),
            payment_status: "unpaid".to_owned(),
            session_status: "expired".to_owned(),
            payment_intent_id: String::new(),
            order_id: String::new(),
        };
        let expired_event = stripe_event_from_checkout_session_snapshot(&expired, "order_002")
            .expect("expired checkout session should produce a payment event");
        assert_eq!(expired_event.order_id, "order_002");
        assert_eq!(expired_event.event_type, "checkout.session.expired");
        assert_eq!(
            stripe_transition(&expired_event),
            ("failed", "canceled", "payment_failed")
        );

        let open = StripeCheckoutSessionSnapshot {
            session_id: "cs_test_003".to_owned(),
            payment_status: "unpaid".to_owned(),
            session_status: "open".to_owned(),
            payment_intent_id: String::new(),
            order_id: "order_003".to_owned(),
        };
        assert!(stripe_event_from_checkout_session_snapshot(&open, "order_003").is_none());
    }

    #[test]
    fn stripe_webhook_rejects_outdated_signature_timestamp() {
        let payload = br#"{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1","metadata":{"order_id":"order_1"}}}}"#;
        let error = construct_stripe_webhook_event(payload, "t=1700000000,v1=00", "whsec_test")
            .expect_err("signature timestamp must be rejected");
        assert!(matches!(
            error,
            StripeWebhookError::TimestampOutsideTolerance
        ));
    }

    #[test]
    fn stripe_webhook_accepts_valid_signature() {
        let payload = br#"{"id":"evt_1","type":"payment_intent.succeeded","data":{"object":{"id":"pi_1","metadata":{"order_id":"order_1"}}}}"#;
        let secret = "whsec_test";
        let timestamp = Utc::now().timestamp();
        let mut signed_payload = timestamp.to_string().into_bytes();
        signed_payload.push(b'.');
        signed_payload.extend_from_slice(payload);

        let mut mac =
            HmacSha256::new_from_slice(secret.as_bytes()).expect("HMAC accepts any key length");
        mac.update(&signed_payload);
        let signature = hex::encode(mac.finalize().into_bytes());
        let header = format!("t={timestamp},v1={signature}");

        let event = construct_stripe_webhook_event(payload, &header, secret)
            .expect("valid signature should be accepted");
        assert_eq!(event.get("id").and_then(JsonValue::as_str), Some("evt_1"));
    }

    #[test]
    fn parse_stripe_event_extracts_core_fields() {
        let payload = br#"{"id":"evt_1","type":"payment_intent.succeeded","created":1770638400,"livemode":false,"object":"event","pending_webhooks":1,"data":{"object":{"id":"pi_1","metadata":{"order_id":"order_1"}}}}"#;
        let payload = serde_json::from_slice::<JsonValue>(payload).expect("payload must parse");
        let event = parse_stripe_event(payload).expect("event must map");
        assert_eq!(event.provider_event_id, "evt_1");
        assert_eq!(event.payment_intent_id, "pi_1");
        assert_eq!(event.order_id, "order_1");
    }

    #[test]
    fn stripe_transition_treats_expired_checkout_session_as_canceled() {
        let event = stripe_webhook_event_fixture("checkout.session.expired");
        assert_eq!(
            stripe_transition(&event),
            ("failed", "canceled", "payment_failed")
        );
    }

    fn valid_seal_designs_request() -> GenerateSealDesignsRequest {
        GenerateSealDesignsRequest {
            input_name: "Michael".to_owned(),
            kanji: "美空".to_owned(),
            shape: "square".to_owned(),
            style: "elegant".to_owned(),
            stroke_weight: "standard".to_owned(),
            balance: "balanced".to_owned(),
            variant_count: Some(3),
            attempt_number: Some(1),
            previous_recipes: None,
            generation_rules: Some(SealGenerationRulesRequest {
                max_characters: Some(2),
                avoid_complex_characters: Some(true),
                engraving_friendly: Some(true),
                avoid_thin_lines: Some(true),
                avoid_decorative_details: Some(true),
                plain_background: Some(true),
            }),
        }
    }

    fn valid_seal_design_recipe_dto() -> SealDesignRecipeDto {
        SealDesignRecipeDto {
            font_profile: "formal_serif".to_owned(),
            impression: "elegant".to_owned(),
            weight: "standard".to_owned(),
            spacing: "balanced".to_owned(),
            texture: "subtle_ink".to_owned(),
            frame: "square_standard".to_owned(),
        }
    }

    fn valid_seal_design_recipe_variant(label: &str) -> SealDesignRecipeVariant {
        SealDesignRecipeVariant {
            label: label.to_owned(),
            recipe: validate_seal_design_recipe(valid_seal_design_recipe_dto())
                .expect("test recipe must be valid"),
        }
    }

    #[test]
    fn validate_generate_seal_designs_request_accepts_contract_payload() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");

        assert_eq!(input.input_name, "Michael");
        assert_eq!(input.kanji, "美空");
        assert_eq!(input.shape, SealShape::Square);
        assert_eq!(input.style, SealStyleName::Elegant);
        assert_eq!(input.stroke_weight, SealStrokeWeight::Standard);
        assert_eq!(input.balance, SealBalance::Balanced);
        assert_eq!(input.variant_count, DEFAULT_SEAL_DESIGN_VARIANT_COUNT);
        assert_eq!(input.attempt_number, 1);
        assert!(input.previous_recipes.is_empty());
        assert_eq!(input.generation_rules.max_characters, 2);
        assert!(input.generation_rules.engraving_friendly);
    }

    #[test]
    fn validate_generate_seal_designs_request_accepts_previous_recipes() {
        let mut request = valid_seal_designs_request();
        request.attempt_number = Some(2);
        request.previous_recipes = Some(vec![valid_seal_design_recipe_dto()]);

        let input =
            validate_generate_seal_designs_request(request).expect("request should be valid");

        assert_eq!(input.attempt_number, 2);
        assert_eq!(input.previous_recipes.len(), 1);
        assert_eq!(
            input.previous_recipes[0].font_profile,
            SealRecipeFontProfile::FormalSerif
        );

        let prompt = build_seal_designs_prompt(&input);
        assert!(prompt.contains("Regeneration attempt: 2"));
        assert!(prompt.contains("Avoid these previously shown full recipe combinations"));
        assert!(
            prompt.contains("formal_serif+elegant+standard+balanced+subtle_ink+square_standard")
        );
    }

    #[test]
    fn validate_generate_seal_designs_request_rejects_invalid_attempt_number() {
        let mut request = valid_seal_designs_request();
        request.attempt_number = Some(MAX_SEAL_GENERATION_ATTEMPTS + 1);

        let err =
            validate_generate_seal_designs_request(request).expect_err("invalid attempt must fail");

        assert!(err.to_string().contains("attempt_number must be between"));
    }

    #[test]
    fn ensure_fresh_seal_design_recipes_replaces_previous_visual_duplicates() {
        let mut request = valid_seal_designs_request();
        request.attempt_number = Some(2);
        request.previous_recipes = Some(vec![valid_seal_design_recipe_dto()]);
        let input =
            validate_generate_seal_designs_request(request).expect("request should be valid");
        let previous_signature = seal_recipe_visual_signature(input.previous_recipes[0]);

        let adjusted = ensure_fresh_seal_design_recipes(
            &input,
            vec![
                valid_seal_design_recipe_variant("Duplicate previous visual"),
                valid_seal_design_recipe_variant("Duplicate response visual"),
            ],
        );

        assert_eq!(adjusted.len(), 2);
        let signatures = adjusted
            .iter()
            .map(|variant| seal_recipe_visual_signature(variant.recipe))
            .collect::<HashSet<_>>();
        assert_eq!(signatures.len(), 2);
        assert!(!signatures.contains(&previous_signature));
    }

    #[test]
    fn m14_t02_validate_seal_design_recipe_accepts_allowed_values() {
        let recipe =
            validate_seal_design_recipe(valid_seal_design_recipe_dto()).expect("recipe is valid");

        assert_eq!(recipe.font_profile, SealRecipeFontProfile::FormalSerif);
        assert_eq!(recipe.font_profile.as_str(), "formal_serif");
        assert_eq!(recipe.impression, SealRecipeImpression::Elegant);
        assert_eq!(recipe.impression.as_str(), "elegant");
        assert_eq!(recipe.weight, SealRecipeWeight::Standard);
        assert_eq!(recipe.weight.as_str(), "standard");
        assert_eq!(recipe.spacing, SealRecipeSpacing::Balanced);
        assert_eq!(recipe.spacing.as_str(), "balanced");
        assert_eq!(recipe.texture, SealRecipeTexture::SubtleInk);
        assert_eq!(recipe.texture.as_str(), "subtle_ink");
        assert_eq!(recipe.frame, SealRecipeFrame::SquareStandard);
        assert_eq!(recipe.frame.as_str(), "square_standard");

        assert_eq!(
            parse_seal_recipe_font_profile(" classic_seal ")
                .expect("profile should parse")
                .as_str(),
            "classic_seal"
        );
        assert_eq!(
            parse_seal_recipe_texture("soft_bleed")
                .expect("texture should parse")
                .as_str(),
            "soft_bleed"
        );
        assert_eq!(
            parse_seal_recipe_frame("round_standard")
                .expect("frame should parse")
                .as_str(),
            "round_standard"
        );
    }

    #[test]
    fn m14_t02_validate_seal_design_recipe_rejects_disallowed_values() {
        let invalid_cases = vec![
            (
                "font_profile",
                SealDesignRecipeDto {
                    font_profile: "blackletter".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "font_profile must be one of formal_serif, soft_sans, bold_brush, classic_seal",
            ),
            (
                "impression",
                SealDesignRecipeDto {
                    impression: "luxury".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "impression must be one of traditional, elegant, soft, bold",
            ),
            (
                "weight",
                SealDesignRecipeDto {
                    weight: "fine".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "weight must be one of standard, bold",
            ),
            (
                "spacing",
                SealDesignRecipeDto {
                    spacing: "crowded".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "spacing must be one of airy, balanced, dense",
            ),
            (
                "texture",
                SealDesignRecipeDto {
                    texture: "paper".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "texture must be one of none, subtle_ink, soft_bleed",
            ),
            (
                "frame",
                SealDesignRecipeDto {
                    frame: "double_square".to_owned(),
                    ..valid_seal_design_recipe_dto()
                },
                "frame must be one of square_standard, round_standard",
            ),
        ];

        for (field, dto, expected) in invalid_cases {
            let err = validate_seal_design_recipe(dto).expect_err("invalid recipe value must fail");
            assert!(
                err.to_string().contains(expected),
                "{field} error should contain {expected}, got {err:#}"
            );
        }
    }

    #[test]
    fn m14_t02_recipe_variant_dto_rejects_unknown_fields() {
        let payload = json!({
            "label": "Formal balanced",
            "recipe": {
                "font_profile": "formal_serif",
                "impression": "traditional",
                "weight": "standard",
                "spacing": "balanced",
                "texture": "none",
                "frame": "square_standard",
                "palette": "red"
            }
        });

        let err = serde_json::from_value::<SealDesignRecipeVariantDto>(payload)
            .expect_err("unknown recipe fields must be rejected");
        assert!(err.to_string().contains("unknown field"));
    }

    #[test]
    fn m14_t02_validate_recipe_variant_trims_label_and_validates_recipe() {
        let variant = validate_seal_design_recipe_variant(SealDesignRecipeVariantDto {
            label: "  Formal balanced  ".to_owned(),
            recipe: SealDesignRecipeDto {
                font_profile: "soft_sans".to_owned(),
                impression: "soft".to_owned(),
                weight: "bold".to_owned(),
                spacing: "airy".to_owned(),
                texture: "soft_bleed".to_owned(),
                frame: "round_standard".to_owned(),
            },
        })
        .expect("variant should be valid");

        assert_eq!(variant.label, "Formal balanced");
        assert_eq!(variant.recipe.font_profile, SealRecipeFontProfile::SoftSans);
        assert_eq!(variant.recipe.impression, SealRecipeImpression::Soft);
        assert_eq!(variant.recipe.weight, SealRecipeWeight::Bold);
        assert_eq!(variant.recipe.spacing, SealRecipeSpacing::Airy);
        assert_eq!(variant.recipe.texture, SealRecipeTexture::SoftBleed);
        assert_eq!(variant.recipe.frame, SealRecipeFrame::RoundStandard);
    }

    #[test]
    fn validate_generate_seal_designs_request_rejects_non_three_variant_count() {
        let mut request = valid_seal_designs_request();
        request.variant_count = Some(2);

        let err = validate_generate_seal_designs_request(request)
            .expect_err("request must fail for non-MVP variant count");
        assert!(err.to_string().contains("variant_count must be 3"));
    }

    #[test]
    fn validate_generate_seal_designs_request_rejects_unsafe_generation_rules() {
        let mut request = valid_seal_designs_request();
        request.generation_rules = Some(SealGenerationRulesRequest {
            max_characters: Some(2),
            avoid_complex_characters: Some(true),
            engraving_friendly: Some(true),
            avoid_thin_lines: Some(false),
            avoid_decorative_details: Some(true),
            plain_background: Some(true),
        });

        let err = validate_generate_seal_designs_request(request)
            .expect_err("request must fail when thin lines are allowed");
        assert!(
            err.to_string()
                .contains("generation_rules.avoid_thin_lines must be true")
        );
    }

    #[test]
    fn m13_t07_seal_design_request_rejects_invalid_kanji_quality_inputs() {
        let mut too_many = valid_seal_designs_request();
        too_many.kanji = "美空翔".to_owned();
        let err = validate_generate_seal_designs_request(too_many).expect_err("3 kanji must fail");
        assert!(
            err.to_string()
                .contains("kanji must be 2 characters or fewer")
        );

        let mut non_han = valid_seal_designs_request();
        non_han.kanji = "M".to_owned();
        let err = validate_generate_seal_designs_request(non_han)
            .expect_err("latin text must not be accepted as seal kanji");
        assert!(
            err.to_string()
                .contains("kanji must contain only CJK Han characters")
        );

        let mut whitespace = valid_seal_designs_request();
        whitespace.kanji = "美 空".to_owned();
        let err = validate_generate_seal_designs_request(whitespace)
            .expect_err("kanji with whitespace must fail");
        assert!(
            err.to_string()
                .contains("kanji must not contain whitespace")
        );
    }

    #[test]
    fn m14_t03_parse_seal_design_recipes_from_gemini_text_accepts_markdown_json() {
        let payload = r#"
```json
{
  "variants": [
    {
      "label": "Elegant and balanced",
      "recipe": {
        "font_profile": "formal_serif",
        "impression": "elegant",
        "weight": "standard",
        "spacing": "balanced",
        "texture": "subtle_ink",
        "frame": "square_standard"
      }
    },
    {
      "label": "Soft spacing",
      "recipe": {
        "font_profile": "soft_sans",
        "impression": "soft",
        "weight": "standard",
        "spacing": "airy",
        "texture": "none",
        "frame": "round_standard"
      }
    },
    {
      "label": "Bold readable seal",
      "recipe": {
        "font_profile": "bold_brush",
        "impression": "bold",
        "weight": "bold",
        "spacing": "dense",
        "texture": "soft_bleed",
        "frame": "square_standard"
      }
    }
  ]
}
```
"#;

        let variants =
            parse_seal_design_recipes_from_gemini_text(payload, 3).expect("payload must parse");
        assert_eq!(variants.len(), 3);
        assert_eq!(variants[0].label, "Elegant and balanced");
        assert_eq!(
            variants[0].recipe.font_profile,
            SealRecipeFontProfile::FormalSerif
        );
        assert_eq!(variants[1].recipe.impression, SealRecipeImpression::Soft);
        assert_eq!(variants[2].recipe.weight, SealRecipeWeight::Bold);
        assert_eq!(variants[2].recipe.texture, SealRecipeTexture::SoftBleed);
    }

    #[test]
    fn m14_t03_parse_seal_design_recipes_rejects_unknown_keys() {
        let payload = r#"
{
  "variants": [
    {
      "label": "Elegant and balanced",
      "recipe": {
        "font_profile": "formal_serif",
        "impression": "elegant",
        "weight": "standard",
        "spacing": "balanced",
        "texture": "none",
        "frame": "square_standard",
        "palette": "red"
      }
    },
    {
      "label": "Soft spacing",
      "recipe": {
        "font_profile": "soft_sans",
        "impression": "soft",
        "weight": "standard",
        "spacing": "airy",
        "texture": "none",
        "frame": "round_standard"
      }
    },
    {
      "label": "Bold readable seal",
      "recipe": {
        "font_profile": "bold_brush",
        "impression": "bold",
        "weight": "bold",
        "spacing": "dense",
        "texture": "soft_bleed",
        "frame": "square_standard"
      }
    }
  ]
}
"#;

        let err = parse_seal_design_recipes_from_gemini_text(payload, 3)
            .expect_err("unknown keys must fail");
        let error_chain = format!("{err:#}");
        assert!(
            error_chain.contains("unknown field"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t03_parse_seal_design_recipes_rejects_disallowed_values() {
        let payload = r#"
{
  "variants": [
    {
      "label": "Elegant and balanced",
      "recipe": {
        "font_profile": "fantasy",
        "impression": "elegant",
        "weight": "standard",
        "spacing": "balanced",
        "texture": "none",
        "frame": "square_standard"
      }
    },
    {
      "label": "Soft spacing",
      "recipe": {
        "font_profile": "soft_sans",
        "impression": "soft",
        "weight": "standard",
        "spacing": "airy",
        "texture": "none",
        "frame": "round_standard"
      }
    },
    {
      "label": "Bold readable seal",
      "recipe": {
        "font_profile": "bold_brush",
        "impression": "bold",
        "weight": "bold",
        "spacing": "dense",
        "texture": "soft_bleed",
        "frame": "square_standard"
      }
    }
  ]
}
"#;

        let err = parse_seal_design_recipes_from_gemini_text(payload, 3)
            .expect_err("disallowed enum values must fail");
        assert!(
            err.to_string().contains("font_profile must be one of"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t03_parse_seal_design_recipes_rejects_count_mismatch() {
        let payload = r#"
{
  "variants": [
    {
      "label": "Elegant and balanced",
      "recipe": {
        "font_profile": "formal_serif",
        "impression": "elegant",
        "weight": "standard",
        "spacing": "balanced",
        "texture": "none",
        "frame": "square_standard"
      }
    }
  ]
}
"#;

        let err = parse_seal_design_recipes_from_gemini_text(payload, 3)
            .expect_err("variant count mismatch must fail");
        assert!(
            err.to_string()
                .contains("gemini returned 1 seal design recipes, expected 3"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t03_build_seal_designs_request_body_enables_recipe_schema_for_three_variants() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");

        let body = build_seal_designs_request_body(&input);
        assert_eq!(
            body["generationConfig"]["thinkingConfig"]["thinkingBudget"],
            json!(DEFAULT_GEMINI_THINKING_BUDGET)
        );
        assert_eq!(
            body["generationConfig"]["responseMimeType"],
            json!("application/json")
        );
        assert_eq!(body["generationConfig"]["temperature"], json!(0.35));
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["minItems"],
            json!(3)
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["maxItems"],
            json!(3)
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["items"]["required"],
            json!(["label", "recipe"])
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["items"]["properties"]
                ["recipe"]["additionalProperties"],
            json!(false)
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["items"]["properties"]
                ["recipe"]["properties"]["font_profile"]["enum"],
            json!(["formal_serif", "soft_sans", "bold_brush", "classic_seal"])
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["items"]["properties"]
                ["recipe"]["properties"]["frame"]["enum"],
            json!(["square_standard"])
        );
        assert!(body["generationConfig"]["responseModalities"].is_null());
        assert!(
            body["contents"][0]["parts"][0]["text"]
                .as_str()
                .expect("prompt must be text")
                .contains("Do not generate, describe, embed, or request final seal images")
        );
    }

    #[test]
    fn build_seal_design_variants_returns_canonical_paths() {
        let recipe_variants = vec![
            valid_seal_design_recipe_variant("Elegant and balanced"),
            valid_seal_design_recipe_variant("Soft spacing"),
            valid_seal_design_recipe_variant("Bold readable seal"),
        ];

        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &recipe_variants);

        assert_eq!(variants.len(), 3);
        assert_eq!(variants[0].id, "seal_variant_001");
        assert_eq!(variants[0].label, "Elegant and balanced");
        assert_eq!(
            variants[0].recipe.font_profile,
            SealRecipeFontProfile::FormalSerif
        );
        assert_eq!(
            variants[0].storage_path,
            "seal_designs/seal_request_001/seal_variant_001.png"
        );
        assert_eq!(
            variants[0].download_url,
            "https://storage.googleapis.com/hanko-assets/seal_designs/seal_request_001/seal_variant_001.png"
        );
        assert_eq!(variants[0].width, SEAL_DESIGN_IMAGE_SIZE);
        assert_eq!(variants[0].height, SEAL_DESIGN_IMAGE_SIZE);
    }

    #[test]
    fn m14_t07_round_shape_recipe_schema_allows_only_round_frame() {
        let mut request = valid_seal_designs_request();
        request.shape = "round".to_owned();
        let input = validate_generate_seal_designs_request(request).expect("request must be valid");

        let body = build_seal_designs_request_body(&input);

        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["variants"]["items"]["properties"]
                ["recipe"]["properties"]["frame"]["enum"],
            json!(["round_standard"])
        );
        let prompt = body["contents"][0]["parts"][0]["text"]
            .as_str()
            .expect("prompt must be text");
        assert!(prompt.contains("frame: round_standard only"));
        assert!(prompt.contains("\"frame\":\"round_standard\""));
    }

    #[test]
    fn m14_t07_renderer_generates_programmatic_png_images_for_recipe_variants() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");
        let recipe_variants = vec![
            valid_seal_design_recipe_variant("Elegant and balanced"),
            SealDesignRecipeVariant {
                label: "Soft spacing".to_owned(),
                recipe: SealDesignRecipe {
                    font_profile: SealRecipeFontProfile::SoftSans,
                    impression: SealRecipeImpression::Soft,
                    weight: SealRecipeWeight::Standard,
                    spacing: SealRecipeSpacing::Airy,
                    texture: SealRecipeTexture::None,
                    frame: SealRecipeFrame::SquareStandard,
                },
            },
            SealDesignRecipeVariant {
                label: "Bold readable".to_owned(),
                recipe: SealDesignRecipe {
                    font_profile: SealRecipeFontProfile::BoldBrush,
                    impression: SealRecipeImpression::Bold,
                    weight: SealRecipeWeight::Bold,
                    spacing: SealRecipeSpacing::Dense,
                    texture: SealRecipeTexture::None,
                    frame: SealRecipeFrame::SquareStandard,
                },
            },
        ];
        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &recipe_variants);

        let images = generate_seal_design_images_with_renderer(&input, &variants)
            .expect("programmatic renderer must generate png variants");

        assert_eq!(images.len(), variants.len());
        for image in images {
            assert_eq!(image.content_type, "image/png");
            validate_generated_seal_image(&image).expect("rendered image must be a valid png");
            let decoded = image::load_from_memory_with_format(&image.bytes, ImageFormat::Png)
                .expect("rendered png must decode");
            assert_eq!(decoded.width(), SEAL_DESIGN_IMAGE_SIZE as u32);
            assert_eq!(decoded.height(), SEAL_DESIGN_IMAGE_SIZE as u32);
        }
    }

    #[test]
    fn m14_t07_renderer_rejects_recipe_frame_that_does_not_match_requested_shape() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");
        let mut recipe_variant = valid_seal_design_recipe_variant("Mismatched round");
        recipe_variant.recipe.frame = SealRecipeFrame::RoundStandard;
        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &[recipe_variant]);

        let err = generate_seal_design_images_with_renderer(&input, &variants)
            .expect_err("mismatched recipe frame must fail");
        let err_chain = format!("{err:#}");

        assert!(
            err_chain.contains("recipe frame round_standard does not match requested shape square"),
            "unexpected error: {err_chain}"
        );
    }

    #[test]
    fn m14_t08_seal_design_failure_codes_match_app_error_states() {
        for phase in [
            SealDesignFailurePhase::RecipeGeneration,
            SealDesignFailurePhase::RecipeValidation,
        ] {
            assert_eq!(phase.error_code(), "gemini_generation_failed");
        }

        for phase in [
            SealDesignFailurePhase::Rendering,
            SealDesignFailurePhase::StorageUpload,
        ] {
            assert_eq!(phase.error_code(), "seal_generation_failed");
        }
    }

    #[test]
    fn m14_t08_seal_design_failure_response_uses_bad_gateway_status() {
        for phase in [
            SealDesignFailurePhase::RecipeGeneration,
            SealDesignFailurePhase::RecipeValidation,
            SealDesignFailurePhase::Rendering,
            SealDesignFailurePhase::StorageUpload,
        ] {
            let response = seal_design_failure_response(phase);
            assert_eq!(response.status(), StatusCode::BAD_GATEWAY);
            assert!(!phase.public_message().is_empty());
            assert!(!phase.log_name().is_empty());
        }
    }

    #[test]
    fn m14_t09_rejects_unapproved_recipe_values() {
        let mut dto = valid_seal_design_recipe_dto();
        dto.font_profile = "ai_invented_font".to_owned();

        let err = validate_seal_design_recipe(dto)
            .expect_err("unapproved recipe values must be rejected");

        assert!(
            err.to_string().contains("font_profile"),
            "unexpected error: {err:#}"
        );
    }

    #[test]
    fn m14_t09_generation_renderer_rejects_missing_glyph() {
        let recipe_variants = vec![valid_seal_design_recipe_variant("Missing glyph")];
        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &recipe_variants);
        let unsupported = find_unsupported_kanji_for_generation_test(&variants)
            .expect("test candidate set should include one unsupported CJK character");
        let mut input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");
        input.kanji = unsupported.to_string();

        let err = generate_seal_design_images_with_renderer(&input, &variants)
            .expect_err("unsupported selected kanji must fail before upload");
        let err_chain = format!("{err:#}");

        assert!(
            err_chain.contains("selected kanji could not be rendered by any approved real font"),
            "unexpected error: {err_chain}"
        );
    }

    #[test]
    fn m14_t09_generated_images_match_fixed_visual_contract() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");
        let recipe_variants = vec![
            valid_seal_design_recipe_variant("Formal"),
            SealDesignRecipeVariant {
                label: "Soft airy".to_owned(),
                recipe: SealDesignRecipe {
                    font_profile: SealRecipeFontProfile::SoftSans,
                    impression: SealRecipeImpression::Soft,
                    weight: SealRecipeWeight::Standard,
                    spacing: SealRecipeSpacing::Airy,
                    texture: SealRecipeTexture::None,
                    frame: SealRecipeFrame::SquareStandard,
                },
            },
            SealDesignRecipeVariant {
                label: "Bold dense".to_owned(),
                recipe: SealDesignRecipe {
                    font_profile: SealRecipeFontProfile::BoldBrush,
                    impression: SealRecipeImpression::Bold,
                    weight: SealRecipeWeight::Bold,
                    spacing: SealRecipeSpacing::Dense,
                    texture: SealRecipeTexture::None,
                    frame: SealRecipeFrame::SquareStandard,
                },
            },
        ];
        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &recipe_variants);

        let images = generate_seal_design_images_with_renderer(&input, &variants)
            .expect("programmatic generation must render test variants");

        assert_eq!(images.len(), 3);
        for image in &images {
            let decoded = decode_generated_seal_image(image);
            assert_square_seal_visual_contract(&decoded);
            assert_red_or_white_antialias_palette(&decoded);
            assert_centered_inner_ink(&decoded);
        }
    }

    #[test]
    fn m14_t09_storage_upload_input_mismatch_is_seal_generation_failure() {
        let recipe_variants = vec![valid_seal_design_recipe_variant("Missing image")];
        let variants =
            build_seal_design_variants("hanko-assets", "seal_request_001", &recipe_variants);
        let err = validate_seal_design_upload_inputs(&variants, &[])
            .expect_err("missing rendered image must reject storage upload");

        assert!(
            err.to_string()
                .contains("seal design image count 0 did not match variant count 1"),
            "unexpected error: {err:#}"
        );
        assert_eq!(
            SealDesignFailurePhase::StorageUpload.error_code(),
            "seal_generation_failed"
        );
    }

    const TEST_SEAL_RED: [u8; 4] = [0x9D, 0x1F, 0x22, 0xFF];
    const TEST_SEAL_WHITE: [u8; 4] = [0xFF, 0xFF, 0xFF, 0xFF];
    const TEST_FRAME_INSET: u32 = 96;
    const TEST_FRAME_STROKE_WIDTH: u32 = 34;

    fn find_unsupported_kanji_for_generation_test(variants: &[SealDesignVariant]) -> Option<char> {
        ['㐀', '㐁', '㐄', '㐅', '㐆', '㐇', '㐈', '㐉', '䶵', '䶴']
            .into_iter()
            .find(|candidate| {
                let mut input =
                    validate_generate_seal_designs_request(valid_seal_designs_request())
                        .expect("request must be valid");
                input.kanji = candidate.to_string();
                let Err(err) = generate_seal_design_images_with_renderer(&input, variants) else {
                    return false;
                };
                format!("{err:#}")
                    .contains("selected kanji could not be rendered by any approved real font")
            })
    }

    fn decode_generated_seal_image(image: &GeneratedSealDesignImage) -> image::RgbaImage {
        assert_eq!(image.content_type, "image/png");
        assert!(image.bytes.starts_with(b"\x89PNG\r\n\x1a\n"));
        image::load_from_memory_with_format(&image.bytes, image::ImageFormat::Png)
            .expect("rendered png must decode")
            .to_rgba8()
    }

    fn assert_square_seal_visual_contract(image: &image::RgbaImage) {
        assert_eq!(image.width(), SEAL_DESIGN_IMAGE_SIZE as u32);
        assert_eq!(image.height(), SEAL_DESIGN_IMAGE_SIZE as u32);
        assert_eq!(image.get_pixel(0, 0).0, TEST_SEAL_WHITE);
        assert_eq!(
            image
                .get_pixel(TEST_FRAME_INSET, SEAL_DESIGN_IMAGE_SIZE as u32 / 2)
                .0,
            TEST_SEAL_RED
        );
        assert_eq!(
            image
                .get_pixel(
                    SEAL_DESIGN_IMAGE_SIZE as u32 - TEST_FRAME_INSET - 1,
                    SEAL_DESIGN_IMAGE_SIZE as u32 / 2
                )
                .0,
            TEST_SEAL_RED
        );
        assert_eq!(
            image
                .get_pixel(SEAL_DESIGN_IMAGE_SIZE as u32 / 2, TEST_FRAME_INSET)
                .0,
            TEST_SEAL_RED
        );
        assert_eq!(
            image
                .get_pixel(
                    TEST_FRAME_INSET + TEST_FRAME_STROKE_WIDTH + 16,
                    TEST_FRAME_INSET + TEST_FRAME_STROKE_WIDTH + 16
                )
                .0,
            TEST_SEAL_WHITE
        );
        assert_eq!(
            count_exact_red_segments_on_row(image, TEST_FRAME_INSET + TEST_FRAME_STROKE_WIDTH / 2),
            1,
            "square seal must have exactly one continuous outer frame segment"
        );
    }

    fn assert_red_or_white_antialias_palette(image: &image::RgbaImage) {
        for pixel in image.pixels() {
            let [red, green, blue, alpha] = pixel.0;
            assert_eq!(alpha, 255);
            if pixel.0 == TEST_SEAL_WHITE || pixel.0 == TEST_SEAL_RED {
                continue;
            }
            assert!(
                red >= green
                    && red >= blue
                    && green >= TEST_SEAL_RED[1]
                    && blue >= TEST_SEAL_RED[2],
                "pixel must be red antialias or white, got rgba({red},{green},{blue},{alpha})"
            );
        }
    }

    fn assert_centered_inner_ink(image: &image::RgbaImage) {
        let inner_edge = TEST_FRAME_INSET + TEST_FRAME_STROKE_WIDTH + 1;
        let search_end = SEAL_DESIGN_IMAGE_SIZE as u32 - inner_edge;
        let mut left = search_end;
        let mut top = search_end;
        let mut right = inner_edge;
        let mut bottom = inner_edge;
        let mut found = false;

        for y in inner_edge..search_end {
            for x in inner_edge..search_end {
                let pixel = image.get_pixel(x, y).0;
                if pixel == TEST_SEAL_WHITE {
                    continue;
                }
                found = true;
                left = left.min(x);
                top = top.min(y);
                right = right.max(x);
                bottom = bottom.max(y);
            }
        }

        assert!(
            found,
            "generated seal should contain centered inner kanji ink"
        );
        assert!(right.saturating_sub(left) > 120);
        assert!(bottom.saturating_sub(top) > 120);

        let ink_center_x = (left + right) as f32 / 2.0;
        let ink_center_y = (top + bottom) as f32 / 2.0;
        let target_center = (SEAL_DESIGN_IMAGE_SIZE as f32 - 1.0) / 2.0;
        assert!(
            (ink_center_x - target_center).abs() <= 24.0,
            "inner ink center x {ink_center_x} should match {target_center}"
        );
        assert!(
            (ink_center_y - target_center).abs() <= 24.0,
            "inner ink center y {ink_center_y} should match {target_center}"
        );
    }

    fn count_exact_red_segments_on_row(image: &image::RgbaImage, row: u32) -> usize {
        let mut count = 0;
        let mut in_segment = false;
        for x in 0..image.width() {
            if image.get_pixel(x, row).0 == TEST_SEAL_RED {
                if !in_segment {
                    count += 1;
                    in_segment = true;
                }
            } else {
                in_segment = false;
            }
        }
        count
    }

    #[test]
    fn m13_t07_seal_quality_prompts_cover_visual_acceptance_criteria() {
        let input = validate_generate_seal_designs_request(valid_seal_designs_request())
            .expect("request must be valid");
        let prompt = build_seal_designs_prompt(&input).to_ascii_lowercase();

        for required in [
            "structured json recipes only",
            "do not generate",
            "api renderer owns",
            "selected unicode kanji",
            "real fonts",
            "2 or fewer cjk han characters",
            "engraving-friendly",
            "fixed size",
            "red ink color (#9d1f22)",
            "pure white background",
            "no background pattern",
            "frame: square_standard only",
        ] {
            assert!(
                prompt.contains(required),
                "AI quality prompt must include {required}"
            );
        }
    }

    #[test]
    fn normalize_generated_seal_image_recolors_and_fixes_square_frame() {
        let mut source = RgbaImage::from_pixel(120, 120, Rgba([246, 214, 214, 255]));
        for y in 12..108 {
            for x in 12..108 {
                let in_outer_frame = !(18..102).contains(&x) || !(18..102).contains(&y);
                let in_inner_frame = (((30..34).contains(&x) || (86..90).contains(&x))
                    && (30..90).contains(&y))
                    || (((30..34).contains(&y) || (86..90).contains(&y)) && (30..90).contains(&x));
                let in_kanji_mark = (54..66).contains(&x) && (44..78).contains(&y);
                if in_outer_frame || in_inner_frame || in_kanji_mark {
                    source.put_pixel(x, y, Rgba([0, 0, 0, 255]));
                }
            }
        }
        for y in 88..92 {
            for x in 48..74 {
                source.put_pixel(x, y, Rgba([210, 160, 160, 255]));
            }
        }

        let mut bytes = Vec::new();
        DynamicImage::ImageRgba8(source)
            .write_to(&mut Cursor::new(&mut bytes), ImageFormat::Png)
            .expect("test png should encode");

        let normalized = normalize_generated_seal_image(
            GeneratedSealDesignImage {
                content_type: "image/png".to_owned(),
                bytes,
            },
            SealShape::Square,
        )
        .expect("seal should normalize");
        validate_generated_seal_image(&normalized).expect("normalized image should validate");

        let decoded = image::load_from_memory_with_format(&normalized.bytes, ImageFormat::Png)
            .expect("normalized png should decode")
            .to_rgba8();
        assert_eq!(decoded.width(), SEAL_DESIGN_IMAGE_SIZE as u32);
        assert_eq!(decoded.height(), SEAL_DESIGN_IMAGE_SIZE as u32);
        assert_eq!(
            decoded
                .get_pixel(SEAL_DESIGN_FRAME_INSET, SEAL_DESIGN_IMAGE_SIZE as u32 / 2)
                .0,
            [
                SEAL_DESIGN_RED[0],
                SEAL_DESIGN_RED[1],
                SEAL_DESIGN_RED[2],
                255
            ]
        );
        assert_eq!(
            decoded
                .get_pixel(
                    SEAL_DESIGN_FRAME_INSET + SEAL_DESIGN_FRAME_STROKE_WIDTH + 70,
                    SEAL_DESIGN_IMAGE_SIZE as u32 / 2
                )
                .0,
            [
                SEAL_DESIGN_BACKGROUND[0],
                SEAL_DESIGN_BACKGROUND[1],
                SEAL_DESIGN_BACKGROUND[2],
                255
            ],
            "normalizer must remove AI-generated inner frames"
        );
        assert_eq!(
            decoded.get_pixel(24, 24).0,
            [
                SEAL_DESIGN_BACKGROUND[0],
                SEAL_DESIGN_BACKGROUND[1],
                SEAL_DESIGN_BACKGROUND[2],
                255
            ]
        );
        let inner_edge = SEAL_DESIGN_FRAME_INSET + SEAL_DESIGN_FRAME_STROKE_WIDTH + 1;
        let kanji_bounds = find_artwork_bounds_in_region(
            &decoded,
            ArtworkBounds {
                left: inner_edge,
                top: inner_edge,
                right: SEAL_DESIGN_IMAGE_SIZE as u32 - inner_edge - 1,
                bottom: SEAL_DESIGN_IMAGE_SIZE as u32 - inner_edge - 1,
            },
            SEAL_DESIGN_BOUNDS_STRENGTH_THRESHOLD,
        )
        .expect("normalized kanji should have visible artwork");
        let center_y = (kanji_bounds.top + kanji_bounds.bottom) as f32 / 2.0;
        assert!(
            (center_y - (SEAL_DESIGN_IMAGE_SIZE as f32 / 2.0)).abs() <= 24.0,
            "normalizer must not let faint lower artifacts push kanji upward"
        );
        assert!(
            decoded.pixels().all(|pixel| pixel.0 != [0, 0, 0, 255]),
            "normalizer must remove black ink"
        );
    }

    #[test]
    fn validate_generated_seal_image_rejects_non_png_payload() {
        let image = GeneratedSealDesignImage {
            content_type: "image/jpeg".to_owned(),
            bytes: vec![0xff, 0xd8, 0xff],
        };

        let err = validate_generated_seal_image(&image).expect_err("jpeg must be rejected");
        assert!(err.to_string().contains("image/png"));
    }

    #[test]
    fn storage_upload_endpoint_rejects_empty_bucket() {
        let err = storage_upload_endpoint("  ").expect_err("empty bucket must fail");
        assert!(err.to_string().contains("storage bucket is required"));
    }

    #[test]
    fn normalize_storage_object_name_rejects_parent_traversal() {
        let err = normalize_storage_object_name("seal_designs/../x.png")
            .expect_err("parent traversal must fail");
        assert!(
            err.to_string()
                .contains("storage object name must not contain parent traversal")
        );
    }

    #[test]
    fn validate_generate_kanji_candidates_request_accepts_defaults() {
        let request = GenerateKanjiCandidatesRequest {
            real_name: "Michael Smith".to_owned(),
            reason_language: None,
            gender: None,
            kanji_style: None,
            count: None,
        };

        let input =
            validate_generate_kanji_candidates_request(request).expect("request must be valid");
        assert_eq!(input.reason_language, "en");
        assert_eq!(input.reason_language_requested, "en");
        assert_eq!(input.reason_language_route_code, "en");
        assert_eq!(input.reason_language_fallback_reason, None);
        assert_eq!(input.gender, CandidateGender::Unspecified);
        assert_eq!(input.kanji_style, KanjiStyle::Japanese);
        assert_eq!(input.count, DEFAULT_KANJI_CANDIDATE_COUNT);
    }

    #[test]
    fn validate_generate_kanji_candidates_request_accepts_gender() {
        let request = GenerateKanjiCandidatesRequest {
            real_name: "John".to_owned(),
            reason_language: Some("en".to_owned()),
            gender: Some("male".to_owned()),
            kanji_style: Some("chinese".to_owned()),
            count: Some(3),
        };

        let input =
            validate_generate_kanji_candidates_request(request).expect("request must be valid");
        assert_eq!(input.gender, CandidateGender::Male);
        assert_eq!(input.kanji_style, KanjiStyle::Chinese);
        assert_eq!(input.count, 3);
    }

    #[test]
    fn validate_generate_kanji_candidates_request_maps_route_code_prompt_language() {
        let request = GenerateKanjiCandidatesRequest {
            real_name: "山田 太郎".to_owned(),
            reason_language: Some("ja-JP".to_owned()),
            gender: None,
            kanji_style: None,
            count: Some(3),
        };

        let input =
            validate_generate_kanji_candidates_request(request).expect("request must be valid");

        assert_eq!(input.reason_language, "ja");
        assert_eq!(input.reason_language_requested, "ja-jp");
        assert_eq!(input.reason_language_route_code, "ja");
        assert_eq!(input.reason_language_fallback_reason, None);
    }

    #[test]
    fn validate_generate_kanji_candidates_request_falls_back_for_unsupported_prompt_language() {
        let request = GenerateKanjiCandidatesRequest {
            real_name: "Mei".to_owned(),
            reason_language: Some("zh_Hant".to_owned()),
            gender: None,
            kanji_style: Some("taiwanese".to_owned()),
            count: Some(3),
        };

        let input =
            validate_generate_kanji_candidates_request(request).expect("request must be valid");

        assert_eq!(input.reason_language, "en");
        assert_eq!(input.reason_language_requested, "zh-hant");
        assert_eq!(input.reason_language_route_code, "zhtw");
        assert_eq!(
            input.reason_language_fallback_reason,
            Some("unsupported_prompt_language")
        );
    }

    #[test]
    fn validate_generate_kanji_candidates_request_rejects_unknown_style() {
        let request = GenerateKanjiCandidatesRequest {
            real_name: "John".to_owned(),
            reason_language: Some("en".to_owned()),
            gender: Some("male".to_owned()),
            kanji_style: Some("korean".to_owned()),
            count: Some(3),
        };

        let err = validate_generate_kanji_candidates_request(request)
            .expect_err("request must fail for unknown style");
        assert!(
            err.to_string()
                .contains("kanji_style must be one of japanese, chinese, taiwanese")
        );
    }

    #[test]
    fn parse_kanji_candidates_from_gemini_text_accepts_markdown_json() {
        let payload = r#"
```json
{
  "candidates": [
    {
      "kanji": "蒼真",
      "reading": "soma",
      "meaning": "Blue truth",
      "impression": ["Clear", "Sincere"],
      "reason": "Balanced and clear for seal engraving.",
      "character_count": 2,
      "stroke_complexity": "medium",
      "engraving_suitability": "high"
    },
    {
      "kanji": "悠花",
      "reading": "yuka",
      "meaning": "Graceful flower",
      "impression": ["Soft", "Elegant"],
      "reason": "Soft sound and elegant strokes.",
      "character_count": 2,
      "stroke_complexity": "low",
      "engraving_suitability": "high"
    }
  ]
}
```
"#;

        let candidates =
            parse_kanji_candidates_from_gemini_text(payload, 5).expect("payload must parse");
        assert_eq!(candidates.len(), 2);
        assert_eq!(candidates[0].kanji, "蒼真");
        assert_eq!(candidates[0].reading, "soma");
        assert_eq!(candidates[0].meaning, "Blue truth");
        assert_eq!(candidates[0].impression, vec!["Clear", "Sincere"]);
        assert_eq!(candidates[0].character_count, 2);
        assert_eq!(candidates[0].stroke_complexity, "medium");
        assert_eq!(candidates[0].engraving_suitability, "high");
    }

    #[test]
    fn m13_t07_kanji_candidate_parser_enforces_quality_contract() {
        let payload = r#"
{
  "candidates": [
    {
      "kanji": "美",
      "reading": "mi",
      "meaning": "Beauty",
      "impression": ["Clear", "Readable"],
      "reason": "One readable kanji for a small seal.",
      "character_count": 1,
      "stroke_complexity": "LOW",
      "engraving_suitability": "HIGH"
    },
    {
      "kanji": "美空",
      "reading": "miku",
      "meaning": "Beautiful sky",
      "impression": ["Balanced", "Open"],
      "reason": "Two clear kanji with balanced strokes.",
      "character_count": 2,
      "stroke_complexity": "medium",
      "engraving_suitability": "high"
    },
    {
      "kanji": "美空翔",
      "reading": "misorasho",
      "meaning": "Too long",
      "impression": ["Crowded", "Complex"],
      "reason": "Should be rejected for character count.",
      "character_count": 3,
      "stroke_complexity": "medium",
      "engraving_suitability": "medium"
    },
    {
      "kanji": "Mika",
      "reading": "mika",
      "meaning": "Not kanji",
      "impression": ["Latin", "Invalid"],
      "reason": "Should be rejected because it is not CJK Han text.",
      "character_count": 2,
      "stroke_complexity": "low",
      "engraving_suitability": "high"
    },
    {
      "kanji": "翔太",
      "reading": "shota",
      "meaning": "Mismatched count",
      "impression": ["Readable", "Mismatch"],
      "reason": "Should be rejected because metadata does not match text.",
      "character_count": 1,
      "stroke_complexity": "medium",
      "engraving_suitability": "high"
    },
    {
      "kanji": "美雨",
      "reading": "miu",
      "meaning": "Out-of-range count",
      "impression": ["Readable", "Invalid"],
      "reason": "Should be rejected because character_count is outside the enum range.",
      "character_count": 3,
      "stroke_complexity": "medium",
      "engraving_suitability": "high"
    },
    {
      "kanji": "美海",
      "reading": "mimi",
      "meaning": "Invalid stroke level",
      "impression": ["Thin", "Invalid"],
      "reason": "Should be rejected because stroke_complexity is outside the enum.",
      "character_count": 2,
      "stroke_complexity": "thin",
      "engraving_suitability": "high"
    },
    {
      "kanji": "悠",
      "reading": "yu",
      "meaning": "Invalid suitability",
      "impression": ["Calm", "Invalid"],
      "reason": "Should be rejected because suitability is outside the enum.",
      "character_count": 1,
      "stroke_complexity": "low",
      "engraving_suitability": "unknown"
    }
  ]
}
"#;

        let candidates =
            parse_kanji_candidates_from_gemini_text(payload, 10).expect("payload must parse");

        assert_eq!(candidates.len(), 2);
        assert_eq!(candidates[0].kanji, "美");
        assert_eq!(candidates[0].character_count, 1);
        assert_eq!(candidates[0].stroke_complexity, "low");
        assert_eq!(candidates[0].engraving_suitability, "high");
        assert_eq!(candidates[1].kanji, "美空");
        assert_eq!(candidates[1].character_count, 2);
    }

    #[test]
    fn build_kanji_candidates_prompt_uses_human_language_names() {
        let input = GenerateKanjiCandidatesInput {
            real_name: "山田 太郎".to_owned(),
            reason_language: "ja".to_owned(),
            reason_language_requested: "ja".to_owned(),
            reason_language_route_code: "ja".to_owned(),
            reason_language_fallback_reason: None,
            gender: CandidateGender::Male,
            kanji_style: KanjiStyle::Japanese,
            count: 6,
        };

        let prompt = build_kanji_candidates_prompt(&input);
        assert!(prompt.contains("written in Japanese"));
        assert!(prompt.contains("meaning: concise literal or symbolic meaning"));
        assert!(prompt.contains("engraving_suitability"));
        assert!(!prompt.contains("written in ja"));
        assert!(prompt.contains("Think through the name fit internally"));
    }

    #[test]
    fn build_kanji_candidates_request_body_enables_thinking_and_schema() {
        let input = GenerateKanjiCandidatesInput {
            real_name: "山田 太郎".to_owned(),
            reason_language: "en".to_owned(),
            reason_language_requested: "zhtw".to_owned(),
            reason_language_route_code: "zhtw".to_owned(),
            reason_language_fallback_reason: Some("unsupported_prompt_language"),
            gender: CandidateGender::Unspecified,
            kanji_style: KanjiStyle::Chinese,
            count: 4,
        };

        let body = build_kanji_candidates_request_body(&input);
        assert_eq!(
            body["generationConfig"]["thinkingConfig"]["thinkingBudget"],
            json!(DEFAULT_GEMINI_THINKING_BUDGET)
        );
        assert_eq!(
            body["generationConfig"]["responseMimeType"],
            json!("application/json")
        );
        assert_eq!(body["generationConfig"]["temperature"], json!(0.4));
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["candidates"]["maxItems"],
            json!(4)
        );
        assert_eq!(
            body["generationConfig"]["responseJsonSchema"]["properties"]["candidates"]["items"]["required"],
            json!([
                "kanji",
                "reading",
                "meaning",
                "impression",
                "reason",
                "character_count",
                "stroke_complexity",
                "engraving_suitability"
            ])
        );
    }
}
