use std::{
    collections::{BTreeMap, HashMap},
    env,
    sync::Arc,
};

use anyhow::{Context, Result, anyhow, bail};
use chrono::{DateTime, Duration, SecondsFormat, Utc};
use firebase_sdk_rust::firebase_firestore::{
    CreateDocumentOptions, Document, FirebaseFirestoreClient, FirebaseFirestoreError,
    GetDocumentOptions, PatchDocumentOptions,
};
use gcp_auth::{CustomServiceAccount, TokenProvider, provider};
use serde::{Deserialize, de::DeserializeOwned};
use serde_json::{Value as JsonValue, json};

#[path = "../language_registry.rs"]
mod language_registry;

const DATASTORE_SCOPE: &str = "https://www.googleapis.com/auth/datastore";
const MATERIALS_I18N_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/catalog/materials.json"
));
const STONE_LISTINGS_I18N_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/catalog/stone_listings.json"
));
const FACET_TAGS_I18N_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/catalog/facet_tags.json"
));
const COUNTRIES_I18N_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/content/i18n/catalog/countries.json"
));

#[derive(Debug, Clone)]
struct SeedConfig {
    project_id: String,
    credentials_file: Option<String>,
}

#[derive(Debug, Clone, Copy)]
struct FontSeed {
    key: &'static str,
    label: &'static str,
    font_family: &'static str,
    font_stylesheet_url: &'static str,
    kanji_style: &'static str,
    sort_order: i64,
}

#[derive(Debug, Clone, Copy)]
struct MaterialSeed {
    key: &'static str,
    sort_order: i64,
}

#[derive(Debug, Clone, Copy)]
struct StoneListingSeed {
    key: &'static str,
    listing_code: &'static str,
    material_key: &'static str,
    size: &'static str,
    color_family: &'static str,
    color_tags: &'static [&'static str],
    pattern_primary: &'static str,
    pattern_tags: &'static [&'static str],
    stone_shape: &'static str,
    translucency: &'static str,
    photo_asset_id: &'static str,
    photo_storage_path: &'static str,
    price_usd: i64,
    price_jpy: i64,
    sort_order: i64,
    published_hours_ago: i64,
}

#[derive(Debug, Clone, Copy)]
struct FacetTagSeed {
    doc_id: &'static str,
    facet_type: &'static str,
    key: &'static str,
    aliases: &'static [&'static str],
    sort_order: i64,
}

#[derive(Debug, Clone, Copy)]
struct CountrySeed {
    code: &'static str,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    sort_order: i64,
}

#[derive(Debug, Clone, Deserialize)]
struct MaterialCopy {
    label: HashMap<String, String>,
    description: HashMap<String, String>,
}

#[derive(Debug, Clone, Deserialize)]
struct StoneListingCopy {
    title: HashMap<String, String>,
    description: HashMap<String, String>,
    story: HashMap<String, String>,
    photo_alt: HashMap<String, String>,
}

#[derive(Debug, Clone, Deserialize)]
struct FacetTagCopy {
    label: HashMap<String, String>,
}

#[derive(Debug, Clone, Deserialize)]
struct CountryCopy {
    label: HashMap<String, String>,
}

#[tokio::main]
async fn main() -> Result<()> {
    let cfg = load_config()?;
    let client = firestore_client(cfg.credentials_file.as_deref()).await?;
    let parent = format!("projects/{}/databases/(default)/documents", cfg.project_id);
    let now = Utc::now();

    println!("seeding Firestore catalog for project {}", cfg.project_id);

    upsert_named_document(
        &client,
        &parent,
        "app_config",
        "public",
        app_config_public_document(now),
    )
    .await
    .context("failed to seed app_config/public")?;

    for font in font_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "fonts",
            font.key,
            font_document(&font, now),
        )
        .await
        .with_context(|| format!("failed to seed fonts/{}", font.key))?;
    }

    let material_copy = material_copy_documents();
    for material in material_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "materials",
            material.key,
            material_document(
                &material,
                required_catalog_copy(&material_copy, "materials", material.key),
                now,
            ),
        )
        .await
        .with_context(|| format!("failed to seed materials/{}", material.key))?;
    }

    let facet_tag_copy = facet_tag_copy_documents();
    for facet_tag in facet_tag_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "facet_tags",
            facet_tag.doc_id,
            facet_tag_document(
                &facet_tag,
                required_catalog_copy(&facet_tag_copy, "facet_tags", facet_tag.doc_id),
                now,
            ),
        )
        .await
        .with_context(|| format!("failed to seed facet_tags/{}", facet_tag.doc_id))?;
    }

    let stone_listing_copy = stone_listing_copy_documents();
    for listing in stone_listing_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "stone_listings",
            listing.key,
            stone_listing_document(
                &listing,
                required_catalog_copy(&stone_listing_copy, "stone_listings", listing.key),
                now,
            ),
        )
        .await
        .with_context(|| format!("failed to seed stone_listings/{}", listing.key))?;
    }

    let country_copy = country_copy_documents();
    for country in country_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "countries",
            country.code,
            country_document(
                &country,
                required_catalog_copy(&country_copy, "countries", country.code),
                now,
            ),
        )
        .await
        .with_context(|| format!("failed to seed countries/{}", country.code))?;
    }

    println!("catalog seed complete");
    Ok(())
}

fn load_config() -> Result<SeedConfig> {
    let project_id = env_first(&[
        "API_FIRESTORE_PROJECT_ID",
        "FIREBASE_PROJECT_ID",
        "GOOGLE_CLOUD_PROJECT",
    ]);
    if project_id.is_empty() {
        bail!("missing Firestore project id env var");
    }

    let credentials_file = first_non_empty(&[
        env::var("API_FIREBASE_CREDENTIALS_FILE").ok(),
        env::var("GOOGLE_APPLICATION_CREDENTIALS").ok(),
    ]);

    Ok(SeedConfig {
        project_id,
        credentials_file,
    })
}

async fn firestore_client(credentials_file: Option<&str>) -> Result<FirebaseFirestoreClient> {
    let token_provider: Arc<dyn TokenProvider> = if let Some(credentials_file) = credentials_file {
        Arc::new(
            CustomServiceAccount::from_file(credentials_file)
                .with_context(|| format!("failed to read credentials file: {credentials_file}"))?,
        )
    } else {
        provider()
            .await
            .context("failed to initialize default GCP auth provider")?
    };

    let access_token = token_provider
        .token(&[DATASTORE_SCOPE])
        .await
        .context("failed to acquire Firestore access token")?;

    firestore_client_from_access_token(access_token.as_str())
}

fn firestore_client_from_access_token(access_token: &str) -> Result<FirebaseFirestoreClient> {
    Ok(FirebaseFirestoreClient::new(access_token.to_owned()))
}

async fn upsert_named_document(
    client: &FirebaseFirestoreClient,
    parent: &str,
    collection: &str,
    doc_id: &str,
    document: Document,
) -> Result<()> {
    let name = format!("{}/{}/{}", parent, collection, doc_id);

    match client
        .get_document(&name, &GetDocumentOptions::default())
        .await
    {
        Ok(existing) => {
            let mut document = document;
            preserve_existing_localized_maps(&existing.fields, &mut document.fields);
            client
                .patch_document(&name, &document, &PatchDocumentOptions::default())
                .await
                .map_err(anyhow::Error::from)?;
        }
        Err(err) if is_not_found(&err) => {
            client
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
                .map_err(anyhow::Error::from)?;
        }
        Err(err) => return Err(anyhow!(err)),
    }

    println!("  upserted {collection}/{doc_id}");
    Ok(())
}

fn preserve_existing_localized_maps(
    existing_fields: &BTreeMap<String, JsonValue>,
    next_fields: &mut BTreeMap<String, JsonValue>,
) {
    for (field_name, next_value) in next_fields {
        if let Some(existing_value) = existing_fields.get(field_name) {
            preserve_existing_localized_value(field_name, existing_value, next_value);
        }
    }
}

fn preserve_existing_localized_value(
    field_name: &str,
    existing_value: &JsonValue,
    next_value: &mut JsonValue,
) {
    if is_localized_string_map_field(field_name) {
        preserve_missing_string_map_entries(existing_value, next_value);
        return;
    }

    if let (Some(existing_fields), Some(next_fields)) = (
        firestore_map_fields(existing_value),
        firestore_map_fields_mut(next_value),
    ) {
        for (nested_field_name, nested_next_value) in next_fields {
            if let Some(nested_existing_value) = existing_fields.get(nested_field_name) {
                preserve_existing_localized_value(
                    nested_field_name,
                    nested_existing_value,
                    nested_next_value,
                );
            }
        }
        return;
    }

    if let (Some(existing_values), Some(next_values)) = (
        firestore_array_values(existing_value),
        firestore_array_values_mut(next_value),
    ) {
        for (index, next_item) in next_values.iter_mut().enumerate() {
            if let Some(existing_item) = existing_values.get(index) {
                preserve_existing_localized_value(field_name, existing_item, next_item);
            }
        }
    }
}

fn preserve_missing_string_map_entries(existing_value: &JsonValue, next_value: &mut JsonValue) {
    let Some(existing_fields) = firestore_map_fields(existing_value) else {
        return;
    };
    let Some(next_fields) = firestore_map_fields_mut(next_value) else {
        return;
    };

    for (locale, existing_locale_value) in existing_fields {
        next_fields
            .entry(locale.clone())
            .or_insert_with(|| existing_locale_value.clone());
    }
}

fn is_localized_string_map_field(field_name: &str) -> bool {
    field_name == "alt_i18n" || field_name.ends_with("_i18n")
}

fn firestore_map_fields(value: &JsonValue) -> Option<&serde_json::Map<String, JsonValue>> {
    value
        .get("mapValue")
        .and_then(|map| map.get("fields"))
        .and_then(JsonValue::as_object)
}

fn firestore_map_fields_mut(
    value: &mut JsonValue,
) -> Option<&mut serde_json::Map<String, JsonValue>> {
    value
        .get_mut("mapValue")
        .and_then(|map| map.get_mut("fields"))
        .and_then(JsonValue::as_object_mut)
}

fn firestore_array_values(value: &JsonValue) -> Option<&Vec<JsonValue>> {
    value
        .get("arrayValue")
        .and_then(|array| array.get("values"))
        .and_then(JsonValue::as_array)
}

fn firestore_array_values_mut(value: &mut JsonValue) -> Option<&mut Vec<JsonValue>> {
    value
        .get_mut("arrayValue")
        .and_then(|array| array.get_mut("values"))
        .and_then(JsonValue::as_array_mut)
}

fn app_config_public_document(now: DateTime<Utc>) -> Document {
    let public_config = language_registry::public_config_from_registry()
        .expect("checked-in language registry should generate public config");
    Document {
        fields: btree_from_pairs(vec![
            (
                "supported_locales",
                fs_string_array(&public_config.supported_locales),
            ),
            ("default_locale", fs_string(public_config.default_locale)),
            (
                "default_currency",
                fs_string(public_config.default_currency),
            ),
            (
                "currency_by_locale",
                fs_owned_string_map(&public_config.currency_by_locale),
            ),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn font_document(font: &FontSeed, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("label", fs_string(font.label)),
            ("font_family", fs_string(font.font_family)),
            ("font_stylesheet_url", fs_string(font.font_stylesheet_url)),
            ("kanji_style", fs_string(font.kanji_style)),
            ("is_active", fs_bool(true)),
            ("sort_order", fs_int(font.sort_order)),
            ("version", fs_int(1)),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn material_document(material: &MaterialSeed, copy: &MaterialCopy, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("label_i18n", fs_owned_string_map(&copy.label)),
            ("description_i18n", fs_owned_string_map(&copy.description)),
            ("comparison_texture_ja", fs_string("")),
            ("comparison_texture_en", fs_string("")),
            ("comparison_weight_ja", fs_string("")),
            ("comparison_weight_en", fs_string("")),
            ("comparison_usage_ja", fs_string("")),
            ("comparison_usage_en", fs_string("")),
            ("shape", fs_string("square")),
            ("photos", fs_array(vec![])),
            ("price_by_currency", fs_int_map(&[("USD", 0), ("JPY", 0)])),
            ("is_active", fs_bool(true)),
            ("sort_order", fs_int(material.sort_order)),
            ("version", fs_int(1)),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn stone_listing_document(
    listing: &StoneListingSeed,
    copy: &StoneListingCopy,
    now: DateTime<Utc>,
) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("listing_code", fs_string(listing.listing_code)),
            ("material_key", fs_string(listing.material_key)),
            ("size", fs_string(listing.size)),
            ("title_i18n", fs_owned_string_map(&copy.title)),
            ("description_i18n", fs_owned_string_map(&copy.description)),
            ("story_i18n", fs_owned_string_map(&copy.story)),
            (
                "facets",
                fs_map(btree_from_pairs(vec![
                    ("color_family", fs_string(listing.color_family)),
                    ("color_tags", fs_string_array(listing.color_tags)),
                    ("pattern_primary", fs_string(listing.pattern_primary)),
                    ("pattern_tags", fs_string_array(listing.pattern_tags)),
                    ("stone_shape", fs_string(listing.stone_shape)),
                    ("translucency", fs_string(listing.translucency)),
                ])),
            ),
            (
                "photos",
                fs_array(vec![fs_map(btree_from_pairs(vec![
                    ("asset_id", fs_string(listing.photo_asset_id)),
                    ("storage_path", fs_string(listing.photo_storage_path)),
                    ("alt_i18n", fs_owned_string_map(&copy.photo_alt)),
                    ("sort_order", fs_int(0)),
                    ("is_primary", fs_bool(true)),
                    ("width", fs_int(1200)),
                    ("height", fs_int(1200)),
                ]))]),
            ),
            (
                "price_by_currency",
                fs_int_map(&[("USD", listing.price_usd), ("JPY", listing.price_jpy)]),
            ),
            ("status", fs_string("published")),
            ("is_active", fs_bool(true)),
            (
                "published_at",
                fs_timestamp(now - Duration::hours(listing.published_hours_ago)),
            ),
            ("sort_order", fs_int(listing.sort_order)),
            ("version", fs_int(1)),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn facet_tag_document(tag: &FacetTagSeed, copy: &FacetTagCopy, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("facet_type", fs_string(tag.facet_type)),
            ("key", fs_string(tag.key)),
            ("label_i18n", fs_owned_string_map(&copy.label)),
            ("aliases", fs_string_array(tag.aliases)),
            ("is_active", fs_bool(true)),
            ("sort_order", fs_int(tag.sort_order)),
            ("version", fs_int(1)),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn country_document(country: &CountrySeed, copy: &CountryCopy, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("label_i18n", fs_owned_string_map(&copy.label)),
            (
                "shipping_fee_by_currency",
                fs_int_map(&[
                    ("USD", country.shipping_fee_usd),
                    ("JPY", country.shipping_fee_jpy),
                ]),
            ),
            ("is_active", fs_bool(true)),
            ("sort_order", fs_int(country.sort_order)),
            ("version", fs_int(1)),
            ("created_at", fs_timestamp(now)),
            ("updated_at", fs_timestamp(now)),
        ]),
        ..Document::default()
    }
}

fn material_copy_documents() -> BTreeMap<String, MaterialCopy> {
    load_catalog_copy_documents(MATERIALS_I18N_JSON, "materials")
}

fn stone_listing_copy_documents() -> BTreeMap<String, StoneListingCopy> {
    load_catalog_copy_documents(STONE_LISTINGS_I18N_JSON, "stone_listings")
}

fn facet_tag_copy_documents() -> BTreeMap<String, FacetTagCopy> {
    load_catalog_copy_documents(FACET_TAGS_I18N_JSON, "facet_tags")
}

fn country_copy_documents() -> BTreeMap<String, CountryCopy> {
    load_catalog_copy_documents(COUNTRIES_I18N_JSON, "countries")
}

fn load_catalog_copy_documents<T>(source: &str, owner: &str) -> BTreeMap<String, T>
where
    T: DeserializeOwned,
{
    serde_json::from_str(source)
        .unwrap_or_else(|error| panic!("failed to parse {owner} catalog copy: {error}"))
}

fn required_catalog_copy<'a, T>(
    documents: &'a BTreeMap<String, T>,
    owner: &str,
    key: &str,
) -> &'a T {
    documents
        .get(key)
        .unwrap_or_else(|| panic!("missing {owner} catalog copy for `{key}`"))
}

fn font_seeds() -> Vec<FontSeed> {
    vec![
        FontSeed {
            key: "zen_maru_gothic",
            label: "Zen Maru Gothic",
            font_family: "'Zen Maru Gothic', sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=Zen+Maru+Gothic:wght@400;700&display=swap",
            kanji_style: "japanese",
            sort_order: 10,
        },
        FontSeed {
            key: "kosugi_maru",
            label: "Kosugi Maru",
            font_family: "'Kosugi Maru', sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=Kosugi+Maru&display=swap",
            kanji_style: "chinese",
            sort_order: 20,
        },
        FontSeed {
            key: "potta_one",
            label: "Potta One",
            font_family: "'Potta One', sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=Potta+One&display=swap",
            kanji_style: "taiwanese",
            sort_order: 30,
        },
        FontSeed {
            key: "kiwi_maru",
            label: "Kiwi Maru",
            font_family: "'Kiwi Maru', sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=Kiwi+Maru:wght@400;700&display=swap",
            kanji_style: "japanese",
            sort_order: 40,
        },
        FontSeed {
            key: "wdxl_lubrifont_jp_n",
            label: "WDXL Lubrifont JP N",
            font_family: "'WDXL Lubrifont JP N', sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=WDXL+Lubrifont+JP+N&display=swap",
            kanji_style: "chinese",
            sort_order: 50,
        },
        FontSeed {
            key: "ai_generated_seal",
            label: "AI generated seal preview",
            font_family: "'Noto Sans JP', system-ui, sans-serif",
            font_stylesheet_url: "https://fonts.googleapis.com/css2?family=Noto+Sans+JP:wght@400;700&display=swap",
            kanji_style: "japanese",
            sort_order: 90,
        },
    ]
}

fn material_seeds() -> Vec<MaterialSeed> {
    vec![
        MaterialSeed {
            key: "wood",
            sort_order: 10,
        },
        MaterialSeed {
            key: "qingtian_stone",
            sort_order: 20,
        },
        MaterialSeed {
            key: "shoushan_stone",
            sort_order: 30,
        },
        MaterialSeed {
            key: "balin_stone",
            sort_order: 40,
        },
        MaterialSeed {
            key: "yili_stone",
            sort_order: 50,
        },
        MaterialSeed {
            key: "laos_stone",
            sort_order: 60,
        },
        MaterialSeed {
            key: "xixia_stone",
            sort_order: 70,
        },
        MaterialSeed {
            key: "frozen_stone",
            sort_order: 80,
        },
    ]
}

fn stone_listing_seeds() -> Vec<StoneListingSeed> {
    vec![
        StoneListingSeed {
            key: "qingtian_stone_01",
            listing_code: "QTN-0001",
            material_key: "qingtian_stone",
            size: "15mm x 15mm x 60mm",
            color_family: "green",
            color_tags: &["soft_green", "gray_green"],
            pattern_primary: "cloud",
            pattern_tags: &["cloud", "mottled"],
            stone_shape: "square",
            translucency: "semi_translucent",
            photo_asset_id: "lst_qingtian_stone_01",
            photo_storage_path: "stone_listings/qingtian_stone/qingtian_stone_01/main.webp",
            price_usd: 21_000,
            price_jpy: 32_000,
            sort_order: 10,
            published_hours_ago: 40,
        },
        StoneListingSeed {
            key: "shoushan_stone_01",
            listing_code: "SHS-0001",
            material_key: "shoushan_stone",
            size: "18mm x 18mm x 60mm",
            color_family: "yellow",
            color_tags: &["warm_yellow", "cream"],
            pattern_primary: "veined",
            pattern_tags: &["veined", "cloud"],
            stone_shape: "square",
            translucency: "opaque",
            photo_asset_id: "lst_shoushan_stone_01",
            photo_storage_path: "stone_listings/shoushan_stone/shoushan_stone_01/main.webp",
            price_usd: 30_000,
            price_jpy: 46_000,
            sort_order: 20,
            published_hours_ago: 30,
        },
        StoneListingSeed {
            key: "frozen_stone_01",
            listing_code: "FRZ-0001",
            material_key: "frozen_stone",
            size: "16mm x 16mm x 60mm",
            color_family: "white",
            color_tags: &["white", "translucent"],
            pattern_primary: "plain",
            pattern_tags: &["plain"],
            stone_shape: "square",
            translucency: "translucent",
            photo_asset_id: "lst_frozen_stone_01",
            photo_storage_path: "stone_listings/frozen_stone/frozen_stone_01/main.webp",
            price_usd: 25_000,
            price_jpy: 38_000,
            sort_order: 30,
            published_hours_ago: 20,
        },
    ]
}

fn facet_tag_seeds() -> Vec<FacetTagSeed> {
    vec![
        FacetTagSeed {
            doc_id: "color:green",
            facet_type: "color",
            key: "green",
            aliases: &["soft_green", "gray_green"],
            sort_order: 10,
        },
        FacetTagSeed {
            doc_id: "color:yellow",
            facet_type: "color",
            key: "yellow",
            aliases: &["warm_yellow", "cream"],
            sort_order: 20,
        },
        FacetTagSeed {
            doc_id: "color:white",
            facet_type: "color",
            key: "white",
            aliases: &["translucent"],
            sort_order: 30,
        },
        FacetTagSeed {
            doc_id: "pattern:cloud",
            facet_type: "pattern",
            key: "cloud",
            aliases: &["mottled"],
            sort_order: 10,
        },
        FacetTagSeed {
            doc_id: "pattern:veined",
            facet_type: "pattern",
            key: "veined",
            aliases: &[],
            sort_order: 20,
        },
        FacetTagSeed {
            doc_id: "pattern:plain",
            facet_type: "pattern",
            key: "plain",
            aliases: &[],
            sort_order: 30,
        },
    ]
}

fn country_seeds() -> Vec<CountrySeed> {
    vec![
        CountrySeed {
            code: "JP",
            shipping_fee_usd: 600,
            shipping_fee_jpy: 600,
            sort_order: 10,
        },
        CountrySeed {
            code: "US",
            shipping_fee_usd: 1_800,
            shipping_fee_jpy: 1_800,
            sort_order: 20,
        },
        CountrySeed {
            code: "CA",
            shipping_fee_usd: 1_900,
            shipping_fee_jpy: 1_900,
            sort_order: 30,
        },
        CountrySeed {
            code: "GB",
            shipping_fee_usd: 2_000,
            shipping_fee_jpy: 2_000,
            sort_order: 40,
        },
        CountrySeed {
            code: "AU",
            shipping_fee_usd: 2_100,
            shipping_fee_jpy: 2_100,
            sort_order: 50,
        },
        CountrySeed {
            code: "SG",
            shipping_fee_usd: 1_300,
            shipping_fee_jpy: 1_300,
            sort_order: 60,
        },
    ]
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

fn first_non_empty(values: &[Option<String>]) -> Option<String> {
    values
        .iter()
        .filter_map(|value| value.as_deref())
        .map(str::trim)
        .find(|value| !value.is_empty())
        .map(ToOwned::to_owned)
}

fn is_not_found(error: &FirebaseFirestoreError) -> bool {
    matches!(
        error,
        FirebaseFirestoreError::UnexpectedStatus { status, .. } if status.as_u16() == 404
    )
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

fn fs_string_array<T: AsRef<str>>(values: &[T]) -> JsonValue {
    fs_array(
        values
            .iter()
            .map(|value| fs_string(value.as_ref()))
            .collect(),
    )
}

fn fs_owned_string_map(values: &HashMap<String, String>) -> JsonValue {
    let mut keys = values.keys().collect::<Vec<_>>();
    keys.sort();
    let mut fields = BTreeMap::new();
    for key in keys {
        if let Some(value) = values.get(key) {
            fields.insert(key.to_owned(), fs_string(value.clone()));
        }
    }
    fs_map(fields)
}

fn fs_int_map(values: &[(&str, i64)]) -> JsonValue {
    let mut fields = BTreeMap::new();
    for (key, value) in values {
        fields.insert((*key).to_owned(), fs_int(*value));
    }
    fs_map(fields)
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
    fn app_config_public_document_matches_language_registry() {
        let now = DateTime::parse_from_rfc3339("2026-06-18T12:00:00Z")
            .expect("timestamp")
            .with_timezone(&Utc);
        let document = app_config_public_document(now);
        let registry_config = language_registry::public_config_from_registry()
            .expect("checked-in registry should generate public config");

        let supported_locales = document.fields["supported_locales"]["arrayValue"]["values"]
            .as_array()
            .expect("supported_locales should be an array")
            .iter()
            .map(|value| {
                value["stringValue"]
                    .as_str()
                    .expect("supported locale should be a string")
                    .to_owned()
            })
            .collect::<Vec<_>>();
        assert_eq!(supported_locales, registry_config.supported_locales);
        assert_eq!(
            document.fields["default_locale"]["stringValue"].as_str(),
            Some(registry_config.default_locale.as_str())
        );
        assert_eq!(
            document.fields["default_currency"]["stringValue"].as_str(),
            Some(registry_config.default_currency.as_str())
        );
        for (locale, currency) in registry_config.currency_by_locale {
            assert_eq!(
                document.fields["currency_by_locale"]["mapValue"]["fields"][&locale]["stringValue"]
                    .as_str(),
                Some(currency.as_str())
            );
        }
    }

    #[test]
    fn font_seeds_include_active_ai_generated_seal_record() {
        let font = font_seeds()
            .into_iter()
            .find(|font| font.key == "ai_generated_seal")
            .expect("ai_generated_seal seed should exist");

        assert_eq!(font.label, "AI generated seal preview");
        assert!(!font.font_family.trim().is_empty());
        assert!(!font.font_stylesheet_url.trim().is_empty());
        assert_eq!(font.kanji_style, "japanese");
    }

    #[test]
    fn ai_generated_seal_document_keeps_font_lookup_fields_active() {
        let font = font_seeds()
            .into_iter()
            .find(|font| font.key == "ai_generated_seal")
            .expect("ai_generated_seal seed should exist");
        let now = DateTime::parse_from_rfc3339("2026-05-21T11:30:00Z")
            .expect("timestamp")
            .with_timezone(&Utc);

        let document = font_document(&font, now);

        assert_eq!(
            document.fields.get("label"),
            Some(&fs_string("AI generated seal preview"))
        );
        assert_eq!(
            document.fields.get("font_family"),
            Some(&fs_string("'Noto Sans JP', system-ui, sans-serif"))
        );
        assert_eq!(
            document.fields.get("kanji_style"),
            Some(&fs_string("japanese"))
        );
        assert_eq!(document.fields.get("is_active"), Some(&fs_bool(true)));
        assert_eq!(document.fields.get("version"), Some(&fs_int(1)));
    }

    #[test]
    fn stone_listing_seeds_include_published_records() {
        let listings = stone_listing_seeds();

        assert!(listings.len() >= 3);
        assert!(
            listings
                .iter()
                .all(|listing| !listing.key.trim().is_empty())
        );
        assert!(listings.iter().all(|listing| listing.price_jpy > 0));
        assert!(listings.iter().all(|listing| listing.price_usd > 0));
    }

    #[test]
    fn catalog_copy_files_cover_seed_records() {
        let material_copy = material_copy_documents();
        for seed in material_seeds() {
            let copy = required_catalog_copy(&material_copy, "materials", seed.key);
            if seed.key == "wood" {
                assert_eq!(copy.label.get("en").map(String::as_str), Some("Wood"));
            }
            assert!(copy.label.contains_key("ja"));
            assert!(copy.description.contains_key("en"));
            assert!(copy.description.contains_key("ja"));
        }

        let listing_copy = stone_listing_copy_documents();
        for seed in stone_listing_seeds() {
            let copy = required_catalog_copy(&listing_copy, "stone_listings", seed.key);
            assert!(copy.title.contains_key("en"));
            assert!(copy.title.contains_key("ja"));
            assert!(copy.description.contains_key("en"));
            assert!(copy.description.contains_key("ja"));
            assert!(copy.story.contains_key("en"));
            assert!(copy.story.contains_key("ja"));
            assert!(copy.photo_alt.contains_key("en"));
            assert!(copy.photo_alt.contains_key("ja"));
        }

        let facet_tag_copy = facet_tag_copy_documents();
        for seed in facet_tag_seeds() {
            let copy = required_catalog_copy(&facet_tag_copy, "facet_tags", seed.doc_id);
            assert!(copy.label.contains_key("en"));
            assert!(copy.label.contains_key("ja"));
        }

        let country_copy = country_copy_documents();
        for seed in country_seeds() {
            let copy = required_catalog_copy(&country_copy, "countries", seed.code);
            assert!(copy.label.contains_key("en"));
            assert!(copy.label.contains_key("ja"));
        }
    }

    #[test]
    fn seed_patch_preserves_unknown_locale_keys_in_localized_maps() {
        let existing = btree_from_pairs(vec![
            (
                "label_i18n",
                fs_owned_string_map(&HashMap::from([
                    ("en".to_owned(), "Old Wood".to_owned()),
                    ("ja".to_owned(), "古い木材".to_owned()),
                    ("fr".to_owned(), "Bois".to_owned()),
                    ("zh".to_owned(), "木材".to_owned()),
                    ("zhtw".to_owned(), "木材".to_owned()),
                ])),
            ),
            (
                "photos",
                fs_array(vec![fs_map(btree_from_pairs(vec![(
                    "alt_i18n",
                    fs_owned_string_map(&HashMap::from([
                        ("en".to_owned(), "Old photo".to_owned()),
                        ("ja".to_owned(), "古い写真".to_owned()),
                        ("zhtw".to_owned(), "照片".to_owned()),
                    ])),
                )]))]),
            ),
        ]);
        let mut next = btree_from_pairs(vec![
            (
                "label_i18n",
                fs_owned_string_map(&HashMap::from([
                    ("en".to_owned(), "Wood".to_owned()),
                    ("ja".to_owned(), "木材".to_owned()),
                ])),
            ),
            (
                "photos",
                fs_array(vec![fs_map(btree_from_pairs(vec![(
                    "alt_i18n",
                    fs_owned_string_map(&HashMap::from([
                        ("en".to_owned(), "New photo".to_owned()),
                        ("ja".to_owned(), "新しい写真".to_owned()),
                    ])),
                )]))]),
            ),
        ]);

        preserve_existing_localized_maps(&existing, &mut next);

        let label_fields = next["label_i18n"]["mapValue"]["fields"]
            .as_object()
            .expect("label_i18n should be a Firestore map");
        assert_eq!(label_fields["en"]["stringValue"], "Wood");
        assert_eq!(label_fields["ja"]["stringValue"], "木材");
        assert_eq!(label_fields["fr"]["stringValue"], "Bois");
        assert_eq!(label_fields["zh"]["stringValue"], "木材");
        assert_eq!(label_fields["zhtw"]["stringValue"], "木材");

        let photo_alt_fields =
            next["photos"]["arrayValue"]["values"][0]["mapValue"]["fields"]["alt_i18n"]["mapValue"]
                ["fields"]
                .as_object()
                .expect("alt_i18n should be a Firestore map");
        assert_eq!(photo_alt_fields["en"]["stringValue"], "New photo");
        assert_eq!(photo_alt_fields["ja"]["stringValue"], "新しい写真");
        assert_eq!(photo_alt_fields["zhtw"]["stringValue"], "照片");
    }

    #[test]
    fn stone_listing_document_contains_app_required_fields() {
        let listing = stone_listing_seeds()
            .into_iter()
            .find(|listing| listing.key == "qingtian_stone_01")
            .expect("qingtian stone listing seed should exist");
        let now = DateTime::parse_from_rfc3339("2026-05-25T12:00:00Z")
            .expect("timestamp")
            .with_timezone(&Utc);

        let listing_copy = stone_listing_copy_documents();
        let document = stone_listing_document(
            &listing,
            required_catalog_copy(&listing_copy, "stone_listings", listing.key),
            now,
        );

        assert_eq!(document.fields.get("status"), Some(&fs_string("published")));
        assert_eq!(document.fields.get("is_active"), Some(&fs_bool(true)));
        assert!(document.fields.contains_key("title_i18n"));
        assert!(document.fields.contains_key("facets"));
        assert!(document.fields.contains_key("photos"));
        assert!(document.fields.contains_key("price_by_currency"));
        assert!(document.fields.contains_key("published_at"));
    }
}
