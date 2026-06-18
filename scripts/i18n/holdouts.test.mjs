import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildHoldoutReview, renderHoldoutReview } from './holdouts.mjs';

test('fails when same-as-English content has no intention sidecar approval', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);

  const report = await buildHoldoutReview({ rootDir, langs: ['all'] });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'intention-missing'));
});

test('summarizes reviewed non-English holdouts and excludes English source sidecars', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_ja_intentions.json', {
    appTitle: 'brand_name',
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/ja_intentions.json', {
    entries: [
      {
        key_path: 'terms.sections[0].body',
        source_value: 'Finite Field, K.K.',
        target_locale: 'ja',
        reason_code: 'legal_entity',
      },
      {
        key_path: 'contact.email',
        source_value: 'support@example.com',
        target_locale: 'ja',
        reason_code: 'url_or_email',
      },
    ],
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/fr_intentions.json', {
    entries: [
      {
        key_path: 'draft.title',
        source_value: 'Draft',
        target_locale: 'fr',
        reason_code: 'locale_not_release_enabled',
      },
    ],
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/en_intentions.json', {
    entries: [
      {
        key_path: 'about.body',
        source_value: 'STONE SIGNATURE',
        target_locale: 'en',
        reason_code: 'brand_name',
      },
    ],
  });

  const report = await buildHoldoutReview({ rootDir, langs: ['all'] });

  assert.equal(report.ok, true);
  assert.deepEqual(report.reason_counts, {
    brand_name: 1,
    legal_entity: 1,
    locale_not_release_enabled: 1,
    url_or_email: 1,
  });
  assert.deepEqual(report.group_counts, {
    brand: 1,
    contact: 1,
    legal: 1,
    release_deferred: 1,
  });
  assert.equal(report.entries.some((entry) => entry.target_locale === 'en'), false);
  assert.equal(report.entries.filter((entry) => !entry.deferred).length, 3);

  const rendered = renderHoldoutReview(report);
  assert.match(rendered, /Result: pass/);
  assert.match(rendered, /Reviewed shared English\/legal entries: 3/);
  assert.match(rendered, /Deferred translation entries: 1/);
  assert.match(rendered, /appTitle/);
});

test('reports malformed sidecars through intention validation instead of throwing', async () => {
  const rootDir = await createTempRoot();
  await writeMinimalContent(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_ja_intentions.json', {
    appTitle: 'brand_name',
  });
  await writeText(rootDir, 'app/assets/i18n/settings/ja_intentions.json', '{');

  const report = await buildHoldoutReview({ rootDir, langs: ['all'] });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.file === 'app/assets/i18n/settings/ja_intentions.json' &&
        issue.code === 'holdout-sidecar-json',
    ),
  );
});

async function createTempRoot() {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-holdouts-'));
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

async function writeMinimalContent(rootDir) {
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', {
    '@@locale': 'en',
    appTitle: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'app/lib/l10n/app_ja.arb', {
    '@@locale': 'ja',
    appTitle: 'STONE SIGNATURE',
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/en.json', {
    terms: { sections: [{ body: 'Finite Field, K.K.' }] },
    contact: { email: 'support@example.com' },
  });
  await writeJson(rootDir, 'app/assets/i18n/settings/ja.json', {
    terms: { sections: [{ body: 'Finite Field, K.K.' }] },
    contact: { email: 'support@example.com' },
  });
}

function createLanguage(routeCode, overrides = {}) {
  return {
    route_code: routeCode,
    bcp47: overrides.bcp47 ?? routeCode,
    flutter: {
      languageCode: routeCode === 'zhtw' ? 'zh' : routeCode,
      scriptCode: null,
      countryCode: null,
    },
    native_name: overrides.native_name ?? routeCode,
    english_name: overrides.english_name ?? routeCode,
    text_direction: 'ltr',
    fallback: Object.hasOwn(overrides, 'fallback') ? overrides.fallback : 'en',
    currency: overrides.currency ?? 'USD',
    web: overrides.web ?? { enabled: false, indexed: false, url_prefix: routeCode },
    app: overrides.app ?? { enabled: false, selectable: false },
    release: {
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
