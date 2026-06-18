import { readdir, readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { flutterArbSuffix } from './arb.mjs';
import { loadLanguageRegistry } from './registry.mjs';
import { parseLangsFilter } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const APP_ARB_BASE = 'app/lib/l10n/app_en.arb';
const APP_SETTINGS_BASE = 'app/assets/i18n/settings/en.json';
const API_CHECKOUT_BASE = 'api/content/i18n/checkout/en.json';
const RELEASE_METADATA_BASE = 'release/store_metadata/source/en.json';
const API_CATALOG_FILES = [
  'api/content/i18n/catalog/materials.json',
  'api/content/i18n/catalog/stone_listings.json',
  'api/content/i18n/catalog/facet_tags.json',
  'api/content/i18n/catalog/countries.json',
];
const INTENTION_ROOTS = [
  'app/lib/l10n',
  'app/assets/i18n',
  'web/content/i18n',
  'api/content/i18n',
  'release/store_metadata/source',
];
const ALLOWED_REASON_CODES = new Set([
  'brand_name',
  'code_literal',
  'code_or_identifier',
  'country_code',
  'currency_code',
  'email',
  'font_name',
  'intentionally_english',
  'kanji_character',
  'law_name',
  'legal_entity',
  'legal_entity_name',
  'locale_not_release_enabled',
  'payment_provider',
  'pending_human_translation',
  'product_model_or_font',
  'product_name',
  'source_not_available',
  'technical_identifier',
  'url',
  'url_or_email',
]);

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} IntentionIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} IntentionReport
 * @property {boolean} ok
 * @property {IntentionIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 * @returns {Promise<IntentionReport>}
 */
export async function validateIntentions({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const fileFilter = normalizeFilterPath(file);
  if (fileFilter && !isIntentionPath(fileFilter) && !isTranslatableContentPath(fileFilter)) {
    return { ok: true, issues: [], parsed_files: [] };
  }

  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  const registry = await loadLanguageRegistry(new URL('config/languages.json', rootUrl));
  const selectedLanguages = selectIntentionLanguages(registry.languages, langs);
  const reader = createJsonReader(rootDir);
  const issues = [];
  const parsed = new Set();
  const sidecarCache = new Map();

  const existingSidecars = await collectIntentionSidecars(rootDir);
  for (const sidecarPath of existingSidecars) {
    if (fileFilter && sidecarPath !== fileFilter) {
      continue;
    }
    const sidecar = await readSidecar(reader, sidecarCache, sidecarPath, inferLocaleFromSidecar(sidecarPath));
    parsed.add(sidecarPath);
    issues.push(...sidecar.issues);
  }
  if (fileFilter && isIntentionPath(fileFilter) && !existingSidecars.includes(fileFilter)) {
    const sidecar = await readSidecar(reader, sidecarCache, fileFilter, inferLocaleFromSidecar(fileFilter));
    if (sidecar.exists) {
      parsed.add(fileFilter);
    }
    issues.push(...sidecar.issues);
    return { ok: issues.length === 0, issues, parsed_files: [...parsed].sort() };
  }

  const descriptors = [
    ...appArbDescriptors(selectedLanguages, langs),
    ...appSettingsDescriptors(selectedLanguages, langs),
    ...(await webContentDescriptors(rootDir, selectedLanguages, langs)),
    ...apiCheckoutDescriptors(selectedLanguages, langs),
    ...(await releaseMetadataDescriptors(rootDir, selectedLanguages, langs, fileFilter)),
  ];

  for (const descriptor of descriptors) {
    if (!matchesFileFilter(fileFilter, descriptor.basePath, descriptor.targetPath, descriptor.sidecarPath)) {
      continue;
    }
    const base = await reader.readOptional(descriptor.basePath);
    const target = await reader.readOptional(descriptor.targetPath);
    if (!base.ok || !target.ok) {
      continue;
    }
    parsed.add(descriptor.basePath);
    parsed.add(descriptor.targetPath);
    const sidecar = await readSidecar(reader, sidecarCache, descriptor.sidecarPath, descriptor.locale);
    if (sidecar.exists) {
      parsed.add(descriptor.sidecarPath);
    }
    issues.push(
      ...sameEnglishIssues({
        file: descriptor.targetPath,
        sidecarPath: descriptor.sidecarPath,
        locale: descriptor.locale,
        baseValues: descriptor.kind === 'arb' ? flattenArb(base.value) : flattenJsonStrings(base.value),
        targetValues: descriptor.kind === 'arb' ? flattenArb(target.value) : flattenJsonStrings(target.value),
        sidecar,
      }),
    );
  }

  for (const catalogPath of API_CATALOG_FILES) {
    if (!matchesFileFilter(fileFilter, catalogPath, catalogSidecarPath(catalogPath))) {
      continue;
    }
    const catalog = await reader.readOptional(catalogPath);
    if (!catalog.ok) {
      continue;
    }
    parsed.add(catalogPath);
    const sidecarPath = catalogSidecarPath(catalogPath);
    const sidecar = await readSidecar(reader, sidecarCache, sidecarPath, null);
    if (sidecar.exists) {
      parsed.add(sidecarPath);
    }
    issues.push(...catalogSameEnglishIssues(catalog.value, catalogPath, sidecarPath, selectedLanguages, sidecar));
  }

  return {
    ok: issues.length === 0,
    issues,
    parsed_files: [...parsed].sort(),
  };
}

export function isAllowedReasonCode(reasonCode) {
  return ALLOWED_REASON_CODES.has(reasonCode);
}

/**
 * @param {unknown} raw
 * @param {{ path: string, targetLocale?: string | null }} options
 * @returns {{ exists: boolean, approvals: Map<string, string>, issues: IntentionIssue[] }}
 */
export function parseIntentionSidecar(raw, { path, targetLocale = null }) {
  const approvals = new Map();
  const issues = [];
  if (!isRecord(raw)) {
    return {
      exists: true,
      approvals,
      issues: [createIssue('intention-format', path, null, 'sidecar top-level value must be an object')],
    };
  }

  if (Array.isArray(raw.entries)) {
    raw.entries.forEach((entry, index) => {
      const prefix = `entries[${index}]`;
      if (!isRecord(entry)) {
        issues.push(createIssue('intention-format', path, prefix, 'entry must be an object'));
        return;
      }
      const keyPath = entry.key_path;
      const reasonCode = entry.reason_code;
      if (typeof keyPath !== 'string' || keyPath.trim() === '') {
        issues.push(createIssue('intention-format', path, prefix, 'entry.key_path must be a non-empty string'));
        return;
      }
      if (typeof reasonCode !== 'string' || reasonCode.trim() === '') {
        issues.push(createIssue('intention-format', path, keyPath, 'entry.reason_code must be a non-empty string'));
        return;
      }
      if (!isAllowedReasonCode(reasonCode)) {
        issues.push(createIssue('intention-reason', path, keyPath, `reason_code "${reasonCode}" is not allowed`));
        return;
      }
      if (
        targetLocale &&
        typeof entry.target_locale === 'string' &&
        entry.target_locale.trim() !== '' &&
        entry.target_locale !== targetLocale
      ) {
        issues.push(
          createIssue('intention-locale', path, keyPath, `target_locale "${entry.target_locale}" does not match ${targetLocale}`),
        );
        return;
      }
      approvals.set(keyPath, reasonCode);
    });
    return { exists: true, approvals, issues };
  }

  for (const [keyPath, reasonCode] of Object.entries(raw)) {
    if (typeof reasonCode !== 'string' || reasonCode.trim() === '') {
      issues.push(createIssue('intention-format', path, keyPath, 'reason code must be a non-empty string'));
      continue;
    }
    if (!isAllowedReasonCode(reasonCode)) {
      issues.push(createIssue('intention-reason', path, keyPath, `reason_code "${reasonCode}" is not allowed`));
      continue;
    }
    approvals.set(keyPath, reasonCode);
  }

  return { exists: true, approvals, issues };
}

function sameEnglishIssues({ file, sidecarPath, locale, baseValues, targetValues, sidecar }) {
  const issues = [];
  for (const [key, baseValue] of baseValues.entries()) {
    if (!targetValues.has(key)) {
      continue;
    }
    const targetValue = targetValues.get(key);
    if (!isActionableSameValue(baseValue, targetValue)) {
      continue;
    }
    if (hasApproval(sidecar, key, locale)) {
      continue;
    }
    issues.push(
      createIssue(
        'intention-missing',
        file,
        key,
        `same-as-English value must be approved in ${sidecarPath}`,
      ),
    );
  }
  return issues;
}

function catalogSameEnglishIssues(catalog, file, sidecarPath, languages, sidecar) {
  const issues = [];
  for (const entry of flattenCatalogEnglishEntries(catalog)) {
    for (const language of languages) {
      const targetValue = catalogValueAt(catalog, `${entry.key}.${language.route_code}`);
      if (!isActionableSameValue(entry.value, targetValue)) {
        continue;
      }
      if (hasApproval(sidecar, `${entry.key}.${language.route_code}`, language.route_code) || hasApproval(sidecar, entry.key, language.route_code)) {
        continue;
      }
      issues.push(
        createIssue(
          'intention-missing',
          file,
          `${entry.key}.${language.route_code}`,
          `same-as-English value must be approved in ${sidecarPath}`,
        ),
      );
    }
  }
  return issues;
}

function hasApproval(sidecar, key, locale) {
  return (
    sidecar.approvals.has(key) ||
    sidecar.approvals.has(`${locale}.${key}`) ||
    sidecar.approvals.has(`${key}.${locale}`)
  );
}

function isActionableSameValue(baseValue, targetValue) {
  return (
    typeof baseValue === 'string' &&
    typeof targetValue === 'string' &&
    baseValue === targetValue &&
    targetValue.trim() !== ''
  );
}

function flattenArb(value) {
  const results = new Map();
  if (!isRecord(value)) {
    return results;
  }
  for (const [key, entryValue] of Object.entries(value)) {
    if (key.startsWith('@')) {
      continue;
    }
    if (typeof entryValue === 'string') {
      results.set(key, entryValue);
    }
  }
  return results;
}

function flattenJsonStrings(value, prefix = '') {
  const results = new Map();
  if (Array.isArray(value)) {
    value.forEach((entry, index) => {
      for (const [key, nestedValue] of flattenJsonStrings(entry, `${prefix}[${index}]`)) {
        results.set(key, nestedValue);
      }
    });
    return results;
  }
  if (isRecord(value)) {
    for (const [key, entryValue] of Object.entries(value)) {
      const nextPrefix = prefix ? `${prefix}.${key}` : key;
      for (const [nestedKey, nestedValue] of flattenJsonStrings(entryValue, nextPrefix)) {
        results.set(nestedKey, nestedValue);
      }
    }
    return results;
  }
  if (prefix && typeof value === 'string') {
    results.set(prefix, value);
  }
  return results;
}

function flattenCatalogEnglishEntries(value, prefix = '') {
  const results = [];
  if (Array.isArray(value)) {
    value.forEach((entry, index) => {
      results.push(...flattenCatalogEnglishEntries(entry, `${prefix}[${index}]`));
    });
    return results;
  }
  if (!isRecord(value)) {
    return results;
  }
  if (Object.hasOwn(value, 'en') && typeof value.en === 'string') {
    results.push({ key: prefix, value: value.en });
    return results;
  }
  for (const [key, entryValue] of Object.entries(value)) {
    const nextPrefix = prefix ? `${prefix}.${key}` : key;
    results.push(...flattenCatalogEnglishEntries(entryValue, nextPrefix));
  }
  return results;
}

function catalogValueAt(catalog, dottedPath) {
  let current = catalog;
  for (const part of dottedPath.split('.')) {
    if (!isRecord(current) || !Object.hasOwn(current, part)) {
      return undefined;
    }
    current = current[part];
  }
  return current;
}

function appArbDescriptors(languages, langs) {
  return languages
    .filter((language) => language.app.enabled || langs)
    .map((language) => {
      const suffix = flutterArbSuffix(language);
      return {
        kind: 'arb',
        locale: language.route_code,
        basePath: APP_ARB_BASE,
        targetPath: `app/lib/l10n/app_${suffix}.arb`,
        sidecarPath: `app/lib/l10n/app_${suffix}_intentions.json`,
      };
    });
}

function appSettingsDescriptors(languages, langs) {
  return languages
    .filter((language) => language.app.enabled || langs)
    .map((language) => ({
      kind: 'json',
      locale: language.route_code,
      basePath: APP_SETTINGS_BASE,
      targetPath: `app/assets/i18n/settings/${language.route_code}.json`,
      sidecarPath: `app/assets/i18n/settings/${language.route_code}_intentions.json`,
    }));
}

async function webContentDescriptors(rootDir, languages, langs) {
  const namespaces = await discoverWebNamespaces(rootDir);
  return languages
    .filter((language) => language.web.enabled || langs)
    .flatMap((language) =>
      namespaces.map((namespace) => ({
        kind: 'json',
        locale: language.route_code,
        basePath: `web/content/i18n/${namespace}/en.json`,
        targetPath: `web/content/i18n/${namespace}/${language.route_code}.json`,
        sidecarPath: `web/content/i18n/${namespace}/${language.route_code}_intentions.json`,
      })),
    );
}

function apiCheckoutDescriptors(languages, langs) {
  return languages
    .filter((language) => language.web.enabled || langs)
    .map((language) => ({
      kind: 'json',
      locale: language.route_code,
      basePath: API_CHECKOUT_BASE,
      targetPath: `api/content/i18n/checkout/${language.route_code}.json`,
      sidecarPath: `api/content/i18n/checkout/${language.route_code}_intentions.json`,
    }));
}

async function releaseMetadataDescriptors(rootDir, languages, langs, fileFilter) {
  const shouldInspectRelease = !fileFilter || fileFilter.startsWith('release/store_metadata/source/');
  if (!shouldInspectRelease && !langs) {
    return [];
  }
  if (!(await fileExists(join(rootDir, RELEASE_METADATA_BASE)))) {
    return [];
  }
  return languages
    .filter((language) => language.release.enabled || langs)
    .map((language) => ({
      kind: 'json',
      locale: language.route_code,
      basePath: RELEASE_METADATA_BASE,
      targetPath: `release/store_metadata/source/${language.route_code}.json`,
      sidecarPath: `release/store_metadata/source/${language.route_code}_intentions.json`,
    }));
}

function selectIntentionLanguages(languages, langs) {
  const baseFiltered = languages.filter((language) => language.route_code !== 'en');
  if (!langs) {
    return baseFiltered.filter(
      (language) => language.app.enabled || language.web.enabled || language.release.enabled,
    );
  }
  if (langs.includes('all')) {
    return baseFiltered;
  }

  const byRouteCode = new Map(languages.map((language) => [language.route_code, language]));
  const unknownCodes = langs.filter((code) => code !== 'en' && !byRouteCode.has(code));
  if (unknownCodes.length > 0) {
    throw new Error(`Unknown LANGS route code(s): ${unknownCodes.join(', ')}`);
  }
  return langs
    .map((code) => byRouteCode.get(code))
    .filter((language) => language && language.route_code !== 'en');
}

async function discoverWebNamespaces(rootDir) {
  const root = 'web/content/i18n';
  let entries;
  try {
    entries = await readdir(join(rootDir, root), { withFileTypes: true });
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return [];
    }
    throw error;
  }

  const namespaces = [];
  for (const entry of entries) {
    if (!entry.isDirectory()) {
      continue;
    }
    const basePath = `${root}/${entry.name}/en.json`;
    if (await fileExists(join(rootDir, basePath))) {
      namespaces.push(entry.name);
    }
  }
  return namespaces.sort();
}

async function collectIntentionSidecars(rootDir) {
  const results = [];
  for (const root of INTENTION_ROOTS) {
    await visit(root);
  }
  return results.sort();

  async function visit(relativeDir) {
    let entries;
    try {
      entries = await readdir(join(rootDir, relativeDir), { withFileTypes: true });
    } catch (error) {
      if (error?.code === 'ENOENT') {
        return;
      }
      throw error;
    }
    for (const entry of entries) {
      const relativePath = `${relativeDir}/${entry.name}`;
      if (entry.isDirectory()) {
        await visit(relativePath);
      } else if (entry.isFile() && isIntentionPath(relativePath)) {
        results.push(relativePath);
      }
    }
  }
}

async function readSidecar(reader, cache, sidecarPath, targetLocale) {
  if (cache.has(sidecarPath)) {
    return cache.get(sidecarPath);
  }
  const raw = await reader.readOptional(sidecarPath);
  if (!raw.ok) {
    const result = { exists: false, approvals: new Map(), issues: [] };
    cache.set(sidecarPath, result);
    return result;
  }
  const result = parseIntentionSidecar(raw.value, { path: sidecarPath, targetLocale });
  cache.set(sidecarPath, result);
  return result;
}

function createJsonReader(rootDir) {
  const cache = new Map();
  return {
    async readOptional(relativePath) {
      if (cache.has(relativePath)) {
        return cache.get(relativePath);
      }
      let rawText;
      try {
        rawText = await readFile(join(rootDir, relativePath), 'utf8');
      } catch (error) {
        if (error?.code === 'ENOENT') {
          const result = { ok: false, code: 'missing-json', message: 'file is missing' };
          cache.set(relativePath, result);
          return result;
        }
        throw error;
      }
      try {
        const result = { ok: true, value: JSON.parse(rawText) };
        cache.set(relativePath, result);
        return result;
      } catch (error) {
        const result = { ok: false, code: 'malformed-json', message: error.message };
        cache.set(relativePath, result);
        return result;
      }
    },
  };
}

async function fileExists(path) {
  try {
    const stats = await stat(path);
    return stats.isFile();
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return false;
    }
    throw error;
  }
}

function matchesFileFilter(fileFilter, ...paths) {
  if (!fileFilter) {
    return true;
  }
  return paths.some((path) => normalizeFilterPath(path) === fileFilter);
}

function isTranslatableContentPath(path) {
  return (
    path.endsWith('.arb') ||
    path.startsWith('app/assets/i18n/') ||
    path.startsWith('web/content/i18n/') ||
    path.startsWith('api/content/i18n/') ||
    path.startsWith('release/store_metadata/source/')
  );
}

function isIntentionPath(path) {
  return path.endsWith('_intentions.json');
}

function catalogSidecarPath(filePath) {
  return filePath.replace(/\.json$/, '_intentions.json');
}

function inferLocaleFromSidecar(sidecarPath) {
  const fileName = sidecarPath.split('/').pop() ?? '';
  if (sidecarPath.startsWith('app/lib/l10n/app_')) {
    return fileName.replace(/^app_/, '').replace(/_intentions\.json$/, '').replace(/^zh_Hant$/, 'zhtw');
  }
  if (fileName.endsWith('_intentions.json')) {
    const locale = fileName.replace(/_intentions\.json$/, '');
    if (/^[a-z][a-z0-9]*$/.test(locale)) {
      return locale;
    }
  }
  return null;
}

function normalizeFilterPath(file) {
  if (!file || !file.trim()) {
    return null;
  }
  return file.trim().replace(/^\.\//, '');
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await validateIntentions({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    if (report.issues.length === 0) {
      process.stdout.write(`Intention validation passed (${report.parsed_files.length} files).\n`);
    } else {
      process.stdout.write(`Intention validation failed (${report.issues.length} issues).\n`);
      for (const issue of report.issues) {
        const key = issue.key ? ` ${issue.key}` : '';
        process.stdout.write(`- ${issue.code}: ${issue.file}${key} - ${issue.message}\n`);
      }
      process.exitCode = 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
