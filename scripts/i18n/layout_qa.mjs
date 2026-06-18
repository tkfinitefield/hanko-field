import { existsSync } from 'node:fs';
import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const DEFAULT_EVIDENCE_PATH = 'doc/qa/m9-t04/layout-qa.json';
const WEB_TEMPLATE_ROOT = 'web/templates';
const WEB_MAIN_PATH = 'web/src/main.rs';
const ANDROID_MANIFEST_PATH = 'app/android/app/src/main/AndroidManifest.xml';

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} LayoutQaIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} locale
 * @property {string} message
 *
 * @typedef {Object} LayoutQaReport
 * @property {boolean} ok
 * @property {LayoutQaIssue[]} issues
 * @property {{ tier1_full: string[], tier2_screenshot: string[], tier3_mechanical: string[] }} expected_tiers
 * @property {string} evidence_file
 * @property {number} parsed_web_templates
 */

/**
 * @param {{ rootDir?: string, evidencePath?: string }} options
 * @returns {Promise<LayoutQaReport>}
 */
export async function buildLayoutQa({ rootDir = REPO_ROOT, evidencePath = DEFAULT_EVIDENCE_PATH } = {}) {
  const registry = await loadRegistry(rootDir);
  const expectedTiers = expectedLayoutQaTiers(registry.languages);
  const issues = [];
  const evidenceResult = await readEvidence(rootDir, evidencePath);
  const parsedWebTemplates = [];

  if (!evidenceResult.ok) {
    issues.push(createIssue('layout-qa-evidence-missing', evidencePath, null, evidenceResult.message));
  } else {
    issues.push(...validateEvidence({ rootDir, evidencePath, evidence: evidenceResult.value, expectedTiers }));
  }

  const templates = await collectWebTemplates(rootDir);
  for (const templatePath of templates) {
    const source = await readFile(join(rootDir, templatePath), 'utf8');
    if (!source.includes('<!doctype html>')) {
      continue;
    }
    parsedWebTemplates.push(templatePath);
    if (!/<html[^>]*lang="\{\{\s*selected_locale\s*\}\}"[^>]*dir="\{\{\s*self\.html_dir\(\)\s*\}\}"/.test(source)) {
      issues.push(
        createIssue(
          'layout-qa-web-dir-missing',
          templatePath,
          null,
          'top-level html tag must bind lang and dir to selected locale helpers',
        ),
      );
    }
  }

  issues.push(...(await validateRtlRuntimeSupport(rootDir, registry.languages)));

  return {
    ok: issues.length === 0,
    issues,
    expected_tiers: expectedTiers,
    evidence_file: evidencePath,
    parsed_web_templates: parsedWebTemplates.length,
  };
}

/**
 * @param {LanguageEntry[]} languages
 */
export function expectedLayoutQaTiers(languages) {
  const tier1 = [];
  const tier2 = [];
  const tier3 = [];

  for (const language of languages) {
    if (language.route_code === 'en') {
      continue;
    }
    if (language.web.indexed || language.release.enabled) {
      tier1.push(language.route_code);
    } else if (language.app.enabled || language.app.selectable || language.web.enabled) {
      tier2.push(language.route_code);
    } else {
      tier3.push(language.route_code);
    }
  }

  return {
    tier1_full: tier1.sort(),
    tier2_screenshot: tier2.sort(),
    tier3_mechanical: tier3.sort(),
  };
}

/**
 * @param {LayoutQaReport} report
 * @returns {string}
 */
export function renderLayoutQa(report) {
  const lines = [
    '# Stone Signature i18n layout QA',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Evidence file: ${report.evidence_file}`,
    `Parsed web templates: ${report.parsed_web_templates}`,
    '',
    '## Expected Tiers',
    '',
    `- Tier 1 full QA: ${formatList(report.expected_tiers.tier1_full)}`,
    `- Tier 2 screenshot QA: ${formatList(report.expected_tiers.tier2_screenshot)}`,
    `- Tier 3 mechanical QA: ${formatList(report.expected_tiers.tier3_mechanical)}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      const locale = issue.locale ? ` ${issue.locale}` : '';
      lines.push(`- ${issue.code}: ${issue.file}${locale} - ${issue.message}`);
    }
  } else {
    lines.push('No layout QA issues.');
  }

  return `${lines.join('\n').trimEnd()}\n`;
}

function validateEvidence({ rootDir, evidencePath, evidence, expectedTiers }) {
  const issues = [];
  if (!isRecord(evidence)) {
    return [createIssue('layout-qa-evidence-format', evidencePath, null, 'top-level value must be an object')];
  }
  if (evidence.format_version !== 1) {
    issues.push(createIssue('layout-qa-evidence-format', evidencePath, null, 'format_version must be 1'));
  }
  if (evidence.task !== 'M9-T04') {
    issues.push(createIssue('layout-qa-evidence-format', evidencePath, null, 'task must be M9-T04'));
  }
  if (!isRecord(evidence.tiers)) {
    issues.push(createIssue('layout-qa-evidence-format', evidencePath, null, 'tiers must be an object'));
    return issues;
  }

  const tier1Entries = readEntries(evidence.tiers.tier1_full);
  const tier2Entries = readEntries(evidence.tiers.tier2_screenshot);
  const tier3Entries = readEntries(evidence.tiers.tier3_mechanical);
  issues.push(...compareLocaleSet(evidencePath, 'tier1_full', entryLocales(tier1Entries), expectedTiers.tier1_full));
  issues.push(...compareLocaleSet(evidencePath, 'tier2_screenshot', entryLocales(tier2Entries), expectedTiers.tier2_screenshot));
  issues.push(...compareLocaleSet(evidencePath, 'tier3_mechanical', groupedLocales(tier3Entries), expectedTiers.tier3_mechanical));

  for (const entry of tier1Entries) {
    issues.push(...validateLocaleEntry(rootDir, evidencePath, entry, 'tier1_full', ['i18n_check', 'app_layout', 'web_layout']));
  }
  for (const entry of tier2Entries) {
    issues.push(...validateLocaleEntry(rootDir, evidencePath, entry, 'tier2_screenshot', ['i18n_check', 'app_layout', 'screenshot_qa']));
  }
  for (const entry of tier3Entries) {
    issues.push(...validateGroupEntry(rootDir, evidencePath, entry, ['mechanical_check', 'stub_check']));
  }

  return issues;
}

function validateLocaleEntry(rootDir, evidencePath, entry, tierName, requiredKinds) {
  const issues = [];
  if (!isRecord(entry)) {
    return [createIssue('layout-qa-evidence-format', evidencePath, null, `${tierName} entries must be objects`)];
  }
  if (entry.status !== 'pass') {
    issues.push(createIssue('layout-qa-status', evidencePath, entry.locale ?? null, `${tierName} status must be pass`));
  }
  if (typeof entry.locale !== 'string' || entry.locale.trim() === '') {
    issues.push(createIssue('layout-qa-evidence-format', evidencePath, null, `${tierName} entry locale is required`));
  }
  issues.push(...validateEvidenceItems(rootDir, evidencePath, entry.locale ?? null, entry.evidence, requiredKinds));
  return issues;
}

function validateGroupEntry(rootDir, evidencePath, entry, requiredKinds) {
  const issues = [];
  if (!isRecord(entry)) {
    return [createIssue('layout-qa-evidence-format', evidencePath, null, 'tier3 entries must be objects')];
  }
  if (entry.status !== 'pass') {
    issues.push(createIssue('layout-qa-status', evidencePath, null, 'tier3 status must be pass'));
  }
  if (!Array.isArray(entry.locales) || entry.locales.some((locale) => typeof locale !== 'string')) {
    issues.push(createIssue('layout-qa-evidence-format', evidencePath, null, 'tier3 locales must be a string array'));
  }
  issues.push(...validateEvidenceItems(rootDir, evidencePath, null, entry.evidence, requiredKinds));
  return issues;
}

function validateEvidenceItems(rootDir, evidencePath, locale, evidenceItems, requiredKinds) {
  const issues = [];
  if (!Array.isArray(evidenceItems)) {
    return [createIssue('layout-qa-evidence-format', evidencePath, locale, 'evidence must be an array')];
  }
  const seenKinds = new Set();
  for (const [index, item] of evidenceItems.entries()) {
    if (!isRecord(item)) {
      issues.push(createIssue('layout-qa-evidence-format', evidencePath, locale, `evidence[${index}] must be an object`));
      continue;
    }
    if (typeof item.kind !== 'string' || item.kind.trim() === '') {
      issues.push(createIssue('layout-qa-evidence-format', evidencePath, locale, `evidence[${index}].kind is required`));
    } else {
      seenKinds.add(item.kind);
    }
    if (typeof item.command !== 'string' || item.command.trim() === '') {
      issues.push(createIssue('layout-qa-evidence-format', evidencePath, locale, `evidence[${index}].command is required`));
    }
    if (item.path !== undefined && (typeof item.path !== 'string' || item.path.trim() === '')) {
      issues.push(createIssue('layout-qa-evidence-format', evidencePath, locale, `evidence[${index}].path must be a non-empty string`));
    }
    if (typeof item.path === 'string' && !fileExistsSyncish(rootDir, item.path)) {
      issues.push(createIssue('layout-qa-evidence-path', evidencePath, locale, `${item.path} does not exist`));
    }
  }

  for (const requiredKind of requiredKinds) {
    if (!seenKinds.has(requiredKind)) {
      issues.push(createIssue('layout-qa-evidence-missing-kind', evidencePath, locale, `missing evidence kind ${requiredKind}`));
    }
  }
  return issues;
}

async function validateRtlRuntimeSupport(rootDir, languages) {
  const issues = [];
  const rtlCodes = languages
    .filter((language) => language.text_direction === 'rtl')
    .map((language) => language.route_code)
    .sort();
  const webMain = await readFile(join(rootDir, WEB_MAIN_PATH), 'utf8');
  for (const routeCode of rtlCodes) {
    if (!webMain.includes(`"${routeCode}"`)) {
      issues.push(createIssue('layout-qa-web-rtl-mapping', WEB_MAIN_PATH, routeCode, 'html_dir_for_locale must map RTL route code'));
    }
  }

  const manifest = await readFile(join(rootDir, ANDROID_MANIFEST_PATH), 'utf8');
  if (!manifest.includes('layoutDirection')) {
    issues.push(createIssue('layout-qa-android-layout-direction', ANDROID_MANIFEST_PATH, null, 'Android activity configChanges must include layoutDirection'));
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

async function collectWebTemplates(rootDir) {
  const results = [];
  await visit(WEB_TEMPLATE_ROOT);
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
      } else if (entry.isFile() && entry.name.endsWith('.html')) {
        results.push(relativePath);
      }
    }
  }
}

function readEntries(value) {
  return Array.isArray(value) ? value : [];
}

function entryLocales(entries) {
  return entries
    .map((entry) => (isRecord(entry) && typeof entry.locale === 'string' ? entry.locale : null))
    .filter(Boolean)
    .sort();
}

function groupedLocales(entries) {
  return entries
    .flatMap((entry) => (isRecord(entry) && Array.isArray(entry.locales) ? entry.locales : []))
    .filter((locale) => typeof locale === 'string')
    .sort();
}

function compareLocaleSet(evidencePath, tierName, actual, expected) {
  if (sameList(actual, expected)) {
    return [];
  }
  return [
    createIssue(
      'layout-qa-tier-mismatch',
      evidencePath,
      null,
      `${tierName} expected ${formatList(expected)} but found ${formatList(actual)}`,
    ),
  ];
}

function fileExistsSyncish(rootDir, relativePath) {
  return existsSync(join(rootDir, relativePath));
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
    const report = await buildLayoutQa({
      evidencePath: process.env.EVIDENCE ?? DEFAULT_EVIDENCE_PATH,
    });
    process.stdout.write(renderLayoutQa(report));
    if (!report.ok || process.argv.includes('--check')) {
      process.exitCode = report.ok ? 0 : 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
