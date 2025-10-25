import 'dart:convert';

import 'package:app/core/storage/offline_cache_repository.dart';
import 'package:shared_preferences/shared_preferences.dart';

class OnboardingLocalDataSource {
  OnboardingLocalDataSource({
    required SharedPreferences preferences,
    required OfflineCacheRepository cacheRepository,
  }) : _preferences = preferences,
       _cacheRepository = cacheRepository;

  static const _prefsKey = 'onboarding.flags';

  final SharedPreferences _preferences;
  final OfflineCacheRepository _cacheRepository;

  Future<OnboardingFlags> load() async {
    final raw = _preferences.getString(_prefsKey);
    if (raw != null) {
      return _decodeFlags(raw);
    }
    final cached = await _cacheRepository.readOnboardingFlags();
    if (cached.hasValue) {
      return cached.value!;
    }
    return OnboardingFlags.initial();
  }

  Future<OnboardingFlags> updateStep(
    OnboardingStep step, {
    bool completed = true,
  }) async {
    final current = await load();
    final updated = current.markStep(step, completed);
    await _persist(updated);
    return updated;
  }

  Future<void> replace(OnboardingFlags flags) async {
    await _persist(flags);
  }

  Future<void> reset() async {
    await _preferences.remove(_prefsKey);
    await _cacheRepository.writeOnboardingFlags(OnboardingFlags.initial());
  }

  Future<void> _persist(OnboardingFlags flags) async {
    final json = jsonEncode(flags.toJson());
    final previous = _preferences.getString(_prefsKey);
    final wrote = await _preferences.setString(_prefsKey, json);
    if (!wrote) {
      throw StateError('Failed to persist onboarding flags to preferences');
    }
    try {
      await _cacheRepository.writeOnboardingFlags(flags);
    } catch (error) {
      if (previous == null) {
        await _preferences.remove(_prefsKey);
      } else {
        await _preferences.setString(_prefsKey, previous);
      }
      rethrow;
    }
  }

  OnboardingFlags _decodeFlags(String raw) {
    return OnboardingFlags.fromJson(
      Map<String, dynamic>.from(jsonDecode(raw) as Map),
    );
  }
}
