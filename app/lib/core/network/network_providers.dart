import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/network/connectivity_service.dart';
import 'package:app/core/network/interceptors/auth_interceptor.dart';
import 'package:app/core/network/interceptors/connectivity_interceptor.dart';
import 'package:app/core/network/interceptors/http_interceptor.dart';
import 'package:app/core/network/interceptors/logging_interceptor.dart';
import 'package:app/core/network/network_client.dart';
import 'package:app/core/network/network_config.dart';
import 'package:app/core/network/retry_policy.dart';
import 'package:app/core/storage/secure_storage_service.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:http/http.dart' as http;
import 'package:logging/logging.dart';

const _appVersion = String.fromEnvironment(
  'APP_VERSION',
  defaultValue: '1.0.0',
);

final networkLoggerProvider = Provider<Logger>((ref) {
  return Logger('network');
});

final connectivityServiceProvider = Provider<ConnectivityService>((ref) {
  return ConnectivityService(null);
});

final networkConfigProvider = Provider<NetworkConfig>((ref) {
  final appConfig = ref.watch(appConfigProvider);
  final localeTag = PlatformDispatcher.instance.locale.toLanguageTag();
  final platform = _platformName();

  return NetworkConfig(
    baseUrl: appConfig.baseUrl,
    userAgent: 'HankoField/${appConfig.displayName}; v=$_appVersion; $platform',
    localeTag: localeTag,
  );
});

final httpClientProvider = Provider<http.Client>((ref) {
  final client = http.Client();
  ref.onDispose(client.close);
  return client;
});

final retryPolicyProvider = Provider<RetryPolicy>((ref) {
  return const RetryPolicy();
});

final networkClientProvider = Provider<NetworkClient>((ref) {
  final client = ref.watch(httpClientProvider);
  final config = ref.watch(networkConfigProvider);
  final logger = ref.watch(networkLoggerProvider);
  final connectivity = ref.watch(connectivityServiceProvider);
  final tokenStorage = ref.watch(authTokenStorageProvider);
  final retryPolicy = ref.watch(retryPolicyProvider);

  final interceptors = <HttpInterceptor>[
    LoggingInterceptor(logger: logger),
    ConnectivityInterceptor(connectivity),
    AuthInterceptor(tokenStorage),
  ];

  return NetworkClient(
    client: client,
    config: config,
    interceptors: interceptors,
    retryPolicy: retryPolicy,
  );
});

String _platformName() {
  try {
    return defaultTargetPlatform.name;
  } catch (_) {
    return 'unknown';
  }
}
