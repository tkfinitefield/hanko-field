import assert from 'node:assert/strict';
import { mkdir, mkdtemp, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';

import { buildMigrationCleanupReport, renderMigrationCleanupReport } from './migration_cleanup.mjs';

test('accepts the checked-in migration cleanup evidence', async () => {
  const report = await buildMigrationCleanupReport();
  const rendered = renderMigrationCleanupReport(report);

  assert.equal(report.ok, true);
  assert.deepEqual(report.issues, []);
  assert.match(rendered, /Result: pass/);
});

test('fails when the old HankoLocalizations typedef returns', async () => {
  const rootDir = await createTempRoot({
    localizationWrapper: [
      "import 'package:flutter/widgets.dart';",
      "import '../../l10n/generated/generated_hanko_localizations.dart';",
      'typedef HankoLocalizations = GeneratedHankoLocalizations;',
      'extension GeneratedHankoLocalizationsBuildContext on BuildContext {',
      '  GeneratedHankoLocalizations get l10n => GeneratedHankoLocalizations.of(this);',
      '}',
    ].join('\n'),
  });

  const report = await buildMigrationCleanupReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'migration-wrapper-typedef'));
});

test('fails when MaterialApp stops using generated localization delegates', async () => {
  const rootDir = await createTempRoot({
    appSource: [
      'class HankoApp {',
      '  final supportedLocales = const [];',
      '  final localizationsDelegates = const [];',
      '}',
    ].join('\n'),
  });

  const report = await buildMigrationCleanupReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'migration-cleanup-generated-locales'));
  assert.ok(report.issues.some((issue) => issue.code === 'migration-cleanup-generated-delegates'));
});

test('fails when evidence omits a required removed wrapper', async () => {
  const evidence = validEvidence();
  evidence.removed_wrappers = ['HankoLocalizations typedef'];
  const rootDir = await createTempRoot({ evidence });

  const report = await buildMigrationCleanupReport({ rootDir });

  assert.equal(report.ok, false);
  assert.ok(report.issues.some((issue) => issue.code === 'migration-cleanup-wrapper-set'));
});

async function createTempRoot({
  evidence = validEvidence(),
  localizationWrapper = validLocalizationWrapper(),
  appSource = validAppSource(),
  registrySource = validRegistrySource(),
  appTestSource = "test('generated localization', () {});",
} = {}) {
  const rootDir = await mkdtemp(join(tmpdir(), 'hanko-field-migration-cleanup-'));
  await writeJson(rootDir, 'doc/qa/m11-t04/migration-cleanup.json', evidence);
  await writeText(rootDir, 'app/lib/app/localization/hanko_localizations.dart', localizationWrapper);
  await writeText(rootDir, 'app/lib/app/app.dart', appSource);
  await writeText(rootDir, 'app/lib/app/localization/language_registry.dart', registrySource);
  await writeText(rootDir, 'app/test/generated_hanko_localizations_test.dart', appTestSource);
  return rootDir;
}

function validEvidence() {
  return {
    format_version: 1,
    task: 'M11-T04',
    reviewed_at: '2026-06-18T23:59:00+09:00',
    removed_wrappers: [
      'HankoLocalizations typedef',
      'hankoSupportedLocales constant',
      'hankoLocalizationsDelegates constant',
    ],
    retained_compatibility: [
      'preferred_language_code upgrade fallback',
    ],
    summary: {
      generated_localization_active: true,
      registry_paths_active: true,
      temporary_wrappers_removed: true,
      durable_upgrade_compatibility_preserved: true,
    },
  };
}

function validLocalizationWrapper() {
  return [
    "import 'package:flutter/widgets.dart';",
    "import '../../l10n/generated/generated_hanko_localizations.dart';",
    'extension GeneratedHankoLocalizationsBuildContext on BuildContext {',
    '  GeneratedHankoLocalizations get l10n => GeneratedHankoLocalizations.of(this);',
    '}',
  ].join('\n');
}

function validAppSource() {
  return [
    'class HankoApp {',
    '  final supportedLocales = GeneratedHankoLocalizations.supportedLocales;',
    '  final localizationsDelegates = GeneratedHankoLocalizations.localizationsDelegates;',
    '}',
  ].join('\n');
}

function validRegistrySource() {
  return "class AppLanguageRegistry { static const assetPath = '../config/languages.json'; }";
}

async function writeJson(rootDir, relativePath, value) {
  await writeText(rootDir, relativePath, `${JSON.stringify(value, null, 2)}\n`);
}

async function writeText(rootDir, relativePath, value) {
  await mkdir(dirname(join(rootDir, relativePath)), { recursive: true });
  await writeFile(join(rootDir, relativePath), value);
}
