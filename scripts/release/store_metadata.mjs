import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const SOURCE_DIR = 'release/store_metadata/source';
const SCHEMA_PATH = `${SOURCE_DIR}/schema.json`;
const REQUIRED_EXAMPLE_LOCALES = ['en', 'ja', 'zh', 'zhtw'];
const REQUIRED_FIELDS = [
  'app_name',
  'subtitle',
  'short_description',
  'full_description',
  'keywords',
  'release_notes',
  'support_url',
  'marketing_url',
  'privacy_policy_url',
  'screenshot_captions',
];
const SCREENSHOT_KEYS = ['design', 'stones', 'checkout'];
const ALLOWED_TOP_LEVEL_KEYS = new Set(REQUIRED_FIELDS);

/**
 * @typedef {Object} StoreMetadataIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} StoreMetadataReport
 * @property {boolean} ok
 * @property {StoreMetadataIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string, requiredLocales?: string[] }} options
 * @returns {Promise<StoreMetadataReport>}
 */
export async function validateStoreMetadataSources({
  rootDir = REPO_ROOT,
  requiredLocales = REQUIRED_EXAMPLE_LOCALES,
} = {}) {
  const issues = [];
  const parsedFiles = [];

  const schemaResult = await readJson(rootDir, SCHEMA_PATH);
  if (!schemaResult.ok) {
    issues.push({
      code: schemaResult.code,
      file: SCHEMA_PATH,
      key: null,
      message: schemaResult.message,
    });
  } else {
    parsedFiles.push(SCHEMA_PATH);
    validateSchemaDocument(schemaResult.value, SCHEMA_PATH, issues);
  }

  const sourceFiles = await discoverSourceFiles(rootDir);
  const sourceLocales = sourceFiles.map((file) => file.replace(/\.json$/, ''));
  for (const locale of requiredLocales) {
    if (!sourceLocales.includes(locale)) {
      issues.push({
        code: 'store-metadata-missing-locale',
        file: `${SOURCE_DIR}/${locale}.json`,
        key: null,
        message: `required M8-T01 example locale is missing: ${locale}`,
      });
    }
  }

  for (const fileName of sourceFiles) {
    const relativePath = `${SOURCE_DIR}/${fileName}`;
    const result = await readJson(rootDir, relativePath);
    if (!result.ok) {
      issues.push({
        code: result.code,
        file: relativePath,
        key: null,
        message: result.message,
      });
      continue;
    }
    parsedFiles.push(relativePath);
    validateStoreMetadataDocument(result.value, relativePath, issues);
  }

  return {
    ok: issues.length === 0,
    issues,
    parsed_files: parsedFiles.sort(),
  };
}

/**
 * @param {unknown} value
 * @param {string} file
 * @param {StoreMetadataIssue[]} issues
 */
export function validateStoreMetadataDocument(value, file, issues) {
  if (!isRecord(value)) {
    addIssue(issues, 'store-metadata-type', file, null, 'top-level value must be an object');
    return;
  }

  for (const key of REQUIRED_FIELDS) {
    if (!Object.hasOwn(value, key)) {
      addIssue(issues, 'store-metadata-required', file, key, 'required field is missing');
    }
  }
  for (const key of Object.keys(value)) {
    if (!ALLOWED_TOP_LEVEL_KEYS.has(key)) {
      addIssue(issues, 'store-metadata-unknown-key', file, key, 'field is not part of the source schema');
    }
  }

  validateString(value.app_name, file, 'app_name', issues, { maxLength: 30 });
  validateString(value.subtitle, file, 'subtitle', issues, { maxLength: 30 });
  validateString(value.short_description, file, 'short_description', issues, { maxLength: 80 });
  validateStringArray(value.full_description, file, 'full_description', issues, { minItems: 2 });
  validateStringArray(value.keywords, file, 'keywords', issues, { minItems: 3, unique: true });
  validateReleaseNotes(value.release_notes, file, issues);
  validateHttpsUrl(value.support_url, file, 'support_url', issues);
  validateHttpsUrl(value.marketing_url, file, 'marketing_url', issues);
  validateHttpsUrl(value.privacy_policy_url, file, 'privacy_policy_url', issues);
  validateScreenshotCaptions(value.screenshot_captions, file, issues);
}

/**
 * @param {unknown} value
 * @param {string} file
 * @param {StoreMetadataIssue[]} issues
 */
function validateSchemaDocument(value, file, issues) {
  if (!isRecord(value)) {
    addIssue(issues, 'store-metadata-schema-type', file, null, 'schema must be an object');
    return;
  }
  if (value.title !== 'STONE SIGNATURE store metadata source') {
    addIssue(issues, 'store-metadata-schema-title', file, 'title', 'schema title is unexpected');
  }
  if (!Array.isArray(value.required)) {
    addIssue(issues, 'store-metadata-schema-required', file, 'required', 'schema required list is missing');
    return;
  }
  for (const key of REQUIRED_FIELDS) {
    if (!value.required.includes(key)) {
      addIssue(issues, 'store-metadata-schema-required', file, key, 'schema does not require this source field');
    }
  }
}

function validateString(value, file, key, issues, { maxLength = null } = {}) {
  if (typeof value !== 'string') {
    addIssue(issues, 'store-metadata-type', file, key, 'must be a string');
    return;
  }
  if (value.trim() === '') {
    addIssue(issues, 'store-metadata-empty', file, key, 'must not be empty');
  }
  if (maxLength !== null && [...value].length > maxLength) {
    addIssue(issues, 'store-metadata-length', file, key, `must be ${maxLength} characters or fewer`);
  }
}

function validateStringArray(value, file, key, issues, { minItems, unique = false }) {
  if (!Array.isArray(value)) {
    addIssue(issues, 'store-metadata-type', file, key, 'must be an array');
    return;
  }
  if (value.length < minItems) {
    addIssue(issues, 'store-metadata-count', file, key, `must have at least ${minItems} entries`);
  }
  const seen = new Set();
  value.forEach((entry, index) => {
    const entryKey = `${key}[${index}]`;
    validateString(entry, file, entryKey, issues);
    if (typeof entry !== 'string') {
      return;
    }
    const normalized = entry.trim().toLocaleLowerCase();
    if (unique && seen.has(normalized)) {
      addIssue(issues, 'store-metadata-duplicate', file, entryKey, 'must not duplicate another entry');
    }
    seen.add(normalized);
  });
}

function validateReleaseNotes(value, file, issues) {
  if (!isRecord(value)) {
    addIssue(issues, 'store-metadata-type', file, 'release_notes', 'must be an object');
    return;
  }
  const versions = Object.keys(value);
  if (versions.length === 0) {
    addIssue(issues, 'store-metadata-empty', file, 'release_notes', 'must include at least one version');
  }
  for (const version of versions) {
    if (!/^\d+\.\d+\.\d+$/.test(version)) {
      addIssue(issues, 'store-metadata-version', file, `release_notes.${version}`, 'version must use MAJOR.MINOR.PATCH');
    }
    validateStringArray(value[version], file, `release_notes.${version}`, issues, { minItems: 1 });
  }
}

function validateHttpsUrl(value, file, key, issues) {
  validateString(value, file, key, issues);
  if (typeof value !== 'string') {
    return;
  }
  let url;
  try {
    url = new URL(value);
  } catch {
    addIssue(issues, 'store-metadata-url', file, key, 'must be a valid URL');
    return;
  }
  if (url.protocol !== 'https:') {
    addIssue(issues, 'store-metadata-url', file, key, 'must use https');
  }
}

function validateScreenshotCaptions(value, file, issues) {
  if (!isRecord(value)) {
    addIssue(issues, 'store-metadata-type', file, 'screenshot_captions', 'must be an object');
    return;
  }
  for (const key of SCREENSHOT_KEYS) {
    if (!Object.hasOwn(value, key)) {
      addIssue(issues, 'store-metadata-required', file, `screenshot_captions.${key}`, 'required screenshot caption is missing');
      continue;
    }
    validateString(value[key], file, `screenshot_captions.${key}`, issues);
  }
  for (const key of Object.keys(value)) {
    if (!SCREENSHOT_KEYS.includes(key)) {
      addIssue(issues, 'store-metadata-unknown-key', file, `screenshot_captions.${key}`, 'unknown screenshot caption key');
    }
  }
}

async function discoverSourceFiles(rootDir) {
  let entries;
  try {
    entries = await readdir(join(rootDir, SOURCE_DIR), { withFileTypes: true });
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return [];
    }
    throw error;
  }
  return entries
    .filter((entry) => entry.isFile())
    .map((entry) => entry.name)
    .filter((fileName) => fileName.endsWith('.json'))
    .filter((fileName) => fileName !== 'schema.json')
    .filter((fileName) => !fileName.endsWith('_intentions.json'))
    .sort();
}

async function readJson(rootDir, relativePath) {
  let rawText;
  try {
    rawText = await readFile(join(rootDir, relativePath), 'utf8');
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return { ok: false, code: 'store-metadata-missing-file', message: 'file is missing' };
    }
    throw error;
  }
  try {
    return { ok: true, value: JSON.parse(rawText) };
  } catch (error) {
    return { ok: false, code: 'store-metadata-json', message: `invalid JSON: ${error.message}` };
  }
}

function addIssue(issues, code, file, key, message) {
  issues.push({ code, file, key, message });
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function renderReport(report) {
  const lines = [
    '# Store metadata source check',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Issues: ${report.issues.length}`,
    `Parsed files: ${report.parsed_files.length}`,
  ];

  if (report.issues.length > 0) {
    lines.push('', '## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.file}${key}: ${issue.code}: ${issue.message}`);
    }
  }

  return `${lines.join('\n')}\n`;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const report = await validateStoreMetadataSources();
  process.stdout.write(renderReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
