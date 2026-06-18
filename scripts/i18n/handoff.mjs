import { mkdir, readdir, readFile, stat, writeFile } from 'node:fs/promises';
import { dirname, isAbsolute, join } from 'node:path';
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

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} HandoffEntry
 * @property {string} key_path
 * @property {string} source_value
 * @property {string} target_value
 * @property {'present' | 'missing'} status
 *
 * @typedef {Object} HandoffFile
 * @property {'arb' | 'json' | 'catalog'} kind
 * @property {string} source_file
 * @property {string} target_file
 * @property {string} target_locale
 * @property {string} sidecar_file
 * @property {HandoffEntry[]} entries
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 */
export async function exportHandoff({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const registry = await loadRegistry(rootDir);
  const languages = selectTargetLanguages(registry.languages, langs);
  const fileFilter = normalizeFilterPath(file);
  const reader = createJsonReader(rootDir);
  const files = [];

  for (const descriptor of await buildDescriptors(rootDir, languages, langs)) {
    if (!matchesFileFilter(fileFilter, descriptor.source_file, descriptor.target_file, descriptor.sidecar_file)) {
      continue;
    }
    const source = await reader.readRequired(descriptor.source_file);
    const target = await reader.readOptional(descriptor.target_file);
    const sourceValues = descriptor.kind === 'arb'
      ? flattenArb(source)
      : flattenJsonStrings(source);
    const targetValues = target.ok
      ? descriptor.kind === 'arb'
        ? flattenArb(target.value)
        : flattenJsonStrings(target.value)
      : new Map();
    files.push({
      kind: descriptor.kind,
      source_file: descriptor.source_file,
      target_file: descriptor.target_file,
      target_locale: descriptor.target_locale,
      sidecar_file: descriptor.sidecar_file,
      entries: [...sourceValues.entries()].map(([keyPath, sourceValue]) => ({
        key_path: keyPath,
        source_value: sourceValue,
        target_value: targetValues.get(keyPath) ?? '',
        status: targetValues.has(keyPath) ? 'present' : 'missing',
      })),
    });
  }

  for (const catalogFile of API_CATALOG_FILES) {
    if (!matchesFileFilter(fileFilter, catalogFile, catalogSidecarPath(catalogFile))) {
      continue;
    }
    const catalog = await reader.readOptional(catalogFile);
    if (!catalog.ok) {
      continue;
    }
    for (const language of languages.filter((entry) => entry.web.enabled || langs)) {
      const entries = flattenCatalogEnglishEntries(catalog.value).map((entry) => {
        const targetValue = catalogValueAt(catalog.value, `${entry.key_path}.${language.route_code}`);
        return {
          key_path: entry.key_path,
          source_value: entry.source_value,
          target_value: typeof targetValue === 'string' ? targetValue : '',
          status: typeof targetValue === 'string' ? 'present' : 'missing',
        };
      });
      files.push({
        kind: 'catalog',
        source_file: catalogFile,
        target_file: catalogFile,
        target_locale: language.route_code,
        sidecar_file: catalogSidecarPath(catalogFile),
        entries,
      });
    }
  }

  files.sort((left, right) =>
    [left.target_file, left.target_locale, left.kind].join('\0').localeCompare(
      [right.target_file, right.target_locale, right.kind].join('\0'),
    ),
  );

  return {
    format_version: 1,
    source_locale: 'en',
    target_locales: languages.map((language) => language.route_code),
    file_filter: fileFilter,
    files,
  };
}

/**
 * @param {{ rootDir?: string, handoff: unknown }} options
 */
export async function importHandoff({ rootDir = REPO_ROOT, handoff }) {
  validateHandoff(handoff);
  const reader = createJsonReader(rootDir);
  const mutableFiles = new Map();

  async function getMutableFile(file) {
    if (mutableFiles.has(file.target_file)) {
      return mutableFiles.get(file.target_file);
    }

    if (file.kind === 'catalog') {
      const catalog = await reader.readRequired(file.target_file);
      mutableFiles.set(file.target_file, catalog);
      return catalog;
    }

    const target = await reader.readOptional(file.target_file);
    const source = await reader.readRequired(file.source_file);
    const value = target.ok ? target.value : cloneTranslatableSkeleton(source, file);
    mutableFiles.set(file.target_file, value);
    return value;
  }

  for (const file of handoff.files) {
    const nextValue = await getMutableFile(file);
    if (file.kind === 'catalog') {
      for (const entry of file.entries) {
        setCatalogValue(nextValue, `${entry.key_path}.${file.target_locale}`, entry.target_value);
      }
      continue;
    }

    for (const entry of file.entries) {
      setPathValue(nextValue, entry.key_path, entry.target_value);
    }
  }

  const changedFiles = [...mutableFiles.keys()].sort();
  for (const targetFile of changedFiles) {
    await writeJson(rootDir, targetFile, mutableFiles.get(targetFile));
  }
  return changedFiles;
}

/**
 * @param {unknown} handoff
 */
export function validateHandoff(handoff) {
  if (!isRecord(handoff)) {
    throw new Error('handoff must be an object');
  }
  if (handoff.format_version !== 1) {
    throw new Error('handoff format_version must be 1');
  }
  if (!Array.isArray(handoff.files)) {
    throw new Error('handoff files must be an array');
  }
  for (const [index, file] of handoff.files.entries()) {
    const prefix = `files[${index}]`;
    if (!isRecord(file)) {
      throw new Error(`${prefix} must be an object`);
    }
    if (!['arb', 'json', 'catalog'].includes(file.kind)) {
      throw new Error(`${prefix}.kind is invalid`);
    }
    for (const field of ['source_file', 'target_file', 'target_locale', 'sidecar_file']) {
      if (typeof file[field] !== 'string' || file[field].trim() === '') {
        throw new Error(`${prefix}.${field} must be a non-empty string`);
      }
    }
    if (!Array.isArray(file.entries)) {
      throw new Error(`${prefix}.entries must be an array`);
    }
    for (const [entryIndex, entry] of file.entries.entries()) {
      const entryPrefix = `${prefix}.entries[${entryIndex}]`;
      if (!isRecord(entry)) {
        throw new Error(`${entryPrefix} must be an object`);
      }
      for (const field of ['key_path', 'source_value', 'target_value']) {
        if (typeof entry[field] !== 'string') {
          throw new Error(`${entryPrefix}.${field} must be a string`);
        }
      }
      if (!['present', 'missing'].includes(entry.status)) {
        throw new Error(`${entryPrefix}.status is invalid`);
      }
    }
  }
}

async function buildDescriptors(rootDir, languages, langs) {
  return [
    ...appArbDescriptors(languages, langs),
    ...appSettingsDescriptors(languages, langs),
    ...(await webContentDescriptors(rootDir, languages, langs)),
    ...apiCheckoutDescriptors(languages, langs),
    ...(await releaseMetadataDescriptors(rootDir, languages, langs)),
  ];
}

function appArbDescriptors(languages, langs) {
  return languages
    .filter((language) => language.app.enabled || langs)
    .map((language) => {
      const suffix = flutterArbSuffix(language);
      return {
        kind: 'arb',
        source_file: APP_ARB_BASE,
        target_file: `app/lib/l10n/app_${suffix}.arb`,
        target_locale: language.route_code,
        sidecar_file: `app/lib/l10n/app_${suffix}_intentions.json`,
      };
    });
}

function appSettingsDescriptors(languages, langs) {
  return languages
    .filter((language) => language.app.enabled || langs)
    .map((language) => ({
      kind: 'json',
      source_file: APP_SETTINGS_BASE,
      target_file: `app/assets/i18n/settings/${language.route_code}.json`,
      target_locale: language.route_code,
      sidecar_file: `app/assets/i18n/settings/${language.route_code}_intentions.json`,
    }));
}

async function webContentDescriptors(rootDir, languages, langs) {
  const namespaces = await discoverWebNamespaces(rootDir);
  return languages
    .filter((language) => language.web.enabled || langs)
    .flatMap((language) =>
      namespaces.map((namespace) => ({
        kind: 'json',
        source_file: `web/content/i18n/${namespace}/en.json`,
        target_file: `web/content/i18n/${namespace}/${language.route_code}.json`,
        target_locale: language.route_code,
        sidecar_file: `web/content/i18n/${namespace}/${language.route_code}_intentions.json`,
      })),
    );
}

function apiCheckoutDescriptors(languages, langs) {
  return languages
    .filter((language) => language.web.enabled || langs)
    .map((language) => ({
      kind: 'json',
      source_file: API_CHECKOUT_BASE,
      target_file: `api/content/i18n/checkout/${language.route_code}.json`,
      target_locale: language.route_code,
      sidecar_file: `api/content/i18n/checkout/${language.route_code}_intentions.json`,
    }));
}

async function releaseMetadataDescriptors(rootDir, languages, langs) {
  if (!(await fileExists(join(rootDir, RELEASE_METADATA_BASE)))) {
    return [];
  }
  return languages
    .filter((language) => language.release.enabled || langs)
    .map((language) => ({
      kind: 'json',
      source_file: RELEASE_METADATA_BASE,
      target_file: `release/store_metadata/source/${language.route_code}.json`,
      target_locale: language.route_code,
      sidecar_file: `release/store_metadata/source/${language.route_code}_intentions.json`,
    }));
}

async function loadRegistry(rootDir) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  return loadLanguageRegistry(new URL('config/languages.json', rootUrl));
}

function selectTargetLanguages(languages, langs) {
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
  if (typeof value.en === 'string') {
    results.push({
      key_path: prefix,
      source_value: value.en,
    });
    return results;
  }
  for (const [key, entryValue] of Object.entries(value)) {
    results.push(...flattenCatalogEnglishEntries(entryValue, prefix ? `${prefix}.${key}` : key));
  }
  return results;
}

function catalogValueAt(catalog, keyPath) {
  let current = catalog;
  for (const segment of parseKeyPath(keyPath)) {
    if (!isRecord(current) && !Array.isArray(current)) {
      return undefined;
    }
    current = current[segment];
  }
  return current;
}

function setCatalogValue(catalog, keyPath, value) {
  setPathValue(catalog, keyPath, value);
}

function setPathValue(root, keyPath, value) {
  const segments = parseKeyPath(keyPath);
  if (segments.length === 0) {
    throw new Error('key_path must not be empty');
  }

  let current = root;
  for (let index = 0; index < segments.length - 1; index += 1) {
    const segment = segments[index];
    const nextSegment = segments[index + 1];
    if (!isRecord(current) && !Array.isArray(current)) {
      throw new Error(`Cannot set ${keyPath}: ${segment} is not a container`);
    }
    if (current[segment] === undefined) {
      current[segment] = typeof nextSegment === 'number' ? [] : {};
    }
    current = current[segment];
  }
  current[segments.at(-1)] = value;
}

function parseKeyPath(keyPath) {
  const segments = [];
  const pattern = /([^.[\]]+)|\[(\d+)\]/g;
  let match;
  while ((match = pattern.exec(keyPath)) !== null) {
    segments.push(match[1] ?? Number(match[2]));
  }
  if (segments.length === 0 && keyPath) {
    throw new Error(`Invalid key_path: ${keyPath}`);
  }
  return segments;
}

function cloneTranslatableSkeleton(source, file) {
  if (file.kind === 'arb') {
    const skeleton = {};
    for (const [key, value] of Object.entries(source)) {
      if (key === '@@locale') {
        skeleton[key] = expectedLocaleFromArbPath(file.target_file) ?? file.target_locale;
      } else if (key.startsWith('@')) {
        skeleton[key] = value;
      } else if (typeof value === 'string') {
        skeleton[key] = '';
      } else {
        skeleton[key] = cloneJson(value);
      }
    }
    return skeleton;
  }
  return cloneJsonStringSkeleton(source);
}

function expectedLocaleFromArbPath(targetFile) {
  const match = targetFile.match(/app_([^/]+)\.arb$/);
  return match?.[1] ?? null;
}

function cloneJsonStringSkeleton(value) {
  if (Array.isArray(value)) {
    return value.map((entry) => cloneJsonStringSkeleton(entry));
  }
  if (isRecord(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, entryValue]) => [key, cloneJsonStringSkeleton(entryValue)]),
    );
  }
  if (typeof value === 'string') {
    return '';
  }
  return cloneJson(value);
}

async function discoverWebNamespaces(rootDir) {
  const baseDir = join(rootDir, 'web/content/i18n');
  try {
    const entries = await readdir(baseDir, { withFileTypes: true });
    const namespaces = [];
    for (const entry of entries) {
      if (!entry.isDirectory()) {
        continue;
      }
      if (await fileExists(join(baseDir, entry.name, 'en.json'))) {
        namespaces.push(entry.name);
      }
    }
    return namespaces.sort();
  } catch (error) {
    if (error && error.code === 'ENOENT') {
      return [];
    }
    throw error;
  }
}

function createJsonReader(rootDir) {
  const cache = new Map();
  return {
    async readRequired(relativePath) {
      const result = await this.readOptional(relativePath);
      if (!result.ok) {
        throw new Error(`Missing required JSON file: ${relativePath}`);
      }
      return result.value;
    },
    async readOptional(relativePath) {
      if (cache.has(relativePath)) {
        return { ok: true, value: cloneJson(cache.get(relativePath)) };
      }
      try {
        const value = JSON.parse(await readFile(join(rootDir, relativePath), 'utf8'));
        cache.set(relativePath, value);
        return { ok: true, value: cloneJson(value) };
      } catch (error) {
        if (error && error.code === 'ENOENT') {
          return { ok: false };
        }
        throw error;
      }
    },
  };
}

async function writeJson(rootDir, relativePath, value) {
  const path = join(rootDir, relativePath);
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeHandoffOutput(rootDir, outputPath, handoff) {
  const serialized = `${JSON.stringify(handoff, null, 2)}\n`;
  if (!outputPath) {
    process.stdout.write(serialized);
    return;
  }
  const path = repoPath(rootDir, outputPath);
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, serialized);
}

async function fileExists(path) {
  try {
    return (await stat(path)).isFile();
  } catch (error) {
    if (error && error.code === 'ENOENT') {
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

function normalizeFilterPath(file) {
  return file ? file.replaceAll('\\', '/').replace(/^\.\//, '') : null;
}

function catalogSidecarPath(filePath) {
  return filePath.replace(/\.json$/, '_intentions.json');
}

function cloneJson(value) {
  return JSON.parse(JSON.stringify(value));
}

function repoPath(rootDir, path) {
  return isAbsolute(path) ? path : join(rootDir, path);
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const command = process.argv[2] ?? 'export';

  if (command === 'export') {
    const handoff = await exportHandoff({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    await writeHandoffOutput(REPO_ROOT, process.env.OUT ?? null, handoff);
  } else if (command === 'import') {
    if (!process.env.IN) {
      throw new Error('IN=<handoff.json> is required for i18n import');
    }
    const handoff = JSON.parse(await readFile(repoPath(REPO_ROOT, process.env.IN), 'utf8'));
    const changedFiles = await importHandoff({ handoff });
    process.stdout.write(`${JSON.stringify({ changed_files: changedFiles }, null, 2)}\n`);
  } else {
    throw new Error(`Unknown i18n handoff command: ${command}`);
  }
}
