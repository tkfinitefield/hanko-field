import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildSupportTriageReport, renderSupportTriageReport } from './support_triage.mjs';

test('accepts the checked-in support feedback triage evidence', async () => {
  const report = await buildSupportTriageReport();
  const rendered = renderSupportTriageReport(report);

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.equal(report.missing_owner_count, 0);
  assert.match(rendered, /Result: pass/);
});

test('fails when a translation issue has no owner', async () => {
  const rootDir = await createTempRoot({
    evidence: validEvidence({
      triageGroups: [
        {
          locale: 'ja',
          platform: 'android',
          screen: 'checkout',
          issues: [
            {
              id: 'SUP-001',
              category: 'translation',
              severity: 'high',
              status: 'open',
              summary: 'Checkout copy is unclear.',
            },
          ],
        },
      ],
    }),
  });

  const report = await buildSupportTriageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.equal(report.missing_owner_count, 1);
  assert.ok(report.issues.some((issue) => issue.code === 'support-triage-owner'));
});

test('fails when a triage group locale is not in the registry', async () => {
  const rootDir = await createTempRoot({
    evidence: validEvidence({
      triageGroups: [
        {
          locale: 'xx',
          platform: 'web',
          screen: 'top',
          issues: [],
        },
      ],
    }),
  });

  const report = await buildSupportTriageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'support-triage-locale'));
});

test('fails when a required support source is missing', async () => {
  const evidence = validEvidence({ triageGroups: [] });
  delete evidence.support_sources.support_form;
  const rootDir = await createTempRoot({ evidence });

  const report = await buildSupportTriageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'support-triage-source'));
});

test('reports malformed triage groups without throwing', async () => {
  const rootDir = await createTempRoot({
    evidence: validEvidence({
      triageGroups: [
        null,
        {
          locale: 'ja',
          platform: 'ios',
          screen: 'settings',
          issues: [],
        },
      ],
    }),
  });

  const report = await buildSupportTriageReport({ rootDir });

  assert.equal(report.ok, false);
  assert.deepEqual(report.triaged_locales, ['ja']);
  assert.ok(report.issues.some((issue) => issue.code === 'support-triage-format'));
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null }),
    createLanguage('ja'),
  ],
  evidence = validEvidence({ triageGroups: [] }),
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-support-triage-'));
  await writeJson(rootDir, 'config/languages.json', languages);
  await writeJson(rootDir, 'doc/qa/m11-t02/support-feedback-triage.json', evidence);
  return rootDir;
}

function validEvidence({ triageGroups }) {
  return {
    format_version: 1,
    task: 'M11-T02',
    reviewed_at: '2026-06-18T23:45:00+09:00',
    support_sources: {
      support_email: {
        status: 'pass',
        records_reviewed: 0,
        query: 'Support mailbox search for locale feedback, to be run after staged rollout starts.',
      },
      support_form: {
        status: 'pass',
        records_reviewed: 0,
        query: 'Support form export for locale feedback, to be run after staged rollout starts.',
      },
      google_play_reviews: {
        status: 'pass',
        records_reviewed: 0,
        query: 'Google Play review export by locale, to be run after staged rollout starts.',
      },
      app_store_reviews: {
        status: 'pass',
        records_reviewed: 0,
        query: 'App Store Connect review export by locale, to be run after staged rollout starts.',
      },
    },
    owner_policy: {
      translation_owner: 'release owner',
      layout_owner: 'release owner',
    },
    triage_groups: triageGroups,
    summary: {
      grouped_by_language_platform_screen: true,
      translation_fixes_have_owners: true,
      layout_fixes_have_owners: true,
    },
  };
}

function createLanguage(routeCode, overrides = {}) {
  return {
    route_code: routeCode,
    bcp47: overrides.bcp47 ?? routeCode,
    flutter: overrides.flutter ?? {
      languageCode: routeCode,
      scriptCode: null,
      countryCode: null,
    },
    native_name: overrides.native_name ?? routeCode,
    english_name: overrides.english_name ?? routeCode,
    text_direction: overrides.text_direction ?? 'ltr',
    fallback: Object.hasOwn(overrides, 'fallback') ? overrides.fallback : 'en',
    currency: overrides.currency ?? 'USD',
    web: overrides.web ?? {
      enabled: false,
      indexed: false,
      url_prefix: routeCode === 'en' ? '' : routeCode,
    },
    app: overrides.app ?? {
      enabled: false,
      selectable: false,
    },
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
