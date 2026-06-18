import { existsSync } from 'node:fs';
import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m9-t05/flag-stages.json';
const STAGE_ORDER = Object.freeze([
  'disabled',
  'render_only',
  'app_selectable',
  'web_indexed',
  'store_release_enabled',
]);
const REQUIRED_EVIDENCE_KINDS = Object.freeze({
  disabled: ['i18n_check', 'stubs_check'],
  render_only: ['i18n_check', 'stubs_check', 'layout_qa'],
  app_selectable: ['i18n_check', 'holdout_review', 'layout_qa', 'app_layout'],
  web_indexed: ['i18n_check', 'holdout_review', 'layout_qa', 'web_layout'],
  store_release_enabled: [
    'i18n_check',
    'holdout_review',
    'layout_qa',
    'store_metadata',
    'fastlane_config',
    'release_secret_guardrails',
  ],
});

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} FlagStageIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} locale
 * @property {string} message
 *
 * @typedef {Object} FlagStageReport
 * @property {boolean} ok
 * @property {FlagStageIssue[]} issues
 * @property {Record<string, string[]>} current_stages
 * @property {string} evidence_file
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<FlagStageReport>}
 */
export async function buildFlagStageReport({
  rootDir = REPO_ROOT,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const registry = await loadRegistry(rootDir);
  const currentStages = classifyLanguageStages(registry.languages);
  const issues = validateFlagInvariants(rootDir, registry.languages);
  const evidenceResult = await readEvidence(rootDir, evidencePath);

  if (!evidenceResult.ok) {
    issues.push(createIssue('flag-stage-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence({ rootDir, evidencePath, evidence: evidenceResult.value, currentStages }));
  }

  return {
    ok: issues.length === 0,
    issues,
    current_stages: currentStages,
    evidence_file: evidencePath,
  };
}

/**
 * @param {LanguageEntry[]} languages
 * @returns {Record<string, string[]>}
 */
export function classifyLanguageStages(languages) {
  const stages = Object.fromEntries(STAGE_ORDER.map((stage) => [stage, []]));

  for (const language of languages) {
    if (language.route_code === 'en') {
      continue;
    }
    stages[stageForLanguage(language)].push(language.route_code);
  }

  return Object.fromEntries(STAGE_ORDER.map((stage) => [stage, stages[stage].sort()]));
}

/**
 * @param {FlagStageReport} report
 * @returns {string}
 */
export function renderFlagStageReport(report) {
  const lines = [
    '# Stone Signature i18n flag stages',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    '',
    '## Current Stages',
    '',
  ];

  for (const stage of STAGE_ORDER) {
    lines.push(`- ${stage}: ${formatList(report.current_stages[stage])}`);
  }

  if (report.issues.length > 0) {
    lines.push('', '## Issues', '');
    for (const issue of report.issues) {
      const locale = issue.locale ? ` ${issue.locale}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${locale} - ${issue.message}`);
    }
  } else {
    lines.push('', 'No flag stage issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

/**
 * @param {LanguageEntry} language
 * @returns {string}
 */
export function stageForLanguage(language) {
  if (language.release.enabled) {
    return 'store_release_enabled';
  }
  if (language.web.indexed) {
    return 'web_indexed';
  }
  if (language.app.selectable) {
    return 'app_selectable';
  }
  if (language.app.enabled || language.web.enabled) {
    return 'render_only';
  }
  return 'disabled';
}

function validateFlagInvariants(rootDir, languages) {
  const issues = [];
  for (const language of languages) {
    if (language.route_code === 'en') {
      continue;
    }
    if (language.app.selectable && !language.app.enabled) {
      issues.push(createIssue('flag-stage-app-selectable-disabled', 'config/languages.json', language.route_code, 'app.selectable requires app.enabled'));
    }
    if (language.web.indexed && !language.web.enabled) {
      issues.push(createIssue('flag-stage-web-indexed-disabled', 'config/languages.json', language.route_code, 'web.indexed requires web.enabled'));
    }
    if (language.release.enabled) {
      if (!language.app.enabled || !language.app.selectable) {
        issues.push(createIssue('flag-stage-release-app-disabled', 'config/languages.json', language.route_code, 'release.enabled requires app.enabled and app.selectable'));
      }
      if (!language.web.enabled || !language.web.indexed) {
        issues.push(createIssue('flag-stage-release-web-unindexed', 'config/languages.json', language.route_code, 'release.enabled requires web.enabled and web.indexed'));
      }
      if (!language.release.android_store_locale || !language.release.ios_store_locale) {
        issues.push(createIssue('flag-stage-release-store-locale', 'config/languages.json', language.route_code, 'release.enabled requires Android and iOS store locale mappings'));
      }
      const metadataPath = `release/store_metadata/source/${language.route_code}.json`;
      if (!existsSync(join(rootDir, metadataPath))) {
        issues.push(createIssue('flag-stage-release-metadata-missing', metadataPath, language.route_code, 'release.enabled requires store metadata source'));
      }
    }
  }
  return issues;
}

function validateEvidence({ rootDir, evidencePath, evidence, currentStages }) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('flag-stage-evidence-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, 'format_version must be 1'));
  }
  if (evidence.task !== 'M9-T05') {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, 'task must be M9-T05'));
  }
  if (!sameList(readStringArray(evidence.transition_order), STAGE_ORDER)) {
    issues.push(createIssue('flag-stage-transition-order', evidencePath, null, `transition_order must be ${STAGE_ORDER.join(', ')}`));
  }
  if (!isRecord(evidence.transition_policy)) {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, 'transition_policy must be an object'));
  } else {
    for (const key of [
      'single_transition_kind_per_commit',
      'no_public_index_or_release_until_prior_stage_passes',
    ]) {
      if (evidence.transition_policy[key] !== true) {
        issues.push(createIssue('flag-stage-policy', evidencePath, null, `transition_policy.${key} must be true`));
      }
    }
    if (typeof evidence.transition_policy.current_task_changes_registry_flags !== 'boolean') {
      issues.push(createIssue('flag-stage-policy', evidencePath, null, 'transition_policy.current_task_changes_registry_flags must be a boolean'));
    }
  }
  if (!isRecord(evidence.current_stages)) {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, 'current_stages must be an object'));
  } else {
    for (const stage of STAGE_ORDER) {
      issues.push(...compareLocaleSet(evidencePath, `current_stages.${stage}`, readStringArray(evidence.current_stages[stage], { sort: true }), currentStages[stage]));
    }
  }
  if (!isRecord(evidence.stages)) {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, 'stages must be an object'));
    return issues;
  }

  for (const stage of STAGE_ORDER) {
    const entries = readEntries(evidence.stages[stage]);
    issues.push(...compareLocaleSet(evidencePath, `stages.${stage}`, groupedLocales(entries), currentStages[stage]));
    for (const entry of entries) {
      issues.push(...validateStageEntry(rootDir, evidencePath, stage, entry));
    }
  }

  return issues;
}

function validateStageEntry(rootDir, evidencePath, stage, entry) {
  const issues = [];
  if (!isRecord(entry)) {
    return [createIssue('flag-stage-evidence-format', evidencePath, null, `${stage} entries must be objects`)];
  }
  if (entry.status !== 'pass') {
    issues.push(createIssue('flag-stage-status', evidencePath, null, `${stage} status must be pass`));
  }
  if (!Array.isArray(entry.locales) || entry.locales.some((locale) => typeof locale !== 'string')) {
    issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.locales must be a string array`));
  }
  issues.push(...validateEvidenceItems(rootDir, evidencePath, stage, entry.evidence, REQUIRED_EVIDENCE_KINDS[stage]));
  return issues;
}

function validateEvidenceItems(rootDir, evidencePath, stage, evidenceItems, requiredKinds) {
  const issues = [];
  if (!Array.isArray(evidenceItems)) {
    return [createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.evidence must be an array`)];
  }
  const seenKinds = new Set();
  for (const [index, item] of evidenceItems.entries()) {
    if (!isRecord(item)) {
      issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.evidence[${index}] must be an object`));
      continue;
    }
    if (typeof item.kind !== 'string' || item.kind.trim() === '') {
      issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.evidence[${index}].kind is required`));
    } else {
      seenKinds.add(item.kind);
    }
    if (typeof item.command !== 'string' || item.command.trim() === '') {
      issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.evidence[${index}].command is required`));
    }
    if (item.path !== undefined && (typeof item.path !== 'string' || item.path.trim() === '')) {
      issues.push(createIssue('flag-stage-evidence-format', evidencePath, null, `${stage}.evidence[${index}].path must be a non-empty string`));
    }
    if (typeof item.path === 'string' && !existsSync(join(rootDir, item.path))) {
      issues.push(createIssue('flag-stage-evidence-path', evidencePath, null, `${item.path} does not exist`));
    }
  }
  for (const requiredKind of requiredKinds) {
    if (!seenKinds.has(requiredKind)) {
      issues.push(createIssue('flag-stage-evidence-missing-kind', evidencePath, null, `${stage} missing evidence kind ${requiredKind}`));
    }
  }
  return issues;
}

async function loadRegistry(rootDir) {
  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  return loadLanguageRegistry(new URL('config/languages.json', rootUrl));
}

async function readEvidence(rootDir, evidencePath) {
  try {
    return { ok: true, value: JSON.parse(await readFile(join(rootDir, evidencePath), 'utf8')) };
  } catch (error) {
    return { ok: false, message: error instanceof Error ? error.message : String(error) };
  }
}

function readEntries(value) {
  return Array.isArray(value) ? value : [];
}

function readStringArray(value, { sort = false } = {}) {
  const items = Array.isArray(value) ? value.filter((entry) => typeof entry === 'string') : [];
  return sort ? items.sort() : items;
}

function groupedLocales(entries) {
  return entries
    .flatMap((entry) => (isRecord(entry) && Array.isArray(entry.locales) ? entry.locales : []))
    .filter((locale) => typeof locale === 'string')
    .sort();
}

function compareLocaleSet(evidencePath, label, actual, expected) {
  if (sameList(actual, expected)) {
    return [];
  }
  return [
    createIssue(
      'flag-stage-locale-mismatch',
      evidencePath,
      null,
      `${label} expected ${formatList(expected)} but found ${formatList(actual)}`,
    ),
  ];
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

function createIssue(code, file, locale, message) {
  return { code, file, locale, message };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await buildFlagStageReport({
      evidencePath: process.env.EVIDENCE ?? DEFAULT_EVIDENCE_PATH,
    });
    process.stdout.write(renderFlagStageReport(report));
    if (!report.ok || process.argv.includes('--check')) {
      process.exitCode = report.ok ? 0 : 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
