import { readdir, readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m11-t04/migration-cleanup.json';
const REQUIRED_REMOVED_WRAPPERS = Object.freeze([
  'HankoLocalizations typedef',
  'hankoSupportedLocales constant',
  'hankoLocalizationsDelegates constant',
]);
const FORBIDDEN_PATTERNS = Object.freeze([
  {
    code: 'migration-wrapper-typedef',
    file: 'app/lib/app/localization/hanko_localizations.dart',
    pattern: /\btypedef\s+HankoLocalizations\b/,
    message: 'HankoLocalizations typedef must be removed',
  },
  {
    code: 'migration-wrapper-supported-locales',
    file: 'app/lib/app/localization/hanko_localizations.dart',
    pattern: /\bhankoSupportedLocales\b/,
    message: 'hardcoded hankoSupportedLocales wrapper must be removed',
  },
  {
    code: 'migration-wrapper-delegates',
    file: 'app/lib/app/localization/hanko_localizations.dart',
    pattern: /\bhankoLocalizationsDelegates\b/,
    message: 'hankoLocalizationsDelegates wrapper must be removed',
  },
  {
    code: 'migration-wrapper-name',
    file: 'app/lib',
    pattern: /\bHankoLocalizations\b/,
    message: 'application code must use GeneratedHankoLocalizations directly',
  },
  {
    code: 'migration-wrapper-test-name',
    file: 'app/test',
    pattern: /\bHankoLocalizations\b|compatibility API/,
    message: 'tests must not preserve HankoLocalizations compatibility API names',
  },
]);

/**
 * @typedef {Object} MigrationCleanupIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} MigrationCleanupReport
 * @property {boolean} ok
 * @property {MigrationCleanupIssue[]} issues
 * @property {string} evidence_file
 * @property {string[]} removed_wrappers
 * @property {string[]} retained_compatibility
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<MigrationCleanupReport>}
 */
export async function buildMigrationCleanupReport({
  rootDir = REPO_ROOT,
  evidencePath = DEFAULT_EVIDENCE_PATH,
} = {}) {
  const issues = [];
  const evidenceResult = await readJson(rootDir, evidencePath, 'migration cleanup evidence is missing');

  if (!evidenceResult.ok) {
    issues.push(createIssue('migration-cleanup-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence(evidencePath, evidenceResult.value));
  }

  issues.push(...(await validateSource(rootDir)));

  return {
    ok: issues.length === 0,
    issues,
    evidence_file: evidencePath,
    removed_wrappers: evidenceResult.ok ? readStringArray(evidenceResult.value.removed_wrappers) : [],
    retained_compatibility: evidenceResult.ok ? readStringArray(evidenceResult.value.retained_compatibility) : [],
  };
}

/**
 * @param {MigrationCleanupReport} report
 * @returns {string}
 */
export function renderMigrationCleanupReport(report) {
  const lines = [
    '# Stone Signature migration wrapper cleanup',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    `Removed wrappers: ${formatList(report.removed_wrappers)}`,
    `Retained compatibility: ${formatList(report.retained_compatibility)}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const key = issue.key ? ` ${issue.key}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${key} - ${issue.message}`);
    }
  } else {
    lines.push('No migration cleanup issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

async function validateSource(rootDir) {
  const issues = [];
  for (const rule of FORBIDDEN_PATTERNS) {
    const source = await readSourceTree(rootDir, rule.file);
    if (!source.ok) {
      issues.push(createIssue('migration-cleanup-source-missing', rule.file, null, source.message));
      continue;
    }
    if (rule.pattern.test(source.value)) {
      issues.push(createIssue(rule.code, rule.file, null, rule.message));
    }
  }

  const appSource = await readSource(rootDir, 'app/lib/app/app.dart');
  if (appSource.ok) {
    if (!appSource.value.includes('GeneratedHankoLocalizations.supportedLocales')) {
      issues.push(createIssue('migration-cleanup-generated-locales', 'app/lib/app/app.dart', null, 'MaterialApp must use generated supportedLocales'));
    }
    if (!appSource.value.includes('GeneratedHankoLocalizations.localizationsDelegates')) {
      issues.push(createIssue('migration-cleanup-generated-delegates', 'app/lib/app/app.dart', null, 'MaterialApp must use generated localizationsDelegates'));
    }
  } else {
    issues.push(createIssue('migration-cleanup-source-missing', 'app/lib/app/app.dart', null, appSource.message));
  }

  const registrySource = await readSource(rootDir, 'app/lib/app/localization/language_registry.dart');
  if (registrySource.ok) {
    if (!registrySource.value.includes("assetPath = '../config/languages.json'")) {
      issues.push(createIssue('migration-cleanup-registry-path', 'app/lib/app/localization/language_registry.dart', null, 'app language registry must load config/languages.json'));
    }
  } else {
    issues.push(createIssue('migration-cleanup-source-missing', 'app/lib/app/localization/language_registry.dart', null, registrySource.message));
  }

  return issues;
}

function validateEvidence(evidencePath, evidence) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('migration-cleanup-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('migration-cleanup-format', evidencePath, 'format_version', 'must be 1'));
  }
  if (evidence.task !== 'M11-T04') {
    issues.push(createIssue('migration-cleanup-format', evidencePath, 'task', 'must be M11-T04'));
  }
  if (!hasNonEmptyString(evidence.reviewed_at)) {
    issues.push(createIssue('migration-cleanup-format', evidencePath, 'reviewed_at', 'must be a non-empty string'));
  }
  issues.push(...compareList(evidencePath, 'removed_wrappers', readStringArray(evidence.removed_wrappers), REQUIRED_REMOVED_WRAPPERS));
  issues.push(...validateSummary(evidencePath, evidence.summary));
  return issues;
}

function validateSummary(evidencePath, summary) {
  const issues = [];
  if (!isRecord(summary)) {
    return [createIssue('migration-cleanup-format', evidencePath, 'summary', 'must be an object')];
  }
  for (const key of [
    'generated_localization_active',
    'registry_paths_active',
    'temporary_wrappers_removed',
    'durable_upgrade_compatibility_preserved',
  ]) {
    if (summary[key] !== true) {
      issues.push(createIssue('migration-cleanup-summary', evidencePath, key, 'must be true'));
    }
  }
  return issues;
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

async function readSource(rootDir, relativePath) {
  try {
    return {
      ok: true,
      value: await readFile(join(rootDir, relativePath), 'utf8'),
    };
  } catch (error) {
    return {
      ok: false,
      message: error?.code === 'ENOENT' ? 'source file is missing' : error.message,
    };
  }
}

async function readSourceTree(rootDir, relativePath) {
  const absolutePath = join(rootDir, relativePath);
  try {
    const fileStat = await stat(absolutePath);
    if (fileStat.isFile()) {
      return {
        ok: true,
        value: await readFile(absolutePath, 'utf8'),
      };
    }
    if (!fileStat.isDirectory()) {
      return {
        ok: false,
        message: 'source path is neither a file nor a directory',
      };
    }
    const files = await collectFiles(absolutePath);
    const chunks = [];
    for (const file of files) {
      chunks.push(await readFile(file, 'utf8'));
    }
    return {
      ok: true,
      value: chunks.join('\n'),
    };
  } catch (error) {
    return {
      ok: false,
      message: error?.code === 'ENOENT' ? 'source path is missing' : error.message,
    };
  }
}

async function collectFiles(absolutePath) {
  const entries = await readdir(absolutePath, { withFileTypes: true });
  const files = [];
  for (const entry of entries) {
    const childPath = join(absolutePath, entry.name);
    if (entry.isDirectory()) {
      files.push(...(await collectFiles(childPath)));
    } else if (entry.isFile()) {
      files.push(childPath);
    }
  }
  return files;
}

function compareList(evidencePath, key, actual, expected) {
  const normalizedExpected = [...expected].sort();
  if (actual.length === normalizedExpected.length && actual.every((value, index) => value === normalizedExpected[index])) {
    return [];
  }
  return [createIssue('migration-cleanup-wrapper-set', evidencePath, key, `expected ${formatList(normalizedExpected)}, got ${formatList(actual)}`)];
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
  const report = await buildMigrationCleanupReport();
  process.stdout.write(renderMigrationCleanupReport(report));
  if (!report.ok) {
    process.exitCode = 1;
  }
}
