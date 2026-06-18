use std::collections::{HashMap, HashSet};

use anyhow::{Context, Result, bail};
use serde::Deserialize;

const LANGUAGE_REGISTRY_JSON: &str = include_str!(concat!(
    env!("CARGO_MANIFEST_DIR"),
    "/../config/languages.json"
));
const DEFAULT_PUBLIC_LOCALE: &str = "ja";
const DEFAULT_PUBLIC_CURRENCY: &str = "USD";

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct RegistryPublicConfig {
    pub supported_locales: Vec<String>,
    pub default_locale: String,
    pub default_currency: String,
    pub currency_by_locale: HashMap<String, String>,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct ReasonLanguageResolution {
    pub requested_locale: String,
    pub route_code: String,
    pub prompt_language: String,
    pub fallback_reason: Option<&'static str>,
}

#[derive(Debug, Deserialize)]
struct LanguageRegistryEntry {
    route_code: String,
    bcp47: String,
    currency: String,
    app: LanguageRegistryAppConfig,
}

#[derive(Debug, Deserialize)]
struct LanguageRegistryAppConfig {
    enabled: bool,
}

pub fn public_config_from_registry() -> Result<RegistryPublicConfig> {
    public_config_from_registry_json(LANGUAGE_REGISTRY_JSON)
}

fn public_config_from_registry_json(source: &str) -> Result<RegistryPublicConfig> {
    let entries = serde_json::from_str::<Vec<LanguageRegistryEntry>>(source)
        .context("failed to parse language registry")?;
    let mut supported_locales = Vec::new();
    let mut currency_by_locale = HashMap::new();
    let mut seen = HashSet::new();

    for entry in entries {
        if !entry.app.enabled {
            continue;
        }

        let route_code = normalize_route_code(&entry.route_code);
        if route_code.is_empty() {
            bail!("language registry contains an empty app-enabled route_code");
        }
        if !seen.insert(route_code.clone()) {
            bail!("language registry contains duplicate app-enabled route_code `{route_code}`");
        }
        let currency = normalize_currency_code(&entry.currency)
            .with_context(|| format!("invalid registry currency for `{route_code}`"))?;
        supported_locales.push(route_code.clone());
        currency_by_locale.insert(route_code, currency);
    }

    if supported_locales.is_empty() {
        bail!("language registry has no app-enabled locales");
    }

    let default_locale = if supported_locales
        .iter()
        .any(|locale| locale == DEFAULT_PUBLIC_LOCALE)
    {
        DEFAULT_PUBLIC_LOCALE.to_owned()
    } else {
        supported_locales[0].clone()
    };

    Ok(RegistryPublicConfig {
        supported_locales,
        default_locale,
        default_currency: DEFAULT_PUBLIC_CURRENCY.to_owned(),
        currency_by_locale,
    })
}

pub fn normalize_route_code(raw: &str) -> String {
    raw.trim().to_lowercase()
}

pub fn route_code_for_locale(raw: &str) -> Option<String> {
    let value = normalize_locale_token(raw)?;
    let entries = registry_entries().ok()?;

    for entry in &entries {
        if normalize_route_code(&entry.route_code) == value {
            return Some(normalize_route_code(&entry.route_code));
        }
    }

    for entry in &entries {
        if normalize_locale_token(&entry.bcp47).as_deref() == Some(value.as_str()) {
            return Some(normalize_route_code(&entry.route_code));
        }
    }

    match value.as_str() {
        "zh-hans" | "zh-cn" | "zh-sg" => return Some("zh".to_owned()),
        "zh-hant" | "zh-tw" | "zh-hk" | "zh-mo" => return Some("zhtw".to_owned()),
        _ => {}
    }

    let primary = value.split('-').next().unwrap_or_default();
    let mut primary_matches = entries
        .iter()
        .filter(|entry| normalize_locale_token(&entry.bcp47).is_some_and(|bcp47| bcp47 == primary))
        .map(|entry| normalize_route_code(&entry.route_code))
        .collect::<Vec<_>>();
    primary_matches.sort();
    primary_matches.dedup();

    if primary_matches.len() == 1 {
        primary_matches.pop()
    } else {
        None
    }
}

pub fn normalize_currency_code(raw: &str) -> Option<String> {
    let value = raw.trim().to_uppercase();
    if value.len() != 3 || !value.chars().all(|ch| ch.is_ascii_alphabetic()) {
        return None;
    }
    Some(value)
}

pub fn reason_language_for_locale(raw: Option<&str>) -> ReasonLanguageResolution {
    let requested = raw
        .map(str::trim)
        .filter(|value| !value.is_empty())
        .unwrap_or("en");
    let requested_locale =
        normalize_locale_token(requested).unwrap_or_else(|| requested.trim().to_lowercase());
    let route_code = match requested_locale.as_str() {
        "english" => Some("en".to_owned()),
        "japanese" => Some("ja".to_owned()),
        _ => route_code_for_locale(requested),
    };

    match route_code {
        Some(route_code) if route_code == "en" || route_code == "ja" => ReasonLanguageResolution {
            requested_locale,
            prompt_language: route_code.clone(),
            route_code,
            fallback_reason: None,
        },
        Some(route_code) => ReasonLanguageResolution {
            requested_locale,
            route_code,
            prompt_language: "en".to_owned(),
            fallback_reason: Some("unsupported_prompt_language"),
        },
        None => ReasonLanguageResolution {
            requested_locale,
            route_code: "en".to_owned(),
            prompt_language: "en".to_owned(),
            fallback_reason: Some("unknown_locale"),
        },
    }
}

fn registry_entries() -> Result<Vec<LanguageRegistryEntry>> {
    serde_json::from_str::<Vec<LanguageRegistryEntry>>(LANGUAGE_REGISTRY_JSON)
        .context("failed to parse language registry")
}

fn normalize_locale_token(raw: &str) -> Option<String> {
    let value = raw.trim().replace('_', "-").to_lowercase();
    if value.is_empty()
        || !value
            .chars()
            .all(|ch| ch.is_ascii_lowercase() || ch.is_ascii_digit() || ch == '-')
    {
        return None;
    }
    Some(value)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn checked_in_registry_generates_public_config() {
        let config = public_config_from_registry().expect("checked-in registry should load");

        assert_eq!(
            config.supported_locales,
            vec!["ar", "en", "ja", "zh", "zhtw"]
        );
        assert_eq!(config.default_locale, "ja");
        assert_eq!(config.default_currency, "USD");
        assert_eq!(
            config.currency_by_locale.get("en").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            config.currency_by_locale.get("ja").map(String::as_str),
            Some("JPY")
        );
        assert_eq!(
            config.currency_by_locale.get("ar").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            config.currency_by_locale.get("zh").map(String::as_str),
            Some("USD")
        );
        assert_eq!(
            config.currency_by_locale.get("zhtw").map(String::as_str),
            Some("USD")
        );
    }

    #[test]
    fn public_config_uses_app_enabled_registry_entries() {
        let source = r#"[
          {"route_code":"en","bcp47":"en","currency":"USD","app":{"enabled":true}},
          {"route_code":"fr","bcp47":"fr","currency":"EUR","app":{"enabled":true}},
          {"route_code":"ja","bcp47":"ja","currency":"JPY","app":{"enabled":false}},
          {"route_code":"zhtw","bcp47":"zh-Hant","currency":"USD","app":{"enabled":true}}
        ]"#;

        let config =
            public_config_from_registry_json(source).expect("fixture registry should load");

        assert_eq!(config.supported_locales, vec!["en", "fr", "zhtw"]);
        assert_eq!(config.default_locale, "en");
        assert_eq!(
            config.currency_by_locale.get("fr").map(String::as_str),
            Some("EUR")
        );
    }

    #[test]
    fn route_code_for_locale_accepts_route_and_bcp47_values() {
        assert_eq!(route_code_for_locale(" ZHTW ").as_deref(), Some("zhtw"));
        assert_eq!(route_code_for_locale("zh_Hant").as_deref(), Some("zhtw"));
        assert_eq!(route_code_for_locale("zh-TW").as_deref(), Some("zhtw"));
        assert_eq!(route_code_for_locale("zh-CN").as_deref(), Some("zh"));
        assert_eq!(route_code_for_locale("ja-JP").as_deref(), Some("ja"));
        assert_eq!(route_code_for_locale("xx").as_deref(), None);
    }

    #[test]
    fn reason_language_for_locale_falls_back_with_diagnostics() {
        assert_eq!(
            reason_language_for_locale(Some("ja-JP")),
            ReasonLanguageResolution {
                requested_locale: "ja-jp".to_owned(),
                route_code: "ja".to_owned(),
                prompt_language: "ja".to_owned(),
                fallback_reason: None,
            }
        );
        assert_eq!(
            reason_language_for_locale(Some("zh_Hant")),
            ReasonLanguageResolution {
                requested_locale: "zh-hant".to_owned(),
                route_code: "zhtw".to_owned(),
                prompt_language: "en".to_owned(),
                fallback_reason: Some("unsupported_prompt_language"),
            }
        );
        assert_eq!(
            reason_language_for_locale(Some("xx")),
            ReasonLanguageResolution {
                requested_locale: "xx".to_owned(),
                route_code: "en".to_owned(),
                prompt_language: "en".to_owned(),
                fallback_reason: Some("unknown_locale"),
            }
        );
    }
}
