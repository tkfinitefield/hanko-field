import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildI18nCheck, renderI18nCheck, validateJsonContentFiles } from './check.mjs';

test('passes when enabled content is present and well formed', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);

  const report = await buildI18nCheck({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.ok(report.parsed_files.includes('config/languages.json'));
});

test('fails when todo reports missing keys for requested languages', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);

  const report = await buildI18nCheck({
    rootDir,
    langs: ['fr'],
    file: 'web/content/i18n/common/fr.json',
  });

  assert.equal(report.ok, false);
  assert.deepEqual(
    report.issues.map((issue) => [issue.type, issue.file, issue.message]),
    [
      ['missing-file', 'web/content/i18n/common/fr.json', 'fr brand_subtitle'],
      ['missing-file', 'web/content/i18n/common/fr.json', 'fr footer_company'],
    ],
  );
});

test('fails when enabled status files are missing', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir, {
    skip: new Set(['web/content/i18n/top/ja.json']),
  });

  const report = await buildI18nCheck({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) => issue.type === 'missing-file' && issue.file === 'web/content/i18n/top/ja.json',
    ),
  );
});

test('fails when checked content is malformed JSON', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);
  await writeText(rootDir, 'web/content/i18n/common/ja.json', '{');

  const report = await buildI18nCheck({
    rootDir,
    file: 'web/content/i18n/common/ja.json',
  });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) => issue.type === 'malformed-json' && issue.file === 'web/content/i18n/common/ja.json',
    ),
  );
});

test('fails when checked ARB placeholders are unsafe', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_ja.arb', {
    '@@locale': 'ja',
    hello: 'こんにちは',
    '@hello': {
      description: 'Greeting with the user name.',
      placeholders: {
        name: {
          type: 'String',
          example: 'Yuki',
        },
      },
    },
  });

  const report = await buildI18nCheck({
    rootDir,
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) => issue.type === 'arb' && issue.file === 'app/lib/l10n/app_ja.arb',
    ),
  );
});

test('validates sidecar and release metadata JSON when present', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);
  await writeText(rootDir, 'app/lib/l10n/app_ja_intentions.json', '{');
  await writeJson(rootDir, 'release/store_metadata/source/ja.json', {
    title: 'STONE SIGNATURE',
  });

  const parsed = await validateJsonContentFiles({ rootDir });

  assert.ok(parsed.some((entry) => entry.path === 'release/store_metadata/source/ja.json' && entry.ok));
  assert.ok(
    parsed.some(
      (entry) => entry.path === 'app/lib/l10n/app_ja_intentions.json' && !entry.ok,
    ),
  );
});

test('renders a stable check summary', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalEnabledContent(rootDir);
  const report = await buildI18nCheck({ rootDir });

  const rendered = renderI18nCheck(report);

  assert.match(rendered, /^# Stone Signature i18n check/);
  assert.match(rendered, /Result: pass/);
  assert.match(rendered, /## Status/);
  assert.match(rendered, /## Todo/);
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-check-'));
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
    createLanguage('fr', {
      bcp47: 'fr',
      native_name: 'Français',
      english_name: 'French',
    }),
  ]);
  return rootDir;
}

async function writeMinimalEnabledContent(rootDir, { skip = new Set() } = {}) {
  const files = new Map([
    [
      'app/lib/l10n/app_en.arb',
      {
        '@@locale': 'en',
        hello: 'Hello {name}',
        '@hello': {
          description: 'Greeting with the user name.',
          placeholders: {
            name: {
              type: 'String',
              example: 'Yuki',
            },
          },
        },
      },
    ],
    [
      'app/lib/l10n/app_ja.arb',
      {
        '@@locale': 'ja',
        hello: 'こんにちは {name}',
        '@hello': {
          description: 'Greeting with the user name.',
          placeholders: {
            name: {
              type: 'String',
              example: 'Yuki',
            },
          },
        },
      },
    ],
    ['app/assets/i18n/settings/en.json', { about: { heading: 'About' } }],
    ['app/assets/i18n/settings/ja.json', { about: { heading: '概要' } }],
    ['api/content/i18n/checkout/en.json', { product_name_template: 'Stone seal' }],
    ['api/content/i18n/checkout/ja.json', { product_name_template: '宝石印鑑' }],
    ['api/content/i18n/catalog/materials.json', { jade: { label: { en: 'Jade', ja: '翡翠' } } }],
    ['api/content/i18n/catalog/stone_listings.json', {}],
    ['api/content/i18n/catalog/facet_tags.json', {}],
    ['api/content/i18n/catalog/countries.json', {}],
  ]);

  for (const namespace of [
    'common',
    'top',
    'design',
    'about',
    'blog_index',
    'payment_success',
    'payment_failure',
    'terms',
    'commercial_transactions',
  ]) {
    files.set(`web/content/i18n/${namespace}/en.json`, {
      brand_subtitle: 'Seal Field',
      footer_company: 'Company',
    });
    files.set(`web/content/i18n/${namespace}/ja.json`, {
      brand_subtitle: '印鑑フィールド',
      footer_company: '会社',
    });
  }

  for (const [relativePath, value] of files) {
    if (!skip.has(relativePath)) {
      await writeJson(rootDir, relativePath, value);
    }
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
