import 'package:app/core/storage/offline_cache_repository.dart';
import 'package:app/core/storage/onboarding_local_data_source.dart';
import 'package:app/core/storage/local_cache_store.dart';
import 'package:app/core/storage/secure_storage_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:shared_preferences/shared_preferences.dart';

final localCacheStoreProvider = Provider<LocalCacheStore>((ref) {
  final secure = ref.watch(secureStorageProvider);
  return LocalCacheStore(secureStorage: secure);
});

final localCacheStoreInitializedProvider = FutureProvider<void>((ref) async {
  final store = ref.watch(localCacheStoreProvider);
  await store.ensureInitialized();
});

final offlineCacheRepositoryProvider = Provider<OfflineCacheRepository>((ref) {
  final store = ref.watch(localCacheStoreProvider);
  return OfflineCacheRepository(store);
});

final sharedPreferencesProvider = FutureProvider<SharedPreferences>((ref) {
  return SharedPreferences.getInstance();
});

final onboardingLocalDataSourceProvider =
    FutureProvider<OnboardingLocalDataSource>((ref) async {
      final prefs = await ref.watch(sharedPreferencesProvider.future);
      final cache = ref.watch(offlineCacheRepositoryProvider);
      return OnboardingLocalDataSource(
        preferences: prefs,
        cacheRepository: cache,
      );
    });
