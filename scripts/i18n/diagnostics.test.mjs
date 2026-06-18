import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { buildLocaleDiagnosticsReport, renderLocaleDiagnosticsReport } from './diagnostics.mjs';

test('accepts the checked-in locale diagnostics review', async () => {
  const report = await buildLocaleDiagnosticsReport();
  const rendered = renderLocaleDiagnosticsReport(report);

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.match(rendered, /Result: pass/);
});

test('fails when evidence omits a required diagnostic stream', async () => {
  const rootDir = await createTempRoot({
    evidence: {
      ...validEvidence({ releaseEnabledLocales: [] }),
      diagnostic_streams: {},
    },
  });

  const report = await buildLocaleDiagnosticsReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'locale-diagnostics-stream'));
});

test('fails when release-enabled fallback spikes are present', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, release: { enabled: true, android_store_locale: 'en-US', ios_store_locale: 'en-US' } }),
      createLanguage('ja', { release: { enabled: true, android_store_locale: 'ja-JP', ios_store_locale: 'ja' } }),
    ],
    evidence: {
      ...validEvidence({ releaseEnabledLocales: ['en', 'ja'] }),
      unexpected_fallback_spikes: [
        {
          locale: 'ja',
          count: 3,
          reason: 'missing_content',
        },
      ],
    },
  });

  const report = await buildLocaleDiagnosticsReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'locale-diagnostics-spike' &&
        issue.message === 'release-enabled locales have unexpected fallback spikes',
    ),
  );
});

test('fails when evidence release-enabled locales differ from registry', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, release: { enabled: true, android_store_locale: 'en-US', ios_store_locale: 'en-US' } }),
      createLanguage('ja', { release: { enabled: true, android_store_locale: 'ja-JP', ios_store_locale: 'ja' } }),
    ],
    evidence: validEvidence({ releaseEnabledLocales: ['en'] }),
  });

  const report = await buildLocaleDiagnosticsReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'locale-diagnostics-locale-set'));
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null }),
    createLanguage('ja'),
  ],
  evidence = validEvidence({ releaseEnabledLocales: [] }),
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-locale-diagnostics-'));
  await mkdir(join(rootDir, 'config'), { recursive: true });
  await mkdir(join(rootDir, 'doc/qa/m11-t01'), { recursive: true });
  await writeFile(join(rootDir, 'config/languages.json'), `${JSON.stringify(languages, null, 2)}\n`);
  await writeFile(join(rootDir, 'doc/qa/m11-t01/locale-diagnostics-review.json'), `${JSON.stringify(evidence, null, 2)}\n`);
  return rootDir;
}

function validEvidence({ releaseEnabledLocales }) {
  return {
    format_version: 1,
    task: 'M11-T01',
    reviewed_at: '2026-06-18T23:00:00+09:00',
    release_enabled_locales: releaseEnabledLocales,
    diagnostic_streams: Object.fromEntries(
      ['unsupported_locale', 'fallback_locale', 'missing_content', 'checkout_locale', 'malformed_translation'].map((stream) => [
        stream,
        {
          status: 'pass',
          events_reviewed: 0,
          query: `logName:${stream}`,
        },
      ]),
    ),
    unexpected_fallback_spikes: [],
    summary: {
      unsupported_locale_reviewed: true,
      fallback_locale_reviewed: true,
      missing_content_reviewed: true,
      checkout_locale_reviewed: true,
      malformed_translation_reviewed: true,
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
