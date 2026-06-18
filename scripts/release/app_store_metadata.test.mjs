import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  AppStoreMetadataError,
  buildAppStoreMetadataFiles,
  checkAppStoreMetadata,
  generateAppStoreMetadata,
} from './app_store_metadata.mjs';

test('builds App Store metadata text files from source metadata', async () => {
  const rootDir = await createTempRoot();

  const files = await buildAppStoreMetadataFiles({ rootDir });

  assert.equal(files.get('release/store_metadata/app_store/en-US/name.txt'), 'STONE SIGNATURE\n');
  assert.equal(
    files.get('release/store_metadata/app_store/en-US/subtitle.txt'),
    'Custom gemstone seals\n',
  );
  assert.equal(
    files.get('release/store_metadata/app_store/en-US/description.txt'),
    'Create a personal seal from natural gemstone.\n\nChoose a one-of-a-kind stone and proceed to secure checkout.\n',
  );
  assert.equal(
    files.get('release/store_metadata/app_store/en-US/keywords.txt'),
    'hanko,seal,gemstone\n',
  );
  assert.equal(
    files.get('release/store_metadata/app_store/en-US/release_notes.txt'),
    'Added multilingual store metadata source files.\n',
  );
  assert.equal(
    files.get('release/store_metadata/app_store/en-US/privacy_url.txt'),
    'https://finitefield.org/privacy/\n',
  );
});

test('generates deterministic App Store metadata and check mode validates it', async () => {
  const rootDir = await createTempRoot();

  await generateAppStoreMetadata({ rootDir });
  const report = await checkAppStoreMetadata({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.equal(
    await readFile(join(rootDir, 'release/store_metadata/app_store/ja/name.txt'), 'utf8'),
    'STONE SIGNATURE\n',
  );
});

test('check mode reports stale generated files', async () => {
  const rootDir = await createTempRoot();
  await generateAppStoreMetadata({ rootDir });
  await writeFile(
    join(rootDir, 'release/store_metadata/app_store/en-US/name.txt'),
    'Old title\n',
  );

  const report = await checkAppStoreMetadata({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'app-store-stale-file' &&
        issue.file === 'release/store_metadata/app_store/en-US/name.txt',
    ),
  );
});

test('fails clearly when a source locale has no App Store locale', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, iosStoreLocale: 'en-US' }),
      createLanguage('ar', { bcp47: 'ar', iosStoreLocale: null }),
    ],
    sources: ['en', 'ar'],
  });

  await assert.rejects(
    () => buildAppStoreMetadataFiles({ rootDir, requiredLocales: ['en', 'ar'] }),
    (error) =>
      error instanceof AppStoreMetadataError &&
      error.issues.some(
        (issue) =>
          issue.code === 'app-store-unsupported-locale' &&
          issue.file === 'release/store_metadata/source/ar.json',
      ),
  );
});

test('fails clearly when an App Store locale is not supported by deliver', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, iosStoreLocale: 'en-US' }),
      createLanguage('xx', { bcp47: 'xx', iosStoreLocale: 'xx-YY' }),
    ],
    sources: ['en', 'xx'],
  });

  await assert.rejects(
    () => buildAppStoreMetadataFiles({ rootDir, requiredLocales: ['en', 'xx'] }),
    (error) =>
      error instanceof AppStoreMetadataError &&
      error.issues.some(
        (issue) =>
          issue.code === 'app-store-unsupported-locale' &&
          issue.key === 'release.ios_store_locale' &&
          issue.message.includes('supported fastlane deliver locale list'),
      ),
  );
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null, iosStoreLocale: 'en-US' }),
    createLanguage('ja', { bcp47: 'ja', iosStoreLocale: 'ja' }),
    createLanguage('zh', { bcp47: 'zh-Hans', iosStoreLocale: 'zh-Hans' }),
    createLanguage('zhtw', { bcp47: 'zh-Hant', iosStoreLocale: 'zh-Hant' }),
  ],
  sources = ['en', 'ja', 'zh', 'zhtw'],
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-app-store-'));
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

function createLanguage(routeCode, { bcp47 = routeCode, fallback = 'en', iosStoreLocale }) {
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
      android_store_locale: null,
      ios_store_locale: iosStoreLocale,
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
