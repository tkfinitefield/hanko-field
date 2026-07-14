import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  buildFreezeReport,
  createFreezeManifest,
  deriveLanguageSet,
  renderFreezeReport,
} from './freeze.mjs';

test('derives the frozen language set from enabled app and web flags', () => {
  const languageSet = deriveLanguageSet(
    [
      createLanguage('en', { fallback: null, web: { enabled: true, indexed: true, url_prefix: '' }, app: { enabled: true, selectable: true } }),
      createLanguage('ja', { web: { enabled: true, indexed: true, url_prefix: 'ja' }, app: { enabled: true, selectable: true } }),
      createLanguage('fr', { web: { enabled: true, indexed: false, url_prefix: 'fr' } }),
      createLanguage('de', {}),
    ],
    ['en', 'ja'],
  );

  assert.deepEqual(languageSet, {
    registry_languages: 4,
    app_enabled: ['en', 'ja'],
    web_enabled: ['en', 'fr', 'ja'],
    release_enabled: [],
    frozen_locales: ['en', 'fr', 'ja'],
    store_metadata_source_locales: ['en', 'ja'],
  });
});

test('passes when the freeze manifest matches current files', async () => {
  const rootDir = await createTempRoot();
  const manifest = await createFreezeManifest({ rootDir, frozenAt: '2026-06-18' });
  await writeJson(rootDir, 'doc/qa/m9-t06/translation-freeze.json', manifest);

  const report = await buildFreezeReport({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.deepEqual(report.frozen_locales, ['en', 'ja']);
  assert.match(renderFreezeReport(report), /Result: pass/);
});

test('fails when a frozen translation file changes after freeze', async () => {
  const rootDir = await createTempRoot();
  const manifest = await createFreezeManifest({ rootDir, frozenAt: '2026-06-18' });
  await writeJson(rootDir, 'doc/qa/m9-t06/translation-freeze.json', manifest);
  await writeJson(rootDir, 'app/assets/i18n/settings/ja.json', {
    about: '変更後',
  });

  const report = await buildFreezeReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'freeze-file-hash'));
});

test('fails when enabled languages change without a freeze manifest update', async () => {
  const rootDir = await createTempRoot();
  const manifest = await createFreezeManifest({ rootDir, frozenAt: '2026-06-18' });
  await writeJson(rootDir, 'doc/qa/m9-t06/translation-freeze.json', manifest);
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
    createLanguage('fr', {
      web: { enabled: true, indexed: false, url_prefix: 'fr' },
      app: { enabled: true, selectable: false },
    }),
  ]);
  await writeJson(rootDir, 'app/lib/l10n/app_fr.arb', {
    '@@locale': 'fr',
    appTitle: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/fr.json', {
    about: 'Francais',
  });
  await writeJson(rootDir, 'web/content/i18n/common/fr.json', {
    headline: 'Francais',
  });
  await writeJson(rootDir, 'api/content/i18n/checkout/fr.json', {
    checkout: 'Francais',
  });

  const report = await buildFreezeReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'freeze-language-set'));
  assert.ok(report.issues.some((issue) => issue.code === 'freeze-file-missing'));
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-freeze-'));
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
    createLanguage('fr', {}),
  ]);
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', {
    '@@locale': 'en',
    appTitle: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'app/lib/l10n/app_ja.arb', {
    '@@locale': 'ja',
    appTitle: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'app/lib/l10n/app_ja_intentions.json', {
    locale: 'ja',
    entries: [{ key: 'appTitle', reason: 'brand_name' }],
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/en.json', {
    about: 'About',
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/ja.json', {
    about: '概要',
  });
  await writeJson(rootDir, 'web/content/i18n/common/en.json', {
    headline: 'Hello',
  });
  await writeJson(rootDir, 'web/content/i18n/common/ja.json', {
    headline: 'こんにちは',
  });
  await writeJson(rootDir, 'api/content/i18n/checkout/en.json', {
    checkout: 'Checkout',
  });
  await writeJson(rootDir, 'api/content/i18n/checkout/ja.json', {
    checkout: '決済',
  });
  await writeJson(rootDir, 'api/content/i18n/catalog/materials.json', {
    stone: {
      en: 'Stone',
      ja: '石',
    },
  });
  await writeJson(rootDir, 'release/store_metadata/source/schema.json', {
    title: 'test schema',
  });
  await writeJson(rootDir, 'release/store_metadata/source/en.json', {
    app_name: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'release/store_metadata/source/ja.json', {
    app_name: 'STONE SIGNATURE',
  });
  await writeText(rootDir, 'release/store_metadata/google_play/en-US/title.txt', 'STONE SIGNATURE\n');
  await writeText(rootDir, 'release/store_metadata/app_store/en-US/name.txt', 'STONE SIGNATURE\n');
  await writeJson(rootDir, 'release/store_metadata/screenshots/manifest.json', {
    screenshots: [],
  });
  return rootDir;
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
