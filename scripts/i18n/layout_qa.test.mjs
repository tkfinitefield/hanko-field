import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildLayoutQa, expectedLayoutQaTiers, renderLayoutQa } from './layout_qa.mjs';

test('classifies current rollout tiers from registry flags', () => {
  const tiers = expectedLayoutQaTiers([
    createLanguage('en', { web: { enabled: true, indexed: true, url_prefix: '' } }),
    createLanguage('ja', { web: { enabled: true, indexed: true, url_prefix: 'ja' } }),
    createLanguage('ar', { web: { enabled: true, indexed: false, url_prefix: 'ar' }, app: { enabled: true, selectable: true } }),
    createLanguage('fr', {}),
  ]);

  assert.deepEqual(tiers, {
    tier1_full: ['ja'],
    tier2_screenshot: ['ar'],
    tier3_mechanical: ['fr'],
  });
});

test('passes when evidence matches tiers and runtime direction hooks exist', async () => {
  const rootDir = await createTempRoot();
  await writeCompleteEvidence(rootDir);

  const report = await buildLayoutQa({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.deepEqual(report.expected_tiers.tier1_full, ['ja']);
  assert.deepEqual(report.expected_tiers.tier2_screenshot, ['ar']);
  assert.deepEqual(report.expected_tiers.tier3_mechanical, ['fr']);
  assert.match(renderLayoutQa(report), /Result: pass/);
});

test('fails when tier evidence omits a locale or required evidence kind', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'doc/qa/m9-t04/layout-qa.json', {
    format_version: 1,
    task: 'M9-T04',
    tiers: {
      tier1_full: [],
      tier2_screenshot: [
        {
          locale: 'ar',
          status: 'pass',
          evidence: [{ kind: 'i18n_check', command: 'make i18n-check' }],
        },
      ],
      tier3_mechanical: [],
    },
  });

  const report = await buildLayoutQa({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'layout-qa-tier-mismatch'));
  assert.ok(report.issues.some((issue) => issue.code === 'layout-qa-evidence-missing-kind'));
});

test('fails when web templates do not bind the locale direction', async () => {
  const rootDir = await createTempRoot({
    webTemplate: '<!doctype html><html lang="{{ selected_locale }}"><body></body></html>',
  });
  await writeCompleteEvidence(rootDir);

  const report = await buildLayoutQa({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'layout-qa-web-dir-missing'));
});

async function createTempRoot({ webTemplate } = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-layout-qa-'));
  await writeJson(rootDir, 'config/languages.json', [
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
      text_direction: 'rtl',
      web: { enabled: true, indexed: false, url_prefix: 'ar' },
      app: { enabled: true, selectable: true },
    }),
    createLanguage('fr', {}),
  ]);
  await writeText(
    rootDir,
    'web/templates/top.html',
    webTemplate ?? '<!doctype html><html lang="{{ selected_locale }}" dir="{{ self.html_dir() }}"><body></body></html>',
  );
  await writeText(rootDir, 'web/src/main.rs', 'fn html_dir_for_locale(locale: &str) { match locale { "ar" => "rtl", _ => "ltr" } }');
  await writeText(
    rootDir,
    'app/android/app/src/main/AndroidManifest.xml',
    '<manifest><application><activity android:configChanges="locale|layoutDirection"/></application></manifest>',
  );
  for (const path of [
    'doc/qa/m7-t05/README.md',
    'doc/qa/m7-t05/screenshots/web-ar-about-mobile.png',
    'app/test/pilot_layout_qa_test.dart',
    'app/test/rtl_overflow_test.dart',
    'app/test/widget_test.dart',
  ]) {
    await writeText(rootDir, path, 'fixture');
  }
  return rootDir;
}

async function writeCompleteEvidence(rootDir) {
  await writeJson(rootDir, 'doc/qa/m9-t04/layout-qa.json', {
    format_version: 1,
    task: 'M9-T04',
    tiers: {
      tier1_full: [
        {
          locale: 'ja',
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check' },
            { kind: 'app_layout', command: 'flutter test test/widget_test.dart', path: 'app/test/widget_test.dart' },
            { kind: 'web_layout', command: 'cargo test --manifest-path web/Cargo.toml', path: 'web/src/main.rs' },
          ],
        },
      ],
      tier2_screenshot: [
        {
          locale: 'ar',
          status: 'pass',
          evidence: [
            { kind: 'i18n_check', command: 'make i18n-check' },
            { kind: 'app_layout', command: 'flutter test test/pilot_layout_qa_test.dart', path: 'app/test/pilot_layout_qa_test.dart' },
            { kind: 'screenshot_qa', command: 'sips -g pixelWidth doc/qa/m7-t05/screenshots/*.png', path: 'doc/qa/m7-t05/screenshots/web-ar-about-mobile.png' },
          ],
        },
      ],
      tier3_mechanical: [
        {
          locales: ['fr'],
          status: 'pass',
          evidence: [
            { kind: 'mechanical_check', command: 'make i18n-check LANGS=all' },
            { kind: 'stub_check', command: 'make i18n-stubs-check' },
          ],
        },
      ],
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
    release: {
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
