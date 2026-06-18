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

#[derive(Debug, Deserialize)]
struct LanguageRegistryEntry {
    route_code: String,
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

pub fn normalize_currency_code(raw: &str) -> Option<String> {
    let value = raw.trim().to_uppercase();
    if value.len() != 3 || !value.chars().all(|ch| ch.is_ascii_alphabetic()) {
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

        assert_eq!(config.supported_locales, vec!["en", "ja"]);
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
    }

    #[test]
    fn public_config_uses_app_enabled_registry_entries() {
        let source = r#"[
          {"route_code":"en","currency":"USD","app":{"enabled":true}},
          {"route_code":"fr","currency":"EUR","app":{"enabled":true}},
          {"route_code":"ja","currency":"JPY","app":{"enabled":false}},
          {"route_code":"zhtw","currency":"USD","app":{"enabled":true}}
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
}
