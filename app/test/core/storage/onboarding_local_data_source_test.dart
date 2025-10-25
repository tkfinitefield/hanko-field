import 'dart:io';

import 'package:app/core/storage/offline_cache_repository.dart'
    show OfflineCacheRepository, OnboardingStep;
import 'package:app/core/storage/onboarding_local_data_source.dart';
import 'package:app/core/storage/local_cache_store.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hive/hive.dart' as hive;
import 'package:shared_preferences/shared_preferences.dart';

import 'fakes.dart';

void main() {
  late Directory tempDir;
  late LocalCacheStore store;
  late OfflineCacheRepository cacheRepository;
  late hive.HiveInterface hiveInstance;
  late SharedPreferences prefs;

  setUp(() async {
    SharedPreferences.setMockInitialValues(<String, Object>{});
    prefs = await SharedPreferences.getInstance();
    tempDir = await Directory.systemTemp.createTemp('onboarding_cache_test');
    hiveInstance = hive.Hive;
    await hiveInstance.close();
    hiveInstance.init(tempDir.path);
    final secure = FakeSecureStorageService();
    store = LocalCacheStore(
      secureStorage: secure,
      hive: hiveInstance,
      initializeHive: (_) async {},
    );
    await store.ensureInitialized();
    cacheRepository = OfflineCacheRepository(store);
  });

  tearDown(() async {
    await store.close();
    await hiveInstance.close();
    await tempDir.delete(recursive: true);
  });

  OnboardingLocalDataSource _buildDataSource() {
    return OnboardingLocalDataSource(
      preferences: prefs,
      cacheRepository: cacheRepository,
    );
  }

  test('returns initial flags when nothing persisted', () async {
    final dataSource = _buildDataSource();
    final flags = await dataSource.load();
    expect(flags.isCompleted, isFalse);
    expect(
      flags.stepCompletion.values.every((completed) => !completed),
      isTrue,
    );
  });

  test('updates steps and mirrors to cache', () async {
    final dataSource = _buildDataSource();
    await dataSource.updateStep(OnboardingStep.locale);

    final raw = prefs.getString('onboarding.flags');
    expect(raw, isNotNull);

    final cached = await cacheRepository.readOnboardingFlags();
    expect(cached.hasValue, isTrue);
    expect(cached.value!.stepCompletion[OnboardingStep.locale], isTrue);
  });

  test('reset clears preferences and seeds cache with defaults', () async {
    final dataSource = _buildDataSource();
    await dataSource.updateStep(OnboardingStep.persona);

    await dataSource.reset();
    expect(prefs.getString('onboarding.flags'), isNull);

    final cached = await cacheRepository.readOnboardingFlags();
    expect(cached.hasValue, isTrue);
    expect(cached.value!.isCompleted, isFalse);
  });
}
