import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import test from 'node:test';

import {
  buildI18nStatus,
  expectedAppFiles,
  expectedReleaseFiles,
  renderI18nStatus,
} from './status.mjs';

test('reports missing files for enabled app, web, and API languages', async () => {
  const rootDir = await createTempRegistryRoot();

  await mkdir(join(rootDir, 'app/lib/l10n'), { recursive: true });
  await writeFile(join(rootDir, 'app/lib/l10n/app_en.arb'), '{}\n');

  const status = await buildI18nStatus({ rootDir });
  const rendered = renderI18nStatus(status);

  assert.equal(status.total_languages, 3);
  assert.deepEqual(status.app_enabled, ['en', 'ja']);
  assert.deepEqual(status.web_enabled, ['en', 'ja']);
  assert.deepEqual(status.release_enabled, []);

  assert.match(rendered, /app: 1\/4 present/);
  assert.match(rendered, /app\/lib\/l10n\/app_ja\.arb/);
  assert.match(rendered, /web\/content\/i18n\/common\/en\.json/);
  assert.match(rendered, /api\/content\/i18n\/checkout\/ja\.json/);
  assert.match(rendered, /release: 0\/0 present/);
  assert.match(rendered, /missing: none/);
});

test('does not modify the workspace while building status', async () => {
  const rootDir = await createTempRegistryRoot();
  const before = await listFiles(rootDir);

  await buildI18nStatus({ rootDir });

  assert.deepEqual(await listFiles(rootDir), before);
});

test('uses expected file names for Flutter and release metadata', () => {
  const languages = [
    createLanguage('zh', {
      flutter: { languageCode: 'zh', scriptCode: 'Hans', countryCode: null },
      app: { enabled: true, selectable: false },
      release: { enabled: true, android_store_locale: 'zh-CN', ios_store_locale: 'zh-Hans' },
    }),
    createLanguage('zhtw', {
      bcp47: 'zh-Hant',
      flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
      app: { enabled: true, selectable: false },
      release: { enabled: true, android_store_locale: 'zh-TW', ios_store_locale: 'zh-Hant' },
    }),
  ];

  assert.deepEqual(
    expectedAppFiles(languages).map((file) => file.path),
    [
      'app/lib/l10n/app_zh.arb',
      'app/assets/i18n/settings/zh.json',
      'app/lib/l10n/app_zh_Hant.arb',
      'app/assets/i18n/settings/zhtw.json',
    ],
  );
  assert.deepEqual(
    expectedReleaseFiles(languages).map((file) => file.path),
    [
      'release/store_metadata/source/zh.json',
      'release/store_metadata/source/zhtw.json',
    ],
  );
});

async function createTempRegistryRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-status-'));
  await mkdir(join(rootDir, 'config'), { recursive: true });
  await writeFile(
    join(rootDir, 'config/languages.json'),
    `${JSON.stringify([
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
    ], null, 2)}\n`,
  );
  return rootDir;
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

async function listFiles(rootDir) {
  const results = [];

  async function visit(relativeDir) {
    const entries = await readdir(join(rootDir, relativeDir), { withFileTypes: true });
    for (const entry of entries) {
      const relativePath = relativeDir ? `${relativeDir}/${entry.name}` : entry.name;
      if (entry.isDirectory()) {
        await visit(relativePath);
      } else if (entry.isFile()) {
        results.push(relativePath);
      }
    }
  }

  await visit('');
  return results.sort();
}
