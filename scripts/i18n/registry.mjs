import { readFile } from 'node:fs/promises';

export const DEFAULT_LANGUAGE_REGISTRY_URL = new URL(
  '../../config/languages.json',
  import.meta.url,
);

/**
 * @typedef {Object} FlutterLocaleConfig
 * @property {string} languageCode
 * @property {string | null} scriptCode
 * @property {string | null} countryCode
 *
 * @typedef {Object} WebLocaleConfig
 * @property {boolean} enabled
 * @property {boolean} indexed
 * @property {string} url_prefix
 *
 * @typedef {Object} AppLocaleConfig
 * @property {boolean} enabled
 * @property {boolean} selectable
 *
 * @typedef {Object} ReleaseLocaleConfig
 * @property {boolean} enabled
 * @property {string | null} android_store_locale
 * @property {string | null} ios_store_locale
 *
 * @typedef {Object} LanguageEntry
 * @property {string} route_code
 * @property {string} bcp47
 * @property {FlutterLocaleConfig} flutter
 * @property {string} native_name
 * @property {string} english_name
 * @property {'ltr' | 'rtl'} text_direction
 * @property {string | null} fallback
 * @property {string} currency
 * @property {WebLocaleConfig} web
 * @property {AppLocaleConfig} app
 * @property {ReleaseLocaleConfig} release
 *
 * @typedef {Object} LanguageRegistry
 * @property {readonly LanguageEntry[]} languages
 * @property {ReadonlyMap<string, LanguageEntry>} byRouteCode
 */

export class RegistryValidationError extends Error {
  /**
   * @param {string[]} errors
   * @param {string} source
   */
  constructor(errors, source = 'language registry') {
    super(`${source} is invalid:\n${errors.map((error) => `- ${error}`).join('\n')}`);
    this.name = 'RegistryValidationError';
    this.errors = errors;
    this.source = source;
  }
}

/**
 * @param {string | URL} filePath
 * @returns {Promise<LanguageRegistry>}
 */
export async function loadLanguageRegistry(filePath = DEFAULT_LANGUAGE_REGISTRY_URL) {
  const source = filePath instanceof URL ? filePath.pathname : filePath;
  let rawText;

  try {
    rawText = await readFile(filePath, 'utf8');
  } catch (error) {
    throw new Error(`Unable to read ${source}: ${error.message}`);
  }

  let parsed;
  try {
    parsed = JSON.parse(rawText);
  } catch (error) {
    throw new RegistryValidationError([`invalid JSON: ${error.message}`], source);
  }

  return parseLanguageRegistry(parsed, { source });
}

/**
 * @param {unknown} raw
 * @param {{ source?: string }} options
 * @returns {LanguageRegistry}
 */
export function parseLanguageRegistry(raw, { source = 'language registry' } = {}) {
  const errors = [];

  if (!Array.isArray(raw)) {
    throw new RegistryValidationError(['top-level value must be an array'], source);
  }

  /** @type {LanguageEntry[]} */
  const languages = [];
  const routeCounts = new Map();

  raw.forEach((entry, index) => {
    const prefix = `entry ${index}`;
    if (!isRecord(entry)) {
      errors.push(`${prefix}: must be an object`);
      return;
    }

    const routeCode = requiredString(entry, 'route_code', prefix, errors);
    const bcp47 = requiredString(entry, 'bcp47', prefix, errors);
    const nativeName = requiredString(entry, 'native_name', prefix, errors);
    const englishName = requiredString(entry, 'english_name', prefix, errors);
    const textDirection = requiredEnum(entry, 'text_direction', ['ltr', 'rtl'], prefix, errors);
    const fallback = optionalString(entry, 'fallback', prefix, errors);
    const currency = requiredString(entry, 'currency', prefix, errors);

    if (routeCode && !/^[a-z][a-z0-9]*$/.test(routeCode)) {
      errors.push(`${prefix}.route_code: must use lowercase route-code characters`);
    }
    if (bcp47 && !/^[A-Za-z]{2,3}(-[A-Za-z0-9]{2,8})*$/.test(bcp47)) {
      errors.push(`${prefix}.bcp47: must look like a BCP-47 language tag`);
    }
    if (currency && !/^[A-Z]{3}$/.test(currency)) {
      errors.push(`${prefix}.currency: must be a three-letter uppercase currency code`);
    }

    const flutter = parseFlutterConfig(entry.flutter, `${prefix}.flutter`, errors);
    const web = parseWebConfig(entry.web, `${prefix}.web`, errors);
    const app = parseAppConfig(entry.app, `${prefix}.app`, errors);
    const release = parseReleaseConfig(entry.release, `${prefix}.release`, errors);

    if (routeCode) {
      routeCounts.set(routeCode, (routeCounts.get(routeCode) ?? 0) + 1);
    }

    if (
      routeCode &&
      bcp47 &&
      nativeName &&
      englishName &&
      textDirection &&
      currency &&
      flutter &&
      web &&
      app &&
      release
    ) {
      languages.push(Object.freeze({
        route_code: routeCode,
        bcp47,
        flutter,
        native_name: nativeName,
        english_name: englishName,
        text_direction: textDirection,
        fallback,
        currency,
        web,
        app,
        release,
      }));
    }
  });

  for (const [routeCode, count] of routeCounts.entries()) {
    if (count > 1) {
      errors.push(`route_code "${routeCode}" is duplicated`);
    }
  }

  const knownRouteCodes = new Set(languages.map((entry) => entry.route_code));
  for (const entry of languages) {
    if (entry.fallback !== null && !knownRouteCodes.has(entry.fallback)) {
      errors.push(`${entry.route_code}.fallback: "${entry.fallback}" does not match a route_code`);
    }
    if (entry.fallback === entry.route_code) {
      errors.push(`${entry.route_code}.fallback: must not point to itself`);
    }
  }

  if (errors.length > 0) {
    throw new RegistryValidationError(errors, source);
  }

  return Object.freeze({
    languages: Object.freeze(languages),
    byRouteCode: new Map(languages.map((entry) => [entry.route_code, entry])),
  });
}

/**
 * @param {LanguageRegistry} registry
 * @param {string} routeCode
 * @returns {LanguageEntry}
 */
export function getLanguageByRouteCode(registry, routeCode) {
  const entry = registry.byRouteCode.get(routeCode);
  if (!entry) {
    throw new Error(`Unknown route code: ${routeCode}`);
  }
  return entry;
}

function parseFlutterConfig(value, prefix, errors) {
  if (!isRecord(value)) {
    errors.push(`${prefix}: must be an object`);
    return null;
  }

  const languageCode = requiredString(value, 'languageCode', prefix, errors);
  const scriptCode = optionalString(value, 'scriptCode', prefix, errors);
  const countryCode = optionalString(value, 'countryCode', prefix, errors);

  if (languageCode && !/^[a-z]{2,3}$/.test(languageCode)) {
    errors.push(`${prefix}.languageCode: must be a lowercase ISO language code`);
  }
  if (scriptCode && !/^[A-Z][a-z]{3}$/.test(scriptCode)) {
    errors.push(`${prefix}.scriptCode: must be a four-letter title-case script code`);
  }
  if (countryCode && !/^[A-Z]{2}$/.test(countryCode)) {
    errors.push(`${prefix}.countryCode: must be a two-letter uppercase country code`);
  }

  if (!languageCode) {
    return null;
  }

  return Object.freeze({ languageCode, scriptCode, countryCode });
}

function parseWebConfig(value, prefix, errors) {
  if (!isRecord(value)) {
    errors.push(`${prefix}: must be an object`);
    return null;
  }

  const enabled = requiredBoolean(value, 'enabled', prefix, errors);
  const indexed = requiredBoolean(value, 'indexed', prefix, errors);
  const urlPrefix = requiredString(value, 'url_prefix', prefix, errors, { allowEmpty: true });

  if (enabled === null || indexed === null || urlPrefix === null) {
    return null;
  }

  return Object.freeze({ enabled, indexed, url_prefix: urlPrefix });
}

function parseAppConfig(value, prefix, errors) {
  if (!isRecord(value)) {
    errors.push(`${prefix}: must be an object`);
    return null;
  }

  const enabled = requiredBoolean(value, 'enabled', prefix, errors);
  const selectable = requiredBoolean(value, 'selectable', prefix, errors);

  if (enabled === null || selectable === null) {
    return null;
  }

  return Object.freeze({ enabled, selectable });
}

function parseReleaseConfig(value, prefix, errors) {
  if (!isRecord(value)) {
    errors.push(`${prefix}: must be an object`);
    return null;
  }

  const enabled = requiredBoolean(value, 'enabled', prefix, errors);
  const androidStoreLocale = optionalString(value, 'android_store_locale', prefix, errors);
  const iosStoreLocale = optionalString(value, 'ios_store_locale', prefix, errors);

  if (enabled === null) {
    return null;
  }

  return Object.freeze({
    enabled,
    android_store_locale: androidStoreLocale,
    ios_store_locale: iosStoreLocale,
  });
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function requiredString(record, field, prefix, errors, { allowEmpty = false } = {}) {
  const value = record[field];
  if (typeof value !== 'string') {
    errors.push(`${prefix}.${field}: must be a string`);
    return null;
  }
  if (!allowEmpty && value.trim() === '') {
    errors.push(`${prefix}.${field}: must not be empty`);
    return null;
  }
  return value;
}

function optionalString(record, field, prefix, errors) {
  const value = record[field];
  if (value === null) {
    return null;
  }
  if (typeof value !== 'string') {
    errors.push(`${prefix}.${field}: must be a string or null`);
    return null;
  }
  if (value.trim() === '') {
    errors.push(`${prefix}.${field}: must not be empty when present`);
    return null;
  }
  return value;
}

function requiredBoolean(record, field, prefix, errors) {
  const value = record[field];
  if (typeof value !== 'boolean') {
    errors.push(`${prefix}.${field}: must be a boolean`);
    return null;
  }
  return value;
}

function requiredEnum(record, field, allowedValues, prefix, errors) {
  const value = record[field];
  if (!allowedValues.includes(value)) {
    errors.push(`${prefix}.${field}: must be one of ${allowedValues.join(', ')}`);
    return null;
  }
  return value;
}
