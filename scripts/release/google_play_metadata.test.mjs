import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  GooglePlayMetadataError,
  buildGooglePlayMetadataFiles,
  checkGooglePlayMetadata,
  generateGooglePlayMetadata,
} from './google_play_metadata.mjs';

test('builds Google Play metadata text files from source metadata', async () => {
  const rootDir = await createTempRoot();

  const files = await buildGooglePlayMetadataFiles({ rootDir });

  assert.equal(files.get('release/store_metadata/google_play/en-US/title.txt'), 'STONE SIGNATURE\n');
  assert.equal(
    files.get('release/store_metadata/google_play/en-US/short_description.txt'),
    'Design and order a custom gemstone seal.\n',
  );
  assert.equal(
    files.get('release/store_metadata/google_play/en-US/changelogs/default.txt'),
    'Added multilingual store metadata source files.\n',
  );
});

test('generates deterministic Google Play metadata and check mode validates it', async () => {
  const rootDir = await createTempRoot();

  await generateGooglePlayMetadata({ rootDir });
  const report = await checkGooglePlayMetadata({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.equal(
    await readFile(join(rootDir, 'release/store_metadata/google_play/ja-JP/title.txt'), 'utf8'),
    'STONE SIGNATURE\n',
  );
});

test('check mode reports stale generated files', async () => {
  const rootDir = await createTempRoot();
  await generateGooglePlayMetadata({ rootDir });
  await writeFile(
    join(rootDir, 'release/store_metadata/google_play/en-US/title.txt'),
    'Old title\n',
  );

  const report = await checkGooglePlayMetadata({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'google-play-stale-file' &&
        issue.file === 'release/store_metadata/google_play/en-US/title.txt',
    ),
  );
});

test('fails clearly when a source locale has no Google Play store locale', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, androidStoreLocale: 'en-US' }),
      createLanguage('ar', { bcp47: 'ar', androidStoreLocale: null }),
    ],
    sources: ['en', 'ar'],
  });

  await assert.rejects(
    () => buildGooglePlayMetadataFiles({ rootDir, requiredLocales: ['en', 'ar'] }),
    (error) =>
      error instanceof GooglePlayMetadataError &&
      error.issues.some(
        (issue) =>
          issue.code === 'google-play-unsupported-locale' &&
          issue.file === 'release/store_metadata/source/ar.json',
      ),
  );
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null, androidStoreLocale: 'en-US' }),
    createLanguage('ja', { bcp47: 'ja', androidStoreLocale: 'ja-JP' }),
    createLanguage('zh', { bcp47: 'zh-Hans', androidStoreLocale: 'zh-CN' }),
    createLanguage('zhtw', { bcp47: 'zh-Hant', androidStoreLocale: 'zh-TW' }),
  ],
  sources = ['en', 'ja', 'zh', 'zhtw'],
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-google-play-'));
  await mkdir(join(rootDir, 'config'), { recursive: true });
  await mkdir(join(rootDir, 'release/store_metadata/source'), { recursive: true });
  await writeJson(join(rootDir, 'config/languages.json'), languages);
  await writeJson(join(rootDir, 'release/store_metadata/source/schema.json'), {
    title: 'STONE SIGNATURE store metadata source',
    required: [
      'app_name',
      'subtitle',
      'short_description',
      'full_description',
      'keywords',
      'release_notes',
      'support_url',
      'marketing_url',
      'privacy_policy_url',
      'screenshot_captions',
    ],
  });
  for (const locale of sources) {
    await writeJson(join(rootDir, `release/store_metadata/source/${locale}.json`), validStoreMetadata());
  }
  return rootDir;
}

function createLanguage(routeCode, { bcp47 = routeCode, fallback = 'en', androidStoreLocale }) {
  return {
    route_code: routeCode,
    bcp47,
    flutter: {
      languageCode: routeCode === 'zhtw' ? 'zh' : routeCode,
      scriptCode: routeCode === 'zhtw' ? 'Hant' : null,
      countryCode: null,
    },
    native_name: routeCode,
    english_name: routeCode,
    text_direction: routeCode === 'ar' ? 'rtl' : 'ltr',
    fallback,
    currency: 'USD',
    web: {
      enabled: true,
      indexed: false,
      url_prefix: routeCode === 'en' ? '' : routeCode,
    },
    app: {
      enabled: true,
      selectable: true,
    },
    release: {
      enabled: false,
      android_store_locale: androidStoreLocale,
      ios_store_locale: null,
    },
  };
}

async function writeJson(path, value) {
  await writeFile(path, `${JSON.stringify(value, null, 2)}\n`);
}

function validStoreMetadata() {
  return {
    app_name: 'STONE SIGNATURE',
    subtitle: 'Custom gemstone seals',
    short_description: 'Design and order a custom gemstone seal.',
    full_description: [
      'Create a personal seal from natural gemstone.',
      'Choose a one-of-a-kind stone and proceed to secure checkout.',
    ],
    keywords: ['hanko', 'seal', 'gemstone'],
    release_notes: {
      '1.0.0': ['Initial release.'],
      '1.1.0': ['Added multilingual store metadata source files.'],
    },
    support_url: 'https://finitefield.org/contact/',
    marketing_url: 'https://finitefield.org/',
    privacy_policy_url: 'https://finitefield.org/privacy/',
    screenshot_captions: {
      design: 'Design your seal impression',
      stones: 'Choose a one-of-a-kind stone',
      checkout: 'Order securely',
    },
  };
}
