import assert from 'node:assert/strict';
import { mkdir, mkdtemp, readFile, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  ScreenshotMetadataError,
  buildScreenshotManifest,
  checkScreenshotMetadata,
  generateScreenshotMetadata,
} from './screenshot_metadata.mjs';

test('builds deterministic screenshot slots from store metadata', async () => {
  const rootDir = await createTempRoot();

  const manifest = await buildScreenshotManifest({ rootDir });
  const en = manifest.locales.find((locale) => locale.route_code === 'en');
  const phone = en.devices.find((device) => device.id === 'phone_6_5');

  assert.equal(manifest.screenshot_keys.join(','), 'design,stones,checkout');
  assert.equal(phone.google_play_image_type, 'phoneScreenshots');
  assert.deepEqual(phone.slots[0], {
    key: 'design',
    position: 1,
    caption: 'Design your seal impression',
    source_path: 'release/store_metadata/screenshots/source/en/phone_6_5/01-design.png',
    google_play_path: 'release/store_metadata/screenshots/google_play/en-US/phone_6_5/01-design.png',
    app_store_path: 'release/store_metadata/screenshots/app_store/en-US/phone_6_5/01-design.png',
  });
});

test('generates screenshot metadata and check mode validates it', async () => {
  const rootDir = await createTempRoot();

  await generateScreenshotMetadata({ rootDir });
  const report = await checkScreenshotMetadata({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.match(
    await readFile(join(rootDir, 'release/store_metadata/screenshots/manifest.json'), 'utf8'),
    /"route_code": "ja"/,
  );
});

test('check mode reports stale screenshot metadata', async () => {
  const rootDir = await createTempRoot();
  await generateScreenshotMetadata({ rootDir });
  await writeFile(join(rootDir, 'release/store_metadata/screenshots/manifest.json'), '{}\n');

  const report = await checkScreenshotMetadata({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'screenshot-metadata-stale-file' &&
        issue.file === 'release/store_metadata/screenshots/manifest.json',
    ),
  );
});

test('fails clearly when a source locale cannot map to store screenshot locales', async () => {
  const rootDir = await createTempRoot({
    languages: [
      createLanguage('en', { fallback: null, androidStoreLocale: 'en-US', iosStoreLocale: 'en-US' }),
      createLanguage('ar', { bcp47: 'ar', androidStoreLocale: null, iosStoreLocale: null }),
    ],
    sources: ['en', 'ar'],
  });

  await assert.rejects(
    () => buildScreenshotManifest({ rootDir, requiredLocales: ['en', 'ar'] }),
    (error) =>
      error instanceof ScreenshotMetadataError &&
      error.issues.some(
        (issue) =>
          issue.code === 'screenshot-metadata-unsupported-locale' &&
          issue.file === 'release/store_metadata/source/ar.json',
      ),
  );
});

async function createTempRoot({
  languages = [
    createLanguage('en', { fallback: null, androidStoreLocale: 'en-US', iosStoreLocale: 'en-US' }),
    createLanguage('ja', { bcp47: 'ja', androidStoreLocale: 'ja-JP', iosStoreLocale: 'ja' }),
    createLanguage('zh', { bcp47: 'zh-Hans', androidStoreLocale: 'zh-CN', iosStoreLocale: 'zh-Hans' }),
    createLanguage('zhtw', { bcp47: 'zh-Hant', androidStoreLocale: 'zh-TW', iosStoreLocale: 'zh-Hant' }),
  ],
  sources = ['en', 'ja', 'zh', 'zhtw'],
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-screenshot-metadata-'));
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

function createLanguage(routeCode, { bcp47 = routeCode, fallback = 'en', androidStoreLocale, iosStoreLocale }) {
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
