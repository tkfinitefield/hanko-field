import { readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const WEB_COPY_NAMESPACES = [
  'common',
  'top',
  'design',
  'about',
  'blog_index',
  'payment_success',
  'payment_failure',
  'terms',
  'commercial_transactions',
];

const API_CATALOG_FILES = [
  'api/content/i18n/catalog/materials.json',
  'api/content/i18n/catalog/stone_listings.json',
  'api/content/i18n/catalog/facet_tags.json',
  'api/content/i18n/catalog/countries.json',
];

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} I18nTodoItem
 * @property {'missing-key' | 'missing-file'} type
 * @property {string} file
 * @property {string} locale
 * @property {string} key
 * @property {string} base_english_value
 * @property {string} fallback_value
 * @property {string} sidecar_path
 *
 * @typedef {Object} I18nTodoReport
 * @property {I18nTodoItem[]} items
 * @property {string[]} languages
 * @property {string | null} file_filter
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 * @returns {Promise<I18nTodoReport>}
 */
export async function buildI18nTodo({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  const registry = await loadLanguageRegistry(new URL('config/languages.json', rootUrl));
  const targetLanguages = selectTargetLanguages(registry.languages, langs);
  const fileFilter = normalizeFilterPath(file);
  const reader = createJsonReader(rootDir);

  const items = [];
  items.push(
    ...(await buildPerLocaleFileTodos({
      rootDir,
      reader,
      languages: targetLanguages.filter((language) => language.app.enabled || langs),
      fileFilter,
      descriptors: appDescriptors(targetLanguages),
    })),
  );
  items.push(
    ...(await buildPerLocaleFileTodos({
      rootDir,
      reader,
      languages: targetLanguages.filter((language) => language.web.enabled || langs),
      fileFilter,
      descriptors: webDescriptors(targetLanguages),
    })),
  );
  items.push(
    ...(await buildPerLocaleFileTodos({
      rootDir,
      reader,
      languages: targetLanguages.filter((language) => language.web.enabled || langs),
      fileFilter,
      descriptors: checkoutDescriptors(targetLanguages),
    })),
  );
  items.push(
    ...(await buildCatalogTodos({
      reader,
      languages: targetLanguages.filter((language) => language.web.enabled || langs),
      fileFilter,
      registryLanguages: registry.languages,
    })),
  );

  items.sort((left, right) =>
    [left.file, left.locale, left.key].join('\0').localeCompare(
      [right.file, right.locale, right.key].join('\0'),
    ),
  );

  return {
    items,
    languages: targetLanguages.map((language) => language.route_code),
    file_filter: fileFilter,
  };
}

/**
 * @param {I18nTodoReport} report
 * @returns {string}
 */
export function renderI18nTodo(report) {
  const lines = [
    '# Stone Signature i18n todo',
    '',
    `Languages: ${report.languages.length === 0 ? 'none' : report.languages.join(', ')}`,
    `File filter: ${report.file_filter ?? 'none'}`,
    `Items: ${report.items.length}`,
    '',
  ];

  if (report.items.length === 0) {
    lines.push('No missing i18n items.');
    return `${lines.join('\n')}\n`;
  }

  lines.push(
    '| file | locale | key | base English value | fallback value | sidecar path |',
    '| --- | --- | --- | --- | --- | --- |',
  );

  for (const item of report.items) {
    lines.push(
      `| ${md(item.file)} | ${md(item.locale)} | ${md(item.key)} | ${md(item.base_english_value)} | ${md(item.fallback_value)} | ${md(item.sidecar_path)} |`,
    );
  }

  return `${lines.join('\n')}\n`;
}

/**
 * @param {string | null | undefined} raw
 * @returns {string[] | null}
 */
export function parseLangsFilter(raw) {
  if (!raw || !raw.trim()) {
    return null;
  }
  return raw
    .split(',')
    .map((code) => code.trim().toLowerCase())
    .filter(Boolean);
}

/**
 * @param {LanguageEntry[]} languages
 * @param {string[] | null} langs
 * @returns {LanguageEntry[]}
 */
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

/**
 * @param {LanguageEntry[]} languages
 */
function appDescriptors(languages) {
  return languages.flatMap((language) => {
    const arbSuffix = flutterArbSuffix(language);
    return [
      {
        kind: 'arb',
        locale: language.route_code,
        basePath: 'app/lib/l10n/app_en.arb',
        targetPath: `app/lib/l10n/app_${arbSuffix}.arb`,
        sidecarPath: `app/lib/l10n/app_${arbSuffix}_intentions.json`,
      },
      {
        kind: 'json',
        locale: language.route_code,
        basePath: 'app/assets/i18n/settings/en.json',
        targetPath: `app/assets/i18n/settings/${language.route_code}.json`,
        sidecarPath: `app/assets/i18n/settings/${language.route_code}_intentions.json`,
      },
    ];
  });
}

/**
 * @param {LanguageEntry[]} languages
 */
function webDescriptors(languages) {
  return languages.flatMap((language) =>
    WEB_COPY_NAMESPACES.map((namespace) => ({
      kind: 'json',
      locale: language.route_code,
      basePath: `web/content/i18n/${namespace}/en.json`,
      targetPath: `web/content/i18n/${namespace}/${language.route_code}.json`,
      sidecarPath: `web/content/i18n/${namespace}/${language.route_code}_intentions.json`,
    })),
  );
}

/**
 * @param {LanguageEntry[]} languages
 */
function checkoutDescriptors(languages) {
  return languages.map((language) => ({
    kind: 'json',
    locale: language.route_code,
    basePath: 'api/content/i18n/checkout/en.json',
    targetPath: `api/content/i18n/checkout/${language.route_code}.json`,
    sidecarPath: `api/content/i18n/checkout/${language.route_code}_intentions.json`,
  }));
}

/**
 * @param {{ rootDir: string, reader: ReturnType<typeof createJsonReader>, languages: LanguageEntry[], fileFilter: string | null, descriptors: ReturnType<typeof appDescriptors> }} options
 * @returns {Promise<I18nTodoItem[]>}
 */
async function buildPerLocaleFileTodos({ rootDir, reader, languages, fileFilter, descriptors }) {
  const languageSet = new Set(languages.map((language) => language.route_code));
  const items = [];

  for (const descriptor of descriptors) {
    if (!languageSet.has(descriptor.locale)) {
      continue;
    }
    if (!matchesFileFilter(fileFilter, descriptor.basePath, descriptor.targetPath, descriptor.sidecarPath)) {
      continue;
    }

    const baseJson = await reader.readRequired(descriptor.basePath);
    const baseValues = descriptor.kind === 'arb' ? flattenArb(baseJson) : flattenJson(baseJson);
    const targetExists = await fileExists(join(rootDir, descriptor.targetPath));
    const targetValues = targetExists
      ? descriptor.kind === 'arb'
        ? flattenArb(await reader.readRequired(descriptor.targetPath))
        : flattenJson(await reader.readRequired(descriptor.targetPath))
      : new Map();
    const fallbackValues = await readFallbackValues({
      reader,
      descriptor,
      locale: descriptor.locale,
      kind: descriptor.kind,
    });

    for (const [key, baseValue] of baseValues) {
      if (targetValues.has(key)) {
        continue;
      }
      items.push({
        type: targetExists ? 'missing-key' : 'missing-file',
        file: descriptor.targetPath,
        locale: descriptor.locale,
        key,
        base_english_value: baseValue,
        fallback_value: fallbackValues.get(key) ?? baseValue,
        sidecar_path: descriptor.sidecarPath,
      });
    }
  }

  return items;
}

/**
 * @param {{ reader: ReturnType<typeof createJsonReader>, languages: LanguageEntry[], fileFilter: string | null, registryLanguages: LanguageEntry[] }} options
 * @returns {Promise<I18nTodoItem[]>}
 */
async function buildCatalogTodos({ reader, languages, fileFilter, registryLanguages }) {
  const items = [];
  const byRouteCode = new Map(registryLanguages.map((language) => [language.route_code, language]));

  for (const filePath of API_CATALOG_FILES) {
    if (!matchesFileFilter(fileFilter, filePath, catalogSidecarPath(filePath))) {
      continue;
    }
    const catalog = await reader.readRequired(filePath);
    const entries = flattenCatalogEnglishEntries(catalog);

    for (const language of languages) {
      for (const entry of entries) {
        if (catalogValueAt(catalog, `${entry.key}.${language.route_code}`) !== undefined) {
          continue;
        }
        const fallbackCode = byRouteCode.get(language.route_code)?.fallback ?? 'en';
        const fallbackValue =
          (fallbackCode ? catalogValueAt(catalog, `${entry.key}.${fallbackCode}`) : undefined) ??
          entry.value;
        items.push({
          type: 'missing-key',
          file: filePath,
          locale: language.route_code,
          key: entry.key,
          base_english_value: entry.value,
          fallback_value: valueToString(fallbackValue),
          sidecar_path: catalogSidecarPath(filePath),
        });
      }
    }
  }

  return items;
}

/**
 * @param {{ reader: ReturnType<typeof createJsonReader>, descriptor: { basePath: string, targetPath: string, locale: string }, locale: string, kind: 'arb' | 'json' }} options
 * @returns {Promise<Map<string, string>>}
 */
async function readFallbackValues({ reader, descriptor, locale, kind }) {
  if (locale === 'ja') {
    return kind === 'arb'
      ? flattenArb(await reader.readRequired(descriptor.basePath))
      : flattenJson(await reader.readRequired(descriptor.basePath));
  }

  return kind === 'arb'
    ? flattenArb(await reader.readRequired(descriptor.basePath))
    : flattenJson(await reader.readRequired(descriptor.basePath));
}

function createJsonReader(rootDir) {
  const cache = new Map();
  return {
    async readRequired(relativePath) {
      if (cache.has(relativePath)) {
        return cache.get(relativePath);
      }
      let rawText;
      try {
        rawText = await readFile(join(rootDir, relativePath), 'utf8');
      } catch (error) {
        throw new Error(`Unable to read ${relativePath}: ${error.message}`);
      }
      let parsed;
      try {
        parsed = JSON.parse(rawText);
      } catch (error) {
        throw new Error(`Invalid JSON in ${relativePath}: ${error.message}`);
      }
      cache.set(relativePath, parsed);
      return parsed;
    },
  };
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
    results.set(key, valueToString(entryValue));
  }
  return results;
}

function flattenJson(value, prefix = '') {
  const results = new Map();
  if (Array.isArray(value)) {
    value.forEach((entry, index) => {
      for (const [key, nestedValue] of flattenJson(entry, `${prefix}[${index}]`)) {
        results.set(key, nestedValue);
      }
    });
    return results;
  }
  if (isRecord(value)) {
    for (const [key, entryValue] of Object.entries(value)) {
      const nextPrefix = prefix ? `${prefix}.${key}` : key;
      for (const [nestedKey, nestedValue] of flattenJson(entryValue, nextPrefix)) {
        results.set(nestedKey, nestedValue);
      }
    }
    return results;
  }
  if (prefix) {
    results.set(prefix, valueToString(value));
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
  if (Object.hasOwn(value, 'en') && !isRecord(value.en) && !Array.isArray(value.en)) {
    results.push({ key: prefix, value: valueToString(value.en) });
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

function flutterArbSuffix(language) {
  const { languageCode, scriptCode, countryCode } = language.flutter;
  if (language.route_code === 'zh') {
    return 'zh';
  }

  return [languageCode, scriptCode, countryCode].filter(Boolean).join('_');
}

function normalizeFilterPath(file) {
  if (!file || !file.trim()) {
    return null;
  }
  return file.trim().replace(/^\.\//, '');
}

function matchesFileFilter(fileFilter, ...paths) {
  if (!fileFilter) {
    return true;
  }
  return paths.some((path) => normalizeFilterPath(path) === fileFilter);
}

function catalogSidecarPath(filePath) {
  return filePath.replace(/\.json$/, '_intentions.json');
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

function valueToString(value) {
  if (typeof value === 'string') {
    return value;
  }
  if (value === null) {
    return 'null';
  }
  return JSON.stringify(value);
}

function md(value) {
  return truncate(String(value)).replaceAll('\\', '\\\\').replaceAll('|', '\\|').replaceAll('\n', '\\n');
}

function truncate(value, limit = 120) {
  return value.length > limit ? `${value.slice(0, limit - 1)}...` : value;
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await buildI18nTodo({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    process.stdout.write(renderI18nTodo(report));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
