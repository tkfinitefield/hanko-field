import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildReleaseRunbookReport, renderReleaseRunbookReport } from './release_runbook.mjs';

test('accepts the checked-in localized release runbook', async () => {
  const report = await buildReleaseRunbookReport();
  const rendered = renderReleaseRunbookReport(report);

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.ok(report.route_code_count > 0);
  assert.match(rendered, /Result: pass/);
});

test('fails when the runbook omits fastlane release steps', async () => {
  const rootDir = await createTempRoot({
    runbook: validRunbook().replace('bundle exec fastlane android production', ''),
  });

  const report = await buildReleaseRunbookReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'release-runbook-content'));
});

test('fails when internal upload remains in validate-only mode', async () => {
  const rootDir = await createTempRoot({
    runbook: validRunbook().replace('SUPPLY_VALIDATE_ONLY=false', ''),
  });

  const report = await buildReleaseRunbookReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.key === 'SUPPLY_VALIDATE_ONLY=false'));
});

test('fails when evidence route code count drifts from the registry', async () => {
  const evidence = validEvidence();
  evidence.route_code_count = 3;
  const rootDir = await createTempRoot({ evidence });

  const report = await buildReleaseRunbookReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'release-runbook-route-count'));
});

test('fails when evidence omits a required store metadata path', async () => {
  const evidence = validEvidence();
  evidence.store_metadata_paths = ['release/store_metadata/source/'];
  const rootDir = await createTempRoot({ evidence });

  const report = await buildReleaseRunbookReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'release-runbook-list'));
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null }),
    createLanguage('ja'),
  ],
  runbook = validRunbook(),
  evidence = validEvidence({ routeCodeCount: languages.length }),
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-release-runbook-'));
  await writeJson(rootDir, 'config/languages.json', languages);
  await writeText(rootDir, 'doc/localized-release-runbook.md', runbook);
  await writeJson(rootDir, 'doc/qa/m11-t05/release-runbook-review.json', evidence);
  return rootDir;
}

function createLanguage(routeCode, { fallback = 'en' } = {}) {
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
    text_direction: 'ltr',
    fallback,
    currency: 'USD',
    web: {
      enabled: routeCode === 'en',
      indexed: routeCode === 'en',
      url_prefix: routeCode,
    },
    app: {
      enabled: routeCode === 'en',
      selectable: routeCode === 'en',
    },
    release: {
      enabled: false,
      android_store_locale: null,
      ios_store_locale: null,
    },
  };
}

function validEvidence({ routeCodeCount = 2 } = {}) {
  return {
    format_version: 1,
    task: 'M11-T05',
    reviewed_at: '2026-06-18T23:59:00+09:00',
    runbook_path: 'doc/localized-release-runbook.md',
    route_code_count: routeCodeCount,
    required_sections: [
      'language_addition_flow',
      'store_metadata_update_flow',
      'fastlane_release_flow',
      'post_release_monitoring_and_cleanup',
      'rollback',
    ],
    required_commands: [
      'make i18n-check',
      'make store-metadata-check',
      'make google-play-metadata-check',
      'make app-store-metadata-check',
      'make screenshot-metadata-check',
      'make android-fastlane-check',
      'make ios-fastlane-check',
      'make release-secret-guardrails-check',
      'make i18n-ci',
    ],
    store_metadata_paths: [
      'release/store_metadata/source/',
      'release/store_metadata/google_play',
      'release/store_metadata/app_store',
    ],
    fastlane_lanes: [
      'app/android: bundle exec fastlane android metadata',
      'app/android: bundle exec fastlane android internal',
      'app/android: bundle exec fastlane android production',
      'app/ios: bundle exec fastlane ios metadata',
      'app/ios: bundle exec fastlane ios testflight_upload',
      'app/ios: bundle exec fastlane ios production',
    ],
    summary: {
      future_language_steps_documented: true,
      store_metadata_steps_documented: true,
      fastlane_release_steps_documented: true,
      post_release_cleanup_documented: true,
      production_secrets_excluded: true,
    },
  };
}

function validRunbook() {
  return `# Localized Release Runbook

## Language Addition Flow

Update config/languages.json, then run make i18n-check.

## Store Metadata Update Flow

Use release/store_metadata/source/, release/store_metadata/google_play, and release/store_metadata/app_store.
Run make store-metadata-check, make google-play-metadata-check, make app-store-metadata-check, and make screenshot-metadata-check.

## fastlane Release Flow

Use app/android/fastlane/Fastfile and app/ios/fastlane/Fastfile.
Set SUPPLY_JSON_KEY, SUPPLY_VALIDATE_ONLY=false, APP_STORE_CONNECT_API_KEY_PATH, RELEASE_SIGNOFF_PATH, and RELEASE_SIGNOFF_CONFIRMATION.
Run bundle exec fastlane android metadata, bundle exec fastlane android internal, bundle exec fastlane android production, bundle exec fastlane ios metadata, bundle exec fastlane ios testflight_upload, and bundle exec fastlane ios production.

## Post-Release Monitoring and Cleanup

Run make i18n-diagnostics-check, make i18n-support-triage-check, make i18n-translation-patches-check, and make i18n-migration-cleanup-check.
Run make i18n-ci and make release-secret-guardrails-check.

## Rollback

Disable release flags and repeat validation.
`;
}

async function writeJson(rootDir, relativePath, value) {
  await writeText(rootDir, relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeText(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), value);
}
