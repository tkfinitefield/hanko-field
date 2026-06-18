import { mkdir, readdir, readFile, stat, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { expectedArbLocale, expectedArbPath } from './arb.mjs';
import { loadLanguageRegistry } from './registry.mjs';
import { parseLangsFilter } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const APP_ARB_BASE = 'app/lib/l10n/app_en.arb';
const APP_SETTINGS_BASE = 'app/assets/i18n/settings/en.json';
const API_CHECKOUT_BASE = 'api/content/i18n/checkout/en.json';
const RELEASE_METADATA_BASE = 'release/store_metadata/source/en.json';

/**
 * @typedef {Object} StubFile
 * @property {'arb' | 'json'} kind
 * @property {string} source_file
 * @property {string} target_file
 * @property {string} target_locale
 *
 * @typedef {Object} StubReport
 * @property {boolean} ok
 * @property {string[]} created_files
 * @property {string[]} missing_files
 * @property {string[]} existing_files
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, check?: boolean }} options
 * @returns {Promise<StubReport>}
 */
export async function ensureLocaleStubs({ rootDir = REPO_ROOT, langs = ['all'], check = false } = {}) {
  const registry = await loadRegistry(rootDir);
  const languages = selectTargetLanguages(registry.languages, langs);
  const descriptors = await buildStubDescriptors(rootDir, languages);
  const reader = createJsonReader(rootDir);
  const createdFiles = [];
  const missingFiles = [];
  const existingFiles = [];

  for (const descriptor of descriptors) {
    if (await fileExists(join(rootDir, descriptor.target_file))) {
      existingFiles.push(descriptor.target_file);
      continue;
    }

    missingFiles.push(descriptor.target_file);
    if (check) {
      continue;
    }

    const source = await reader.readRequired(descriptor.source_file);
    const stub = descriptor.kind === 'arb'
      ? createArbStub(source, descriptor)
      : createJsonStub(source);
    await writeJson(rootDir, descriptor.target_file, stub);
    createdFiles.push(descriptor.target_file);
  }

  return {
    ok: check ? missingFiles.length === 0 : true,
    created_files: createdFiles.sort(),
    missing_files: missingFiles.sort(),
    existing_files: existingFiles.sort(),
  };
}

async function buildStubDescriptors(rootDir, languages) {
  return [
    ...appArbDescriptors(languages),
    ...appSettingsDescriptors(languages),
    ...(await webContentDescriptors(rootDir, languages)),
    ...apiCheckoutDescriptors(languages),
    ...(await releaseMetadataDescriptors(rootDir, languages)),
  ];
}

function appArbDescriptors(languages) {
  return languages
    .filter((language) => language.app.enabled || language.route_code !== 'en')
    .map((language) => ({
      kind: 'arb',
      source_file: APP_ARB_BASE,
      target_file: expectedArbPath(language),
      target_locale: language.route_code,
      arb_locale: expectedArbLocale(language),
    }));
}

function appSettingsDescriptors(languages) {
  return languages
    .filter((language) => language.app.enabled || language.route_code !== 'en')
    .map((language) => ({
      kind: 'json',
      source_file: APP_SETTINGS_BASE,
      target_file: `app/assets/i18n/settings/${language.route_code}.json`,
      target_locale: language.route_code,
    }));
}

async function webContentDescriptors(rootDir, languages) {
  const namespaces = await discoverWebNamespaces(rootDir);
  return languages
    .filter((language) => language.web.enabled || language.route_code !== 'en')
    .flatMap((language) =>
      namespaces.map((namespace) => ({
        kind: 'json',
        source_file: `web/content/i18n/${namespace}/en.json`,
        target_file: `web/content/i18n/${namespace}/${language.route_code}.json`,
        target_locale: language.route_code,
      })),
    );
}

function apiCheckoutDescriptors(languages) {
  return languages
    .filter((language) => language.web.enabled || language.route_code !== 'en')
    .map((language) => ({
      kind: 'json',
      source_file: API_CHECKOUT_BASE,
      target_file: `api/content/i18n/checkout/${language.route_code}.json`,
      target_locale: language.route_code,
    }));
}

async function releaseMetadataDescriptors(rootDir, languages) {
  if (!(await fileExists(join(rootDir, RELEASE_METADATA_BASE)))) {
    return [];
  }
  return languages
    .filter((language) => language.release.enabled)
    .map((language) => ({
      kind: 'json',
      source_file: RELEASE_METADATA_BASE,
      target_file: `release/store_metadata/source/${language.route_code}.json`,
      target_locale: language.route_code,
    }));
}

function createArbStub(source, descriptor) {
  const stub = {};
  for (const [key, value] of Object.entries(source)) {
    if (key === '@@locale') {
      stub[key] = descriptor.arb_locale;
    } else if (key.startsWith('@')) {
      stub[key] = cloneJson(value);
    } else if (typeof value !== 'string') {
      stub[key] = cloneJson(value);
    }
  }
  return stub;
}

function createJsonStub(value) {
  if (Array.isArray(value)) {
    return value.map((entry) => createJsonStub(entry));
  }
  if (isRecord(value)) {
    const entries = Object.entries(value)
      .filter(([, entryValue]) => typeof entryValue !== 'string')
      .map(([key, entryValue]) => [key, createJsonStub(entryValue)]);
    return Object.fromEntries(entries);
  }
  return typeof value === 'string' ? undefined : cloneJson(value);
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
    if (error?.code === 'ENOENT') {
      return [];
    }
    throw error;
  }
}

function selectTargetLanguages(languages, langs) {
  const requested = langs ?? ['all'];
  const baseFiltered = languages.filter((language) => language.route_code !== 'en');
  if (requested.includes('all')) {
    return baseFiltered;
  }

  const byRouteCode = new Map(languages.map((language) => [language.route_code, language]));
  const unknownCodes = requested.filter((code) => code !== 'en' && !byRouteCode.has(code));
  if (unknownCodes.length > 0) {
    throw new Error(`Unknown LANGS route code(s): ${unknownCodes.join(', ')}`);
  }
  return requested
    .map((code) => byRouteCode.get(code))
    .filter((language) => language && language.route_code !== 'en');
}

async function loadRegistry(rootDir) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  return loadLanguageRegistry(new URL('config/languages.json', rootUrl));
}

function createJsonReader(rootDir) {
  const cache = new Map();
  return {
    async readRequired(relativePath) {
      if (cache.has(relativePath)) {
        return cloneJson(cache.get(relativePath));
      }
      const value = JSON.parse(await readFile(join(rootDir, relativePath), 'utf8'));
      cache.set(relativePath, value);
      return cloneJson(value);
    },
  };
}

async function writeJson(rootDir, relativePath, value) {
  const path = join(rootDir, relativePath);
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`);
}

async function fileExists(path) {
  try {
    return (await stat(path)).isFile();
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return false;
    }
    throw error;
  }
}

function cloneJson(value) {
  return JSON.parse(JSON.stringify(value));
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function renderReport(report, mode) {
  const lines = [
    '# Locale stubs',
    '',
    `Mode: ${mode}`,
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Created files: ${report.created_files.length}`,
    `Missing files: ${report.missing_files.length}`,
    `Existing files: ${report.existing_files.length}`,
  ];
  if (report.missing_files.length > 0) {
    lines.push('', '## Missing files', '');
    for (const file of report.missing_files) {
      lines.push(`- ${file}`);
    }
  }
  return `${lines.join('\n')}\n`;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const check = process.argv.includes('--check');
  const report = await ensureLocaleStubs({
    langs: parseLangsFilter(process.env.LANGS) ?? ['all'],
    check,
  });
  process.stdout.write(renderReport(report, check ? 'check' : 'generate'));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
