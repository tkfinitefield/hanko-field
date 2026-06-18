import { readdir, readFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

import { validateArbFiles } from './arb.mjs';
import { validateIntentions } from './intentions.mjs';
import { validateJsonShapes } from './json_shape.mjs';
import { buildI18nStatus, renderI18nStatus } from './status.mjs';
import { buildI18nTodo, parseLangsFilter, renderI18nTodo } from './todo.mjs';

const REPO_ROOT = fileURLToPath(new URL('../..', import.meta.url));

const VALIDATED_CONTENT_ROOTS = [
  'config',
  'app/lib/l10n',
  'app/assets/i18n',
  'web/content/i18n',
  'api/content/i18n',
  'release/store_metadata/source',
];

/**
 * @typedef {Object} I18nCheckIssue
 * @property {'missing-file' | 'missing-key' | 'malformed-json' | 'validation-error' | 'arb' | 'json-shape' | 'intention'} type
 * @property {string} file
 * @property {string} message
 *
 * @typedef {Object} I18nCheckReport
 * @property {boolean} ok
 * @property {I18nCheckIssue[]} issues
 * @property {import('./status.mjs').I18nStatus} status
 * @property {import('./todo.mjs').I18nTodoReport} todo
 * @property {string[]} parsed_files
 */

/**
 * @param {{ rootDir?: string, langs?: string[] | null, file?: string | null }} options
 * @returns {Promise<I18nCheckReport>}
 */
export async function buildI18nCheck({ rootDir = REPO_ROOT, langs = null, file = null } = {}) {
  const [statusResult, todoResult, parsedFiles, arbResult, jsonShapeResult, intentionResult] = await Promise.all([
    settleCheckStep('status', () => buildI18nStatus({ rootDir })),
    settleCheckStep('todo', () => buildI18nTodo({ rootDir, langs, file })),
    validateJsonContentFiles({ rootDir, file }),
    settleCheckStep('arb', () => validateArbFiles({ rootDir, langs, file })),
    settleCheckStep('json-shape', () => validateJsonShapes({ rootDir, langs, file })),
    settleCheckStep('intention', () => validateIntentions({ rootDir, langs, file })),
  ]);
  const status = statusResult.value ?? emptyStatus();
  const todo = todoResult.value ?? emptyTodo(langs, file);
  const arb = arbResult.value ?? emptyArb();
  const jsonShape = jsonShapeResult.value ?? emptyJsonShape();
  const intention = intentionResult.value ?? emptyIntention();

  const issues = [
    ...stepIssues(statusResult),
    ...stepIssues(todoResult),
    ...stepIssues(arbResult),
    ...stepIssues(jsonShapeResult),
    ...stepIssues(intentionResult),
    ...statusMissingFileIssues(status),
    ...todoMissingIssues(todo),
    ...parsedFiles.flatMap((entry) =>
      entry.ok
        ? []
        : [
            {
              type: 'malformed-json',
              file: entry.path,
              message: entry.message,
            },
          ],
    ),
    ...arb.issues.map((issue) => ({
      type: 'arb',
      file: issue.file,
      message: issue.key
        ? `${issue.key}: ${issue.code}: ${issue.message}`
        : `${issue.code}: ${issue.message}`,
    })),
    ...jsonShape.issues.map((issue) => ({
      type: 'json-shape',
      file: issue.file,
      message: issue.key
        ? `${issue.key}: ${issue.code}: ${issue.message}`
        : `${issue.code}: ${issue.message}`,
    })),
    ...intention.issues.map((issue) => ({
      type: 'intention',
      file: issue.file,
      message: issue.key
        ? `${issue.key}: ${issue.code}: ${issue.message}`
        : `${issue.code}: ${issue.message}`,
    })),
  ];

  const parsedFileSet = new Set([
    ...parsedFiles.filter((entry) => entry.ok).map((entry) => entry.path),
    ...arb.parsed_files,
    ...jsonShape.parsed_files,
    ...intention.parsed_files,
  ]);

  return {
    ok: issues.length === 0,
    issues,
    status,
    todo,
    parsed_files: [...parsedFileSet].sort(),
  };
}

/**
 * @param {I18nCheckReport} report
 * @returns {string}
 */
export function renderI18nCheck(report) {
  const lines = [
    '# Stone Signature i18n check',
    '',
    `Result: ${report.ok ? 'pass' : 'fail'}`,
    `Issues: ${report.issues.length}`,
    `Parsed files: ${report.parsed_files.length}`,
    '',
  ];

  if (report.issues.length > 0) {
    lines.push('## Issues', '');
    for (const issue of report.issues) {
      lines.push(`- ${issue.type}: ${issue.file} - ${issue.message}`);
    }
    lines.push('');
  }

  lines.push('## Status', '', renderI18nStatus(report.status).trimEnd(), '');
  lines.push('## Todo', '', renderI18nTodo(report.todo).trimEnd(), '');

  return `${lines.join('\n').trimEnd()}\n`;
}

/**
 * @param {{ rootDir: string, file?: string | null }} options
 * @returns {Promise<Array<{ ok: true, path: string } | { ok: false, path: string, message: string }>>}
 */
export async function validateJsonContentFiles({ rootDir, file = null }) {
  const fileFilter = normalizeFilterPath(file);
  const paths = fileFilter
    ? [fileFilter]
    : await collectJsonLikeFiles(rootDir, VALIDATED_CONTENT_ROOTS);

  const results = [];
  for (const relativePath of paths) {
    if (!isJsonLikePath(relativePath)) {
      continue;
    }
    try {
      const rawText = await readFile(join(rootDir, relativePath), 'utf8');
      JSON.parse(rawText);
      results.push({ ok: true, path: relativePath });
    } catch (error) {
      if (error?.code === 'ENOENT') {
        continue;
      }
      results.push({
        ok: false,
        path: relativePath,
        message: error.message,
      });
    }
  }
  return results;
}

function statusMissingFileIssues(status) {
  return status.sections.flatMap((section) =>
    section.items
      .filter((item) => item.status === 'missing')
      .map((item) => ({
        type: 'missing-file',
        file: item.path,
        message: `${section.name}: ${item.label}`,
      })),
  );
}

function todoMissingIssues(todo) {
  return todo.items.map((item) => ({
    type: item.type,
    file: item.file,
    message: `${item.locale} ${item.key}`,
  }));
}

async function settleCheckStep(label, fn) {
  try {
    return { ok: true, value: await fn() };
  } catch (error) {
    return {
      ok: false,
      label,
      message: error instanceof Error ? error.message : String(error),
    };
  }
}

function stepIssues(result) {
  if (result.ok) {
    return [];
  }
  return [
    {
      type: result.message.includes('Invalid JSON') ? 'malformed-json' : 'validation-error',
      file: result.label,
      message: result.message,
    },
  ];
}

function emptyStatus() {
  return {
    total_languages: 0,
    app_enabled: [],
    web_enabled: [],
    release_enabled: [],
    sections: [],
  };
}

function emptyTodo(langs, file) {
  return {
    items: [],
    languages: langs ?? [],
    file_filter: normalizeFilterPath(file),
  };
}

function emptyArb() {
  return {
    ok: true,
    issues: [],
    parsed_files: [],
  };
}

function emptyJsonShape() {
  return {
    ok: true,
    issues: [],
    parsed_files: [],
  };
}

function emptyIntention() {
  return {
    ok: true,
    issues: [],
    parsed_files: [],
  };
}

async function collectJsonLikeFiles(rootDir, relativeRoots) {
  const results = [];

  for (const relativeRoot of relativeRoots) {
    await visit(relativeRoot);
  }

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
      } else if (entry.isFile() && isJsonLikePath(relativePath)) {
        results.push(relativePath);
      }
    }
  }

  return results.sort();
}

function isJsonLikePath(path) {
  return path.endsWith('.json') || path.endsWith('.arb');
}

function normalizeFilterPath(file) {
  if (!file || !file.trim()) {
    return null;
  }
  return file.trim().replace(/^\.\//, '');
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    const report = await buildI18nCheck({
      langs: parseLangsFilter(process.env.LANGS),
      file: process.env.FILE ?? null,
    });
    process.stdout.write(renderI18nCheck(report));
    if (!report.ok) {
      process.exitCode = 1;
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
