import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m11-t01/locale-diagnostics-review.json';
const REQUIRED_DIAGNOSTIC_STREAMS = Object.freeze([
  'unsupported_locale',
  'fallback_locale',
  'missing_content',
  'checkout_locale',
  'malformed_translation',
]);

/**
 * @typedef {Object} LocaleDiagnosticsIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} LocaleDiagnosticsReport
 * @property {boolean} ok
 * @property {LocaleDiagnosticsIssue[]} issues
 * @property {string} evidence_file
 * @property {string[]} release_enabled_locales
 * @property {string[]} diagnostic_streams
 * @property {number} unexpected_fallback_spikes
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<LocaleDiagnosticsReport>}
 */
export async function buildLocaleDiagnosticsReport({
  rootDir = REPO_ROOT,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const registry = await loadLanguageRegistry(join(rootDir, 'config/languages.json'));
  const releaseEnabledLocales = registry.languages
    .filter((language) => language.release.enabled)
    .map((language) => language.route_code)
    .sort();
  const issues = [];
  const evidenceResult = await readEvidence(rootDir, evidencePath);

  if (!evidenceResult.ok) {
    issues.push(createIssue('locale-diagnostics-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence(evidencePath, evidenceResult.value, releaseEnabledLocales));
  }

  const diagnosticStreams = evidenceResult.ok
    ? Object.keys(asRecord(evidenceResult.value.diagnostic_streams)).sort()
    : [];
  const unexpectedFallbackSpikes = evidenceResult.ok
    ? readArray(evidenceResult.value.unexpected_fallback_spikes).length
    : 0;

  return {
    ok: issues.length === 0,
    issues,
    evidence_file: evidencePath,
    release_enabled_locales: releaseEnabledLocales,
    diagnostic_streams: diagnosticStreams,
    unexpected_fallback_spikes: unexpectedFallbackSpikes,
  };
}

/**
 * @param {LocaleDiagnosticsReport} report
 * @returns {string}
 */
export function renderLocaleDiagnosticsReport(report) {
  const lines = [
    '# Stone Signature locale diagnostics',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    `Release-enabled locales: ${formatList(report.release_enabled_locales)}`,
    `Diagnostic streams: ${formatList(report.diagnostic_streams)}`,
    `Unexpected fallback spikes: ${report.unexpected_fallback_spikes}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No locale diagnostic issues.');
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
      message: error?.code === 'ENOENT' ? 'locale diagnostics evidence is missing' : error.message,
    };
  }
}

function validateEvidence(evidencePath, evidence, releaseEnabledLocales) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('locale-diagnostics-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('locale-diagnostics-format', evidencePath, 'format_version', 'must be 1'));
  }
  if (evidence.task !== 'M11-T01') {
    issues.push(createIssue('locale-diagnostics-format', evidencePath, 'task', 'must be M11-T01'));
  }
  if (typeof evidence.reviewed_at !== 'string' || evidence.reviewed_at.trim() === '') {
    issues.push(createIssue('locale-diagnostics-format', evidencePath, 'reviewed_at', 'must be a non-empty string'));
  }
  issues.push(...compareList(evidencePath, 'release_enabled_locales', readStringArray(evidence.release_enabled_locales), releaseEnabledLocales));
  issues.push(...validateDiagnosticStreams(evidencePath, evidence.diagnostic_streams));
  issues.push(...validateUnexpectedFallbackSpikes(evidencePath, evidence.unexpected_fallback_spikes, releaseEnabledLocales));
  issues.push(...validateSummary(evidencePath, evidence.summary));
  return issues;
}

function validateDiagnosticStreams(evidencePath, value) {
  const issues = [];
  if (!isRecord(value)) {
    return [createIssue('locale-diagnostics-format', evidencePath, 'diagnostic_streams', 'must be an object')];
  }
  for (const stream of REQUIRED_DIAGNOSTIC_STREAMS) {
    const entry = value[stream];
    if (!isRecord(entry)) {
      issues.push(createIssue('locale-diagnostics-stream', evidencePath, stream, 'required diagnostic stream is missing'));
      continue;
    }
    if (entry.status !== 'pass') {
      issues.push(createIssue('locale-diagnostics-stream', evidencePath, stream, 'status must be pass'));
    }
    if (!Number.isInteger(entry.events_reviewed) || entry.events_reviewed < 0) {
      issues.push(createIssue('locale-diagnostics-stream', evidencePath, stream, 'events_reviewed must be a non-negative integer'));
    }
    if (typeof entry.query !== 'string' || entry.query.trim() === '') {
      issues.push(createIssue('locale-diagnostics-stream', evidencePath, stream, 'query must describe the reviewed log query'));
    }
  }
  return issues;
}

function validateUnexpectedFallbackSpikes(evidencePath, value, releaseEnabledLocales) {
  const issues = [];
  if (!Array.isArray(value)) {
    return [createIssue('locale-diagnostics-format', evidencePath, 'unexpected_fallback_spikes', 'must be an array')];
  }
  const releaseEnabledSet = new Set(releaseEnabledLocales);
  for (const entry of value) {
    if (!isRecord(entry)) {
      issues.push(createIssue('locale-diagnostics-spike', evidencePath, null, 'unexpected fallback spike entries must be objects'));
      continue;
    }
    const locale = typeof entry.locale === 'string' ? entry.locale : null;
    if (!locale || !releaseEnabledSet.has(locale)) {
      issues.push(createIssue('locale-diagnostics-spike', evidencePath, locale, 'spike locale must be release-enabled'));
    }
    if (!Number.isInteger(entry.count) || entry.count <= 0) {
      issues.push(createIssue('locale-diagnostics-spike', evidencePath, locale, 'spike count must be a positive integer'));
    }
  }
  if (value.length > 0) {
    issues.push(createIssue('locale-diagnostics-spike', evidencePath, null, 'release-enabled locales have unexpected fallback spikes'));
  }
  return issues;
}

function validateSummary(evidencePath, summary) {
  const issues = [];
  if (!isRecord(summary)) {
    return [createIssue('locale-diagnostics-format', evidencePath, 'summary', 'must be an object')];
  }
  for (const key of [
    'unsupported_locale_reviewed',
    'fallback_locale_reviewed',
    'missing_content_reviewed',
    'checkout_locale_reviewed',
    'malformed_translation_reviewed',
  ]) {
    if (summary[key] !== true) {
      issues.push(createIssue('locale-diagnostics-summary', evidencePath, key, 'must be true'));
    }
  }
  return issues;
}

function compareList(evidencePath, key, actual, expected) {
  if (sameList(actual, expected)) {
    return [];
  }
  return [createIssue('locale-diagnostics-locale-set', evidencePath, key, `expected ${formatList(expected)}, got ${formatList(actual)}`)];
}

function sameList(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
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

function asRecord(value) {
  return isRecord(value) ? value : {};
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

function formatList(values) {
  return values.length > 0 ? values.join(', ') : 'none';
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const report = await buildLocaleDiagnosticsReport();
  process.stdout.write(renderLocaleDiagnosticsReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
