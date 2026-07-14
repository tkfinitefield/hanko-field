import { mkdir, readdir, readFile, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from '../i18n/registry.mjs';
import {
  validateStoreMetadataDocument,
  validateStoreMetadataSources,
} from './store_metadata.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const SOURCE_DIR = 'release/store_metadata/source';
const OUTPUT_PATH = 'release/store_metadata/screenshots/manifest.json';
const SCREENSHOT_KEYS = ['design', 'stones', 'checkout'];
const DEVICE_PROFILES = [
  {
    id: 'phone_6_5',
    label: 'Phone 6.5 inch',
    capture_size: '1290x2796',
    google_play_image_type: 'phoneScreenshots',
    app_store_device: 'iphone_6_5',
  },
  {
    id: 'tablet_12_9',
    label: 'Tablet 12.9 inch',
    capture_size: '2048x2732',
    google_play_image_type: 'tenInchScreenshots',
    app_store_device: 'ipad_12_9',
  },
];

/**
 * @typedef {Object} ScreenshotMetadataIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} ScreenshotMetadataReport
 * @property {boolean} ok
 * @property {ScreenshotMetadataIssue[]} issues
 * @property {string[]} files
 */

/**
 * @param {{ rootDir?: string, requiredLocales?: string[] }} options
 * @returns {Promise<Object>}
 */
export async function buildScreenshotManifest({
  rootDir = REPO_ROOT,
  requiredLocales = undefined,
} = {}) {
  const validation = await validateStoreMetadataSources({ rootDir, requiredLocales });
  if (!validation.ok) {
    throw new ScreenshotMetadataError(
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
  const locales = [];

  for (const sourceFile of await discoverSourceFiles(rootDir)) {
    const routeCode = sourceFile.replace(/\.json$/, '');
    const sourcePath = `${SOURCE_DIR}/${sourceFile}`;
    const language = byRouteCode.get(routeCode);
    if (!language) {
      throw new ScreenshotMetadataError([
        {
          code: 'screenshot-metadata-unknown-locale',
          file: sourcePath,
          key: null,
          message: `source locale does not exist in config/languages.json: ${routeCode}`,
        },
      ]);
    }

    const metadata = await readJson(join(rootDir, sourcePath));
    const issues = [];
    validateStoreMetadataDocument(metadata, sourcePath, issues);
    if (issues.length > 0) {
      throw new ScreenshotMetadataError(issues);
    }

    const androidStoreLocale = language.release.android_store_locale;
    const iosStoreLocale = language.release.ios_store_locale;
    if (!androidStoreLocale) {
      throw new ScreenshotMetadataError([
        {
          code: 'screenshot-metadata-unsupported-locale',
          file: sourcePath,
          key: 'release.android_store_locale',
          message: `route_code "${routeCode}" does not define a Google Play store locale`,
        },
      ]);
    }
    if (!iosStoreLocale) {
      throw new ScreenshotMetadataError([
        {
          code: 'screenshot-metadata-unsupported-locale',
          file: sourcePath,
          key: 'release.ios_store_locale',
          message: `route_code "${routeCode}" does not define an App Store locale`,
        },
      ]);
    }

    locales.push({
      route_code: routeCode,
      bcp47: language.bcp47,
      text_direction: language.text_direction,
      google_play_locale: androidStoreLocale,
      app_store_locale: iosStoreLocale,
      devices: DEVICE_PROFILES.map((device) => ({
        id: device.id,
        google_play_image_type: device.google_play_image_type,
        app_store_device: device.app_store_device,
        slots: SCREENSHOT_KEYS.map((key, index) =>
          buildScreenshotSlot({
            routeCode,
            googlePlayLocale: androidStoreLocale,
            appStoreLocale: iosStoreLocale,
            device,
            key,
            index,
            caption: metadata.screenshot_captions[key],
          }),
        ),
      })),
    });
  }

  return {
    schema_version: 1,
    source_pattern: 'release/store_metadata/screenshots/source/{route_code}/{device}/{NN}-{key}.png',
    prepared_pattern: 'release/store_metadata/screenshots/{platform}/{store_locale}/{device}/{NN}-{key}.png',
    screenshot_keys: SCREENSHOT_KEYS,
    devices: DEVICE_PROFILES,
    locales,
  };
}

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<ScreenshotMetadataReport>}
 */
export async function generateScreenshotMetadata({ rootDir = REPO_ROOT } = {}) {
  const manifest = await buildScreenshotManifest({ rootDir });
  const output = `${stableJson(manifest)}\n`;
  const absolutePath = join(rootDir, OUTPUT_PATH);
  await mkdir(dirname(absolutePath), { recursive: true });
  await writeFile(absolutePath, output);
  return {
    ok: true,
    issues: [],
    files: [OUTPUT_PATH],
  };
}

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<ScreenshotMetadataReport>}
 */
export async function checkScreenshotMetadata({ rootDir = REPO_ROOT } = {}) {
  const manifest = await buildScreenshotManifest({ rootDir });
  const expected = `${stableJson(manifest)}\n`;
  const issues = [];
  let actual;
  try {
    actual = await readFile(join(rootDir, OUTPUT_PATH), 'utf8');
  } catch (error) {
    if (error?.code !== 'ENOENT') {
      throw error;
    }
    issues.push({
      code: 'screenshot-metadata-missing-file',
      file: OUTPUT_PATH,
      key: null,
      message: 'generated screenshot manifest is missing',
    });
  }

  if (actual !== undefined && actual !== expected) {
    issues.push({
      code: 'screenshot-metadata-stale-file',
      file: OUTPUT_PATH,
      key: null,
      message: 'generated screenshot manifest is not in sync with source metadata',
    });
  }

  return {
    ok: issues.length === 0,
    issues,
    files: [OUTPUT_PATH],
  };
}

export class ScreenshotMetadataError extends Error {
  constructor(issues) {
    super(`Screenshot metadata generation failed:\n${issues.map(formatIssue).join('\n')}`);
    this.name = 'ScreenshotMetadataError';
    this.issues = issues;
  }
}

function buildScreenshotSlot({
  routeCode,
  googlePlayLocale,
  appStoreLocale,
  device,
  key,
  index,
  caption,
}) {
  const fileName = `${String(index + 1).padStart(2, '0')}-${key}.png`;
  return {
    key,
    position: index + 1,
    caption,
    source_path: `release/store_metadata/screenshots/source/${routeCode}/${device.id}/${fileName}`,
    google_play_path: `release/store_metadata/screenshots/google_play/${googlePlayLocale}/${device.id}/${fileName}`,
    app_store_path: `release/store_metadata/screenshots/app_store/${appStoreLocale}/${device.id}/${fileName}`,
  };
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

async function readJson(path) {
  return JSON.parse(await readFile(path, 'utf8'));
}

function stableJson(value) {
  return JSON.stringify(value, null, 2);
}

function formatIssue(issue) {
  const key = issue.key ? ` ${issue.key}` : '';
  return `- ${issue.file}${key}: ${issue.code}: ${issue.message}`;
}

function renderReport(report, mode) {
  const lines = [
    '# Screenshot metadata',
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
        ? await checkScreenshotMetadata()
        : await generateScreenshotMetadata();
    process.stdout.write(renderReport(report, mode));
    if (!report.ok) {
      process.exitCode = 1;
    }
  } catch (error) {
    if (error instanceof ScreenshotMetadataError) {
      process.stderr.write(`${error.message}\n`);
      process.exitCode = 1;
    } else {
      throw error;
    }
  }
}
