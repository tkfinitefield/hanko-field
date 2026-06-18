import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

import {
  RegistryValidationError,
  getLanguageByRouteCode,
  loadLanguageRegistry,
  parseLanguageRegistry,
} from './registry.mjs';

const CORE_FIXTURE_URL = new URL('./fixtures/registry-core.json', import.meta.url);

test('loads the checked-in 68-language registry', async () => {
  const registry = await loadLanguageRegistry();

  assert.equal(registry.languages.length, 68);
  assert.equal(getLanguageByRouteCode(registry, 'no').bcp47, 'no');

  const zh = getLanguageByRouteCode(registry, 'zh');
  assert.equal(zh.bcp47, 'zh-Hans');
  assert.equal(zh.flutter.languageCode, 'zh');
  assert.equal(zh.flutter.scriptCode, 'Hans');

  const zhtw = getLanguageByRouteCode(registry, 'zhtw');
  assert.equal(zhtw.bcp47, 'zh-Hant');
  assert.equal(zhtw.flutter.languageCode, 'zh');
  assert.equal(zhtw.flutter.scriptCode, 'Hant');

  const rtlCodes = registry.languages
    .filter((language) => language.text_direction === 'rtl')
    .map((language) => language.route_code);
  assert.deepEqual(rtlCodes, ['ar', 'fa', 'he', 'ps', 'ur']);
});

test('rejects duplicate route codes', async () => {
  const registry = await loadLanguageRegistry();
  const duplicated = registry.languages.map(cloneLanguage);
  duplicated[1].route_code = duplicated[0].route_code;

  assert.throws(
    () => parseLanguageRegistry(duplicated, { source: 'duplicate fixture' }),
    (error) => {
      assert.ok(error instanceof RegistryValidationError);
      assert.ok(error.errors.some((message) => message.includes('duplicated')));
      return true;
    },
  );
});

test('rejects fallback values that do not point to a route code', async () => {
  const registry = await loadLanguageRegistry();
  const invalidFallback = registry.languages.map(cloneLanguage);
  invalidFallback.find((language) => language.route_code === 'ja').fallback = 'missing';

  assert.throws(
    () => parseLanguageRegistry(invalidFallback, { source: 'fallback fixture' }),
    (error) => {
      assert.ok(error instanceof RegistryValidationError);
      assert.ok(error.errors.some((message) => message.includes('does not match a route_code')));
      return true;
    },
  );
});

test('rejects fallback values that point to the same route code', async () => {
  const registry = await loadLanguageRegistry();
  const selfFallback = registry.languages.map(cloneLanguage);
  selfFallback.find((language) => language.route_code === 'ja').fallback = 'ja';

  assert.throws(
    () => parseLanguageRegistry(selfFallback, { source: 'self fallback fixture' }),
    (error) => {
      assert.ok(error instanceof RegistryValidationError);
      assert.ok(error.errors.some((message) => message.includes('must not point to itself')));
      return true;
    },
  );
});

test('core fixture covers route, BCP-47, Flutter, and store fields', async () => {
  const registry = await loadFixtureRegistry();

  assert.deepEqual(
    registry.languages.map((language) => language.route_code),
    ['en', 'ja', 'zh', 'zhtw', 'no', 'ar', 'fa', 'he', 'ps', 'ur'],
  );

  assertLanguage(registry, 'en', {
    bcp47: 'en',
    flutter: { languageCode: 'en', scriptCode: null, countryCode: null },
    text_direction: 'ltr',
    fallback: null,
    currency: 'USD',
    url_prefix: '',
    android_store_locale: 'en-US',
    ios_store_locale: 'en-US',
  });
  assertLanguage(registry, 'ja', {
    bcp47: 'ja',
    flutter: { languageCode: 'ja', scriptCode: null, countryCode: null },
    text_direction: 'ltr',
    fallback: 'en',
    currency: 'JPY',
    url_prefix: 'ja',
    android_store_locale: 'ja-JP',
    ios_store_locale: 'ja',
  });
  assertLanguage(registry, 'zh', {
    bcp47: 'zh-Hans',
    flutter: { languageCode: 'zh', scriptCode: 'Hans', countryCode: null },
    text_direction: 'ltr',
    fallback: 'en',
    currency: 'USD',
    url_prefix: 'zh',
    android_store_locale: 'zh-CN',
    ios_store_locale: 'zh-Hans',
  });
  assertLanguage(registry, 'zhtw', {
    bcp47: 'zh-Hant',
    flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
    text_direction: 'ltr',
    fallback: 'en',
    currency: 'USD',
    url_prefix: 'zhtw',
    android_store_locale: 'zh-TW',
    ios_store_locale: 'zh-Hant',
  });
  assertLanguage(registry, 'no', {
    bcp47: 'no',
    flutter: { languageCode: 'no', scriptCode: null, countryCode: null },
    text_direction: 'ltr',
    fallback: 'en',
    currency: 'USD',
    url_prefix: 'no',
    android_store_locale: null,
    ios_store_locale: null,
  });

  for (const routeCode of ['ar', 'fa', 'he', 'ps', 'ur']) {
    const language = getLanguageByRouteCode(registry, routeCode);
    assert.equal(language.text_direction, 'rtl');
    assert.equal(language.bcp47, routeCode);
    assert.deepEqual(language.flutter, {
      languageCode: routeCode,
      scriptCode: null,
      countryCode: null,
    });
    assert.equal(language.release.android_store_locale, null);
    assert.equal(language.release.ios_store_locale, null);
  }
});

test('core fixture mirrors the checked-in registry for covered route codes', async () => {
  const checkedInRegistry = await loadLanguageRegistry();
  const fixtureRegistry = await loadFixtureRegistry();

  for (const fixtureLanguage of fixtureRegistry.languages) {
    assert.deepEqual(
      getLanguageByRouteCode(checkedInRegistry, fixtureLanguage.route_code),
      fixtureLanguage,
    );
  }
});

function cloneLanguage(language) {
  return JSON.parse(JSON.stringify(language));
}

async function loadFixtureRegistry() {
  return parseLanguageRegistry(JSON.parse(await readFile(CORE_FIXTURE_URL, 'utf8')), {
    source: CORE_FIXTURE_URL.pathname,
  });
}

function assertLanguage(registry, routeCode, expected) {
  const language = getLanguageByRouteCode(registry, routeCode);
  assert.equal(language.bcp47, expected.bcp47);
  assert.deepEqual(language.flutter, expected.flutter);
  assert.equal(language.text_direction, expected.text_direction);
  assert.equal(language.fallback, expected.fallback);
  assert.equal(language.currency, expected.currency);
  assert.equal(language.web.url_prefix, expected.url_prefix);
  assert.equal(language.release.android_store_locale, expected.android_store_locale);
  assert.equal(language.release.ios_store_locale, expected.ios_store_locale);
}
