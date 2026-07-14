import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildTranslationPatchReport, renderTranslationPatchReport } from './translation_patches.mjs';

test('accepts the checked-in high-priority translation patch evidence', async () => {
  const report = await buildTranslationPatchReport();
  const rendered = renderTranslationPatchReport(report);

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.equal(report.high_priority_translation_issue_count, 0);
  assert.equal(report.patch_count, 0);
  assert.match(rendered, /Result: pass/);
});

test('fails when a high-priority translation issue has no patch', async () => {
  const rootDir = await createTempRoot({
    triage: validTriage({
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
              owner: 'release owner',
              summary: 'Checkout copy is unclear.',
            },
          ],
        },
      ],
    }),
    evidence: validEvidence({ patches: [] }),
  });

  const report = await buildTranslationPatchReport({ rootDir });

  assert.equal(report.ok, false);
  assert.equal(report.high_priority_translation_issue_count, 1);
  assert.equal(report.unresolved_issue_count, 1);
  assert.ok(report.issues.some((issue) => issue.code === 'translation-patch-unresolved'));
});

test('passes when a high-priority translation issue has a content-only patch', async () => {
  const rootDir = await createTempRoot({
    triage: validTriage({
      triageGroups: [
        {
          locale: 'ja',
          platform: 'web',
          screen: 'payment_success',
          issues: [
            {
              id: 'SUP-002',
              category: 'translation',
              severity: 'critical',
              status: 'open',
              owner: 'release owner',
              summary: 'Payment success copy is wrong.',
            },
          ],
        },
      ],
    }),
    evidence: validEvidence({
      patches: [
        {
          source_issue_id: 'SUP-002',
          locale: 'ja',
          owner: 'release owner',
          status: 'pass',
          files: ['web/content/i18n/payment_success/ja.json'],
          validation: [
            {
              command: 'make i18n-check',
              status: 'pass',
            },
          ],
        },
      ],
    }),
  });

  const report = await buildTranslationPatchReport({ rootDir });

  assert.equal(report.ok, true);
  assert.equal(report.patch_count, 1);
  assert.deepEqual(report.patched_locales, ['ja']);
});

test('fails when a patch lists non-content files', async () => {
  const rootDir = await createTempRoot({
    triage: validTriage({
      triageGroups: [
        {
          locale: 'ja',
          platform: 'ios',
          screen: 'settings',
          issues: [
            {
              id: 'SUP-003',
              category: 'translation',
              severity: 'high',
              status: 'open',
              owner: 'release owner',
              summary: 'Settings label is wrong.',
            },
          ],
        },
      ],
    }),
    evidence: validEvidence({
      patches: [
        {
          source_issue_id: 'SUP-003',
          locale: 'ja',
          owner: 'release owner',
          status: 'pass',
          files: ['app/lib/features/settings/settings_screen.dart'],
          validation: [
            {
              command: 'make i18n-check',
              status: 'pass',
            },
          ],
        },
      ],
    }),
  });

  const report = await buildTranslationPatchReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'translation-patch-content-only'));
});

test('fails when a patch locale differs from the source issue locale', async () => {
  const rootDir = await createTempRoot({
    triage: validTriage({
      triageGroups: [
        {
          locale: 'ja',
          platform: 'web',
          screen: 'top',
          issues: [
            {
              id: 'SUP-004',
              category: 'translation',
              severity: 'high',
              status: 'open',
              owner: 'release owner',
              summary: 'Top page copy is wrong.',
            },
          ],
        },
      ],
    }),
    evidence: validEvidence({
      patches: [
        {
          source_issue_id: 'SUP-004',
          locale: 'en',
          owner: 'release owner',
          status: 'pass',
          files: ['web/content/i18n/common/en.json'],
          validation: [
            {
              command: 'make i18n-check',
              status: 'pass',
            },
          ],
        },
      ],
    }),
  });

  const report = await buildTranslationPatchReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'translation-patch-locale'));
});

test('fails when evidence omits passing i18n-check validation', async () => {
  const evidence = validEvidence({ patches: [] });
  evidence.validation = [];
  const rootDir = await createTempRoot({ evidence });

  const report = await buildTranslationPatchReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'translation-patch-validation'));
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null }),
    createLanguage('ja'),
  ],
  triage = validTriage({ triageGroups: [] }),
  evidence = validEvidence({ patches: [] }),
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-translation-patches-'));
  await writeJson(rootDir, 'config/languages.json', languages);
  await writeJson(rootDir, 'doc/qa/m11-t02/support-feedback-triage.json', triage);
  await writeJson(rootDir, 'doc/qa/m11-t03/translation-patch-review.json', evidence);
  return rootDir;
}

function validTriage({ triageGroups }) {
  return {
    format_version: 1,
    task: 'M11-T02',
    reviewed_at: '2026-06-18T23:45:00+09:00',
    triage_groups: triageGroups,
  };
}

function validEvidence({ patches }) {
  return {
    format_version: 1,
    task: 'M11-T03',
    reviewed_at: '2026-06-18T23:55:00+09:00',
    source_triage_evidence: 'doc/qa/m11-t02/support-feedback-triage.json',
    store_release_enabled_locales: [],
    patches,
    validation: [
      {
        command: 'make i18n-check',
        status: 'pass',
      },
    ],
    summary: {
      high_priority_translation_issues_patched: true,
      content_only_patches: true,
      i18n_check_passed: true,
      store_release_enabled_languages_clean: true,
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
