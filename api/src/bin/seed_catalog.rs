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
use serde_json::{Value as JsonValue, json};

#[path = "../language_registry.rs"]
mod language_registry;

const DATASTORE_SCOPE: &str = "https://www.googleapis.com/auth/datastore";

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
    label_ja: &'static str,
    label_en: &'static str,
    description_ja: &'static str,
    description_en: &'static str,
    sort_order: i64,
}

#[derive(Debug, Clone, Copy)]
struct StoneListingSeed {
    key: &'static str,
    listing_code: &'static str,
    material_key: &'static str,
    size: &'static str,
    title_ja: &'static str,
    title_en: &'static str,
    description_ja: &'static str,
    description_en: &'static str,
    story_ja: &'static str,
    story_en: &'static str,
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
    label_ja: &'static str,
    label_en: &'static str,
    aliases: &'static [&'static str],
    sort_order: i64,
}

#[derive(Debug, Clone, Copy)]
struct CountrySeed {
    code: &'static str,
    label_ja: &'static str,
    label_en: &'static str,
    shipping_fee_usd: i64,
    shipping_fee_jpy: i64,
    sort_order: i64,
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

    for material in material_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "materials",
            material.key,
            material_document(&material, now),
        )
        .await
        .with_context(|| format!("failed to seed materials/{}", material.key))?;
    }

    for facet_tag in facet_tag_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "facet_tags",
            facet_tag.doc_id,
            facet_tag_document(&facet_tag, now),
        )
        .await
        .with_context(|| format!("failed to seed facet_tags/{}", facet_tag.doc_id))?;
    }

    for listing in stone_listing_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "stone_listings",
            listing.key,
            stone_listing_document(&listing, now),
        )
        .await
        .with_context(|| format!("failed to seed stone_listings/{}", listing.key))?;
    }

    for country in country_seeds() {
        upsert_named_document(
            &client,
            &parent,
            "countries",
            country.code,
            country_document(&country, now),
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
        Ok(_) => {
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

fn material_document(material: &MaterialSeed, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            (
                "label_i18n",
                fs_string_map(&[("ja", material.label_ja), ("en", material.label_en)]),
            ),
            (
                "description_i18n",
                fs_string_map(&[
                    ("ja", material.description_ja),
                    ("en", material.description_en),
                ]),
            ),
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

fn stone_listing_document(listing: &StoneListingSeed, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("listing_code", fs_string(listing.listing_code)),
            ("material_key", fs_string(listing.material_key)),
            ("size", fs_string(listing.size)),
            (
                "title_i18n",
                fs_string_map(&[("ja", listing.title_ja), ("en", listing.title_en)]),
            ),
            (
                "description_i18n",
                fs_string_map(&[
                    ("ja", listing.description_ja),
                    ("en", listing.description_en),
                ]),
            ),
            (
                "story_i18n",
                fs_string_map(&[("ja", listing.story_ja), ("en", listing.story_en)]),
            ),
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
                    (
                        "alt_i18n",
                        fs_string_map(&[("ja", listing.title_ja), ("en", listing.title_en)]),
                    ),
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

fn facet_tag_document(tag: &FacetTagSeed, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            ("facet_type", fs_string(tag.facet_type)),
            ("key", fs_string(tag.key)),
            (
                "label_i18n",
                fs_string_map(&[("ja", tag.label_ja), ("en", tag.label_en)]),
            ),
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

fn country_document(country: &CountrySeed, now: DateTime<Utc>) -> Document {
    Document {
        fields: btree_from_pairs(vec![
            (
                "label_i18n",
                fs_string_map(&[("ja", country.label_ja), ("en", country.label_en)]),
            ),
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
            label_ja: "木材",
            label_en: "Wood",
            description_ja: "自然な木目と軽さがあり、あたたかみのある印象に仕上がる材質です。",
            description_en: "A lightweight material with natural grain that gives the seal a warm, organic feel.",
            sort_order: 10,
        },
        MaterialSeed {
            key: "qingtian_stone",
            label_ja: "青田石",
            label_en: "Qingtian Stone",
            description_ja: "中国浙江省青田産として知られる代表的な篆刻石で、きめ細かく彫りやすい石材です。",
            description_en: "A classic seal-carving stone associated with Qingtian, Zhejiang, known for its fine texture and ease of carving.",
            sort_order: 20,
        },
        MaterialSeed {
            key: "shoushan_stone",
            label_ja: "寿山石",
            label_en: "Shoushan Stone",
            description_ja: "中国福建省寿山産として知られる印材石で、色味の幅と滑らかな彫り心地が特徴です。",
            description_en: "A seal stone associated with Shoushan, Fujian, valued for its range of colors and smooth carving feel.",
            sort_order: 30,
        },
        MaterialSeed {
            key: "balin_stone",
            label_ja: "巴林石",
            label_en: "Balin Stone",
            description_ja: "中国内モンゴル産として知られる印材石で、色柄の変化とほどよい硬さを持つ材質です。",
            description_en: "A seal stone associated with Inner Mongolia, known for varied colors and patterns with moderate hardness.",
            sort_order: 40,
        },
        MaterialSeed {
            key: "yili_stone",
            label_ja: "伊犁石",
            label_en: "Yili Stone",
            description_ja: "中国新疆ウイグル自治区の伊犁地域に由来する印材石で、落ち着いた色味と素朴な石質が特徴です。",
            description_en: "A seal stone associated with the Yili region of Xinjiang, with subdued colors and a natural stone texture.",
            sort_order: 50,
        },
        MaterialSeed {
            key: "laos_stone",
            label_ja: "ラオス石",
            label_en: "Laos Stone",
            description_ja: "ラオス産として流通する印材石で、やわらかめの石質と彫刻しやすさが特徴です。",
            description_en: "A seal stone commonly traded as Laos stone, known for its softer texture and ease of carving.",
            sort_order: 60,
        },
        MaterialSeed {
            key: "xixia_stone",
            label_ja: "西峡石",
            label_en: "Xixia Stone",
            description_ja: "中国河南省西峡産として知られる印材石で、緻密で落ち着いた質感を持つ材質です。",
            description_en: "A seal stone associated with Xixia, Henan, with a dense structure and calm, refined texture.",
            sort_order: 70,
        },
        MaterialSeed {
            key: "frozen_stone",
            label_ja: "凍石",
            label_en: "Frozen Stone",
            description_ja: "凍ったような半透明感としっとりした質感が特徴の、篆刻向けの石材です。",
            description_en: "A seal-carving stone with a moist, semi-translucent appearance reminiscent of frozen stone.",
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
            title_ja: "青田石の一点物 01",
            title_en: "One-of-a-kind Qingtian Stone 01",
            description_ja: "淡い緑と灰色の揺らぎが入った、落ち着きのある印材です。",
            description_en: "A calm seal stone with soft green and gray natural movement.",
            story_ja: "きめ細かな石質で、日常使いにも贈り物にも合わせやすい一本です。",
            story_en: "Its fine texture makes it suitable for daily use or a thoughtful gift.",
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
            title_ja: "寿山石の一点物 01",
            title_en: "One-of-a-kind Shoushan Stone 01",
            description_ja: "あたたかな黄味に自然な筋が走る、存在感のある個体です。",
            description_en: "A warm yellow piece with natural veining and a strong presence.",
            story_ja: "柔らかな色味と彫り心地のよさが魅力の、表情豊かな石です。",
            story_en: "A characterful stone known for its gentle color and smooth carving feel.",
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
            title_ja: "凍石の一点物 01",
            title_en: "One-of-a-kind Frozen Stone 01",
            description_ja: "白く半透明な奥行きがあり、清らかな印象に仕上がる石です。",
            description_en: "A white, translucent stone that gives the finished seal a clear impression.",
            story_ja: "凍った光のような質感が、印面の朱色を引き立てます。",
            story_en: "Its frozen-light texture sets off the red seal impression beautifully.",
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
            label_ja: "緑",
            label_en: "Green",
            aliases: &["soft_green", "gray_green"],
            sort_order: 10,
        },
        FacetTagSeed {
            doc_id: "color:yellow",
            facet_type: "color",
            key: "yellow",
            label_ja: "黄",
            label_en: "Yellow",
            aliases: &["warm_yellow", "cream"],
            sort_order: 20,
        },
        FacetTagSeed {
            doc_id: "color:white",
            facet_type: "color",
            key: "white",
            label_ja: "白",
            label_en: "White",
            aliases: &["translucent"],
            sort_order: 30,
        },
        FacetTagSeed {
            doc_id: "pattern:cloud",
            facet_type: "pattern",
            key: "cloud",
            label_ja: "雲状",
            label_en: "Cloud",
            aliases: &["mottled"],
            sort_order: 10,
        },
        FacetTagSeed {
            doc_id: "pattern:veined",
            facet_type: "pattern",
            key: "veined",
            label_ja: "筋",
            label_en: "Veined",
            aliases: &[],
            sort_order: 20,
        },
        FacetTagSeed {
            doc_id: "pattern:plain",
            facet_type: "pattern",
            key: "plain",
            label_ja: "無地",
            label_en: "Plain",
            aliases: &[],
            sort_order: 30,
        },
    ]
}

fn country_seeds() -> Vec<CountrySeed> {
    vec![
        CountrySeed {
            code: "JP",
            label_ja: "日本",
            label_en: "Japan",
            shipping_fee_usd: 600,
            shipping_fee_jpy: 600,
            sort_order: 10,
        },
        CountrySeed {
            code: "US",
            label_ja: "アメリカ",
            label_en: "United States",
            shipping_fee_usd: 1_800,
            shipping_fee_jpy: 1_800,
            sort_order: 20,
        },
        CountrySeed {
            code: "CA",
            label_ja: "カナダ",
            label_en: "Canada",
            shipping_fee_usd: 1_900,
            shipping_fee_jpy: 1_900,
            sort_order: 30,
        },
        CountrySeed {
            code: "GB",
            label_ja: "イギリス",
            label_en: "United Kingdom",
            shipping_fee_usd: 2_000,
            shipping_fee_jpy: 2_000,
            sort_order: 40,
        },
        CountrySeed {
            code: "AU",
            label_ja: "オーストラリア",
            label_en: "Australia",
            shipping_fee_usd: 2_100,
            shipping_fee_jpy: 2_100,
            sort_order: 50,
        },
        CountrySeed {
            code: "SG",
            label_ja: "シンガポール",
            label_en: "Singapore",
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

fn fs_string_map(values: &[(&str, &str)]) -> JsonValue {
    let mut fields = BTreeMap::new();
    for (key, value) in values {
        fields.insert((*key).to_owned(), fs_string(*value));
    }
    fs_map(fields)
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
    fn stone_listing_document_contains_app_required_fields() {
        let listing = stone_listing_seeds()
            .into_iter()
            .find(|listing| listing.key == "qingtian_stone_01")
            .expect("qingtian stone listing seed should exist");
        let now = DateTime::parse_from_rfc3339("2026-05-25T12:00:00Z")
            .expect("timestamp")
            .with_timezone(&Utc);

        let document = stone_listing_document(&listing, now);

        assert_eq!(document.fields.get("status"), Some(&fs_string("published")));
        assert_eq!(document.fields.get("is_active"), Some(&fs_bool(true)));
        assert!(document.fields.contains_key("title_i18n"));
        assert!(document.fields.contains_key("facets"));
        assert!(document.fields.contains_key("photos"));
        assert!(document.fields.contains_key("price_by_currency"));
        assert!(document.fields.contains_key("published_at"));
    }
}
