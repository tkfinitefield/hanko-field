import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildI18nTodo } from './todo.mjs';
import { ensureLocaleStubs } from './stubs.mjs';

test('creates missing locale files as translatable stubs', async () => {
  const rootDir = await createTempRoot();

  const report = await ensureLocaleStubs({ rootDir, langs: ['fr'] });

  assert.equal(report.created_files.length, 4);
  assert.deepEqual(report.missing_files, report.created_files);

  const arb = JSON.parse(await readFile(join(rootDir, 'app/lib/l10n/app_fr.arb'), 'utf8'));
  assert.deepEqual(arb, {
    '@@locale': 'fr',
    '@helloUser': { description: 'Greeting', placeholders: { name: {} } },
    count: 2,
  });

  const settings = JSON.parse(await readFile(join(rootDir, 'app/assets/i18n/settings/fr.json'), 'utf8'));
  assert.deepEqual(settings, {
    about: {
      points: [{}, {}],
      count: 2,
    },
  });

  const todo = await buildI18nTodo({
    rootDir,
    langs: ['fr'],
    file: 'app/lib/l10n/app_fr.arb',
  });
  assert.equal(todo.items.some((item) => item.type === 'missing-file'), false);
  assert.ok(todo.items.some((item) => item.type === 'missing-key' && item.file === 'app/lib/l10n/app_fr.arb'));
});

test('check mode fails before generation and passes after generation', async () => {
  const rootDir = await createTempRoot();

  const before = await ensureLocaleStubs({ rootDir, langs: ['fr'], check: true });
  assert.equal(before.ok, false);
  assert.equal(before.missing_files.length, 4);

  await ensureLocaleStubs({ rootDir, langs: ['fr'] });
  const after = await ensureLocaleStubs({ rootDir, langs: ['fr'], check: true });

  assert.equal(after.ok, true);
  assert.deepEqual(after.missing_files, []);
  assert.equal(after.existing_files.length, 4);
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-stubs-'));
  await writeJson(rootDir, 'config/languages.json', [
    createLanguage('en', { fallback: null }),
    createLanguage('fr', {}),
  ]);
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', {
    '@@locale': 'en',
    helloUser: 'Hello {name}',
    '@helloUser': { description: 'Greeting', placeholders: { name: {} } },
    count: 2,
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/en.json', {
    about: {
      heading: 'About',
      points: [{ title: 'First' }, { title: 'Second' }],
      count: 2,
    },
  });
  await writeJson(rootDir, 'web/content/i18n/common/en.json', {
    nav: { home: 'Home' },
  });
  await writeJson(rootDir, 'api/content/i18n/checkout/en.json', {
    product_name_template: 'Stone seal',
  });
  return rootDir;
}

function createLanguage(routeCode, { fallback = 'en' }) {
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
    text_direction: 'ltr',
    fallback,
    currency: 'USD',
    web: {
      enabled: routeCode === 'en',
      indexed: false,
      url_prefix: routeCode === 'en' ? '' : routeCode,
    },
    app: {
      enabled: routeCode === 'en',
      selectable: false,
    },
    release: {
      enabled: false,
      android_store_locale: null,
      ios_store_locale: null,
    },
  };
}

async function writeJson(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), `${JSON.stringify(value, null, 2)}\n`);
}
