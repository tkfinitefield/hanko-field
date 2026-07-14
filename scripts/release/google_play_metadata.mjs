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
const OUTPUT_DIR = 'release/store_metadata/google_play';
const GOOGLE_PLAY_TEXT_FILES = [
  'title.txt',
  'short_description.txt',
  'full_description.txt',
  'changelogs/default.txt',
];

/**
 * @typedef {Object} GooglePlayMetadataIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} GooglePlayMetadataReport
 * @property {boolean} ok
 * @property {GooglePlayMetadataIssue[]} issues
 * @property {string[]} files
 */

/**
 * @param {{ rootDir?: string, requiredLocales?: string[] }} options
 * @returns {Promise<Map<string, string>>}
 */
export async function buildGooglePlayMetadataFiles({
  rootDir = REPO_ROOT,
  requiredLocales = undefined,
} = {}) {
  const validation = await validateStoreMetadataSources({ rootDir, requiredLocales });
  if (!validation.ok) {
    throw new GooglePlayMetadataError(
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
      throw new GooglePlayMetadataError([
        {
          code: 'google-play-unknown-locale',
          file: sourcePath,
          key: null,
          message: `source locale does not exist in config/languages.json: ${routeCode}`,
        },
      ]);
    }

    const googlePlayLocale = language.release.android_store_locale;
    if (!googlePlayLocale) {
      throw new GooglePlayMetadataError([
        {
          code: 'google-play-unsupported-locale',
          file: sourcePath,
          key: 'release.android_store_locale',
          message: `route_code "${routeCode}" does not define a Google Play store locale`,
        },
      ]);
    }
    if (!/^[a-z]{2,3}(-[A-Z]{2})?$/.test(googlePlayLocale)) {
      throw new GooglePlayMetadataError([
        {
          code: 'google-play-locale-format',
          file: sourcePath,
          key: 'release.android_store_locale',
          message: `Google Play store locale must look like en-US or ja-JP: ${googlePlayLocale}`,
        },
      ]);
    }

    const metadata = await readJson(join(rootDir, sourcePath));
    const issues = [];
    validateStoreMetadataDocument(metadata, sourcePath, issues);
    if (issues.length > 0) {
      throw new GooglePlayMetadataError(issues);
    }

    const basePath = `${OUTPUT_DIR}/${googlePlayLocale}`;
    files.set(`${basePath}/title.txt`, textFile(metadata.app_name));
    files.set(`${basePath}/short_description.txt`, textFile(metadata.short_description));
    files.set(`${basePath}/full_description.txt`, `${metadata.full_description.map(cleanText).join('\n\n')}\n`);
    files.set(
      `${basePath}/changelogs/default.txt`,
      `${latestReleaseNotes(metadata.release_notes).map(cleanText).join('\n')}\n`,
    );
  }

  return new Map([...files.entries()].sort(([a], [b]) => a.localeCompare(b)));
}

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<GooglePlayMetadataReport>}
 */
export async function generateGooglePlayMetadata({ rootDir = REPO_ROOT } = {}) {
  const expectedFiles = await buildGooglePlayMetadataFiles({ rootDir });
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
 * @returns {Promise<GooglePlayMetadataReport>}
 */
export async function checkGooglePlayMetadata({ rootDir = REPO_ROOT } = {}) {
  const expectedFiles = await buildGooglePlayMetadataFiles({ rootDir });
  const actualFiles = await readGeneratedFiles(rootDir);
  const issues = [];

  for (const [relativePath, expectedContents] of expectedFiles.entries()) {
    if (!actualFiles.has(relativePath)) {
      issues.push({
        code: 'google-play-missing-file',
        file: relativePath,
        key: null,
        message: 'generated file is missing',
      });
      continue;
    }
    if (actualFiles.get(relativePath) !== expectedContents) {
      issues.push({
        code: 'google-play-stale-file',
        file: relativePath,
        key: null,
        message: 'generated file is not in sync with source metadata',
      });
    }
  }

  for (const relativePath of actualFiles.keys()) {
    if (!expectedFiles.has(relativePath)) {
      issues.push({
        code: 'google-play-extra-file',
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

export class GooglePlayMetadataError extends Error {
  constructor(issues) {
    super(`Google Play metadata generation failed:\n${issues.map(formatIssue).join('\n')}`);
    this.name = 'GooglePlayMetadataError';
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
    '# Google Play metadata',
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
        ? await checkGooglePlayMetadata()
        : await generateGooglePlayMetadata();
    process.stdout.write(renderReport(report, mode));
    if (!report.ok) {
      process.exitCode = 1;
    }
  } catch (error) {
    if (error instanceof GooglePlayMetadataError) {
      process.stderr.write(`${error.message}\n`);
      process.exitCode = 1;
    } else {
      throw error;
    }
  }
}
