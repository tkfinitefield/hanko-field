import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  isAllowedReasonCode,
  parseIntentionSidecar,
  validateIntentions,
} from './intentions.mjs';

test('accepts allowed reason codes from map and entries sidecars', () => {
  assert.equal(isAllowedReasonCode('brand_name'), true);
  assert.equal(isAllowedReasonCode('url_or_email'), true);
  assert.equal(isAllowedReasonCode('not_a_reason'), false);

  const mapSidecar = parseIntentionSidecar(
    {
      appTitle: 'brand_name',
    },
    { path: 'app/lib/l10n/app_ja_intentions.json', targetLocale: 'ja' },
  );
  assert.deepEqual(mapSidecar.issues, []);
  assert.equal(mapSidecar.approvals.get('appTitle'), 'brand_name');

  const entriesSidecar = parseIntentionSidecar(
    {
      entries: [
        {
          key_path: 'contact.options[1].value',
          target_locale: 'ja',
          reason_code: 'url_or_email',
        },
      ],
    },
    { path: 'app/assets/i18n/settings/ja_intentions.json', targetLocale: 'ja' },
  );
  assert.deepEqual(entriesSidecar.issues, []);
  assert.equal(entriesSidecar.approvals.get('contact.options[1].value'), 'url_or_email');
});

test('rejects invalid reason codes and mismatched target locale', () => {
  const parsed = parseIntentionSidecar(
    {
      entries: [
        {
          key_path: 'appTitle',
          target_locale: 'zh',
          reason_code: 'brand_name',
        },
        {
          key_path: 'orderNoHint',
          target_locale: 'ja',
          reason_code: 'unknown_reason',
        },
      ],
    },
    { path: 'app/lib/l10n/app_ja_intentions.json', targetLocale: 'ja' },
  );

  assert.deepEqual(
    parsed.issues.map((issue) => [issue.code, issue.key]),
    [
      ['intention-locale', 'appTitle'],
      ['intention-reason', 'orderNoHint'],
    ],
  );
});

test('fails when target ARB keeps English without an approved sidecar', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);

  const report = await validateIntentions({
    rootDir,
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'intention-missing' &&
        issue.file === 'app/lib/l10n/app_ja.arb' &&
        issue.key === 'appTitle',
    ),
  );
});

test('passes when exact-English ARB values are approved per key', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_ja_intentions.json', {
    appTitle: 'brand_name',
  });

  const report = await validateIntentions({
    rootDir,
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
});

test('passes when exact-English JSON values are approved by entries sidecar', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'web/content/i18n/common/ja_intentions.json', {
    entries: [
      {
        key_path: 'language_en_label',
        target_locale: 'ja',
        reason_code: 'intentionally_english',
      },
    ],
  });

  const report = await validateIntentions({
    rootDir,
    file: 'web/content/i18n/common/ja.json',
  });

  assert.equal(report.ok, true);
});

test('ignores empty strings when checking same-as-English values', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir, {
    appEn: {
      '@@locale': 'en',
      appTitle: 'STONE SIGNATURE',
      emptyNotice: '',
    },
    appJa: {
      '@@locale': 'ja',
      appTitle: 'STONE SIGNATURE',
      emptyNotice: '',
    },
  });
  await writeJson(rootDir, 'app/lib/l10n/app_ja_intentions.json', {
    appTitle: 'brand_name',
  });

  const report = await validateIntentions({
    rootDir,
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
});

test('validates catalog same-as-English values with route-specific approvals', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'api/content/i18n/catalog/materials_intentions.json', {
    entries: [
      {
        key_path: 'jade.label',
        target_locale: 'ja',
        reason_code: 'product_name',
      },
    ],
  });

  const report = await validateIntentions({
    rootDir,
    file: 'api/content/i18n/catalog/materials.json',
  });

  assert.equal(report.ok, true);
});

test('does not infer a target locale from catalog sidecar file names', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'api/content/i18n/catalog/materials_intentions.json', {
    entries: [
      {
        key_path: 'jade.label',
        target_locale: 'ar',
        reason_code: 'locale_not_release_enabled',
      },
    ],
  });

  const report = await validateIntentions({
    rootDir,
    file: 'api/content/i18n/catalog/materials_intentions.json',
  });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-intentions-'));
  await writeJson(rootDir, 'config/languages.json', [
    createLanguage('en', {
      fallback: null,
      web: { enabled: true, indexed: true, url_prefix: '' },
      app: { enabled: true, selectable: true },
    }),
    createLanguage('ja', {
      bcp47: 'ja',
      native_name: '日本語',
      english_name: 'Japanese',
      currency: 'JPY',
      web: { enabled: true, indexed: true, url_prefix: 'ja' },
      app: { enabled: true, selectable: true },
    }),
  ]);
  return rootDir;
}

async function writeMinimalContent(rootDir, overrides = {}) {
  const files = new Map([
    [
      'app/lib/l10n/app_en.arb',
      overrides.appEn ?? {
        '@@locale': 'en',
        appTitle: 'STONE SIGNATURE',
        hello: 'Hello',
      },
    ],
    [
      'app/lib/l10n/app_ja.arb',
      overrides.appJa ?? {
        '@@locale': 'ja',
        appTitle: 'STONE SIGNATURE',
        hello: 'こんにちは',
      },
    ],
    ['app/assets/i18n/settings/en.json', { contact: { email: 'dev@finitefield.org' } }],
    ['app/assets/i18n/settings/ja.json', { contact: { email: 'dev@finitefield.org' } }],
    ['app/assets/i18n/settings/ja_intentions.json', { contact: { email: 'url_or_email' } }],
    ['api/content/i18n/checkout/en.json', { product_name_template: 'Stone seal' }],
    ['api/content/i18n/checkout/ja.json', { product_name_template: '宝石印鑑' }],
    ['api/content/i18n/catalog/materials.json', { jade: { label: { en: 'Jade', ja: 'Jade' } } }],
    ['api/content/i18n/catalog/stone_listings.json', {}],
    ['api/content/i18n/catalog/facet_tags.json', {}],
    ['api/content/i18n/catalog/countries.json', {}],
    ['web/content/i18n/common/en.json', { language_en_label: 'English', footer_company: 'Company' }],
    ['web/content/i18n/common/ja.json', { language_en_label: 'English', footer_company: '会社' }],
  ]);

  for (const [relativePath, value] of files) {
    await writeJson(rootDir, relativePath, value);
  }
}

async function writeJson(rootDir, relativePath, value) {
  await writeText(rootDir, relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeText(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), value);
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
    fallback: overrides.fallback === undefined ? 'en' : overrides.fallback,
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
