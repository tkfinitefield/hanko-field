import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { validateIntentions } from './intentions.mjs';
import { loadLanguageRegistry } from './registry.mjs';
import { parseLangsFilter } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const INTENTION_ROOTS = [
  'app/lib/l10n',
  'app/assets/i18n',
  'web/content/i18n',
  'api/content/i18n',
  'release/store_metadata/source',
];
const DEFERRED_REASON_CODES = new Set([
  'locale_not_release_enabled',
  'pending_human_translation',
  'source_not_available',
]);

/**
 * @typedef {Object} HoldoutEntry
 * @property {string} file
 * @property {string} key_path
 * @property {string} target_locale
 * @property {string} reason_code
 * @property {string} reason_group
 * @property {string} source_value
 * @property {boolean} deferred
 *
 * @typedef {Object} HoldoutReviewReport
 * @property {boolean} ok
 * @property {import('./intentions.mjs').IntentionIssue[]} issues
 * @property {string[]} languages
 * @property {HoldoutEntry[]} entries
 * @property {Record<string, number>} reason_counts
 * @property {Record<string, number>} group_counts
 * @property {number} parsed_sidecars
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null }} options
 * @returns {Promise<HoldoutReviewReport>}
 */
export async function buildHoldoutReview({ rootDir = REPO_ROOT, langs = ['all'] } = {}) {
  const registry = await loadRegistry(rootDir);
  const languages = selectTargetLanguages(registry.languages, langs);
  const languageSet = new Set(languages.map((language) => language.route_code));
  const validation = await validateIntentions({ rootDir, langs });
  const sidecarPaths = await collectIntentionSidecars(rootDir);
  const sidecarIssues = [];
  const entries = [];

  for (const sidecarPath of sidecarPaths) {
    const raw = await readOptionalJson(rootDir, sidecarPath);
    if (!raw.ok) {
      sidecarIssues.push({
        code: 'holdout-sidecar-json',
        file: sidecarPath,
        key: null,
        message: raw.message,
      });
      continue;
    }
    const inferredLocale = inferLocaleFromSidecar(sidecarPath);
    entries.push(...sidecarEntries(raw.value, sidecarPath, inferredLocale, languageSet));
  }

  entries.sort((left, right) =>
    [
      left.deferred ? '1' : '0',
      left.reason_group,
      left.reason_code,
      left.target_locale,
      left.file,
      left.key_path,
    ]
      .join('\0')
      .localeCompare(
        [
          right.deferred ? '1' : '0',
          right.reason_group,
          right.reason_code,
          right.target_locale,
          right.file,
          right.key_path,
        ].join('\0'),
      ),
  );

  const issues = [...validation.issues, ...sidecarIssues];
  return {
    ok: issues.length === 0,
    issues,
    languages: languages.map((language) => language.route_code),
    entries,
    reason_counts: countBy(entries, (entry) => entry.reason_code),
    group_counts: countBy(entries, (entry) => entry.reason_group),
    parsed_sidecars: sidecarPaths.length,
  };
}

/**
 * @param {HoldoutReviewReport} report
 * @returns {string}
 */
export function renderHoldoutReview(report) {
  const reviewedEntries = report.entries.filter((entry) => !entry.deferred);
  const deferredEntries = report.entries.filter((entry) => entry.deferred);
  const lines = [
    '# Stone Signature i18n holdout review',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Languages: ${report.languages.length === 0 ? 'none' : report.languages.join(', ')}`,
    `Parsed sidecars: ${report.parsed_sidecars}`,
    `Approved holdout entries: ${report.entries.length}`,
    `Reviewed shared English/legal entries: ${reviewedEntries.length}`,
    `Deferred translation entries: ${deferredEntries.length}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
    lines.push('');
  }

  lines.push('## Reason Summary', '', '| reason code | entries |', '| --- | ---: |');
  for (const [reasonCode, count] of Object.entries(report.reason_counts).sort()) {
    lines.push(`| ${md(reasonCode)} | ${count} |`);
  }

  lines.push('', '## Group Summary', '', '| group | entries |', '| --- | ---: |');
  for (const [group, count] of Object.entries(report.group_counts).sort()) {
    lines.push(`| ${md(group)} | ${count} |`);
  }

  lines.push('', '## Reviewed Holdouts', '');
  if (reviewedEntries.length === 0) {
    lines.push('No reviewed shared English or legal holdouts.');
  } else {
    lines.push('| file | locale | key | reason | source value |');
    lines.push('| --- | --- | --- | --- | --- |');
    for (const entry of reviewedEntries) {
      lines.push(
        `| ${md(entry.file)} | ${md(entry.target_locale)} | ${md(entry.key_path)} | ${md(entry.reason_code)} | ${md(entry.source_value)} |`,
      );
    }
  }

  if (deferredEntries.length > 0) {
    lines.push(
      '',
      '## Deferred Entries',
      '',
      'Deferred entries are tracked by intention sidecars but are not approved release copy. They must be translated or re-reviewed before their locale is made app-selectable, web-indexed, or release-enabled.',
    );
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

async function loadRegistry(rootDir) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  return loadLanguageRegistry(new URL('config/languages.json', rootUrl));
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

async function collectIntentionSidecars(rootDir) {
  const results = [];
  for (const root of INTENTION_ROOTS) {
    await visit(root);
  }
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
      } else if (entry.isFile() && relativePath.endsWith('_intentions.json')) {
        results.push(relativePath);
      }
    }
  }
}

async function readOptionalJson(rootDir, relativePath) {
  try {
    return { ok: true, value: JSON.parse(await readFile(join(rootDir, relativePath), 'utf8')) };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function sidecarEntries(raw, sidecarPath, inferredLocale, languageSet) {
  if (!isRecord(raw)) {
    return [];
  }

  if (Array.isArray(raw.entries)) {
    return raw.entries.flatMap((entry) => {
      if (!isRecord(entry) || typeof entry.key_path !== 'string' || typeof entry.reason_code !== 'string') {
        return [];
      }
      const targetLocale =
        typeof entry.target_locale === 'string' && entry.target_locale.trim() !== ''
          ? entry.target_locale
          : inferredLocale;
      return createEntry({
        sidecarPath,
        keyPath: entry.key_path,
        targetLocale,
        reasonCode: entry.reason_code,
        sourceValue: typeof entry.source_value === 'string' ? entry.source_value : '',
        languageSet,
      });
    });
  }

  return Object.entries(raw).flatMap(([keyPath, reasonCode]) =>
    createEntry({
      sidecarPath,
      keyPath,
      targetLocale: inferredLocale,
      reasonCode,
      sourceValue: '',
      languageSet,
    }),
  );
}

function createEntry({ sidecarPath, keyPath, targetLocale, reasonCode, sourceValue, languageSet }) {
  if (
    typeof targetLocale !== 'string' ||
    targetLocale === 'en' ||
    !languageSet.has(targetLocale) ||
    typeof reasonCode !== 'string'
  ) {
    return [];
  }
  return [
    {
      file: sidecarPath,
      key_path: keyPath,
      target_locale: targetLocale,
      reason_code: reasonCode,
      reason_group: reasonGroup(reasonCode),
      source_value: sourceValue,
      deferred: DEFERRED_REASON_CODES.has(reasonCode),
    },
  ];
}

function reasonGroup(reasonCode) {
  if (['brand_name'].includes(reasonCode)) {
    return 'brand';
  }
  if (['product_name', 'product_model_or_font'].includes(reasonCode)) {
    return 'product';
  }
  if (['law_name', 'legal_entity', 'legal_entity_name'].includes(reasonCode)) {
    return 'legal';
  }
  if (['payment_provider'].includes(reasonCode)) {
    return 'provider';
  }
  if (['email', 'url', 'url_or_email'].includes(reasonCode)) {
    return 'contact';
  }
  if (
    [
      'code_literal',
      'code_or_identifier',
      'country_code',
      'currency_code',
      'font_name',
      'kanji_character',
      'technical_identifier',
    ].includes(reasonCode)
  ) {
    return 'technical';
  }
  if (reasonCode === 'intentionally_english') {
    return 'english_label';
  }
  if (reasonCode === 'locale_not_release_enabled') {
    return 'release_deferred';
  }
  if (['pending_human_translation', 'source_not_available'].includes(reasonCode)) {
    return 'translation_deferred';
  }
  return 'other';
}

function countBy(entries, keyFn) {
  const counts = {};
  for (const entry of entries) {
    const key = keyFn(entry);
    counts[key] = (counts[key] ?? 0) + 1;
  }
  return counts;
}

function inferLocaleFromSidecar(sidecarPath) {
  const fileName = sidecarPath.split('/').pop() ?? '';
  if (sidecarPath.startsWith('api/content/i18n/catalog/')) {
    return null;
  }
  if (sidecarPath.startsWith('app/lib/l10n/app_')) {
    return fileName.replace(/^app_/, '').replace(/_intentions\.json$/, '').replace(/^zh_Hant$/, 'zhtw');
  }
  if (fileName.endsWith('_intentions.json')) {
    const locale = fileName.replace(/_intentions\.json$/, '');
    if (/^[a-z][a-z0-9]*$/.test(locale)) {
      return locale;
    }
  }
  return null;
}

function md(value) {
  return truncate(String(value)).replaceAll('\\', '\\\\').replaceAll('|', '\\|').replaceAll('\n', '\\n');
}

function truncate(value, limit = 140) {
  return value.length > limit ? `${value.slice(0, limit - 1)}...` : value;
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await buildHoldoutReview({
      langs: parseLangsFilter(process.env.LANGS) ?? ['all'],
    });
    process.stdout.write(renderHoldoutReview(report));
    if (!report.ok || process.argv.includes('--check')) {
      process.exitCode = report.ok ? 0 : 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
