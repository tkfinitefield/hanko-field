import 'dart:convert';
import 'dart:io';

import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';
import 'package:sqflite/sqflite.dart' as sqflite;

import '../domain/local_seal_design.dart';

abstract interface class LocalSealDesignRepository {
  Future<List<LocalSealDesign>> listLocalSealDesigns();

  Future<LocalSealDesign?> getLocalSealDesign(String id);

  Future<void> saveLocalSealDesign(LocalSealDesign design);

  Future<void> deleteLocalSealDesign(String id);
}

class InMemoryLocalSealDesignRepository implements LocalSealDesignRepository {
  InMemoryLocalSealDesignRepository([
    Iterable<LocalSealDesign> designs = const [],
  ]) : _designs = {for (final design in designs) design.id: design};

  final Map<String, LocalSealDesign> _designs;

  @override
  Future<List<LocalSealDesign>> listLocalSealDesigns() async {
    final designs = _designs.values.toList(growable: false)
      ..sort((a, b) {
        final updatedComparison = b.updatedAt.compareTo(a.updatedAt);
        if (updatedComparison != 0) {
          return updatedComparison;
        }
        final createdComparison = b.createdAt.compareTo(a.createdAt);
        if (createdComparison != 0) {
          return createdComparison;
        }
        return a.id.compareTo(b.id);
      });
    return designs;
  }

  @override
  Future<LocalSealDesign?> getLocalSealDesign(String id) async {
    return _designs[id];
  }

  @override
  Future<void> saveLocalSealDesign(LocalSealDesign design) async {
    _designs[design.id] = design;
  }

  @override
  Future<void> deleteLocalSealDesign(String id) async {
    _designs.remove(id);
  }
}

class SqfliteLocalSealDesignRepository implements LocalSealDesignRepository {
  SqfliteLocalSealDesignRepository({
    sqflite.DatabaseFactory? databaseFactory,
    Future<String> Function()? databasePathResolver,
    LocalSealImageStore? imageStore,
  }) : _databaseFactory = databaseFactory ?? sqflite.databaseFactory,
       _databasePathResolver = databasePathResolver ?? _defaultDatabasePath,
       _imageStore = imageStore ?? LocalSealImageStore();

  static const _databaseName = 'hanko_field_local_seals.db';
  static const _table = 'local_seal_designs';
  static const _databaseVersion = 2;
  static const _legacyDateFallback = '1970-01-01T00:00:00.000Z';
  static const _columnMigrations = <String, String>{
    'input_name': "input_name TEXT NOT NULL DEFAULT ''",
    'selected_kanji': "selected_kanji TEXT NOT NULL DEFAULT ''",
    'reading': "reading TEXT NOT NULL DEFAULT ''",
    'meaning': 'meaning TEXT',
    'impression_json': "impression_json TEXT NOT NULL DEFAULT '[]'",
    'character_count': 'character_count INTEGER NOT NULL DEFAULT 0',
    'stroke_complexity': 'stroke_complexity TEXT',
    'engraving_suitability': 'engraving_suitability TEXT',
    'shape': "shape TEXT NOT NULL DEFAULT 'square'",
    'style': "style TEXT NOT NULL DEFAULT 'elegant'",
    'stroke_weight': "stroke_weight TEXT NOT NULL DEFAULT 'standard'",
    'balance': "balance TEXT NOT NULL DEFAULT 'balanced'",
    'ai_generation_id': "ai_generation_id TEXT NOT NULL DEFAULT ''",
    'ai_variant_id': "ai_variant_id TEXT NOT NULL DEFAULT ''",
    'preview_image_storage_path':
        "preview_image_storage_path TEXT NOT NULL DEFAULT ''",
    'preview_image_download_url':
        "preview_image_download_url TEXT NOT NULL DEFAULT ''",
    'local_image_path': "local_image_path TEXT NOT NULL DEFAULT ''",
    'is_favorite': 'is_favorite INTEGER NOT NULL DEFAULT 0',
    'created_at': "created_at TEXT NOT NULL DEFAULT '$_legacyDateFallback'",
    'updated_at': "updated_at TEXT NOT NULL DEFAULT '$_legacyDateFallback'",
  };

  final sqflite.DatabaseFactory _databaseFactory;
  final Future<String> Function() _databasePathResolver;
  final LocalSealImageStore _imageStore;

  @override
  Future<List<LocalSealDesign>> listLocalSealDesigns() async {
    final db = await _openDatabase();
    final rows = await db.query(
      _table,
      orderBy: 'updated_at DESC, created_at DESC, id ASC',
    );

    return rows
        .map(_LocalSealDesignRecord.tryFromMap)
        .whereType<_LocalSealDesignRecord>()
        .map((record) => record.toDomain())
        .toList(growable: false);
  }

  @override
  Future<LocalSealDesign?> getLocalSealDesign(String id) async {
    final db = await _openDatabase();
    final rows = await db.query(
      _table,
      where: 'id = ?',
      whereArgs: [id],
      limit: 1,
    );
    if (rows.isEmpty) {
      return null;
    }

    return _LocalSealDesignRecord.tryFromMap(rows.first)?.toDomain();
  }

  @override
  Future<void> saveLocalSealDesign(LocalSealDesign design) async {
    final db = await _openDatabase();
    await db.insert(
      _table,
      _LocalSealDesignRecord.fromDomain(design).toMap(),
      conflictAlgorithm: sqflite.ConflictAlgorithm.replace,
    );
  }

  @override
  Future<void> deleteLocalSealDesign(String id) async {
    final design = await getLocalSealDesign(id);
    final db = await _openDatabase();
    await db.delete(_table, where: 'id = ?', whereArgs: [id]);
    if (design == null) {
      return;
    }
    await _imageStore.deleteSealImage(design.localImagePath);
  }

  Future<sqflite.Database> _openDatabase() async {
    final databasePath = await _databasePathResolver();
    return _databaseFactory.openDatabase(
      databasePath,
      options: sqflite.OpenDatabaseOptions(
        version: _databaseVersion,
        onCreate: (db, _) => _ensureSchema(db),
        onUpgrade: (db, _, _) => _ensureSchema(db),
        onOpen: _ensureSchema,
      ),
    );
  }

  Future<void> _ensureSchema(sqflite.Database db) async {
    await db.execute('''
CREATE TABLE IF NOT EXISTS $_table (
  id TEXT PRIMARY KEY,
  input_name TEXT NOT NULL,
  selected_kanji TEXT NOT NULL,
  reading TEXT NOT NULL,
  meaning TEXT,
  impression_json TEXT NOT NULL,
  character_count INTEGER NOT NULL,
  stroke_complexity TEXT,
  engraving_suitability TEXT,
  shape TEXT NOT NULL,
  style TEXT NOT NULL,
  stroke_weight TEXT NOT NULL,
  balance TEXT NOT NULL,
  ai_generation_id TEXT NOT NULL,
  ai_variant_id TEXT NOT NULL,
  preview_image_storage_path TEXT NOT NULL,
  preview_image_download_url TEXT NOT NULL,
  local_image_path TEXT NOT NULL,
  is_favorite INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
)
''');
    await _ensureMigratedColumns(db);
    await _backfillMigratedRows(db);
  }

  Future<void> _ensureMigratedColumns(sqflite.Database db) async {
    final columns = await db.rawQuery('PRAGMA table_info($_table)');
    final existingColumnNames = columns
        .map((column) => column['name'])
        .whereType<String>()
        .toSet();

    for (final entry in _columnMigrations.entries) {
      if (existingColumnNames.contains(entry.key)) {
        continue;
      }
      await db.execute('ALTER TABLE $_table ADD COLUMN ${entry.value}');
    }
  }

  Future<void> _backfillMigratedRows(sqflite.Database db) async {
    await db.execute('''
UPDATE $_table
SET character_count = length(selected_kanji)
WHERE character_count = 0 AND selected_kanji <> ''
''');
    await db.execute('''
UPDATE $_table
SET ai_generation_id = id
WHERE ai_generation_id = ''
''');
    await db.execute('''
UPDATE $_table
SET ai_variant_id = id
WHERE ai_variant_id = ''
''');
  }

  static Future<String> _defaultDatabasePath() async {
    final documentsDirectory = await getApplicationDocumentsDirectory();
    return p.join(documentsDirectory.path, _databaseName);
  }
}

class LocalSealImageStore {
  LocalSealImageStore({
    Future<Directory> Function()? documentsDirectoryProvider,
  }) : _documentsDirectoryProvider =
           documentsDirectoryProvider ?? getApplicationDocumentsDirectory;

  static const _imageDirectoryName = 'seal_designs';

  final Future<Directory> Function() _documentsDirectoryProvider;

  Future<String> saveSealImage({
    required String localSealId,
    required List<int> bytes,
    String fileExtension = 'png',
  }) async {
    final directory = await _imageDirectory();
    await directory.create(recursive: true);

    final filename =
        '${_safeFilename(localSealId)}.${_safeExtension(fileExtension)}';
    final file = File(p.join(directory.path, filename));
    await file.writeAsBytes(bytes, flush: true);
    return file.path;
  }

  Future<void> deleteSealImage(String localImagePath) async {
    final trimmedPath = localImagePath.trim();
    if (trimmedPath.isEmpty) {
      return;
    }

    final directory = await _imageDirectory();
    final normalizedDirectory = p.normalize(directory.path);
    final normalizedPath = p.normalize(trimmedPath);
    if (!p.isWithin(normalizedDirectory, normalizedPath)) {
      return;
    }

    final file = File(normalizedPath);
    if (await file.exists()) {
      await file.delete();
    }
  }

  Future<Directory> _imageDirectory() async {
    final documentsDirectory = await _documentsDirectoryProvider();
    return Directory(p.join(documentsDirectory.path, _imageDirectoryName));
  }

  String _safeFilename(String value) {
    final sanitized = value.replaceAll(RegExp(r'[^A-Za-z0-9_-]'), '_');
    return sanitized.isEmpty ? 'seal_design' : sanitized;
  }

  String _safeExtension(String value) {
    final sanitized = value
        .replaceFirst(RegExp(r'^\.'), '')
        .replaceAll(RegExp(r'[^A-Za-z0-9]'), '')
        .toLowerCase();
    return sanitized.isEmpty ? 'png' : sanitized;
  }
}

class _LocalSealDesignRecord {
  const _LocalSealDesignRecord({
    required this.id,
    required this.inputName,
    required this.selectedKanji,
    required this.reading,
    required this.meaning,
    required this.impression,
    required this.characterCount,
    required this.strokeComplexity,
    required this.engravingSuitability,
    required this.shape,
    required this.style,
    required this.strokeWeight,
    required this.balance,
    required this.aiGenerationId,
    required this.aiVariantId,
    required this.previewImageStoragePath,
    required this.previewImageDownloadUrl,
    required this.localImagePath,
    required this.isFavorite,
    required this.createdAt,
    required this.updatedAt,
  });

  factory _LocalSealDesignRecord.fromDomain(LocalSealDesign design) {
    return _LocalSealDesignRecord(
      id: design.id,
      inputName: design.inputName,
      selectedKanji: design.selectedKanji,
      reading: design.reading,
      meaning: design.meaning,
      impression: design.impression,
      characterCount: design.characterCount,
      strokeComplexity: design.strokeComplexity,
      engravingSuitability: design.engravingSuitability,
      shape: design.shape,
      style: design.style,
      strokeWeight: design.strokeWeight,
      balance: design.balance,
      aiGenerationId: design.aiGenerationId,
      aiVariantId: design.aiVariantId,
      previewImageStoragePath: design.previewImageStoragePath,
      previewImageDownloadUrl: design.previewImageDownloadUrl,
      localImagePath: design.localImagePath,
      isFavorite: design.isFavorite,
      createdAt: design.createdAt,
      updatedAt: design.updatedAt,
    );
  }

  static _LocalSealDesignRecord? tryFromMap(Map<String, Object?> map) {
    try {
      return _LocalSealDesignRecord(
        id: _readString(map, 'id'),
        inputName: _readString(map, 'input_name'),
        selectedKanji: _readString(map, 'selected_kanji'),
        reading: _readString(map, 'reading'),
        meaning: _readNullableString(map, 'meaning'),
        impression: _readStringListJson(map, 'impression_json'),
        characterCount: _readInt(map, 'character_count'),
        strokeComplexity: _readNullableString(map, 'stroke_complexity'),
        engravingSuitability: _readNullableString(map, 'engraving_suitability'),
        shape: _readString(map, 'shape'),
        style: _readString(map, 'style'),
        strokeWeight: _readString(map, 'stroke_weight'),
        balance: _readString(map, 'balance'),
        aiGenerationId: _readString(map, 'ai_generation_id'),
        aiVariantId: _readString(map, 'ai_variant_id'),
        previewImageStoragePath: _readString(
          map,
          'preview_image_storage_path',
          allowEmpty: true,
        ),
        previewImageDownloadUrl: _readString(
          map,
          'preview_image_download_url',
          allowEmpty: true,
        ),
        localImagePath: _readString(map, 'local_image_path', allowEmpty: true),
        isFavorite: _readBool(map, 'is_favorite'),
        createdAt: DateTime.parse(_readString(map, 'created_at')),
        updatedAt: DateTime.parse(_readString(map, 'updated_at')),
      );
    } on FormatException {
      return null;
    } on TypeError {
      return null;
    }
  }

  final String id;
  final String inputName;
  final String selectedKanji;
  final String reading;
  final String? meaning;
  final List<String> impression;
  final int characterCount;
  final String? strokeComplexity;
  final String? engravingSuitability;
  final String shape;
  final String style;
  final String strokeWeight;
  final String balance;
  final String aiGenerationId;
  final String aiVariantId;
  final String previewImageStoragePath;
  final String previewImageDownloadUrl;
  final String localImagePath;
  final bool isFavorite;
  final DateTime createdAt;
  final DateTime updatedAt;

  Map<String, Object?> toMap() {
    return {
      'id': id,
      'input_name': inputName,
      'selected_kanji': selectedKanji,
      'reading': reading,
      'meaning': meaning,
      'impression_json': jsonEncode(impression),
      'character_count': characterCount,
      'stroke_complexity': strokeComplexity,
      'engraving_suitability': engravingSuitability,
      'shape': shape,
      'style': style,
      'stroke_weight': strokeWeight,
      'balance': balance,
      'ai_generation_id': aiGenerationId,
      'ai_variant_id': aiVariantId,
      'preview_image_storage_path': previewImageStoragePath,
      'preview_image_download_url': previewImageDownloadUrl,
      'local_image_path': localImagePath,
      'is_favorite': isFavorite ? 1 : 0,
      'created_at': createdAt.toIso8601String(),
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  LocalSealDesign toDomain() {
    return LocalSealDesign(
      id: id,
      inputName: inputName,
      selectedKanji: selectedKanji,
      reading: reading,
      meaning: meaning,
      impression: impression,
      characterCount: characterCount,
      strokeComplexity: strokeComplexity,
      engravingSuitability: engravingSuitability,
      shape: shape,
      style: style,
      strokeWeight: strokeWeight,
      balance: balance,
      aiGenerationId: aiGenerationId,
      aiVariantId: aiVariantId,
      previewImageStoragePath: previewImageStoragePath,
      previewImageDownloadUrl: previewImageDownloadUrl,
      localImagePath: localImagePath,
      isFavorite: isFavorite,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  static String _readString(
    Map<String, Object?> map,
    String key, {
    bool allowEmpty = false,
  }) {
    final value = map[key];
    if (value is String && (allowEmpty || value.isNotEmpty)) {
      return value;
    }
    throw FormatException('Invalid local seal string: $key');
  }

  static String? _readNullableString(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value == null) {
      return null;
    }
    if (value is String) {
      return value.isEmpty ? null : value;
    }
    throw FormatException('Invalid local seal nullable string: $key');
  }

  static int _readInt(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value is int) {
      return value;
    }
    throw FormatException('Invalid local seal int: $key');
  }

  static bool _readBool(Map<String, Object?> map, String key) {
    final value = map[key];
    if (value is int) {
      return value == 1;
    }
    if (value is bool) {
      return value;
    }
    throw FormatException('Invalid local seal bool: $key');
  }

  static List<String> _readStringListJson(
    Map<String, Object?> map,
    String key,
  ) {
    final rawValue = _readString(map, key);
    final decoded = jsonDecode(rawValue);
    if (decoded is! List) {
      throw FormatException('Invalid local seal string list: $key');
    }
    return decoded.whereType<String>().toList(growable: false);
  }
}
