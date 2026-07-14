import { readFile, stat } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { loadLanguageRegistry } from './registry.mjs';
import { parseLangsFilter } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));
const ARB_ROOT = 'app/lib/l10n';
const GENERATED_LOCALIZATIONS_PATH = 'app/lib/l10n/generated/generated_hanko_localizations.dart';
const IDENTIFIER_PATTERN = /^[A-Za-z_][A-Za-z0-9_]*$/;
const PLURAL_SELECTORS = new Set(['zero', 'one', 'two', 'few', 'many', 'other']);

/**
 * @typedef {import('./registry.mjs').LanguageEntry} LanguageEntry
 *
 * @typedef {Object} ArbValidationIssue
 * @property {string} code
 * @property {string} file
 * @property {string | null} key
 * @property {string} message
 *
 * @typedef {Object} ArbValidationReport
 * @property {boolean} ok
 * @property {ArbValidationIssue[]} issues
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 * @returns {Promise<ArbValidationReport>}
 */
export async function validateArbFiles({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const fileFilter = normalizeFilterPath(file);
  if (fileFilter && !fileFilter.endsWith('.arb')) {
    return {
      ok: true,
      issues: [],
      parsed_files: [],
    };
  }

  const rootUrl = pathToFileURL(rootDir.endsWith('/') ? rootDir : `${rootDir}/`);
  const registry = await loadLanguageRegistry(new URL('config/languages.json', rootUrl));
  const selectedLanguages = selectArbLanguages(registry.languages, langs, fileFilter);
  const expectedByPath = new Map(
    registry.languages.map((language) => [expectedArbPath(language), language]),
  );
  const paths = fileFilter && fileFilter.endsWith('.arb')
    ? [fileFilter]
    : selectedLanguages.map((language) => expectedArbPath(language));
  const baseLanguage = registry.byRouteCode.get('en');
  const basePath = baseLanguage ? expectedArbPath(baseLanguage) : `${ARB_ROOT}/app_en.arb`;
  const parsed = new Map();
  const issues = [];

  if (!baseLanguage) {
    issues.push(createIssue('arb-locale-mapping', 'config/languages.json', null, 'missing en route_code'));
  }

  const requiredPaths = new Set(paths);
  if ([...requiredPaths].some((path) => path !== basePath)) {
    requiredPaths.add(basePath);
  }

  for (const path of requiredPaths) {
    const language = expectedByPath.get(path) ?? null;
    const expectedLocale = language ? expectedArbLocale(language) : null;
    const readResult = await readArb(rootDir, path);
    if (!readResult.ok) {
      issues.push(createIssue(readResult.code, path, null, readResult.message));
      continue;
    }
    parsed.set(path, readResult.value);
    issues.push(...validateArbDocument(path, readResult.value, { expectedLocale, requireMetadata: path === basePath }));
    if (!language && fileFilter === path) {
      issues.push(createIssue('arb-locale-mapping', path, null, 'ARB file does not match any registry Flutter locale mapping'));
    }
  }

  const baseArb = parsed.get(basePath);
  if (baseArb) {
    for (const path of paths) {
      if (path === basePath) {
        continue;
      }
      const targetArb = parsed.get(path);
      if (!targetArb) {
        continue;
      }
      issues.push(...validatePlaceholderParity(basePath, baseArb, path, targetArb));
    }
  }

  const generatedLocaleReport = await validateGeneratedLocaleMapping(rootDir, selectedLanguages);
  issues.push(...generatedLocaleReport.issues);

  return {
    ok: issues.length === 0,
    issues,
    parsed_files: [...new Set([...parsed.keys(), ...generatedLocaleReport.parsed_files])].sort(),
  };
}

/**
 * @param {LanguageEntry} language
 * @returns {string}
 */
export function expectedArbPath(language) {
  return `${ARB_ROOT}/app_${flutterArbSuffix(language)}.arb`;
}

/**
 * @param {LanguageEntry} language
 * @returns {string}
 */
export function expectedArbLocale(language) {
  return flutterArbSuffix(language);
}

/**
 * @param {LanguageEntry} language
 * @returns {string}
 */
export function flutterArbSuffix(language) {
  const { languageCode, scriptCode, countryCode } = language.flutter;
  if (language.route_code === 'zh') {
    return 'zh';
  }

  return [languageCode, scriptCode, countryCode].filter(Boolean).join('_');
}

/**
 * @param {string} message
 * @returns {{ placeholders: Set<string>, issues: string[] }}
 */
export function analyzeArbMessage(message) {
  return analyzeSegment(message);
}

/**
 * @param {string} path
 * @param {Record<string, unknown>} arb
 * @param {{ expectedLocale: string | null, requireMetadata: boolean }} options
 * @returns {ArbValidationIssue[]}
 */
export function validateArbDocument(path, arb, { expectedLocale, requireMetadata }) {
  const issues = [];
  if (!isRecord(arb)) {
    return [createIssue('arb-format', path, null, 'ARB top-level value must be an object')];
  }

  if (expectedLocale && arb['@@locale'] !== expectedLocale) {
    issues.push(
      createIssue(
        'arb-locale-mapping',
        path,
        '@@locale',
        `expected @@locale "${expectedLocale}", found ${formatValue(arb['@@locale'])}`,
      ),
    );
  }

  for (const [key, value] of Object.entries(arb)) {
    if (key.startsWith('@')) {
      continue;
    }
    if (typeof value !== 'string') {
      issues.push(createIssue('arb-format', path, key, 'message value must be a string'));
      continue;
    }

    const analysis = analyzeArbMessage(value);
    for (const message of analysis.issues) {
      issues.push(createIssue('arb-icu', path, key, message));
    }

    const metadataKey = `@${key}`;
    const metadata = arb[metadataKey];
    const metadataPlaceholders = readMetadataPlaceholders(metadata);
    if (metadataPlaceholders.error) {
      issues.push(createIssue('arb-metadata', path, key, metadataPlaceholders.error));
      continue;
    }

    const valuePlaceholders = sortedSet(analysis.placeholders);
    const metadataNames = metadataPlaceholders.names;
    if (requireMetadata && valuePlaceholders.length > 0 && !isRecord(metadata)) {
      issues.push(createIssue('arb-metadata', path, key, 'message with placeholders must define metadata'));
      continue;
    }
    if (requireMetadata && valuePlaceholders.length > 0 && metadataNames.length === 0) {
      issues.push(createIssue('arb-metadata', path, key, 'message placeholders must be listed in metadata.placeholders'));
    }
    if (metadataNames.length > 0 && !sameList(valuePlaceholders, metadataNames)) {
      issues.push(
        createIssue(
          'arb-metadata',
          path,
          key,
          `metadata placeholders ${formatList(metadataNames)} do not match message placeholders ${formatList(valuePlaceholders)}`,
        ),
      );
    }
  }

  return issues;
}

/**
 * @param {string} basePath
 * @param {Record<string, unknown>} baseArb
 * @param {string} targetPath
 * @param {Record<string, unknown>} targetArb
 * @returns {ArbValidationIssue[]}
 */
export function validatePlaceholderParity(basePath, baseArb, targetPath, targetArb) {
  const issues = [];
  for (const [key, baseValue] of Object.entries(baseArb)) {
    if (key.startsWith('@') || typeof baseValue !== 'string') {
      continue;
    }
    const targetValue = targetArb[key];
    if (typeof targetValue !== 'string') {
      continue;
    }

    const basePlaceholders = sortedSet(analyzeArbMessage(baseValue).placeholders);
    const targetPlaceholders = sortedSet(analyzeArbMessage(targetValue).placeholders);
    if (!sameList(basePlaceholders, targetPlaceholders)) {
      issues.push(
        createIssue(
          'arb-placeholder-mismatch',
          targetPath,
          key,
          `placeholders ${formatList(targetPlaceholders)} do not match ${basePath} placeholders ${formatList(basePlaceholders)}`,
        ),
      );
    }
  }
  return issues;
}

/**
 * @param {string} rootDir
 * @param {LanguageEntry[]} languages
 * @returns {Promise<{ issues: ArbValidationIssue[], parsed_files: string[] }>}
 */
export async function validateGeneratedLocaleMapping(rootDir, languages) {
  let rawText;
  try {
    rawText = await readFile(join(rootDir, GENERATED_LOCALIZATIONS_PATH), 'utf8');
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return { issues: [], parsed_files: [] };
    }
    throw error;
  }

  const supportedLocales = parseGeneratedSupportedLocales(rawText);
  const issues = [];
  for (const language of languages) {
    const expectedLocale = expectedArbLocale(language);
    if (!supportedLocales.has(expectedLocale)) {
      issues.push(
        createIssue(
          'arb-generated-locale-mapping',
          GENERATED_LOCALIZATIONS_PATH,
          language.route_code,
          `generated supportedLocales is missing ${expectedLocale}`,
        ),
      );
    }
  }

  return {
    issues,
    parsed_files: [GENERATED_LOCALIZATIONS_PATH],
  };
}

/**
 * @param {string} source
 * @returns {Set<string>}
 */
export function parseGeneratedSupportedLocales(source) {
  const supportedBlock = source.match(/supportedLocales\s*=\s*<Locale>\s*\[([\s\S]*?)\];/);
  if (!supportedBlock) {
    return new Set();
  }

  const body = supportedBlock[1];
  const locales = new Set();
  for (const match of body.matchAll(/Locale\('([^']+)'(?:,\s*'([^']+)')?\)/g)) {
    locales.add([match[1], match[2]].filter(Boolean).join('_'));
  }
  for (const match of body.matchAll(/Locale\.fromSubtags\(([^)]*)\)/g)) {
    const args = match[1];
    const languageCode = namedStringArgument(args, 'languageCode');
    if (!languageCode) {
      continue;
    }
    locales.add(
      [
        languageCode,
        namedStringArgument(args, 'scriptCode'),
        namedStringArgument(args, 'countryCode'),
      ]
        .filter(Boolean)
        .join('_'),
    );
  }

  return locales;
}

/**
 * @param {LanguageEntry[]} languages
 * @param {string[] | null} langs
 * @param {string | null} fileFilter
 * @returns {LanguageEntry[]}
 */
function selectArbLanguages(languages, langs, fileFilter) {
  const byRouteCode = new Map(languages.map((language) => [language.route_code, language]));
  const byArbPath = new Map(languages.map((language) => [expectedArbPath(language), language]));
  if (fileFilter?.endsWith('.arb')) {
    const language = byArbPath.get(fileFilter);
    return language ? [language] : [];
  }
  if (!langs) {
    return languages.filter((language) => language.app.enabled);
  }
  if (langs.includes('all')) {
    return languages.filter((language) => language.route_code !== 'en' || language.app.enabled);
  }

  const unknownCodes = langs.filter((code) => !byRouteCode.has(code));
  if (unknownCodes.length > 0) {
    throw new Error(`Unknown LANGS route code(s): ${unknownCodes.join(', ')}`);
  }
  return langs.map((code) => byRouteCode.get(code)).filter(Boolean);
}

async function readArb(rootDir, relativePath) {
  try {
    const stats = await stat(join(rootDir, relativePath));
    if (!stats.isFile()) {
      return { ok: false, code: 'arb-missing-file', message: 'not a file' };
    }
  } catch (error) {
    if (error?.code === 'ENOENT') {
      return { ok: false, code: 'arb-missing-file', message: 'file is missing' };
    }
    throw error;
  }

  try {
    return { ok: true, value: JSON.parse(await readFile(join(rootDir, relativePath), 'utf8')) };
  } catch (error) {
    return { ok: false, code: 'arb-malformed-json', message: error.message };
  }
}

function analyzeSegment(segment) {
  const placeholders = new Set();
  const issues = [];
  let index = 0;

  while (index < segment.length) {
    const char = segment[index];
    if (char === '}') {
      issues.push('unmatched closing brace');
      index += 1;
      continue;
    }
    if (char !== '{') {
      index += 1;
      continue;
    }

    const close = findMatchingBrace(segment, index);
    if (close === -1) {
      issues.push('unmatched opening brace');
      break;
    }

    const expression = segment.slice(index + 1, close).trim();
    const parts = splitFirstTopLevelCommas(expression, 2);
    if (parts.length === 1) {
      if (IDENTIFIER_PATTERN.test(parts[0].trim())) {
        placeholders.add(parts[0].trim());
      } else if (parts[0].trim()) {
        issues.push(`invalid placeholder expression "{${expression}}"`);
      }
      index = close + 1;
      continue;
    }

    const name = parts[0].trim();
    const type = parts[1].trim();
    const body = parts[2]?.trim() ?? '';
    if (!IDENTIFIER_PATTERN.test(name)) {
      issues.push(`invalid ICU argument name "${name}"`);
    } else {
      placeholders.add(name);
    }

    if (type !== 'plural' && type !== 'select') {
      issues.push(`unsupported ICU type "${type}"`);
      index = close + 1;
      continue;
    }

    const optionResult = analyzeIcuOptions(type, body);
    for (const placeholder of optionResult.placeholders) {
      placeholders.add(placeholder);
    }
    issues.push(...optionResult.issues);
    index = close + 1;
  }

  return { placeholders, issues };
}

function analyzeIcuOptions(type, body) {
  const placeholders = new Set();
  const issues = [];
  let index = 0;
  let optionCount = 0;
  let hasOther = false;

  while (index < body.length) {
    while (index < body.length && /\s/.test(body[index])) {
      index += 1;
    }
    if (index >= body.length) {
      break;
    }

    const selectorStart = index;
    while (index < body.length && body[index] !== '{' && !/\s/.test(body[index])) {
      index += 1;
    }
    const selector = body.slice(selectorStart, index).trim();
    while (index < body.length && /\s/.test(body[index])) {
      index += 1;
    }

    if (!selector) {
      issues.push(`${type} option is missing a selector`);
      break;
    }
    if (!isValidSelector(type, selector)) {
      issues.push(`invalid ${type} selector "${selector}"`);
    }
    if (selector === 'other') {
      hasOther = true;
    }
    if (body[index] !== '{') {
      issues.push(`${type} selector "${selector}" is missing a message block`);
      break;
    }

    const close = findMatchingBrace(body, index);
    if (close === -1) {
      issues.push(`${type} selector "${selector}" has an unmatched opening brace`);
      break;
    }

    const branch = analyzeSegment(body.slice(index + 1, close));
    for (const placeholder of branch.placeholders) {
      placeholders.add(placeholder);
    }
    issues.push(...branch.issues);
    optionCount += 1;
    index = close + 1;
  }

  if (optionCount === 0) {
    issues.push(`${type} ICU block must define at least one option`);
  }
  if (!hasOther) {
    issues.push(`${type} ICU block must define an other option`);
  }

  return { placeholders, issues };
}

function findMatchingBrace(text, openIndex) {
  let depth = 0;
  for (let index = openIndex; index < text.length; index += 1) {
    if (text[index] === '{') {
      depth += 1;
    } else if (text[index] === '}') {
      depth -= 1;
      if (depth === 0) {
        return index;
      }
    }
  }
  return -1;
}

function splitFirstTopLevelCommas(text, maxCommas) {
  const parts = [];
  let depth = 0;
  let start = 0;
  let commas = 0;

  for (let index = 0; index < text.length; index += 1) {
    if (text[index] === '{') {
      depth += 1;
    } else if (text[index] === '}') {
      depth -= 1;
    } else if (text[index] === ',' && depth === 0 && commas < maxCommas) {
      parts.push(text.slice(start, index));
      start = index + 1;
      commas += 1;
    }
  }

  parts.push(text.slice(start));
  return parts;
}

function isValidSelector(type, selector) {
  if (type === 'plural') {
    return PLURAL_SELECTORS.has(selector) || /^=\d+$/.test(selector);
  }
  return IDENTIFIER_PATTERN.test(selector);
}

function namedStringArgument(args, name) {
  return args.match(new RegExp(`${name}:\\s*'([^']+)'`))?.[1] ?? null;
}

function readMetadataPlaceholders(metadata) {
  if (!isRecord(metadata)) {
    return { names: [], error: null };
  }
  if (metadata.placeholders === undefined) {
    return { names: [], error: null };
  }
  if (!isRecord(metadata.placeholders)) {
    return { names: [], error: 'metadata.placeholders must be an object' };
  }

  const names = Object.keys(metadata.placeholders).sort();
  const invalid = names.filter((name) => !IDENTIFIER_PATTERN.test(name));
  if (invalid.length > 0) {
    return { names, error: `metadata placeholder names are invalid: ${invalid.join(', ')}` };
  }
  return { names, error: null };
}

function createIssue(code, file, key, message) {
  return { code, file, key, message };
}

function normalizeFilterPath(file) {
  if (!file || !file.trim()) {
    return null;
  }
  return file.trim().replace(/^\.\//, '');
}

function isRecord(value) {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function sortedSet(values) {
  return [...values].sort();
}

function sameList(left, right) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}

function formatList(values) {
  return values.length === 0 ? '(none)' : values.join(', ');
}

function formatValue(value) {
  return typeof value === 'string' ? `"${value}"` : String(value);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await validateArbFiles({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    if (report.issues.length === 0) {
      process.stdout.write(`ARB validation passed (${report.parsed_files.length} files).\n`);
    } else {
      process.stdout.write(`ARB validation failed (${report.issues.length} issues).\n`);
      for (const issue of report.issues) {
        const key = issue.key ? ` ${issue.key}` : '';
        process.stdout.write(`- ${issue.code}: ${issue.file}${key} - ${issue.message}\n`);
      }
      process.exitCode = 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
