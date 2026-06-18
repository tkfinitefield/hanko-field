import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import {
  analyzeArbMessage,
  expectedArbLocale,
  expectedArbPath,
  parseGeneratedSupportedLocales,
  validateArbFiles,
} from './arb.mjs';

test('passes for matching placeholders and metadata', async () => {
  const rootDir = await createTempRoot();
  await writeBaseAndTargetArbs(rootDir);

  const report = await validateArbFiles({ rootDir });

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.ok(report.parsed_files.includes('app/lib/l10n/app_en.arb'));
  assert.ok(report.parsed_files.includes('app/lib/l10n/app_ja.arb'));
});

test('fails when target placeholders do not match English', async () => {
  const rootDir = await createTempRoot();
  await writeBaseAndTargetArbs(rootDir, {
    ja: {
      '@@locale': 'ja',
      helloUser: 'こんにちは',
      '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
    },
  });

  const report = await validateArbFiles({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'arb-placeholder-mismatch' &&
        issue.file === 'app/lib/l10n/app_ja.arb' &&
        issue.key === 'helloUser',
    ),
  );
});

test('fails when English placeholder metadata is missing or extra', async () => {
  const rootDir = await createTempRoot();
  await writeBaseAndTargetArbs(rootDir, {
    en: {
      '@@locale': 'en',
      helloUser: 'Hello {name}',
      '@helloUser': placeholderMetadata('Greeting with the user name.', ['unused']),
    },
  });

  const report = await validateArbFiles({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) => issue.code === 'arb-metadata' && issue.file === 'app/lib/l10n/app_en.arb',
    ),
  );
});

test('validates plural and select ICU syntax', () => {
  const valid = analyzeArbMessage(
    '{count, plural, =0{No items} one{One item} other{{count} items for {name}}}',
  );
  assert.deepEqual([...valid.placeholders].sort(), ['count', 'name']);
  assert.deepEqual(valid.issues, []);

  const invalid = analyzeArbMessage('{count, plural, one{One item}}');
  assert.ok(invalid.issues.includes('plural ICU block must define an other option'));
});

test('fails on malformed ICU braces before runtime', async () => {
  const rootDir = await createTempRoot();
  await writeBaseAndTargetArbs(rootDir, {
    ja: {
      '@@locale': 'ja',
      helloUser: 'こんにちは {name',
      '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
    },
  });

  const report = await validateArbFiles({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) => issue.code === 'arb-icu' && issue.file === 'app/lib/l10n/app_ja.arb',
    ),
  );
});

test('validates registry Flutter locale mapping for requested languages', async () => {
  const rootDir = await createTempRoot({
    extraLanguages: [
      createLanguage('zhtw', {
        bcp47: 'zh-Hant',
        flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
      }),
    ],
  });
  await writeBaseAndTargetArbs(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_zh_Hant.arb', {
    '@@locale': 'zh',
    helloUser: 'Hello {name}',
    '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
  });

  const report = await validateArbFiles({ rootDir, langs: ['zhtw'] });

  assert.equal(expectedArbPath(createLanguage('zhtw', {
    flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
  })), 'app/lib/l10n/app_zh_Hant.arb');
  assert.equal(expectedArbLocale(createLanguage('zhtw', {
    flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
  })), 'zh_Hant');
  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'arb-locale-mapping' &&
        issue.file === 'app/lib/l10n/app_zh_Hant.arb' &&
        issue.key === '@@locale',
    ),
  );
});

test('fails when generated supportedLocales is missing a requested ARB locale', async () => {
  const rootDir = await createTempRoot({
    extraLanguages: [
      createLanguage('zhtw', {
        bcp47: 'zh-Hant',
        flutter: { languageCode: 'zh', scriptCode: 'Hant', countryCode: null },
      }),
    ],
  });
  await writeBaseAndTargetArbs(rootDir);
  await writeJson(rootDir, 'app/lib/l10n/app_zh_Hant.arb', {
    '@@locale': 'zh_Hant',
    helloUser: 'Hello {name}',
    '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
  });
  await writeText(
    rootDir,
    'app/lib/l10n/generated/generated_hanko_localizations.dart',
    `
class GeneratedHankoLocalizations {
  static const List<Locale> supportedLocales = <Locale>[
    Locale('en'),
    Locale('ja'),
    Locale('zh'),
  ];
}
`,
  );

  const report = await validateArbFiles({ rootDir, langs: ['zhtw'] });

  assert.equal(report.ok, false);
  assert.ok(
    report.issues.some(
      (issue) =>
        issue.code === 'arb-generated-locale-mapping' &&
        issue.file === 'app/lib/l10n/generated/generated_hanko_localizations.dart' &&
        issue.key === 'zhtw',
    ),
  );
});

test('parses generated Locale.fromSubtags entries', () => {
  const locales = parseGeneratedSupportedLocales(`
static const List<Locale> supportedLocales = <Locale>[
  Locale('en'),
  Locale('ja'),
  Locale('zh'),
  Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
  Locale.fromSubtags(languageCode: 'pt', countryCode: 'BR'),
];
`);

  assert.deepEqual([...locales].sort(), ['en', 'ja', 'pt_BR', 'zh', 'zh_Hant']);
});

test('file filter validates the target ARB against English', async () => {
  const rootDir = await createTempRoot();
  await writeBaseAndTargetArbs(rootDir, {
    ja: {
      '@@locale': 'ja',
      helloUser: 'こんにちは',
      '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
    },
  });

  const report = await validateArbFiles({
    rootDir,
    file: 'app/lib/l10n/app_ja.arb',
  });

  assert.equal(report.ok, false);
  assert.deepEqual(report.parsed_files.sort(), [
    'app/lib/l10n/app_en.arb',
    'app/lib/l10n/app_ja.arb',
  ]);
  assert.ok(report.issues.some((issue) => issue.code === 'arb-placeholder-mismatch'));
});

async function createTempRoot({ extraLanguages = [] } = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-i18n-arb-'));
  await writeJson(rootDir, 'config/languages.json', [
    createLanguage('en', { fallback: null, app: { enabled: true, selectable: true } }),
    createLanguage('ja', {
      bcp47: 'ja',
      native_name: '日本語',
      english_name: 'Japanese',
      currency: 'JPY',
      app: { enabled: true, selectable: true },
    }),
    ...extraLanguages,
  ]);
  return rootDir;
}

async function writeBaseAndTargetArbs(rootDir, overrides = {}) {
  const en = overrides.en ?? {
    '@@locale': 'en',
    helloUser: 'Hello {name}',
    '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
  };
  const ja = overrides.ja ?? {
    '@@locale': 'ja',
    helloUser: 'こんにちは {name}',
    '@helloUser': placeholderMetadata('Greeting with the user name.', ['name']),
  };
  await writeJson(rootDir, 'app/lib/l10n/app_en.arb', en);
  await writeJson(rootDir, 'app/lib/l10n/app_ja.arb', ja);
}

function placeholderMetadata(description, names) {
  return {
    description,
    placeholders: Object.fromEntries(
      names.map((name) => [
        name,
        {
          type: 'String',
          example: name,
        },
      ]),
    ),
  };
}

async function writeJson(rootDir, relativePath, value) {
  await writeText(rootDir, relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeText(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), value);
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
