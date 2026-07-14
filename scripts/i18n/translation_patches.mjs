import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m11-t03/translation-patch-review.json';
const DEFAULT_TRIAGE_PATH = 'doc/qa/m11-t02/support-feedback-triage.json';
const HIGH_PRIORITY_SEVERITIES = new Set(['critical', 'high']);
const CONTENT_PREFIXES = Object.freeze([
  'app/lib/l10n/',
  'app/assets/i18n/',
  'api/content/i18n/',
  'web/content/i18n/',
  'release/store_metadata/source/',
]);

/**
 * @typedef {Object} TranslationPatchIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} TranslationPatchReport
 * @property {boolean} ok
 * @property {TranslationPatchIssue[]} issues
 * @property {string} evidence_file
 * @property {string} source_triage_evidence
 * @property {string[]} store_release_enabled_locales
 * @property {number} high_priority_translation_issue_count
 * @property {number} patch_count
 * @property {number} unresolved_issue_count
 * @property {string[]} patched_locales
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<TranslationPatchReport>}
 */
export async function buildTranslationPatchReport({
  rootDir = REPO_ROOT,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const registry = await loadLanguageRegistry(join(rootDir, 'config/languages.json'));
  const knownLocales = new Set(registry.languages.map((language) => language.route_code));
  const storeReleaseEnabledLocales = registry.languages
    .filter((language) => language.release.enabled)
    .map((language) => language.route_code)
    .sort();
  const issues = [];
  const evidenceResult = await readJson(rootDir, evidencePath, 'translation patch evidence is missing');
  const evidence = evidenceResult.ok ? evidenceResult.value : null;
  const sourceTriagePath = isRecord(evidence) && hasNonEmptyString(evidence.source_triage_evidence)
    ? evidence.source_triage_evidence
    : DEFAULT_TRIAGE_PATH;
  const triageResult = await readJson(rootDir, sourceTriagePath, 'support triage evidence is missing');
  const highPriorityTranslationIssues = triageResult.ok ? collectHighPriorityTranslationIssues(triageResult.value) : [];

  if (!evidenceResult.ok) {
    issues.push(createIssue('translation-patch-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence({
      evidencePath,
      evidence: evidenceResult.value,
      knownLocales,
      storeReleaseEnabledLocales,
      highPriorityTranslationIssues,
    }));
  }

  if (!triageResult.ok) {
    issues.push(createIssue('translation-patch-triage-missing', sourceTriagePath, null, triageResult.message));
  } else {
    issues.push(...validateTriageEvidence(sourceTriagePath, triageResult.value));
  }

  const patches = evidenceResult.ok ? readArray(evidenceResult.value.patches).filter(isRecord) : [];
  const patchedIssueIds = new Set(patches.map((patch) => patch.source_issue_id).filter((value) => typeof value === 'string'));
  const unresolvedIssues = highPriorityTranslationIssues.filter((issue) => !patchedIssueIds.has(issue.id));

  return {
    ok: issues.length === 0,
    issues,
    evidence_file: evidencePath,
    source_triage_evidence: sourceTriagePath,
    store_release_enabled_locales: storeReleaseEnabledLocales,
    high_priority_translation_issue_count: highPriorityTranslationIssues.length,
    patch_count: patches.length,
    unresolved_issue_count: unresolvedIssues.length,
    patched_locales: uniqueSorted(patches.map((patch) => patch.locale).filter((locale) => typeof locale === 'string')),
  };
}

/**
 * @param {TranslationPatchReport} report
 * @returns {string}
 */
export function renderTranslationPatchReport(report) {
  const lines = [
    '# Stone Signature high-priority translation patches',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    `Source triage evidence: ${report.source_triage_evidence}`,
    `Store-release-enabled locales: ${formatList(report.store_release_enabled_locales)}`,
    `High-priority translation issues: ${report.high_priority_translation_issue_count}`,
    `Patches: ${report.patch_count}`,
    `Unresolved issues: ${report.unresolved_issue_count}`,
    `Patched locales: ${formatList(report.patched_locales)}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No high-priority translation patch issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

async function readJson(rootDir, relativePath, missingMessage) {
  try {
    return {
      ok: true,
      value: JSON.parse(await readFile(join(rootDir, relativePath), 'utf8')),
    };
  } catch (error) {
    return {
      ok: false,
      message: error?.code === 'ENOENT' ? missingMessage : error.message,
    };
  }
}

function validateEvidence({
  evidencePath,
  evidence,
  knownLocales,
  storeReleaseEnabledLocales,
  highPriorityTranslationIssues,
}) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('translation-patch-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('translation-patch-format', evidencePath, 'format_version', 'must be 1'));
  }
  if (evidence.task !== 'M11-T03') {
    issues.push(createIssue('translation-patch-format', evidencePath, 'task', 'must be M11-T03'));
  }
  if (!hasNonEmptyString(evidence.reviewed_at)) {
    issues.push(createIssue('translation-patch-format', evidencePath, 'reviewed_at', 'must be a non-empty string'));
  }
  if (!hasNonEmptyString(evidence.source_triage_evidence)) {
    issues.push(createIssue('translation-patch-format', evidencePath, 'source_triage_evidence', 'must be a non-empty string'));
  }
  issues.push(...compareList(evidencePath, 'store_release_enabled_locales', readStringArray(evidence.store_release_enabled_locales), storeReleaseEnabledLocales));
  issues.push(...validatePatches(evidencePath, evidence.patches, knownLocales, highPriorityTranslationIssues));
  issues.push(...validateValidationEvidence(evidencePath, evidence.validation));
  issues.push(...validateSummary(evidencePath, evidence.summary));
  return issues;
}

function validateTriageEvidence(evidencePath, value) {
  const issues = [];
  if (!isRecord(value)) {
    return [createIssue('translation-patch-triage-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (value.task !== 'M11-T02') {
    issues.push(createIssue('translation-patch-triage-format', evidencePath, 'task', 'source triage evidence must be M11-T02'));
  }
  if (!Array.isArray(value.triage_groups)) {
    issues.push(createIssue('translation-patch-triage-format', evidencePath, 'triage_groups', 'must be an array'));
  }
  return issues;
}

function collectHighPriorityTranslationIssues(triageEvidence) {
  if (!isRecord(triageEvidence)) {
    return [];
  }
  const collected = [];
  for (const group of readArray(triageEvidence.triage_groups)) {
    if (!isRecord(group)) {
      continue;
    }
    for (const issue of readArray(group.issues)) {
      if (!isRecord(issue)) {
        continue;
      }
      if (issue.category === 'translation' && HIGH_PRIORITY_SEVERITIES.has(issue.severity) && hasNonEmptyString(issue.id)) {
        collected.push({
          id: issue.id,
          locale: group.locale,
          platform: group.platform,
          screen: group.screen,
        });
      }
    }
  }
  return collected;
}

function validatePatches(evidencePath, value, knownLocales, highPriorityTranslationIssues) {
  const issues = [];
  if (!Array.isArray(value)) {
    return [createIssue('translation-patch-format', evidencePath, 'patches', 'must be an array')];
  }
  const highPriorityIssueById = new Map(highPriorityTranslationIssues.map((issue) => [issue.id, issue]));
  const patchedIssueIds = new Set();

  for (const [index, patch] of value.entries()) {
    const key = `patches[${index}]`;
    if (!isRecord(patch)) {
      issues.push(createIssue('translation-patch-format', evidencePath, key, 'patch must be an object'));
      continue;
    }
    if (patch.status !== 'pass') {
      issues.push(createIssue('translation-patch-status', evidencePath, `${key}.status`, 'status must be pass'));
    }
    if (!hasNonEmptyString(patch.source_issue_id)) {
      issues.push(createIssue('translation-patch-format', evidencePath, `${key}.source_issue_id`, 'must be a non-empty string'));
    } else {
      patchedIssueIds.add(patch.source_issue_id);
      const sourceIssue = highPriorityIssueById.get(patch.source_issue_id);
      if (!sourceIssue) {
        issues.push(createIssue('translation-patch-source', evidencePath, `${key}.source_issue_id`, 'must reference a high-priority translation issue from M11-T02'));
      } else if (patch.locale !== sourceIssue.locale) {
        issues.push(createIssue('translation-patch-locale', evidencePath, `${key}.locale`, `must match source issue locale ${sourceIssue.locale}`));
      }
    }
    if (!hasNonEmptyString(patch.owner)) {
      issues.push(createIssue('translation-patch-owner', evidencePath, `${key}.owner`, 'must be a non-empty string'));
    }
    if (!hasNonEmptyString(patch.locale) || !knownLocales.has(patch.locale)) {
      issues.push(createIssue('translation-patch-locale', evidencePath, `${key}.locale`, 'locale must match config/languages.json route_code'));
    }
    issues.push(...validatePatchFiles(evidencePath, key, patch.files));
    issues.push(...validatePatchValidation(evidencePath, key, patch.validation));
  }

  for (const issue of highPriorityTranslationIssues) {
    if (!patchedIssueIds.has(issue.id)) {
      issues.push(createIssue('translation-patch-unresolved', evidencePath, issue.id, 'high-priority translation issue has no patch'));
    }
  }

  return issues;
}

function validatePatchFiles(evidencePath, patchKey, files) {
  const issues = [];
  if (!Array.isArray(files) || files.length === 0) {
    return [createIssue('translation-patch-files', evidencePath, `${patchKey}.files`, 'must list one or more content files')];
  }
  for (const [index, file] of files.entries()) {
    const key = `${patchKey}.files[${index}]`;
    if (!hasNonEmptyString(file)) {
      issues.push(createIssue('translation-patch-files', evidencePath, key, 'must be a non-empty string'));
    } else if (!CONTENT_PREFIXES.some((prefix) => file.startsWith(prefix))) {
      issues.push(createIssue('translation-patch-content-only', evidencePath, key, 'patch files must be localization content paths'));
    }
  }
  return issues;
}

function validatePatchValidation(evidencePath, patchKey, validation) {
  const issues = [];
  if (!Array.isArray(validation) || validation.length === 0) {
    return [createIssue('translation-patch-validation', evidencePath, `${patchKey}.validation`, 'must include validation commands')];
  }
  if (!validation.some((entry) => isRecord(entry) && entry.status === 'pass' && entry.command === 'make i18n-check')) {
    issues.push(createIssue('translation-patch-validation', evidencePath, patchKey, 'must include passing make i18n-check evidence'));
  }
  return issues;
}

function validateValidationEvidence(evidencePath, validation) {
  const issues = [];
  if (!Array.isArray(validation)) {
    return [createIssue('translation-patch-validation', evidencePath, 'validation', 'must be an array')];
  }
  if (!validation.some((entry) => isRecord(entry) && entry.status === 'pass' && entry.command === 'make i18n-check')) {
    issues.push(createIssue('translation-patch-validation', evidencePath, 'validation', 'must include passing make i18n-check evidence'));
  }
  return issues;
}

function validateSummary(evidencePath, summary) {
  const issues = [];
  if (!isRecord(summary)) {
    return [createIssue('translation-patch-format', evidencePath, 'summary', 'must be an object')];
  }
  for (const key of [
    'high_priority_translation_issues_patched',
    'content_only_patches',
    'i18n_check_passed',
    'store_release_enabled_languages_clean',
  ]) {
    if (summary[key] !== true) {
      issues.push(createIssue('translation-patch-summary', evidencePath, key, 'must be true'));
    }
  }
  return issues;
}

function compareList(evidencePath, key, actual, expected) {
  if (actual.length === expected.length && actual.every((value, index) => value === expected[index])) {
    return [];
  }
  return [createIssue('translation-patch-locale-set', evidencePath, key, `expected ${formatList(expected)}, got ${formatList(actual)}`)];
}

function readStringArray(value) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((entry) => typeof entry === 'string').sort();
}

function readArray(value) {
  return Array.isArray(value) ? value : [];
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function hasNonEmptyString(value) {
  return typeof value === 'string' && value.trim() !== '';
}

function uniqueSorted(values) {
  return Array.from(new Set(values)).sort();
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

function formatList(values) {
  return values.length > 0 ? values.join(', ') : 'none';
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const report = await buildTranslationPatchReport();
  process.stdout.write(renderTranslationPatchReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
