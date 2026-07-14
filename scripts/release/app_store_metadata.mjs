import { mkdir, readdir, readFile, rm, writeFile } from 'node:fs/promises';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from '../i18n/registry.mjs';
import {
  validateStoreMetadataDocument,
  validateStoreMetadataSources,
} from './store_metadata.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const SOURCE_DIR = 'release/store_metadata/source';
const OUTPUT_DIR = 'release/store_metadata/app_store';
const IOS_STORE_LOCALES = new Set([
  'ar-SA',
  'bn-BD',
  'ca',
  'cs',
  'da',
  'de-DE',
  'el',
  'en-AU',
  'en-CA',
  'en-GB',
  'en-US',
  'es-ES',
  'es-MX',
  'fi',
  'fr-CA',
  'fr-FR',
  'gu-IN',
  'he',
  'hi',
  'hr',
  'hu',
  'id',
  'it',
  'ja',
  'kn-IN',
  'ko',
  'ml-IN',
  'mr-IN',
  'ms',
  'nl-NL',
  'no',
  'or-IN',
  'pa-IN',
  'pl',
  'pt-BR',
  'pt-PT',
  'ro',
  'ru',
  'sk',
  'sl-SI',
  'sv',
  'ta-IN',
  'te-IN',
  'th',
  'tr',
  'uk',
  'ur-PK',
  'vi',
  'zh-Hans',
  'zh-Hant',
]);

/**
 * @typedef {Object} AppStoreMetadataIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} AppStoreMetadataReport
 * @property {boolean} ok
 * @property {AppStoreMetadataIssue[]} issues
 * @property {string[]} files
 */

/**
 * @param {{ rootDir?: string, requiredLocales?: string[] }} options
 * @returns {Promise<Map<string, string>>}
 */
export async function buildAppStoreMetadataFiles({
  rootDir = REPO_ROOT,
  requiredLocales = undefined,
} = {}) {
  const validation = await validateStoreMetadataSources({ rootDir, requiredLocales });
  if (!validation.ok) {
    throw new AppStoreMetadataError(
      validation.issues.map((issue) => ({
        code: issue.code,
        file: issue.file,
        key: issue.key,
        message: issue.message,
      })),
    );
  }

  const registry = await loadLanguageRegistry(join(rootDir, 'config/languages.json'));
  const byRouteCode = new Map(registry.languages.map((language) => [language.route_code, language]));
  const files = new Map();

  for (const sourceFile of await discoverSourceFiles(rootDir)) {
    const routeCode = sourceFile.replace(/\.json$/, '');
    const sourcePath = `${SOURCE_DIR}/${sourceFile}`;
    const language = byRouteCode.get(routeCode);
    if (!language) {
      throw new AppStoreMetadataError([
        {
          code: 'app-store-unknown-locale',
          file: sourcePath,
          key: null,
          message: `source locale does not exist in config/languages.json: ${routeCode}`,
        },
      ]);
    }

    const appStoreLocale = language.release.ios_store_locale;
    if (!appStoreLocale) {
      throw new AppStoreMetadataError([
        {
          code: 'app-store-unsupported-locale',
          file: sourcePath,
          key: 'release.ios_store_locale',
          message: `route_code "${routeCode}" does not define an App Store locale`,
        },
      ]);
    }
    if (!IOS_STORE_LOCALES.has(appStoreLocale)) {
      throw new AppStoreMetadataError([
        {
          code: 'app-store-unsupported-locale',
          file: sourcePath,
          key: 'release.ios_store_locale',
          message: `App Store locale is not in the supported fastlane deliver locale list: ${appStoreLocale}`,
        },
      ]);
    }

    const metadata = await readJson(join(rootDir, sourcePath));
    const issues = [];
    validateStoreMetadataDocument(metadata, sourcePath, issues);
    if (issues.length > 0) {
      throw new AppStoreMetadataError(issues);
    }

    const basePath = `${OUTPUT_DIR}/${appStoreLocale}`;
    files.set(`${basePath}/name.txt`, textFile(metadata.app_name));
    files.set(`${basePath}/subtitle.txt`, textFile(metadata.subtitle));
    files.set(`${basePath}/description.txt`, `${metadata.full_description.map(cleanText).join('\n\n')}\n`);
    files.set(`${basePath}/keywords.txt`, textFile(metadata.keywords.map(cleanText).join(',')));
    files.set(`${basePath}/release_notes.txt`, `${latestReleaseNotes(metadata.release_notes).map(cleanText).join('\n')}\n`);
    files.set(`${basePath}/support_url.txt`, textFile(metadata.support_url));
    files.set(`${basePath}/marketing_url.txt`, textFile(metadata.marketing_url));
    files.set(`${basePath}/privacy_url.txt`, textFile(metadata.privacy_policy_url));
  }

  return new Map([...files.entries()].sort(([a], [b]) => a.localeCompare(b)));
}

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<AppStoreMetadataReport>}
 */
export async function generateAppStoreMetadata({ rootDir = REPO_ROOT } = {}) {
  const expectedFiles = await buildAppStoreMetadataFiles({ rootDir });
  await rm(join(rootDir, OUTPUT_DIR), { recursive: true, force: true });
  for (const [relativePath, contents] of expectedFiles.entries()) {
    const absolutePath = join(rootDir, relativePath);
    await mkdir(dirname(absolutePath), { recursive: true });
    await writeFile(absolutePath, contents);
  }
  return {
    ok: true,
    issues: [],
    files: [...expectedFiles.keys()],
  };
}

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<AppStoreMetadataReport>}
 */
export async function checkAppStoreMetadata({ rootDir = REPO_ROOT } = {}) {
  const expectedFiles = await buildAppStoreMetadataFiles({ rootDir });
  const actualFiles = await readGeneratedFiles(rootDir);
  const issues = [];

  for (const [relativePath, expectedContents] of expectedFiles.entries()) {
    if (!actualFiles.has(relativePath)) {
      issues.push({
        code: 'app-store-missing-file',
        file: relativePath,
        key: null,
        message: 'generated file is missing',
      });
      continue;
    }
    if (actualFiles.get(relativePath) !== expectedContents) {
      issues.push({
        code: 'app-store-stale-file',
        file: relativePath,
        key: null,
        message: 'generated file is not in sync with source metadata',
      });
    }
  }

  for (const relativePath of actualFiles.keys()) {
    if (!expectedFiles.has(relativePath)) {
      issues.push({
        code: 'app-store-extra-file',
        file: relativePath,
        key: null,
        message: 'generated file is not expected from source metadata',
      });
    }
  }

  return {
    ok: issues.length === 0,
    issues,
    files: [...expectedFiles.keys()],
  };
}

export class AppStoreMetadataError extends Error {
  constructor(issues) {
    super(`App Store metadata generation failed:\n${issues.map(formatIssue).join('\n')}`);
    this.name = 'AppStoreMetadataError';
    this.issues = issues;
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

async function readGeneratedFiles(rootDir) {
  const files = new Map();
  await walkGeneratedDirectory(rootDir, join(rootDir, OUTPUT_DIR), files);
  return new Map([...files.entries()].sort(([a], [b]) => a.localeCompare(b)));
}

async function walkGeneratedDirectory(rootDir, absoluteDir, files) {
  let entries;
  try {
    entries = await readdir(absoluteDir, { withFileTypes: true });
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return;
    }
    throw error;
  }

  for (const entry of entries) {
    const absolutePath = join(absoluteDir, entry.name);
    if (entry.isDirectory()) {
      await walkGeneratedDirectory(rootDir, absolutePath, files);
      continue;
    }
    if (!entry.isFile()) {
      continue;
    }
    const relativePath = relative(rootDir, absolutePath).replaceAll('\\', '/');
    files.set(relativePath, await readFile(absolutePath, 'utf8'));
  }
}

async function readJson(path) {
  return JSON.parse(await readFile(path, 'utf8'));
}

function latestReleaseNotes(releaseNotes) {
  const latestVersion = Object.keys(releaseNotes).sort(compareSemver).at(-1);
  return releaseNotes[latestVersion] ?? [];
}

function compareSemver(a, b) {
  const aParts = a.split('.').map(Number);
  const bParts = b.split('.').map(Number);
  for (let index = 0; index < Math.max(aParts.length, bParts.length); index += 1) {
    const diff = (aParts[index] ?? 0) - (bParts[index] ?? 0);
    if (diff !== 0) {
      return diff;
    }
  }
  return 0;
}

function textFile(value) {
  return `${value.trim()}\n`;
}

function cleanText(value) {
  return value.trim();
}

function formatIssue(issue) {
  const key = issue.key ? ` ${issue.key}` : '';
  return `- ${issue.file}${key}: ${issue.code}: ${issue.message}`;
}

function renderReport(report, mode) {
  const lines = [
    '# App Store metadata',
    '',
    `Mode: ${mode}`,
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Issues: ${report.issues.length}`,
    `Files: ${report.files.length}`,
  ];
  if (report.issues.length > 0) {
    lines.push('', '## Issues', '');
    for (const issue of report.issues) {
      lines.push(formatIssue(issue));
    }
  }
  return `${lines.join('\n')}\n`;
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const mode = process.argv.includes('--check') ? 'check' : 'generate';
  try {
    const report =
      mode === 'check'
        ? await checkAppStoreMetadata()
        : await generateAppStoreMetadata();
    process.stdout.write(renderReport(report, mode));
    if (!report.ok) {
      process.exitCode = 1;
    }
  } catch (error) {
    if (error instanceof AppStoreMetadataError) {
      process.stderr.write(`${error.message}\n`);
      process.exitCode = 1;
    } else {
      throw error;
    }
  }
}
