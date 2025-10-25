import 'dart:io';
import 'dart:math';

import 'package:app/core/storage/cache_bucket.dart';
import 'package:app/core/storage/local_cache_store.dart';
import 'package:clock/clock.dart' as clock_package;
import 'package:flutter_test/flutter_test.dart';
import 'package:hive/hive.dart' as hive;

import 'fakes.dart';

void main() {
  late Directory tempDir;
  late LocalCacheStore store;
  late FakeSecureStorageService secureStorage;
  late hive.HiveInterface hiveInstance;
  late DateTime now;
  late clock_package.Clock testClock;

  setUp(() async {
    tempDir = await Directory.systemTemp.createTemp('cache_store_test');
    hiveInstance = hive.Hive;
    await hiveInstance.close();
    hiveInstance.init(tempDir.path);
    secureStorage = FakeSecureStorageService();
    now = DateTime(2024, 1, 1, 12);
    testClock = clock_package.Clock(() => now);
    store = LocalCacheStore(
      secureStorage: secureStorage,
      hive: hiveInstance,
      initializeHive: (_) async {},
      clock: testClock,
      random: Random(42),
    );
    await store.ensureInitialized();
  });

  tearDown(() async {
    await store.close();
    await hiveInstance.close();
    await tempDir.delete(recursive: true);
  });

  test('returns cache miss when bucket is empty', () async {
    final result = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.designs,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(result.isMiss, isTrue);
  });

  test('honours ttl and stale windows', () async {
    await store.write<Map<String, dynamic>>(
      bucket: CacheBucket.designs,
      encoder: (value) => value,
      value: {'id': 'd1'},
    );

    var snapshot = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.designs,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(snapshot.isFresh, isTrue);

    now = now.add(const Duration(hours: 11));
    snapshot = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.designs,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(snapshot.isStale, isTrue);

    now = now.add(const Duration(hours: 3));
    snapshot = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.designs,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(snapshot.isMiss, isTrue);
  });

  test('supports encrypted buckets', () async {
    await store.write(
      bucket: CacheBucket.cart,
      encoder: (value) => value,
      value: {'lines': 2},
    );

    final snapshot = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.cart,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(snapshot.value?['lines'], 2);
  });

  test('invalidate removes entry', () async {
    await store.write(
      bucket: CacheBucket.guides,
      encoder: (value) => value,
      value: {'guides': []},
    );

    await store.invalidate(CacheBucket.guides);
    final snapshot = await store.read<Map<String, dynamic>>(
      bucket: CacheBucket.guides,
      decoder: (data) => Map<String, dynamic>.from(data as Map),
    );
    expect(snapshot.isMiss, isTrue);
  });
}
