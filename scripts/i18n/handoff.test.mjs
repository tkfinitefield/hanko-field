import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { exportHandoff, importHandoff, validateHandoff } from './handoff.mjs';

test('exports deterministic handoff files with source, target, and sidecar data', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'web/content/i18n/common/en.json', {
    nav: { home: 'Home', about: 'About' },
  });
  await writeJson(rootDir, 'web/content/i18n/common/ja.json', {
    nav: { home: 'ホーム' },
  });

  const options = {
    rootDir,
    langs: ['ja'],
    file: 'web/content/i18n/common/ja.json',
  };
  const first = await exportHandoff(options);
  const second = await exportHandoff(options);

  assert.deepEqual(second, first);
  assert.deepEqual(first.target_locales, ['ja']);
  assert.deepEqual(first.files, [
    {
      kind: 'json',
      source_file: 'web/content/i18n/common/en.json',
      target_file: 'web/content/i18n/common/ja.json',
      target_locale: 'ja',
      sidecar_file: 'web/content/i18n/common/ja_intentions.json',
      entries: [
        {
          key_path: 'nav.home',
          source_value: 'Home',
          target_value: 'ホーム',
          status: 'present',
        },
        {
          key_path: 'nav.about',
          source_value: 'About',
          target_value: '',
          status: 'missing',
        },
      ],
    },
  ]);
});

test('imports handoff values without reordering existing JSON keys', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'web/content/i18n/common/en.json', {
    first: 'First',
    second: 'Second',
    nested: { alpha: 'Alpha', beta: 'Beta' },
  });
  await writeJson(rootDir, 'web/content/i18n/common/ja.json', {
    second: '古い2',
    first: '古い1',
    nested: { beta: '古いB', alpha: '古いA' },
  });

  const changedFiles = await importHandoff({
    rootDir,
    handoff: createHandoff([
      {
        kind: 'json',
        source_file: 'web/content/i18n/common/en.json',
        target_file: 'web/content/i18n/common/ja.json',
        target_locale: 'ja',
        sidecar_file: 'web/content/i18n/common/ja_intentions.json',
        entries: [
          entry('first', 'First', '新しい1'),
          entry('second', 'Second', '新しい2'),
          entry('nested.alpha', 'Alpha', '新しいA'),
          entry('nested.beta', 'Beta', '新しいB'),
        ],
      },
    ]),
  });

  assert.deepEqual(changedFiles, ['web/content/i18n/common/ja.json']);
  const raw = await readFile(join(rootDir, 'web/content/i18n/common/ja.json'), 'utf8');
  const imported = JSON.parse(raw);
  assert.equal(imported.first, '新しい1');
  assert.equal(imported.second, '新しい2');
  assert.equal(imported.nested.alpha, '新しいA');
  assert.equal(imported.nested.beta, '新しいB');
  assert.ok(raw.indexOf('"second"') < raw.indexOf('"first"'));
  assert.ok(raw.indexOf('"beta"') < raw.indexOf('"alpha"'));
});

test('imports missing target files from source skeleton in source key order', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'app/assets/i18n/settings/en.json', {
    first: 'First',
    second: 'Second',
    count: 2,
    nested: { label: 'Label' },
  });

  await importHandoff({
    rootDir,
    handoff: createHandoff([
      {
        kind: 'json',
        source_file: 'app/assets/i18n/settings/en.json',
        target_file: 'app/assets/i18n/settings/fr.json',
        target_locale: 'fr',
        sidecar_file: 'app/assets/i18n/settings/fr_intentions.json',
        entries: [
          entry('first', 'First', 'Premier'),
          entry('second', 'Second', 'Deuxieme'),
          entry('nested.label', 'Label', 'Libelle'),
        ],
      },
    ]),
  });

  const raw = await readFile(join(rootDir, 'app/assets/i18n/settings/fr.json'), 'utf8');
  assert.ok(raw.indexOf('"first"') < raw.indexOf('"second"'));
  assert.ok(raw.indexOf('"second"') < raw.indexOf('"count"'));
  assert.deepEqual(JSON.parse(raw), {
    first: 'Premier',
    second: 'Deuxieme',
    count: 2,
    nested: { label: 'Libelle' },
  });
});

test('imports missing ARB files with the target locale metadata', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', {
    '@@locale': 'en',
    hello: 'Hello',
    '@hello': { description: 'Greeting' },
  });

  await importHandoff({
    rootDir,
    handoff: createHandoff([
      {
        kind: 'arb',
        source_file: 'app/lib/l10n/app_en.arb',
        target_file: 'app/lib/l10n/app_zh_Hant.arb',
        target_locale: 'zhtw',
        sidecar_file: 'app/lib/l10n/app_zh_Hant_intentions.json',
        entries: [entry('hello', 'Hello', '您好')],
      },
    ]),
  });

  assert.deepEqual(JSON.parse(await readFile(join(rootDir, 'app/lib/l10n/app_zh_Hant.arb'), 'utf8')), {
    '@@locale': 'zh_Hant',
    hello: '您好',
    '@hello': { description: 'Greeting' },
  });
});

test('imports multiple catalog locales into the same file in one pass', async () => {
  const rootDir = await createTempRoot();
  await writeJson(rootDir, 'api/content/i18n/catalog/materials.json', {
    jade: {
      label: { en: 'Jade' },
      description: { en: 'Green stone' },
    },
  });

  const changedFiles = await importHandoff({
    rootDir,
    handoff: createHandoff([
      {
        kind: 'catalog',
        source_file: 'api/content/i18n/catalog/materials.json',
        target_file: 'api/content/i18n/catalog/materials.json',
        target_locale: 'ja',
        sidecar_file: 'api/content/i18n/catalog/materials_intentions.json',
        entries: [entry('jade.label', 'Jade', '翡翠')],
      },
      {
        kind: 'catalog',
        source_file: 'api/content/i18n/catalog/materials.json',
        target_file: 'api/content/i18n/catalog/materials.json',
        target_locale: 'fr',
        sidecar_file: 'api/content/i18n/catalog/materials_intentions.json',
        entries: [entry('jade.label', 'Jade', 'Jade FR')],
      },
    ]),
  });

  assert.deepEqual(changedFiles, ['api/content/i18n/catalog/materials.json']);
  const catalog = JSON.parse(await readFile(join(rootDir, 'api/content/i18n/catalog/materials.json'), 'utf8'));
  assert.equal(catalog.jade.label.en, 'Jade');
  assert.equal(catalog.jade.label.ja, '翡翠');
  assert.equal(catalog.jade.label.fr, 'Jade FR');
});

test('validates the handoff envelope before import', () => {
  assert.throws(() => validateHandoff({ format_version: 2, files: [] }), /format_version must be 1/);
  assert.throws(
    () =>
      validateHandoff({
        format_version: 1,
        files: [{ kind: 'json', source_file: '', target_file: 'x', target_locale: 'ja', sidecar_file: 'x', entries: [] }],
      }),
    /source_file must be a non-empty string/,
  );
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-handoff-'));
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
      web: { enabled: true, indexed: true, url_prefix: 'ja' },
      app: { enabled: true, selectable: true },
    }),
    createLanguage('fr', {
      bcp47: 'fr',
      native_name: 'Français',
      english_name: 'French',
      web: { enabled: true, indexed: false, url_prefix: 'fr' },
      app: { enabled: true, selectable: false },
    }),
  ]);
  return rootDir;
}

function createHandoff(files) {
  return {
    format_version: 1,
    source_locale: 'en',
    target_locales: [...new Set(files.map((file) => file.target_locale))],
    file_filter: null,
    files,
  };
}

function entry(keyPath, sourceValue, targetValue) {
  return {
    key_path: keyPath,
    source_value: sourceValue,
    target_value: targetValue,
    status: targetValue ? 'present' : 'missing',
  };
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
