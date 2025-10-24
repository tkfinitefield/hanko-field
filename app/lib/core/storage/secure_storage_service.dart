import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// Minimal wrapper around [FlutterSecureStorage] that centralises the
/// platform-specific configuration for Android/iOS/macOS key stores.
class SecureStorageService {
  SecureStorageService({FlutterSecureStorage? storage})
    : _storage =
          storage ??
          const FlutterSecureStorage(
            aOptions: AndroidOptions(encryptedSharedPreferences: true),
            iOptions: IOSOptions(
              accessibility: KeychainAccessibility.first_unlock,
            ),
            mOptions: MacOsOptions(
              accessibility: KeychainAccessibility.first_unlock,
            ),
          );

  final FlutterSecureStorage _storage;

  Future<void> write({required String key, required String value}) {
    return _storage.write(key: key, value: value);
  }

  Future<String?> read({required String key}) {
    return _storage.read(key: key);
  }

  Future<void> delete({required String key}) {
    return _storage.delete(key: key);
  }

  Future<void> deleteAll() {
    return _storage.deleteAll();
  }
}

final secureStorageProvider = Provider<SecureStorageService>(
  (ref) => SecureStorageService(),
);

/// Repository abstraction for authentication tokens written to secure storage.
class AuthTokenStorage {
  AuthTokenStorage(this._secureStorage);

  static const _accessTokenKey = 'auth.access_token';
  static const _refreshTokenKey = 'auth.refresh_token';

  final SecureStorageService _secureStorage;

  Future<void> saveTokens({
    required String accessToken,
    required String refreshToken,
  }) async {
    await Future.wait([
      _secureStorage.write(key: _accessTokenKey, value: accessToken),
      _secureStorage.write(key: _refreshTokenKey, value: refreshToken),
    ]);
  }

  Future<String?> readAccessToken() {
    return _secureStorage.read(key: _accessTokenKey);
  }

  Future<String?> readRefreshToken() {
    return _secureStorage.read(key: _refreshTokenKey);
  }

  Future<void> clearTokens() async {
    await Future.wait([
      _secureStorage.delete(key: _accessTokenKey),
      _secureStorage.delete(key: _refreshTokenKey),
    ]);
  }
}

final authTokenStorageProvider = Provider<AuthTokenStorage>((ref) {
  final secure = ref.watch(secureStorageProvider);
  return AuthTokenStorage(secure);
});
