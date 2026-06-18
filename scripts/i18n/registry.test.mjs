import assert from 'node:assert/strict';
import test from 'node:test';

import {
  RegistryValidationError,
  getLanguageByRouteCode,
  loadLanguageRegistry,
  parseLanguageRegistry,
} from './registry.mjs';

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

function cloneLanguage(language) {
  return JSON.parse(JSON.stringify(language));
}
