import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m11-t02/support-feedback-triage.json';
const REQUIRED_SUPPORT_SOURCES = Object.freeze([
  'support_email',
  'support_form',
  'google_play_reviews',
  'app_store_reviews',
]);
const VALID_PLATFORMS = new Set([
  'android',
  'ios',
  'web',
  'api',
  'google_play',
  'app_store',
  'unknown',
]);
const OWNER_REQUIRED_CATEGORIES = new Set(['translation', 'layout']);

/**
 * @typedef {Object} SupportTriageIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} SupportTriageReport
 * @property {boolean} ok
 * @property {SupportTriageIssue[]} issues
 * @property {string} evidence_file
 * @property {number} support_issue_count
 * @property {number} translation_issue_count
 * @property {number} layout_issue_count
 * @property {number} missing_owner_count
 * @property {string[]} triaged_locales
 * @property {string[]} triaged_platforms
 * @property {string[]} support_sources
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<SupportTriageReport>}
 */
export async function buildSupportTriageReport({
  rootDir = REPO_ROOT,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const registry = await loadLanguageRegistry(join(rootDir, 'config/languages.json'));
  const knownLocales = new Set(registry.languages.map((language) => language.route_code));
  const reportIssues = [];
  const evidenceResult = await readEvidence(rootDir, evidencePath);

  if (!evidenceResult.ok) {
    reportIssues.push(createIssue('support-triage-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    reportIssues.push(...validateEvidence(evidencePath, evidenceResult.value, knownLocales));
  }

  const entries = evidenceResult.ok ? readArray(evidenceResult.value.triage_groups).filter(isRecord) : [];
  const supportIssues = entries.flatMap((entry) => readArray(entry.issues));
  const translationIssues = supportIssues.filter((issue) => issue.category === 'translation');
  const layoutIssues = supportIssues.filter((issue) => issue.category === 'layout');
  const missingOwnerIssues = supportIssues.filter((issue) => OWNER_REQUIRED_CATEGORIES.has(issue.category) && !hasNonEmptyString(issue.owner));

  return {
    ok: reportIssues.length === 0,
    issues: reportIssues,
    evidence_file: evidencePath,
    support_issue_count: supportIssues.length,
    translation_issue_count: translationIssues.length,
    layout_issue_count: layoutIssues.length,
    missing_owner_count: missingOwnerIssues.length,
    triaged_locales: uniqueSorted(entries.map((entry) => entry.locale).filter((locale) => typeof locale === 'string')),
    triaged_platforms: uniqueSorted(entries.map((entry) => entry.platform).filter((platform) => typeof platform === 'string')),
    support_sources: evidenceResult.ok ? Object.keys(asRecord(evidenceResult.value.support_sources)).sort() : [],
  };
}

/**
 * @param {SupportTriageReport} report
 * @returns {string}
 */
export function renderSupportTriageReport(report) {
  const lines = [
    '# Stone Signature support feedback triage',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    `Support sources: ${formatList(report.support_sources)}`,
    `Triaged locales: ${formatList(report.triaged_locales)}`,
    `Triaged platforms: ${formatList(report.triaged_platforms)}`,
    `Support issues: ${report.support_issue_count}`,
    `Translation issues: ${report.translation_issue_count}`,
    `Layout issues: ${report.layout_issue_count}`,
    `Missing owners: ${report.missing_owner_count}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No support triage issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

async function readEvidence(rootDir, evidencePath) {
  try {
    return {
      ok: true,
      value: JSON.parse(await readFile(join(rootDir, evidencePath), 'utf8')),
    };
  } catch (error) {
    return {
      ok: false,
      message: error?.code === 'ENOENT' ? 'support triage evidence is missing' : error.message,
    };
  }
}

function validateEvidence(evidencePath, evidence, knownLocales) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('support-triage-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('support-triage-format', evidencePath, 'format_version', 'must be 1'));
  }
  if (evidence.task !== 'M11-T02') {
    issues.push(createIssue('support-triage-format', evidencePath, 'task', 'must be M11-T02'));
  }
  if (!hasNonEmptyString(evidence.reviewed_at)) {
    issues.push(createIssue('support-triage-format', evidencePath, 'reviewed_at', 'must be a non-empty string'));
  }
  issues.push(...validateSupportSources(evidencePath, evidence.support_sources));
  issues.push(...validateOwnerPolicy(evidencePath, evidence.owner_policy));
  issues.push(...validateTriageGroups(evidencePath, evidence.triage_groups, knownLocales));
  issues.push(...validateSummary(evidencePath, evidence.summary));
  return issues;
}

function validateSupportSources(evidencePath, value) {
  const issues = [];
  if (!isRecord(value)) {
    return [createIssue('support-triage-format', evidencePath, 'support_sources', 'must be an object')];
  }
  for (const source of REQUIRED_SUPPORT_SOURCES) {
    const entry = value[source];
    if (!isRecord(entry)) {
      issues.push(createIssue('support-triage-source', evidencePath, source, 'required support source is missing'));
      continue;
    }
    if (entry.status !== 'pass') {
      issues.push(createIssue('support-triage-source', evidencePath, source, 'status must be pass'));
    }
    if (!Number.isInteger(entry.records_reviewed) || entry.records_reviewed < 0) {
      issues.push(createIssue('support-triage-source', evidencePath, source, 'records_reviewed must be a non-negative integer'));
    }
    if (!hasNonEmptyString(entry.query)) {
      issues.push(createIssue('support-triage-source', evidencePath, source, 'query must describe the reviewed support source'));
    }
  }
  return issues;
}

function validateOwnerPolicy(evidencePath, value) {
  const issues = [];
  if (!isRecord(value)) {
    return [createIssue('support-triage-format', evidencePath, 'owner_policy', 'must be an object')];
  }
  for (const key of ['translation_owner', 'layout_owner']) {
    if (!hasNonEmptyString(value[key])) {
      issues.push(createIssue('support-triage-owner-policy', evidencePath, key, 'must be a non-empty string'));
    }
  }
  return issues;
}

function validateTriageGroups(evidencePath, value, knownLocales) {
  const issues = [];
  if (!Array.isArray(value)) {
    return [createIssue('support-triage-format', evidencePath, 'triage_groups', 'must be an array')];
  }
  for (const [groupIndex, group] of value.entries()) {
    const groupKey = `triage_groups[${groupIndex}]`;
    if (!isRecord(group)) {
      issues.push(createIssue('support-triage-format', evidencePath, groupKey, 'group must be an object'));
      continue;
    }
    if (!hasNonEmptyString(group.locale) || !knownLocales.has(group.locale)) {
      issues.push(createIssue('support-triage-locale', evidencePath, `${groupKey}.locale`, 'locale must match config/languages.json route_code'));
    }
    if (!hasNonEmptyString(group.platform) || !VALID_PLATFORMS.has(group.platform)) {
      issues.push(createIssue('support-triage-platform', evidencePath, `${groupKey}.platform`, 'platform is not supported'));
    }
    if (!hasNonEmptyString(group.screen)) {
      issues.push(createIssue('support-triage-format', evidencePath, `${groupKey}.screen`, 'screen must be a non-empty string'));
    }
    const issueItems = readArray(group.issues);
    if (!Array.isArray(group.issues)) {
      issues.push(createIssue('support-triage-format', evidencePath, `${groupKey}.issues`, 'issues must be an array'));
      continue;
    }
    for (const [issueIndex, issue] of issueItems.entries()) {
      issues.push(...validateTriageIssue(evidencePath, `${groupKey}.issues[${issueIndex}]`, issue));
    }
  }
  return issues;
}

function validateTriageIssue(evidencePath, key, issue) {
  const issues = [];
  if (!isRecord(issue)) {
    return [createIssue('support-triage-format', evidencePath, key, 'issue must be an object')];
  }
  for (const field of ['id', 'category', 'severity', 'status', 'summary']) {
    if (!hasNonEmptyString(issue[field])) {
      issues.push(createIssue('support-triage-format', evidencePath, `${key}.${field}`, 'must be a non-empty string'));
    }
  }
  if (OWNER_REQUIRED_CATEGORIES.has(issue.category) && !hasNonEmptyString(issue.owner)) {
    issues.push(createIssue('support-triage-owner', evidencePath, `${key}.owner`, 'translation and layout issues must have an owner'));
  }
  return issues;
}

function validateSummary(evidencePath, summary) {
  const issues = [];
  if (!isRecord(summary)) {
    return [createIssue('support-triage-format', evidencePath, 'summary', 'must be an object')];
  }
  for (const key of [
    'grouped_by_language_platform_screen',
    'translation_fixes_have_owners',
    'layout_fixes_have_owners',
  ]) {
    if (summary[key] !== true) {
      issues.push(createIssue('support-triage-summary', evidencePath, key, 'must be true'));
    }
  }
  return issues;
}

function readArray(value) {
  return Array.isArray(value) ? value : [];
}

function asRecord(value) {
  return isRecord(value) ? value : {};
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
  const report = await buildSupportTriageReport();
  process.stdout.write(renderSupportTriageReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
