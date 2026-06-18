import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  compareJsonShape,
  validateCatalogLanguageMaps,
  validateFallbackChains,
  validateJsonShapes,
} from './json_shape.mjs';

test('passes when enabled JSON content has matching shape', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);

  const report = await validateJsonShapes({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.ok(report.parsed_files.includes('app/assets/i18n/settings/en.json'));
  assert.ok(report.parsed_files.includes('web/content/i18n/common/ja.json'));
});

test('fails when a target JSON scalar type differs from English', () => {
  const issues = compareJsonShape(
    { about: { heading: 'About' } },
    { about: { heading: { text: '概要' } } },
    'app/assets/i18n/settings/ja.json',
    'app/assets/i18n/settings/en.json',
  );

  assert.deepEqual(
    issues.map((issue) => [issue.code, issue.key]),
    [['json-shape-type', 'about.heading']],
  );
});

test('fails when target arrays do not preserve source shape', () => {
  const issues = compareJsonShape(
    { points: [{ title: 'One', body: 'Body' }, { title: 'Two', body: 'Body' }] },
    { points: [{ title: '一', body: '本文' }] },
    'app/assets/i18n/settings/ja.json',
    'app/assets/i18n/settings/en.json',
  );

  assert.ok(issues.some((issue) => issue.code === 'json-shape-array-length' && issue.key === 'points'));
});

test('fails when catalog language map values change type', () => {
  const issues = validateCatalogLanguageMaps(
    {
      jade: {
        label: {
          en: 'Jade',
          ja: { text: '翡翠' },
        },
      },
    },
    'api/content/i18n/catalog/materials.json',
    [createLanguage('ja', { web: { enabled: true, indexed: true, url_prefix: 'ja' } })],
  );

  assert.deepEqual(
    issues.map((issue) => [issue.code, issue.key]),
    [['json-shape-catalog-type', 'jade.label.ja']],
  );
});

test('fails when fallback points to a disabled language for an enabled surface', () => {
  const issues = validateFallbackChains([
    createLanguage('en', { fallback: null, web: { enabled: true, indexed: true, url_prefix: '' } }),
    createLanguage('ja', { fallback: 'fr', web: { enabled: true, indexed: true, url_prefix: 'ja' } }),
    createLanguage('fr', { web: { enabled: false, indexed: false, url_prefix: 'fr' } }),
  ]);

  assert.ok(
    issues.some(
      (issue) => issue.code === 'fallback-disabled' && issue.key === 'ja' && issue.message === 'web fallback fr is disabled',
    ),
  );
});

test('fails when fallback chains contain a cycle', () => {
  const issues = validateFallbackChains([
    createLanguage('en', { fallback: 'ja' }),
    createLanguage('ja', { fallback: 'en' }),
  ]);

  assert.ok(issues.some((issue) => issue.code === 'fallback-cycle' && issue.key === 'en'));
});

test('file filter validates only the requested JSON family', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'web/content/i18n/common/ja.json', {
    brand_subtitle: ['印鑑フィールド'],
    footer_company: '会社',
  });

  const report = await validateJsonShapes({
    rootDir,
    file: 'web/content/i18n/common/ja.json',
  });

  assert.equal(report.ok, false);
  assert.deepEqual(report.parsed_files.sort(), [
    'web/content/i18n/common/en.json',
    'web/content/i18n/common/ja.json',
  ]);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'json-shape-type' &&
        issue.file === 'web/content/i18n/common/ja.json' &&
        issue.key === 'brand_subtitle',
    ),
  );
});

test('validates release metadata shape when source metadata exists', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, release: { enabled: true, android_store_locale: 'en-US', ios_store_locale: 'en-US' } }),
      createLanguage('ja', { release: { enabled: true, android_store_locale: 'ja-JP', ios_store_locale: 'ja' } }),
    ],
  });
  await writeJson(rootDir, 'release/store_metadata/source/en.json', {
    title: 'STONE SIGNATURE',
    screenshots: ['home', 'settings'],
  });
  await writeJson(rootDir, 'release/store_metadata/source/ja.json', {
    title: 'STONE SIGNATURE',
    screenshots: { home: 'ホーム' },
  });

  const report = await validateJsonShapes({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'json-shape-type' &&
        issue.file === 'release/store_metadata/source/ja.json' &&
        issue.key === 'screenshots',
    ),
  );
});

async function createTempRoot({ languages = null } = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-json-shape-'));
  await writeJson(rootDir, 'config/languages.json', languages ?? [
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

async function writeMinimalContent(rootDir) {
  const files = new Map([
    ['app/assets/i18n/settings/en.json', { about: { heading: 'About' } }],
    ['app/assets/i18n/settings/ja.json', { about: { heading: '概要' } }],
    ['api/content/i18n/checkout/en.json', { product_name_template: 'Stone seal' }],
    ['api/content/i18n/checkout/ja.json', { product_name_template: '宝石印鑑' }],
    ['api/content/i18n/catalog/materials.json', { jade: { label: { en: 'Jade', ja: '翡翠' } } }],
    ['api/content/i18n/catalog/stone_listings.json', {}],
    ['api/content/i18n/catalog/facet_tags.json', {}],
    ['api/content/i18n/catalog/countries.json', {}],
    ['web/content/i18n/common/en.json', { brand_subtitle: 'Seal Field', footer_company: 'Company' }],
    ['web/content/i18n/common/ja.json', { brand_subtitle: '印鑑フィールド', footer_company: '会社' }],
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
