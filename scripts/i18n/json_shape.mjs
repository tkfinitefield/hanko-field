import { readdir, readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';
import { parseLangsFilter } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
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
 * @typedef {Object} JsonShapeIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} JsonShapeReport
 * @property {boolean} ok
 * @property {JsonShapeIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 * @returns {Promise<JsonShapeReport>}
 */
export async function validateJsonShapes({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const fileFilter = normalizeFilterPath(file);
  if (fileFilter && !fileFilter.endsWith('.json')) {
    return { ok: true, issues: [], parsed_files: [] };
  }

  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  const registry = await loadLanguageRegistry(new URL('config/languages.json', rootUrl));
  const selectedLanguages = selectShapeLanguages(registry.languages, langs);
  const reader = createJsonReader(rootDir);
  const issues = [
    ...validateFallbackChains(registry.languages),
  ];
  const parsed = new Set();

  const descriptors = [
    ...appSettingsDescriptors(selectedLanguages, langs),
    ...(await webContentDescriptors(rootDir, selectedLanguages, langs)),
    ...apiCheckoutDescriptors(selectedLanguages, langs),
    ...(await releaseMetadataDescriptors(rootDir, selectedLanguages, langs, fileFilter)),
  ];

  for (const descriptor of descriptors) {
    if (!matchesFileFilter(fileFilter, descriptor.basePath, descriptor.targetPath)) {
      continue;
    }
    const base = await reader.readOptional(descriptor.basePath);
    if (!base.ok) {
      if (base.code !== 'missing-json') {
        issues.push(createIssue(base.code, descriptor.basePath, null, base.message));
      } else if (fileFilter === descriptor.targetPath || fileFilter === descriptor.basePath) {
        issues.push(createIssue(base.code, descriptor.basePath, null, base.message));
      }
      continue;
    }
    parsed.add(descriptor.basePath);

    const target = await reader.readOptional(descriptor.targetPath);
    if (!target.ok) {
      if (target.code !== 'missing-json') {
        issues.push(createIssue(target.code, descriptor.targetPath, null, target.message));
      }
      continue;
    }
    parsed.add(descriptor.targetPath);
    issues.push(...compareJsonShape(base.value, target.value, descriptor.targetPath, descriptor.basePath));
  }

  for (const catalogPath of API_CATALOG_FILES) {
    if (!matchesFileFilter(fileFilter, catalogPath)) {
      continue;
    }
    const catalog = await reader.readOptional(catalogPath);
    if (!catalog.ok) {
      if (catalog.code !== 'missing-json') {
        issues.push(createIssue(catalog.code, catalogPath, null, catalog.message));
      }
      continue;
    }
    parsed.add(catalogPath);
    issues.push(...validateCatalogLanguageMaps(catalog.value, catalogPath, selectedLanguages));
  }

  return {
    ok: issues.length === 0,
    issues,
    parsed_files: [...parsed].sort(),
  };
}

/**
 * @param {unknown} base
 * @param {unknown} target
 * @param {string} targetPath
 * @param {string} basePath
 * @param {string} key
 * @returns {JsonShapeIssue[]}
 */
export function compareJsonShape(base, target, targetPath, basePath, key = '') {
  const issues = [];
  const baseType = shapeType(base);
  const targetType = shapeType(target);
  if (baseType !== targetType) {
    return [
      createIssue(
        'json-shape-type',
        targetPath,
        key || null,
        `expected ${baseType} to match ${basePath}, found ${targetType}`,
      ),
    ];
  }

  if (Array.isArray(base)) {
    if (base.length !== target.length) {
      issues.push(
        createIssue(
          'json-shape-array-length',
          targetPath,
          key || null,
          `expected ${base.length} items to match ${basePath}, found ${target.length}`,
        ),
      );
    }
    const sharedLength = Math.min(base.length, target.length);
    for (let index = 0; index < sharedLength; index += 1) {
      issues.push(...compareJsonShape(base[index], target[index], targetPath, basePath, `${key}[${index}]`));
    }
    return issues;
  }

  if (isRecord(base)) {
    for (const baseKey of Object.keys(base)) {
      const childKey = key ? `${key}.${baseKey}` : baseKey;
      if (!Object.hasOwn(target, baseKey)) {
        issues.push(createIssue('json-shape-missing-key', targetPath, childKey, `missing key from ${basePath}`));
        continue;
      }
      issues.push(...compareJsonShape(base[baseKey], target[baseKey], targetPath, basePath, childKey));
    }
    for (const targetKey of Object.keys(target)) {
      if (!Object.hasOwn(base, targetKey)) {
        const childKey = key ? `${key}.${targetKey}` : targetKey;
        issues.push(createIssue('json-shape-extra-key', targetPath, childKey, `extra key not present in ${basePath}`));
      }
    }
  }

  return issues;
}

/**
 * @param {LanguageEntry[]} languages
 * @returns {JsonShapeIssue[]}
 */
export function validateFallbackChains(languages) {
  const issues = [];
  const byRouteCode = new Map(languages.map((language) => [language.route_code, language]));

  for (const language of languages) {
    if (!language.fallback) {
      continue;
    }
    const fallback = byRouteCode.get(language.fallback);
    if (!fallback) {
      issues.push(
        createIssue('fallback-missing', 'config/languages.json', language.route_code, `fallback ${language.fallback} is not in the registry`),
      );
      continue;
    }
    for (const surface of ['app', 'web', 'release']) {
      if (language[surface].enabled && !fallback[surface].enabled) {
        issues.push(
          createIssue(
            'fallback-disabled',
            'config/languages.json',
            language.route_code,
            `${surface} fallback ${language.fallback} is disabled`,
          ),
        );
      }
    }
  }

  for (const language of languages) {
    const seen = new Set();
    let current = language;
    while (current?.fallback) {
      if (seen.has(current.route_code)) {
        issues.push(
          createIssue('fallback-cycle', 'config/languages.json', language.route_code, `fallback chain cycles at ${current.route_code}`),
        );
        break;
      }
      seen.add(current.route_code);
      current = byRouteCode.get(current.fallback);
    }
  }

  return issues;
}

/**
 * @param {unknown} value
 * @param {string} file
 * @param {LanguageEntry[]} languages
 * @returns {JsonShapeIssue[]}
 */
export function validateCatalogLanguageMaps(value, file, languages) {
  const issues = [];
  visitCatalog(value);
  return issues;

  function visitCatalog(current, key = '') {
    if (Array.isArray(current)) {
      current.forEach((entry, index) => visitCatalog(entry, `${key}[${index}]`));
      return;
    }
    if (!isRecord(current)) {
      return;
    }

    if (Object.hasOwn(current, 'en') && isScalar(current.en)) {
      const baseType = shapeType(current.en);
      for (const language of languages) {
        if (!Object.hasOwn(current, language.route_code)) {
          continue;
        }
        const targetType = shapeType(current[language.route_code]);
        if (targetType !== baseType) {
          issues.push(
            createIssue(
              'json-shape-catalog-type',
              file,
              `${key}.${language.route_code}`,
              `expected ${baseType} to match ${key}.en, found ${targetType}`,
            ),
          );
        }
      }
      return;
    }

    for (const [childKey, childValue] of Object.entries(current)) {
      visitCatalog(childValue, key ? `${key}.${childKey}` : childKey);
    }
  }
}

/**
 * @param {LanguageEntry[]} languages
 * @param {string[] | null} langs
 * @returns {LanguageEntry[]}
 */
function selectShapeLanguages(languages, langs) {
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

function appSettingsDescriptors(languages, langs) {
  return languages
    .filter((language) => language.app.enabled || langs)
    .map((language) => ({
      basePath: APP_SETTINGS_BASE,
      targetPath: `app/assets/i18n/settings/${language.route_code}.json`,
    }));
}

async function webContentDescriptors(rootDir, languages, langs) {
  const namespaces = await discoverWebNamespaces(rootDir);
  return languages
    .filter((language) => language.web.enabled || langs)
    .flatMap((language) =>
      namespaces.map((namespace) => ({
        basePath: `web/content/i18n/${namespace}/en.json`,
        targetPath: `web/content/i18n/${namespace}/${language.route_code}.json`,
      })),
    );
}

function apiCheckoutDescriptors(languages, langs) {
  return languages
    .filter((language) => language.web.enabled || langs)
    .map((language) => ({
      basePath: API_CHECKOUT_BASE,
      targetPath: `api/content/i18n/checkout/${language.route_code}.json`,
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
      basePath: RELEASE_METADATA_BASE,
      targetPath: `release/store_metadata/source/${language.route_code}.json`,
    }));
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

function normalizeFilterPath(file) {
  if (!file || !file.trim()) {
    return null;
  }
  return file.trim().replace(/^\.\//, '');
}

function shapeType(value) {
  if (Array.isArray(value)) {
    return 'array';
  }
  if (value === null) {
    return 'null';
  }
  return typeof value;
}

function isScalar(value) {
  return !Array.isArray(value) && !isRecord(value);
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await validateJsonShapes({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    if (report.issues.length === 0) {
      process.stdout.write(`JSON shape validation passed (${report.parsed_files.length} files).\n`);
    } else {
      process.stdout.write(`JSON shape validation failed (${report.issues.length} issues).\n`);
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
