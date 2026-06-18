import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:path/path.dart' as p;
import 'package:sqflite_common_ffi/sqflite_ffi.dart';

import 'package:hankofield/features/common/data/app_launch_store.dart';

void main() {
  setUpAll(sqfliteFfiInit);

  late Directory tempDirectory;
  late String databasePath;
  late AppLaunchStore store;

  setUp(() async {
    tempDirectory = await Directory.systemTemp.createTemp(
      'hanko_launch_store_test_',
    );
    databasePath = p.join(tempDirectory.path, 'app_launch.db');
    store = AppLaunchStore(
      databaseFactory: databaseFactoryFfi,
      databasePathResolver: () async => databasePath,
    );
  });

  tearDown(() async {
    await databaseFactoryFfi.deleteDatabase(databasePath);
    if (await tempDirectory.exists()) {
      await tempDirectory.delete(recursive: true);
    }
  });

  test('stores preferred route code using the new registry key', () async {
    await store.setPreferredRouteCode(' JA ');

    expect(await store.preferredRouteCode(), 'ja');

    final db = await databaseFactoryFfi.openDatabase(databasePath);
    final rows = await db.query(
      'app_settings',
      columns: ['value'],
      where: 'key = ?',
      whereArgs: ['preferred_route_code'],
      limit: 1,
    );
    await db.close();

    expect(rows.single['value'], 'ja');
  });

  test('falls back to old preferred language code values', () async {
    final db = await databaseFactoryFfi.openDatabase(databasePath);
    await db.execute('''
CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
)
''');
    await db.insert('app_settings', {
      'key': 'preferred_language_code',
      'value': ' EN ',
    });
    await db.close();

    expect(await store.preferredRouteCode(), 'en');
  });

  test(
    'prefers route code over legacy language code when both exist',
    () async {
      final db = await databaseFactoryFfi.openDatabase(databasePath);
      await db.execute('''
CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
)
''');
      await db.insert('app_settings', {
        'key': 'preferred_language_code',
        'value': 'ja',
      });
      await db.insert('app_settings', {
        'key': 'preferred_route_code',
        'value': 'en',
      });
      await db.close();

      expect(await store.preferredRouteCode(), 'en');
    },
  );

  test('falls back to legacy language code when route code is empty', () async {
    final db = await databaseFactoryFfi.openDatabase(databasePath);
    await db.execute('''
CREATE TABLE IF NOT EXISTS app_settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
)
''');
    await db.insert('app_settings', {
      'key': 'preferred_language_code',
      'value': 'ja',
    });
    await db.insert('app_settings', {
      'key': 'preferred_route_code',
      'value': ' ',
    });
    await db.close();

    expect(await store.preferredRouteCode(), 'ja');
  });

  test('ignores empty preferred route code writes', () async {
    await store.setPreferredRouteCode(' ');

    expect(await store.preferredRouteCode(), isNull);
  });
}
