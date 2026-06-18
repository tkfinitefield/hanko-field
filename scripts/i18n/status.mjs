import { stat } from 'node:fs/promises';
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

const API_CONTENT_PATHS = [
  'api/content/i18n/catalog/materials.json',
  'api/content/i18n/catalog/stone_listings.json',
  'api/content/i18n/catalog/facet_tags.json',
  'api/content/i18n/catalog/countries.json',
];

/**
 * @typedef {Object} StatusItem
 * @property {'present' | 'missing'} status
 * @property {string} path
 * @property {string} label
 * @property {string | null} route_code
 *
 * @typedef {Object} StatusSection
 * @property {string} name
 * @property {StatusItem[]} items
 *
 * @typedef {Object} I18nStatus
 * @property {number} total_languages
 * @property {string[]} app_enabled
 * @property {string[]} web_enabled
 * @property {string[]} release_enabled
 * @property {StatusSection[]} sections
 */

/**
 * @param {{ rootDir?: string }} options
 * @returns {Promise<I18nStatus>}
 */
export async function buildI18nStatus({ rootDir = REPO_ROOT } = {}) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  const registry = await loadLanguageRegistry(new URL('config/languages.json', rootUrl));
  const appEnabled = registry.languages.filter((language) => language.app.enabled);
  const webEnabled = registry.languages.filter((language) => language.web.enabled);
  const releaseEnabled = registry.languages.filter((language) => language.release.enabled);

  const sections = [
    await buildSection(rootDir, 'app', expectedAppFiles(appEnabled)),
    await buildSection(rootDir, 'web', expectedWebFiles(webEnabled)),
    await buildSection(rootDir, 'api', expectedApiFiles(webEnabled)),
    await buildSection(rootDir, 'release', expectedReleaseFiles(releaseEnabled)),
  ];

  return {
    total_languages: registry.languages.length,
    app_enabled: appEnabled.map((language) => language.route_code),
    web_enabled: webEnabled.map((language) => language.route_code),
    release_enabled: releaseEnabled.map((language) => language.route_code),
    sections,
  };
}

/**
 * @param {I18nStatus} status
 * @returns {string}
 */
export function renderI18nStatus(status) {
  const lines = [
    'Stone Signature i18n status',
    `Registry languages: ${status.total_languages}`,
    `App enabled: ${formatCodes(status.app_enabled)}`,
    `Web enabled: ${formatCodes(status.web_enabled)}`,
    `Release enabled: ${formatCodes(status.release_enabled)}`,
    '',
  ];

  for (const section of status.sections) {
    const missing = section.items.filter((item) => item.status === 'missing');
    const present = section.items.length - missing.length;
    lines.push(`${section.name}: ${present}/${section.items.length} present`);

    if (missing.length === 0) {
      lines.push('  missing: none');
    } else {
      lines.push('  missing:');
      for (const item of missing) {
        lines.push(`    - ${item.path} (${item.label})`);
      }
    }

    lines.push('');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

export function expectedAppFiles(languages) {
  return languages.flatMap((language) => [
    {
      path: `app/lib/l10n/app_${flutterArbSuffix(language)}.arb`,
      label: `${language.route_code} ARB`,
      route_code: language.route_code,
    },
    {
      path: `app/assets/i18n/settings/${language.route_code}.json`,
      label: `${language.route_code} settings content`,
      route_code: language.route_code,
    },
  ]);
}

export function expectedWebFiles(languages) {
  return languages.flatMap((language) =>
    WEB_COPY_NAMESPACES.map((namespace) => ({
      path: `web/content/i18n/${namespace}/${language.route_code}.json`,
      label: `${language.route_code} ${namespace} copy`,
      route_code: language.route_code,
    })),
  );
}

export function expectedApiFiles(languages) {
  return [
    ...API_CONTENT_PATHS.map((path) => ({
      path,
      label: 'registry-backed catalog content',
      route_code: null,
    })),
    ...languages.map((language) => ({
      path: `api/content/i18n/checkout/${language.route_code}.json`,
      label: `${language.route_code} checkout copy`,
      route_code: language.route_code,
    })),
  ];
}

export function expectedReleaseFiles(languages) {
  return languages.map((language) => ({
    path: `release/store_metadata/source/${language.route_code}.json`,
    label: `${language.route_code} store metadata source`,
    route_code: language.route_code,
  }));
}

async function buildSection(rootDir, name, expectedFiles) {
  const items = await Promise.all(
    expectedFiles.map(async (file) => ({
      ...file,
      status: (await fileExists(join(rootDir, file.path))) ? 'present' : 'missing',
    })),
  );

  return { name, items };
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

function flutterArbSuffix(language) {
  const { languageCode, scriptCode, countryCode } = language.flutter;
  if (language.route_code === 'zh') {
    return 'zh';
  }

  return [languageCode, scriptCode, countryCode].filter(Boolean).join('_');
}

function formatCodes(codes) {
  return codes.length === 0 ? 'none' : codes.join(', ');
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const status = await buildI18nStatus();
    process.stdout.write(renderI18nStatus(status));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
