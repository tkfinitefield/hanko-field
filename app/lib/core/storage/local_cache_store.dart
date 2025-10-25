import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:app/core/storage/cache_bucket.dart';
import 'package:app/core/storage/cache_policy.dart';
import 'package:app/core/storage/secure_storage_service.dart';
import 'package:clock/clock.dart' as clock_package;
import 'package:hive_flutter/hive_flutter.dart';

typedef CacheEncoder<T> = Object? Function(T value);
typedef CacheDecoder<T> = T Function(Object? data);

class LocalCacheStore {
  LocalCacheStore({
    required SecureStorageService secureStorage,
    HiveInterface? hive,
    Future<void> Function(HiveInterface hive)? initializeHive,
    clock_package.Clock? clock,
    Random? random,
  }) : _secureStorage = secureStorage,
       _hive = hive ?? Hive,
       _initializeHive = initializeHive ?? _defaultInitialize,
       _clock = clock ?? clock_package.clock,
       _random = random ?? Random.secure();

  static const defaultEntryKey = 'default';

  final SecureStorageService _secureStorage;
  final HiveInterface _hive;
  final Future<void> Function(HiveInterface hive) _initializeHive;
  final clock_package.Clock _clock;
  final Random _random;

  final Map<CacheBucket, Box<dynamic>> _boxes = {};
  Future<void>? _initialization;

  Future<void> ensureInitialized() {
    return _initialization ??= _init();
  }

  Future<void> _init() async {
    await _initializeHive(_hive);
    for (final bucket in CacheBucket.values) {
      _boxes[bucket] = await _openBox(bucket);
    }
  }

  static Future<void> _defaultInitialize(HiveInterface hive) async {
    if (identical(hive, Hive)) {
      await Hive.initFlutter();
    }
  }

  Future<Box<dynamic>> _openBox(CacheBucket bucket) async {
    if (_hive.isBoxOpen(bucket.boxName)) {
      return _hive.box<dynamic>(bucket.boxName);
    }
    final cipher = bucket.encrypted
        ? HiveAesCipher(await _loadEncryptionKey(bucket))
        : null;
    return _hive.openBox<dynamic>(bucket.boxName, encryptionCipher: cipher);
  }

  Future<Uint8List> _loadEncryptionKey(CacheBucket bucket) async {
    final storageKey = 'local_cache.key.${bucket.boxName}';
    final existing = await _secureStorage.read(key: storageKey);
    if (existing != null) {
      return Uint8List.fromList(base64Decode(existing));
    }
    final bytes = List<int>.generate(32, (_) => _random.nextInt(256));
    await _secureStorage.write(key: storageKey, value: base64Encode(bytes));
    return Uint8List.fromList(bytes);
  }

  Future<CacheReadResult<T>> read<T>({
    required CacheBucket bucket,
    required CacheDecoder<T> decoder,
    String key = defaultEntryKey,
  }) async {
    await ensureInitialized();
    final box = _boxes[bucket];
    if (box == null) {
      return const CacheReadResult.miss();
    }
    final raw = box.get(key);
    if (raw == null) {
      return const CacheReadResult.miss();
    }
    final record = _CacheRecord.fromMap(Map<dynamic, dynamic>.from(raw as Map));
    final state = bucket.policy.evaluate(record.updatedAt, _clock.now());
    if (state == CacheState.expired) {
      await box.delete(key);
      return const CacheReadResult.miss();
    }
    final value = decoder(record.data);
    return CacheReadResult.value(
      value: value,
      state: state,
      lastUpdated: record.updatedAt,
    );
  }

  Future<void> write<T>({
    required CacheBucket bucket,
    required CacheEncoder<T> encoder,
    required T value,
    String key = defaultEntryKey,
    String? etag,
  }) async {
    await ensureInitialized();
    final box = _boxes[bucket];
    if (box == null) {
      throw StateError('Cache bucket ${bucket.boxName} not opened');
    }
    final record = _CacheRecord(
      updatedAt: _clock.now(),
      data: encoder(value),
      etag: etag,
    );
    await box.put(key, record.toMap());
  }

  Future<void> invalidate(
    CacheBucket bucket, {
    String key = defaultEntryKey,
  }) async {
    await ensureInitialized();
    final box = _boxes[bucket];
    if (box == null) {
      return;
    }
    await box.delete(key);
  }

  Future<void> clear(CacheBucket bucket) async {
    await ensureInitialized();
    final box = _boxes[bucket];
    if (box == null) {
      return;
    }
    await box.clear();
  }

  Future<void> clearAll() async {
    for (final bucket in CacheBucket.values) {
      await clear(bucket);
    }
  }

  Future<void> close() async {
    for (final bucket in CacheBucket.values) {
      if (_hive.isBoxOpen(bucket.boxName)) {
        await _hive.box(bucket.boxName).close();
      }
    }
    _initialization = null;
    _boxes.clear();
  }
}

class _CacheRecord {
  _CacheRecord({required this.updatedAt, required this.data, this.etag});

  factory _CacheRecord.fromMap(Map<dynamic, dynamic> map) {
    return _CacheRecord(
      updatedAt: DateTime.parse(map['updatedAt'] as String),
      data: map['data'],
      etag: map['etag'] as String?,
    );
  }

  final DateTime updatedAt;
  final Object? data;
  final String? etag;

  Map<String, dynamic> toMap() {
    return <String, dynamic>{
      'updatedAt': updatedAt.toIso8601String(),
      'data': data,
      'etag': etag,
    };
  }
}
