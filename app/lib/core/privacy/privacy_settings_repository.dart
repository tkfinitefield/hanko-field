import 'package:app/core/storage/secure_storage_service.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

class PrivacySettingsRepository {
  PrivacySettingsRepository(this._secureStorage);

  static const _crashReportingKey = 'privacy.crash_reporting_allowed';
  static const _analyticsKey = 'privacy.analytics_allowed';

  final SecureStorageService _secureStorage;

  Future<bool> isCrashReportingAllowed() => _readFlag(_crashReportingKey);

  Future<void> setCrashReportingAllowed(bool allowed) =>
      _writeFlag(_crashReportingKey, allowed);

  Future<bool> isAnalyticsAllowed() => _readFlag(_analyticsKey);

  Future<void> setAnalyticsAllowed(bool allowed) =>
      _writeFlag(_analyticsKey, allowed);

  Future<bool> _readFlag(String key) async {
    final raw = await _secureStorage.read(key: key);
    if (raw == null) {
      return false;
    }
    return raw == 'true';
  }

  Future<void> _writeFlag(String key, bool value) {
    return _secureStorage.write(key: key, value: value.toString());
  }
}

final privacySettingsRepositoryProvider = Provider<PrivacySettingsRepository>((
  ref,
) {
  final secureStorage = ref.watch(secureStorageProvider);
  return PrivacySettingsRepository(secureStorage);
});
