import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:sqflite/sqflite.dart' as sqflite;

class AppLaunchStore {
  const AppLaunchStore({this.databaseFactory, this.databasePathResolver});

  final sqflite.DatabaseFactory? databaseFactory;
  final Future<String> Function()? databasePathResolver;

  static const _databaseName = 'hanko_field_app.db';
  static const _settingsTable = 'app_settings';
  static const _keyColumn = 'key';
  static const _valueColumn = 'value';
  static const _hasSeenOnboardingKey = 'has_seen_onboarding';
  static const _preferredRouteCodeKey = 'preferred_route_code';
  static const _legacyPreferredLanguageCodeKey = 'preferred_language_code';

  Future<bool> hasSeenOnboarding() async {
    final db = await _openDatabase();
    final rows = await db.query(
      _settingsTable,
      columns: [_valueColumn],
      where: '$_keyColumn = ?',
      whereArgs: [_hasSeenOnboardingKey],
      limit: 1,
    );

    if (rows.isEmpty) {
      return false;
    }

    return rows.first[_valueColumn] == 'true';
  }

  Future<void> setHasSeenOnboarding(bool value) async {
    final db = await _openDatabase();
    await db.insert(_settingsTable, {
      _keyColumn: _hasSeenOnboardingKey,
      _valueColumn: value ? 'true' : 'false',
    }, conflictAlgorithm: sqflite.ConflictAlgorithm.replace);
  }

  Future<String?> preferredRouteCode() async {
    final routeCode = _normalizeRouteCode(
      await _settingValue(_preferredRouteCodeKey),
    );
    if (routeCode != null) {
      return routeCode;
    }
    return _normalizeRouteCode(
      await _settingValue(_legacyPreferredLanguageCodeKey),
    );
  }

  Future<void> setPreferredRouteCode(String routeCode) async {
    final normalized = _normalizeRouteCode(routeCode);
    if (normalized == null) {
      return;
    }
    final db = await _openDatabase();
    await db.insert(_settingsTable, {
      _keyColumn: _preferredRouteCodeKey,
      _valueColumn: normalized,
    }, conflictAlgorithm: sqflite.ConflictAlgorithm.replace);
  }

  Future<String?> _settingValue(String key) async {
    final db = await _openDatabase();
    final rows = await db.query(
      _settingsTable,
      columns: [_valueColumn],
      where: '$_keyColumn = ?',
      whereArgs: [key],
      limit: 1,
    );

    if (rows.isEmpty) {
      return null;
    }

    return rows.first[_valueColumn]?.toString();
  }

  Future<sqflite.Database> _openDatabase() async {
    final databasePath = await _resolveDatabasePath();
    final factory = databaseFactory ?? sqflite.databaseFactory;
    return factory.openDatabase(
      databasePath,
      options: sqflite.OpenDatabaseOptions(
        version: 1,
        onCreate: (db, _) => _ensureSchema(db),
        onOpen: _ensureSchema,
      ),
    );
  }

  Future<String> _resolveDatabasePath() async {
    final resolver = databasePathResolver;
    if (resolver != null) {
      return resolver();
    }

    final documentsDirectory = await getApplicationDocumentsDirectory();
    return p.join(documentsDirectory.path, _databaseName);
  }

  Future<void> _ensureSchema(sqflite.Database db) {
    return db.execute('''
CREATE TABLE IF NOT EXISTS $_settingsTable (
  $_keyColumn TEXT PRIMARY KEY,
  $_valueColumn TEXT NOT NULL
)
''');
  }
}

String? _normalizeRouteCode(String? routeCode) {
  final normalized = routeCode?.trim().toLowerCase();
  if (normalized == null || normalized.isEmpty) {
    return null;
  }
  return normalized;
}
