import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import {
  validateStoreMetadataDocument,
  validateStoreMetadataSources,
} from './store_metadata.mjs';

test('validates a complete store metadata document', () => {
  const issues = [];

  validateStoreMetadataDocument(validStoreMetadata(), 'release/store_metadata/source/en.json', issues);

  assert.deepEqual(issues, []);
});

test('rejects missing required fields and invalid URLs', () => {
  const issues = [];
  const metadata = validStoreMetadata();
  delete metadata.short_description;
  metadata.support_url = 'http://example.test/support';

  validateStoreMetadataDocument(metadata, 'release/store_metadata/source/en.json', issues);

  assert.ok(issues.some((issue) => issue.code === 'store-metadata-required' && issue.key === 'short_description'));
  assert.ok(issues.some((issue) => issue.code === 'store-metadata-url' && issue.key === 'support_url'));
});

test('requires the M8-T01 example locales', async () => {
  const rootDir = await createTempRoot();
  await writeSource(rootDir, 'en', validStoreMetadata());
  await writeSource(rootDir, 'ja', validStoreMetadata());
  await writeSource(rootDir, 'zh', validStoreMetadata());

  const report = await validateStoreMetadataSources({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'store-metadata-missing-locale' &&
        issue.file === 'release/store_metadata/source/zhtw.json',
    ),
  );
});

test('accepts a complete source directory', async () => {
  const rootDir = await createTempRoot();
  for (const locale of ['en', 'ja', 'zh', 'zhtw']) {
    await writeSource(rootDir, locale, validStoreMetadata());
  }

  const report = await validateStoreMetadataSources({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-store-metadata-'));
  await mkdir(join(rootDir, 'release/store_metadata/source'), { recursive: true });
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
  return rootDir;
}

async function writeSource(rootDir, locale, metadata) {
  await writeJson(join(rootDir, `release/store_metadata/source/${locale}.json`), metadata);
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
