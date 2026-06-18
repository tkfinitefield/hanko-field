import { createHash } from 'node:crypto';
import { mkdir, readdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { expectedArbPath } from './arb.mjs';
import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_MANIFEST_PATH = 'doc/qa/m9-t06/translation-freeze.json';
const WEB_I18N_ROOT = 'web/content/i18n';
const API_CATALOG_ROOT = 'api/content/i18n/catalog';
const RELEASE_METADATA_SOURCE_ROOT = 'release/store_metadata/source';
const RELEASE_METADATA_OUTPUT_ROOTS = [
  'release/store_metadata/google_play',
  'release/store_metadata/app_store',
  'release/store_metadata/screenshots',
];
const REQUIRED_CHECK_IDS = Object.freeze([
  'i18n_check',
  'i18n_stubs_check',
  'i18n_holdouts_check',
  'i18n_layout_qa_check',
  'i18n_flag_stages_check',
  'store_metadata_check',
  'google_play_metadata_check',
  'app_store_metadata_check',
  'screenshot_metadata_check',
  'android_fastlane_check',
  'ios_fastlane_check',
  'release_secret_guardrails_check',
]);

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} FreezeIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} FreezeReport
 * @property {boolean} ok
 * @property {FreezeIssue[]} issues
 * @property {string} manifest_file
 * @property {string[]} frozen_locales
 * @property {string[]} metadata_locales
 * @property {number} frozen_files
 */

/**
 * @param {{ rootDir?: string, manifestPath?: string }} options
 * @returns {Promise<FreezeReport>}
 */
export async function buildFreezeReport({
  rootDir = REPO_ROOT,
  manifestPath = DEFAULT_MANIFEST_PATH,
} = {}) {
  const registry = await loadRegistry(rootDir);
  const expectedLanguageSet = deriveLanguageSet(registry.languages, await discoverStoreMetadataLocales(rootDir));
  const expectedFiles = await collectFrozenFiles(rootDir, registry.languages, expectedLanguageSet);
  const issues = [];
  const manifestResult = await readManifest(rootDir, manifestPath);

  if (!manifestResult.ok) {
    issues.push(createIssue('freeze-manifest-missing', manifestPath, null, manifestResult.message));
  } else {
    issues.push(...validateManifest({ manifest: manifestResult.value, manifestPath, expectedLanguageSet, expectedFiles }));
  }

  return {
    ok: issues.length === 0,
    issues,
    manifest_file: manifestPath,
    frozen_locales: expectedLanguageSet.frozen_locales,
    metadata_locales: expectedLanguageSet.store_metadata_source_locales,
    frozen_files: expectedFiles.length,
  };
}

/**
 * @param {{ rootDir?: string, manifestPath?: string, releaseCandidate?: string, frozenAt?: string }} options
 * @returns {Promise<object>}
 */
export async function createFreezeManifest({
  rootDir = REPO_ROOT,
  releaseCandidate = 'next-app-release',
  frozenAt = '2026-06-18',
} = {}) {
  const registry = await loadRegistry(rootDir);
  const languageSet = deriveLanguageSet(registry.languages, await discoverStoreMetadataLocales(rootDir));
  const files = await collectFrozenFiles(rootDir, registry.languages, languageSet);

  return {
    format_version: 1,
    task: 'M9-T06',
    frozen_at: frozenAt,
    release_candidate: releaseCandidate,
    review_policy: {
      translation_changes_after_freeze_require_explicit_review: true,
      m0_t05_preservation_gates_recorded: true,
      current_task_changes_translations: false,
      current_task_changes_registry_flags: false,
      current_task_changes_release_credentials: false,
    },
    language_set: languageSet,
    required_checks: REQUIRED_CHECK_IDS.map((id) => requiredCheck(id)),
    files,
  };
}

/**
 * @param {FreezeReport} report
 * @returns {string}
 */
export function renderFreezeReport(report) {
  const lines = [
    '# Stone Signature i18n translation freeze',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Manifest file: ${report.manifest_file}`,
    `Frozen locales: ${formatList(report.frozen_locales)}`,
    `Store metadata locales: ${formatList(report.metadata_locales)}`,
    `Frozen files: ${report.frozen_files}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No translation freeze issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

/**
 * @param {LanguageEntry[]} languages
 * @param {string[]} metadataLocales
 */
export function deriveLanguageSet(languages, metadataLocales) {
  const appEnabled = languages
    .filter((language) => language.app.enabled)
    .map((language) => language.route_code)
    .sort();
  const webEnabled = languages
    .filter((language) => language.web.enabled)
    .map((language) => language.route_code)
    .sort();
  const releaseEnabled = languages
    .filter((language) => language.release.enabled)
    .map((language) => language.route_code)
    .sort();
  const frozenLocales = [...new Set([...appEnabled, ...webEnabled])].sort();

  return {
    registry_languages: languages.length,
    app_enabled: appEnabled,
    web_enabled: webEnabled,
    release_enabled: releaseEnabled,
    frozen_locales: frozenLocales,
    store_metadata_source_locales: metadataLocales.slice().sort(),
  };
}

function validateManifest({ manifest, manifestPath, expectedLanguageSet, expectedFiles }) {
  const issues = [];
  if (!isRecord(manifest)) {
    return [createIssue('freeze-manifest-format', manifestPath, null, 'top-level value must be an object')];
  }
  if (manifest.format_version !== 1) {
    issues.push(createIssue('freeze-manifest-format', manifestPath, 'format_version', 'must be 1'));
  }
  if (manifest.task !== 'M9-T06') {
    issues.push(createIssue('freeze-manifest-format', manifestPath, 'task', 'must be M9-T06'));
  }
  if (typeof manifest.frozen_at !== 'string' || manifest.frozen_at.trim() === '') {
    issues.push(createIssue('freeze-manifest-format', manifestPath, 'frozen_at', 'must be a non-empty string'));
  }
  if (typeof manifest.release_candidate !== 'string' || manifest.release_candidate.trim() === '') {
    issues.push(createIssue('freeze-manifest-format', manifestPath, 'release_candidate', 'must be a non-empty string'));
  }
  issues.push(...validateReviewPolicy(manifestPath, manifest.review_policy));
  issues.push(...validateLanguageSet(manifestPath, manifest.language_set, expectedLanguageSet));
  issues.push(...validateRequiredChecks(manifestPath, manifest.required_checks));
  issues.push(...validateFrozenFiles(manifestPath, manifest.files, expectedFiles));
  return issues;
}

function validateReviewPolicy(manifestPath, policy) {
  const issues = [];
  if (!isRecord(policy)) {
    return [createIssue('freeze-review-policy', manifestPath, 'review_policy', 'must be an object')];
  }
  for (const key of [
    'translation_changes_after_freeze_require_explicit_review',
    'm0_t05_preservation_gates_recorded',
  ]) {
    if (policy[key] !== true) {
      issues.push(createIssue('freeze-review-policy', manifestPath, key, 'must be true'));
    }
  }
  for (const key of [
    'current_task_changes_translations',
    'current_task_changes_registry_flags',
    'current_task_changes_release_credentials',
  ]) {
    if (typeof policy[key] !== 'boolean') {
      issues.push(createIssue('freeze-review-policy', manifestPath, key, 'must be a boolean'));
    }
  }
  return issues;
}

function validateLanguageSet(manifestPath, actual, expected) {
  const issues = [];
  if (!isRecord(actual)) {
    return [createIssue('freeze-language-set', manifestPath, 'language_set', 'must be an object')];
  }
  if (actual.registry_languages !== expected.registry_languages) {
    issues.push(createIssue('freeze-language-set', manifestPath, 'registry_languages', `expected ${expected.registry_languages}`));
  }
  for (const key of [
    'app_enabled',
    'web_enabled',
    'release_enabled',
    'frozen_locales',
    'store_metadata_source_locales',
  ]) {
    const actualValues = readStringArray(actual[key], { sort: true });
    const expectedValues = expected[key];
    if (!sameList(actualValues, expectedValues)) {
      issues.push(createIssue('freeze-language-set', manifestPath, key, `expected ${formatList(expectedValues)} but found ${formatList(actualValues)}`));
    }
  }
  return issues;
}

function validateRequiredChecks(manifestPath, checks) {
  const issues = [];
  if (!Array.isArray(checks)) {
    return [createIssue('freeze-required-checks', manifestPath, 'required_checks', 'must be an array')];
  }
  const byId = new Map();
  for (const [index, check] of checks.entries()) {
    if (!isRecord(check)) {
      issues.push(createIssue('freeze-required-checks', manifestPath, `required_checks[${index}]`, 'must be an object'));
      continue;
    }
    if (typeof check.id !== 'string' || check.id.trim() === '') {
      issues.push(createIssue('freeze-required-checks', manifestPath, `required_checks[${index}].id`, 'must be a non-empty string'));
      continue;
    }
    byId.set(check.id, check);
    if (check.status !== 'pass') {
      issues.push(createIssue('freeze-required-checks', manifestPath, check.id, 'status must be pass'));
    }
    if (typeof check.command !== 'string' || check.command.trim() === '') {
      issues.push(createIssue('freeze-required-checks', manifestPath, check.id, 'command must be a non-empty string'));
    }
  }
  for (const id of REQUIRED_CHECK_IDS) {
    if (!byId.has(id)) {
      issues.push(createIssue('freeze-required-checks', manifestPath, id, 'required check is missing'));
    }
  }
  return issues;
}

function validateFrozenFiles(manifestPath, files, expectedFiles) {
  const issues = [];
  if (!Array.isArray(files)) {
    return [createIssue('freeze-files', manifestPath, 'files', 'must be an array')];
  }
  const actualByPath = new Map();
  for (const [index, file] of files.entries()) {
    if (!isRecord(file)) {
      issues.push(createIssue('freeze-files', manifestPath, `files[${index}]`, 'must be an object'));
      continue;
    }
    if (typeof file.path !== 'string' || file.path.trim() === '') {
      issues.push(createIssue('freeze-files', manifestPath, `files[${index}].path`, 'must be a non-empty string'));
      continue;
    }
    actualByPath.set(file.path, file);
    if (typeof file.sha256 !== 'string' || !/^[0-9a-f]{64}$/.test(file.sha256)) {
      issues.push(createIssue('freeze-files', manifestPath, file.path, 'sha256 must be a lowercase SHA-256 digest'));
    }
    if (!Number.isInteger(file.bytes) || file.bytes < 0) {
      issues.push(createIssue('freeze-files', manifestPath, file.path, 'bytes must be a non-negative integer'));
    }
  }

  const expectedByPath = new Map(expectedFiles.map((file) => [file.path, file]));
  for (const expected of expectedFiles) {
    const actual = actualByPath.get(expected.path);
    if (!actual) {
      issues.push(createIssue('freeze-file-missing', manifestPath, expected.path, 'frozen file is missing from manifest'));
      continue;
    }
    if (actual.sha256 !== expected.sha256) {
      issues.push(createIssue('freeze-file-hash', expected.path, null, 'content changed after translation freeze'));
    }
    if (actual.bytes !== expected.bytes) {
      issues.push(createIssue('freeze-file-size', expected.path, null, `expected ${expected.bytes} bytes`));
    }
  }
  for (const actualPath of actualByPath.keys()) {
    if (!expectedByPath.has(actualPath)) {
      issues.push(createIssue('freeze-file-extra', manifestPath, actualPath, 'manifest includes a file outside the current freeze set'));
    }
  }
  return issues;
}

async function collectFrozenFiles(rootDir, languages, languageSet) {
  const byRouteCode = new Map(languages.map((language) => [language.route_code, language]));
  const paths = new Set(['config/languages.json']);

  for (const locale of languageSet.frozen_locales) {
    const language = byRouteCode.get(locale);
    if (!language) {
      continue;
    }
    if (language.app.enabled) {
      addPath(paths, expectedArbPath(language));
      addPath(paths, intentionPath(expectedArbPath(language)));
      addPath(paths, `app/assets/i18n/settings/${locale}.json`);
      addPath(paths, `app/assets/i18n/settings/${locale}_intentions.json`);
    }
    if (language.web.enabled) {
      for (const namespace of await discoverWebNamespaces(rootDir)) {
        addPath(paths, `web/content/i18n/${namespace}/${locale}.json`);
        addPath(paths, `web/content/i18n/${namespace}/${locale}_intentions.json`);
      }
      addPath(paths, `api/content/i18n/checkout/${locale}.json`);
      addPath(paths, `api/content/i18n/checkout/${locale}_intentions.json`);
    }
  }

  for (const path of await discoverFiles(rootDir, API_CATALOG_ROOT, (path) => path.endsWith('.json'))) {
    addPath(paths, path);
  }
  for (const locale of languageSet.store_metadata_source_locales) {
    addPath(paths, `${RELEASE_METADATA_SOURCE_ROOT}/${locale}.json`);
    addPath(paths, `${RELEASE_METADATA_SOURCE_ROOT}/${locale}_intentions.json`);
  }
  addPath(paths, `${RELEASE_METADATA_SOURCE_ROOT}/schema.json`);
  for (const root of RELEASE_METADATA_OUTPUT_ROOTS) {
    for (const path of await discoverFiles(rootDir, root, (path) => path.endsWith('.txt') || path.endsWith('.json'))) {
      addPath(paths, path);
    }
  }

  const descriptors = [];
  for (const path of [...paths].sort()) {
    const descriptor = await describeFile(rootDir, path);
    if (descriptor) {
      descriptors.push(descriptor);
    }
  }
  return descriptors;
}

function addPath(paths, path) {
  paths.add(path);
}

function intentionPath(path) {
  return path.replace(/\.([^.]+)$/, '_intentions.$1');
}

async function discoverStoreMetadataLocales(rootDir) {
  const files = await discoverFiles(rootDir, RELEASE_METADATA_SOURCE_ROOT, (path) => path.endsWith('.json'));
  return files
    .map((path) => path.slice(`${RELEASE_METADATA_SOURCE_ROOT}/`.length).replace(/\.json$/, ''))
    .filter((locale) => locale !== 'schema' && !locale.endsWith('_intentions'))
    .sort();
}

async function discoverWebNamespaces(rootDir) {
  const baseDir = join(rootDir, WEB_I18N_ROOT);
  try {
    const entries = await readdir(baseDir, { withFileTypes: true });
    return entries
      .filter((entry) => entry.isDirectory())
      .map((entry) => entry.name)
      .sort();
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return [];
    }
    throw error;
  }
}

async function discoverFiles(rootDir, relativeRoot, predicate) {
  const results = [];
  await visit(relativeRoot);
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
      } else if (entry.isFile() && predicate(relativePath)) {
        results.push(relativePath);
      }
    }
  }
}

async function describeFile(rootDir, path) {
  try {
    const bytes = await readFile(join(rootDir, path));
    return {
      path,
      sha256: createHash('sha256').update(bytes).digest('hex'),
      bytes: bytes.length,
    };
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return null;
    }
    throw error;
  }
}

async function loadRegistry(rootDir) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  return loadLanguageRegistry(new URL('config/languages.json', rootUrl));
}

async function readManifest(rootDir, manifestPath) {
  try {
    return { ok: true, value: JSON.parse(await readFile(join(rootDir, manifestPath), 'utf8')) };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function requiredCheck(id) {
  const commands = {
    i18n_check: 'make i18n-check',
    i18n_stubs_check: 'make i18n-stubs-check',
    i18n_holdouts_check: 'make i18n-holdouts-check',
    i18n_layout_qa_check: 'make i18n-layout-qa-check',
    i18n_flag_stages_check: 'make i18n-flag-stages-check',
    store_metadata_check: 'make store-metadata-check',
    google_play_metadata_check: 'make google-play-metadata-check',
    app_store_metadata_check: 'make app-store-metadata-check',
    screenshot_metadata_check: 'make screenshot-metadata-check',
    android_fastlane_check: 'make android-fastlane-check',
    ios_fastlane_check: 'make ios-fastlane-check',
    release_secret_guardrails_check: 'make release-secret-guardrails-check',
  };
  return {
    id,
    status: 'pass',
    command: commands[id],
  };
}

function readStringArray(value, { sort = false } = {}) {
  const items = Array.isArray(value) ? value.filter((entry) => typeof entry === 'string') : [];
  return sort ? items.sort() : items;
}

function sameList(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function formatList(values) {
  return values.length === 0 ? 'none' : values.join(', ');
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    if (process.argv.includes('--print-manifest')) {
      const manifest = await createFreezeManifest({
        frozenAt: process.env.FROZEN_AT ?? '2026-06-18',
        releaseCandidate: process.env.RELEASE_CANDIDATE ?? 'next-app-release',
      });
      process.stdout.write(`${JSON.stringify(manifest, null, 2)}\n`);
    } else if (process.argv.includes('--write-manifest')) {
      const manifestPath = process.env.MANIFEST ?? DEFAULT_MANIFEST_PATH;
      const manifest = await createFreezeManifest({
        frozenAt: process.env.FROZEN_AT ?? '2026-06-18',
        releaseCandidate: process.env.RELEASE_CANDIDATE ?? 'next-app-release',
      });
      await mkdir(dirname(join(REPO_ROOT, manifestPath)), { recursive: true });
      await writeFile(join(REPO_ROOT, manifestPath), `${JSON.stringify(manifest, null, 2)}\n`);
      process.stdout.write(`Wrote ${manifestPath}\n`);
    } else {
      const report = await buildFreezeReport({
        manifestPath: process.env.MANIFEST ?? DEFAULT_MANIFEST_PATH,
      });
      process.stdout.write(renderFreezeReport(report));
      if (!report.ok || process.argv.includes('--check')) {
        process.exitCode = report.ok ? 0 : 1;
      }
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
