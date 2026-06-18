import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readdir, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildI18nTodo, parseLangsFilter, renderI18nTodo } from './todo.mjs';

test('reports missing keys with English and fallback values', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', {
    '@@locale': 'en',
    hello: 'Hello',
    '@hello': { description: 'Greeting' },
    goodbye: 'Goodbye',
  });
  await writeJson(rootDir, 'app/lib/l10n/app_ja.arb', {
    '@@locale': 'ja',
    hello: 'こんにちは',
    '@hello': { description: 'Greeting' },
  });

  const report = await buildI18nTodo({
    rootDir,
    langs: ['ja'],
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.deepEqual(
    report.items.map((item) => ({
      file: item.file,
      locale: item.locale,
      key: item.key,
      base: item.base_english_value,
      fallback: item.fallback_value,
      sidecar: item.sidecar_path,
    })),
    [
      {
        file: 'app/lib/l10n/app_ja.arb',
        locale: 'ja',
        key: 'goodbye',
        base: 'Goodbye',
        fallback: 'Goodbye',
        sidecar: 'app/lib/l10n/app_ja_intentions.json',
      },
    ],
  );
});

test('LANGS and FILE filters isolate the requested target file', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'web/content/i18n/common/en.json', {
    nav: { home: 'Home', about: 'About' },
  });
  await writeJson(rootDir, 'web/content/i18n/common/ja.json', {
    nav: { home: 'ホーム' },
  });
  await writeJson(rootDir, 'web/content/i18n/top/en.json', {
    hero: 'Custom gemstone seal',
  });

  const report = await buildI18nTodo({
    rootDir,
    langs: parseLangsFilter('ja,fr'),
    file: 'web/content/i18n/common/ja.json',
  });

  assert.deepEqual(report.languages, ['ja', 'fr']);
  assert.deepEqual(
    report.items.map((item) => `${item.locale}:${item.file}:${item.key}`),
    ['ja:web/content/i18n/common/ja.json:nav.about'],
  );
});

test('reports missing files as missing keys from the English base file', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'api/content/i18n/checkout/en.json', {
    product_name_template: 'Stone seal ({listing_label})',
    product_description_template: 'Custom stone seal order',
  });

  const report = await buildI18nTodo({
    rootDir,
    langs: ['fr'],
    file: 'api/content/i18n/checkout/fr.json',
  });

  assert.deepEqual(
    report.items.map((item) => [item.type, item.file, item.locale, item.key]),
    [
      [
        'missing-file',
        'api/content/i18n/checkout/fr.json',
        'fr',
        'product_description_template',
      ],
      ['missing-file', 'api/content/i18n/checkout/fr.json', 'fr', 'product_name_template'],
    ],
  );
});

test('reports catalog language-map gaps from embedded English values', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'api/content/i18n/catalog/materials.json', {
    jade: {
      label: { en: 'Jade', ja: '翡翠' },
      description: { en: 'A green stone.' },
    },
  });

  const report = await buildI18nTodo({
    rootDir,
    langs: ['ja'],
    file: 'api/content/i18n/catalog/materials.json',
  });

  assert.deepEqual(
    report.items.map((item) => ({
      file: item.file,
      locale: item.locale,
      key: item.key,
      base: item.base_english_value,
      fallback: item.fallback_value,
      sidecar: item.sidecar_path,
    })),
    [
      {
        file: 'api/content/i18n/catalog/materials.json',
        locale: 'ja',
        key: 'jade.description',
        base: 'A green stone.',
        fallback: 'A green stone.',
        sidecar: 'api/content/i18n/catalog/materials_intentions.json',
      },
    ],
  );
});

test('renders a stable Markdown report', async () => {
  const rendered = renderI18nTodo({
    languages: ['ja'],
    file_filter: null,
    items: [
      {
        type: 'missing-key',
        file: 'web/content/i18n/common/ja.json',
        locale: 'ja',
        key: 'footer.company',
        base_english_value: 'Company',
        fallback_value: 'Company',
        sidecar_path: 'web/content/i18n/common/ja_intentions.json',
      },
    ],
  });

  assert.match(rendered, /^# Stone Signature i18n todo/);
  assert.match(rendered, /Items: 1/);
  assert.match(rendered, /\| file \| locale \| key \| base English value \| fallback value \| sidecar path \|/);
  assert.match(rendered, /web\/content\/i18n\/common\/ja\.json/);
});

test('rejects unknown LANGS route codes', async () => {
  const rootDir = await createTempRoot();

  await assert.rejects(
    () => buildI18nTodo({ rootDir, langs: ['missing'] }),
    /Unknown LANGS route code\(s\): missing/,
  );
});

test('does not modify the workspace while building todo', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', { hello: 'Hello' });
  const before = await listFiles(rootDir);

  await buildI18nTodo({ rootDir, langs: ['ja'], file: 'app/lib/l10n/app_ja.arb' });

  assert.deepEqual(await listFiles(rootDir), before);
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-todo-'));
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

async function writeJson(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), `${JSON.stringify(value, null, 2)}\n`);
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
