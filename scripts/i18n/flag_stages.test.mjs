import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  buildFlagStageReport,
  classifyLanguageStages,
  renderFlagStageReport,
  stageForLanguage,
} from './flag_stages.mjs';

test('classifies staged locale flags from registry state', () => {
  const languages = [
    createLanguage('en', { fallback: null, web: { enabled: true, indexed: true, url_prefix: '' }, app: { enabled: true, selectable: true } }),
    createLanguage('ja', { web: { enabled: true, indexed: true, url_prefix: 'ja' }, app: { enabled: true, selectable: true } }),
    createLanguage('ar', { web: { enabled: true, indexed: false, url_prefix: 'ar' }, app: { enabled: true, selectable: true } }),
    createLanguage('fr', { web: { enabled: true, indexed: false, url_prefix: 'fr' }, app: { enabled: true, selectable: false } }),
    createLanguage('de', {}),
    createLanguage('it', {
      web: { enabled: true, indexed: true, url_prefix: 'it' },
      app: { enabled: true, selectable: true },
      release: { enabled: true, android_store_locale: 'it-IT', ios_store_locale: 'it' },
    }),
  ];

  assert.equal(stageForLanguage(languages[1]), 'web_indexed');
  assert.equal(stageForLanguage(languages[2]), 'app_selectable');
  assert.equal(stageForLanguage(languages[3]), 'render_only');
  assert.equal(stageForLanguage(languages[4]), 'disabled');
  assert.equal(stageForLanguage(languages[5]), 'store_release_enabled');
  assert.deepEqual(classifyLanguageStages(languages), {
    disabled: ['de'],
    render_only: ['fr'],
    app_selectable: ['ar'],
    web_indexed: ['ja'],
    store_release_enabled: ['it'],
  });
});

test('passes when evidence matches staged locale flags', async () => {
  const rootDir = await createTempRoot();
  await writeCompleteEvidence(rootDir);

  const report = await buildFlagStageReport({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.deepEqual(report.current_stages.app_selectable, ['ar']);
  assert.match(renderFlagStageReport(report), /Result: pass/);
});

test('allows future transition evidence to mark that registry flags changed', async () => {
  const rootDir = await createTempRoot();
  await writeCompleteEvidence(rootDir, { currentTaskChangesRegistryFlags: true });

  const report = await buildFlagStageReport({ rootDir });

  assert.equal(report.ok, true);
});

test('fails when evidence omits current locales or required checks', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'doc/qa/m9-t05/flag-stages.json', {
    format_version: 1,
    task: 'M9-T05',
    transition_order: ['disabled', 'render_only', 'app_selectable', 'web_indexed', 'store_release_enabled'],
    transition_policy: {
      single_transition_kind_per_commit: true,
      no_public_index_or_release_until_prior_stage_passes: true,
      current_task_changes_registry_flags: false,
    },
    current_stages: {
      disabled: [],
      render_only: [],
      app_selectable: [],
      web_indexed: ['ja'],
      store_release_enabled: [],
    },
    stages: {
      disabled: [],
      render_only: [],
      app_selectable: [
        {
          locales: ['ar'],
          status: 'pass',
          evidence: [{ kind: 'i18n_check', command: 'make i18n-check LANGS=ar' }],
        },
      ],
      web_indexed: [],
      store_release_enabled: [],
    },
  });

  const report = await buildFlagStageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'flag-stage-locale-mismatch'));
  assert.ok(report.issues.some((issue) => issue.code === 'flag-stage-evidence-missing-kind'));
});

test('fails when release flags skip prerequisite mappings and metadata', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, web: { enabled: true, indexed: true, url_prefix: '' }, app: { enabled: true, selectable: true } }),
      createLanguage('fr', {
        web: { enabled: true, indexed: true, url_prefix: 'fr' },
        app: { enabled: true, selectable: false },
        release: { enabled: true, android_store_locale: null, ios_store_locale: null },
      }),
    ],
  });
  await writeJson(rootDir, 'doc/qa/m9-t05/flag-stages.json', {
    format_version: 1,
    task: 'M9-T05',
    transition_order: ['disabled', 'render_only', 'app_selectable', 'web_indexed', 'store_release_enabled'],
    transition_policy: {
      single_transition_kind_per_commit: true,
      no_public_index_or_release_until_prior_stage_passes: true,
      current_task_changes_registry_flags: false,
    },
    current_stages: {
      disabled: [],
      render_only: [],
      app_selectable: [],
      web_indexed: [],
      store_release_enabled: ['fr'],
    },
    stages: {
      disabled: [],
      render_only: [],
      app_selectable: [],
      web_indexed: [],
      store_release_enabled: [
        {
          locales: ['fr'],
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check LANGS=fr' },
            { kind: 'holdout_review', command: 'make i18n-holdouts-check' },
            { kind: 'layout_qa', command: 'make i18n-layout-qa-check' },
            { kind: 'store_metadata', command: 'make store-metadata-check' },
            { kind: 'fastlane_config', command: 'make android-fastlane-check && make ios-fastlane-check' },
            { kind: 'release_secret_guardrails', command: 'make release-secret-guardrails-check' },
          ],
        },
      ],
    },
  });

  const report = await buildFlagStageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'flag-stage-release-app-disabled'));
  assert.ok(report.issues.some((issue) => issue.code === 'flag-stage-release-store-locale'));
  assert.ok(report.issues.some((issue) => issue.code === 'flag-stage-release-metadata-missing'));
});

async function createTempRoot({ languages } = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-flag-stages-'));
  await writeJson(
    rootDir,
    'config/languages.json',
    languages ?? [
      createLanguage('en', {
        fallback: null,
        web: { enabled: true, indexed: true, url_prefix: '' },
        app: { enabled: true, selectable: true },
      }),
      createLanguage('ja', {
        web: { enabled: true, indexed: true, url_prefix: 'ja' },
        app: { enabled: true, selectable: true },
      }),
      createLanguage('ar', {
        web: { enabled: true, indexed: false, url_prefix: 'ar' },
        app: { enabled: true, selectable: true },
      }),
      createLanguage('fr', {}),
    ],
  );
  for (const path of [
    'scripts/i18n/check.mjs',
    'scripts/i18n/stubs.mjs',
    'scripts/i18n/holdouts.mjs',
    'scripts/i18n/layout_qa.mjs',
    'app/test/pilot_layout_qa_test.dart',
    'web/src/main.rs',
  ]) {
    await writeText(rootDir, path, 'fixture');
  }
  return rootDir;
}

async function writeCompleteEvidence(rootDir, { currentTaskChangesRegistryFlags = false } = {}) {
  await writeJson(rootDir, 'doc/qa/m9-t05/flag-stages.json', {
    format_version: 1,
    task: 'M9-T05',
    transition_order: ['disabled', 'render_only', 'app_selectable', 'web_indexed', 'store_release_enabled'],
    transition_policy: {
      single_transition_kind_per_commit: true,
      no_public_index_or_release_until_prior_stage_passes: true,
      current_task_changes_registry_flags: currentTaskChangesRegistryFlags,
    },
    current_stages: {
      disabled: ['fr'],
      render_only: [],
      app_selectable: ['ar'],
      web_indexed: ['ja'],
      store_release_enabled: [],
    },
    stages: {
      disabled: [
        {
          locales: ['fr'],
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check LANGS=all', path: 'scripts/i18n/check.mjs' },
            { kind: 'stubs_check', command: 'make i18n-stubs-check', path: 'scripts/i18n/stubs.mjs' },
          ],
        },
      ],
      render_only: [],
      app_selectable: [
        {
          locales: ['ar'],
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check LANGS=ar', path: 'scripts/i18n/check.mjs' },
            { kind: 'holdout_review', command: 'make i18n-holdouts-check', path: 'scripts/i18n/holdouts.mjs' },
            { kind: 'layout_qa', command: 'make i18n-layout-qa-check', path: 'scripts/i18n/layout_qa.mjs' },
            { kind: 'app_layout', command: 'flutter test test/pilot_layout_qa_test.dart', path: 'app/test/pilot_layout_qa_test.dart' },
          ],
        },
      ],
      web_indexed: [
        {
          locales: ['ja'],
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check LANGS=ja', path: 'scripts/i18n/check.mjs' },
            { kind: 'holdout_review', command: 'make i18n-holdouts-check', path: 'scripts/i18n/holdouts.mjs' },
            { kind: 'layout_qa', command: 'make i18n-layout-qa-check', path: 'scripts/i18n/layout_qa.mjs' },
            { kind: 'web_layout', command: 'cargo test --manifest-path web/Cargo.toml', path: 'web/src/main.rs' },
          ],
        },
      ],
      store_release_enabled: [],
    },
  });
}

function createLanguage(routeCode, overrides = {}) {
  return {
    route_code: routeCode,
    bcp47: routeCode,
    flutter: {
      languageCode: routeCode,
      scriptCode: null,
      countryCode: null,
    },
    native_name: routeCode,
    english_name: routeCode,
    text_direction: overrides.text_direction ?? 'ltr',
    fallback: Object.hasOwn(overrides, 'fallback') ? overrides.fallback : 'en',
    currency: 'USD',
    web: overrides.web ?? { enabled: false, indexed: false, url_prefix: routeCode },
    app: overrides.app ?? { enabled: false, selectable: false },
    release: overrides.release ?? {
      enabled: false,
      android_store_locale: null,
      ios_store_locale: null,
    },
  };
}

async function writeJson(rootDir, relativePath, value) {
  await writeText(rootDir, relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeText(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), value);
}
