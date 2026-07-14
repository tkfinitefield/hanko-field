import { readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_RUNBOOK_PATH = 'doc/localized-release-runbook.md';
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m11-t05/release-runbook-review.json';

const REQUIRED_RUNBOOK_TOKENS = Object.freeze([
  '# Localized Release Runbook',
  '## Language Addition Flow',
  '## Store Metadata Update Flow',
  '## fastlane Release Flow',
  '## Post-Release Monitoring and Cleanup',
  '## Rollback',
  'config/languages.json',
  'release/store_metadata/source/',
  'release/store_metadata/google_play',
  'release/store_metadata/app_store',
  'app/android/fastlane/Fastfile',
  'app/ios/fastlane/Fastfile',
  'SUPPLY_JSON_KEY',
  'SUPPLY_VALIDATE_ONLY=false',
  'APP_STORE_CONNECT_API_KEY_PATH',
  'RELEASE_SIGNOFF_PATH',
  'RELEASE_SIGNOFF_CONFIRMATION',
  'bundle exec fastlane android metadata',
  'bundle exec fastlane android internal',
  'bundle exec fastlane android production',
  'bundle exec fastlane ios metadata',
  'bundle exec fastlane ios testflight_upload',
  'bundle exec fastlane ios production',
  'make i18n-check',
  'make i18n-ci',
  'make release-secret-guardrails-check',
  'make i18n-diagnostics-check',
  'make i18n-support-triage-check',
  'make i18n-translation-patches-check',
  'make i18n-migration-cleanup-check',
]);

const REQUIRED_EVIDENCE_SECTIONS = Object.freeze([
  'language_addition_flow',
  'store_metadata_update_flow',
  'fastlane_release_flow',
  'post_release_monitoring_and_cleanup',
  'rollback',
]);

const REQUIRED_EVIDENCE_COMMANDS = Object.freeze([
  'make i18n-check',
  'make store-metadata-check',
  'make google-play-metadata-check',
  'make app-store-metadata-check',
  'make screenshot-metadata-check',
  'make android-fastlane-check',
  'make ios-fastlane-check',
  'make release-secret-guardrails-check',
  'make i18n-ci',
]);

const REQUIRED_FASTLANE_LANES = Object.freeze([
  'app/android: bundle exec fastlane android metadata',
  'app/android: bundle exec fastlane android internal',
  'app/android: bundle exec fastlane android production',
  'app/ios: bundle exec fastlane ios metadata',
  'app/ios: bundle exec fastlane ios testflight_upload',
  'app/ios: bundle exec fastlane ios production',
]);

const REQUIRED_STORE_METADATA_PATHS = Object.freeze([
  'release/store_metadata/source/',
  'release/store_metadata/google_play',
  'release/store_metadata/app_store',
]);

/**
 * @typedef {Object} ReleaseRunbookIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} ReleaseRunbookReport
 * @property {boolean} ok
 * @property {ReleaseRunbookIssue[]} issues
 * @property {string} runbook_file
 * @property {string} evidence_file
 * @property {number} route_code_count
 * @property {string[]} required_sections
 * @property {string[]} required_commands
 * @property {string[]} fastlane_lanes
 */

/**
 * @param {{ rootDir?: string, runbookPath?: string, evidencePath?: string }} options
 * @returns {Promise<ReleaseRunbookReport>}
 */
export async function buildReleaseRunbookReport({
  rootDir = REPO_ROOT,
  runbookPath = DEFAULT_RUNBOOK_PATH,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const registry = await loadLanguageRegistry(join(rootDir, 'config/languages.json'));
  const issues = [];
  const runbookResult = await readText(rootDir, runbookPath, 'localized release runbook is missing');
  const evidenceResult = await readJson(rootDir, evidencePath, 'release runbook review evidence is missing');

  if (!runbookResult.ok) {
    issues.push(createIssue('release-runbook-missing', runbookPath, null, runbookResult.message));
  } else {
    issues.push(...validateRunbookText(runbookPath, runbookResult.value));
  }

  if (!evidenceResult.ok) {
    issues.push(createIssue('release-runbook-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence({
      evidencePath,
      evidence: evidenceResult.value,
      runbookPath,
      routeCodeCount: registry.languages.length,
    }));
  }

  return {
    ok: issues.length === 0,
    issues,
    runbook_file: runbookPath,
    evidence_file: evidencePath,
    route_code_count: registry.languages.length,
    required_sections: [...REQUIRED_EVIDENCE_SECTIONS],
    required_commands: [...REQUIRED_EVIDENCE_COMMANDS],
    fastlane_lanes: [...REQUIRED_FASTLANE_LANES],
  };
}

/**
 * @param {ReleaseRunbookReport} report
 * @returns {string}
 */
export function renderReleaseRunbookReport(report) {
  const lines = [
    '# Stone Signature localized release runbook',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Runbook file: ${report.runbook_file}`,
    `Evidence file: ${report.evidence_file}`,
    `Route codes: ${report.route_code_count}`,
    `Required sections: ${formatList(report.required_sections)}`,
    `Required commands: ${formatList(report.required_commands)}`,
    `fastlane lanes: ${formatList(report.fastlane_lanes)}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No release runbook issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

function validateRunbookText(runbookPath, text) {
  const issues = [];
  for (const token of REQUIRED_RUNBOOK_TOKENS) {
    if (!text.includes(token)) {
      issues.push(createIssue('release-runbook-content', runbookPath, token, 'required runbook content is missing'));
    }
  }
  return issues;
}

function validateEvidence({
  evidencePath,
  evidence,
  runbookPath,
  routeCodeCount,
}) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('release-runbook-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('release-runbook-format', evidencePath, 'format_version', 'must be 1'));
  }
  if (evidence.task !== 'M11-T05') {
    issues.push(createIssue('release-runbook-format', evidencePath, 'task', 'must be M11-T05'));
  }
  if (!hasNonEmptyString(evidence.reviewed_at)) {
    issues.push(createIssue('release-runbook-format', evidencePath, 'reviewed_at', 'must be a non-empty string'));
  }
  if (evidence.runbook_path !== runbookPath) {
    issues.push(createIssue('release-runbook-format', evidencePath, 'runbook_path', `must be ${runbookPath}`));
  }
  if (evidence.route_code_count !== routeCodeCount) {
    issues.push(createIssue('release-runbook-route-count', evidencePath, 'route_code_count', `must match config/languages.json count ${routeCodeCount}`));
  }

  issues.push(...compareList(evidencePath, 'required_sections', readStringArray(evidence.required_sections), REQUIRED_EVIDENCE_SECTIONS));
  issues.push(...compareList(evidencePath, 'required_commands', readStringArray(evidence.required_commands), REQUIRED_EVIDENCE_COMMANDS));
  issues.push(...compareList(evidencePath, 'fastlane_lanes', readStringArray(evidence.fastlane_lanes), REQUIRED_FASTLANE_LANES));
  issues.push(...compareList(evidencePath, 'store_metadata_paths', readStringArray(evidence.store_metadata_paths), REQUIRED_STORE_METADATA_PATHS));
  issues.push(...validateSummary(evidencePath, evidence.summary));
  return issues;
}

function validateSummary(evidencePath, summary) {
  const issues = [];
  if (!isRecord(summary)) {
    return [createIssue('release-runbook-format', evidencePath, 'summary', 'must be an object')];
  }
  for (const key of [
    'future_language_steps_documented',
    'store_metadata_steps_documented',
    'fastlane_release_steps_documented',
    'post_release_cleanup_documented',
    'production_secrets_excluded',
  ]) {
    if (summary[key] !== true) {
      issues.push(createIssue('release-runbook-summary', evidencePath, key, 'must be true'));
    }
  }
  return issues;
}

async function readText(rootDir, relativePath, missingMessage) {
  try {
    return {
      ok: true,
      value: await readFile(join(rootDir, relativePath), 'utf8'),
    };
  } catch (error) {
    return {
      ok: false,
      message: error?.code === 'ENOENT' ? missingMessage : error.message,
    };
  }
}

async function readJson(rootDir, relativePath, missingMessage) {
  const result = await readText(rootDir, relativePath, missingMessage);
  if (!result.ok) {
    return result;
  }
  try {
    return {
      ok: true,
      value: JSON.parse(result.value),
    };
  } catch (error) {
    return {
      ok: false,
      message: error.message,
    };
  }
}

function compareList(evidencePath, key, actual, expected) {
  const normalizedExpected = [...expected].sort();
  if (actual.length === normalizedExpected.length && actual.every((value, index) => value === normalizedExpected[index])) {
    return [];
  }
  return [createIssue('release-runbook-list', evidencePath, key, `expected ${formatList(normalizedExpected)}, got ${formatList(actual)}`)];
}

function readStringArray(value) {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((entry) => typeof entry === 'string').sort();
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function hasNonEmptyString(value) {
  return typeof value === 'string' && value.trim() !== '';
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

function formatList(values) {
  return values.length > 0 ? values.join(', ') : 'none';
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const report = await buildReleaseRunbookReport();
  process.stdout.write(renderReleaseRunbookReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
