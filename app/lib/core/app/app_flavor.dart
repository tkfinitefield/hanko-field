import 'package:flutter_riverpod/flutter_riverpod.dart';

enum AppFlavor { dev, stg, prod }

final appFlavorProvider = Provider<AppFlavor>((ref) {
  const fromEnv = String.fromEnvironment('FLAVOR', defaultValue: 'dev');
  switch (fromEnv) {
    case 'prod':
      return AppFlavor.prod;
    case 'stg':
      return AppFlavor.stg;
    default:
      return AppFlavor.dev;
  }
});

class AppConfig {
  AppConfig({required this.baseUrl, required this.displayName});
  final String baseUrl;
  final String displayName;
}

final appConfigProvider = Provider<AppConfig>((ref) {
  final flavor = ref.watch(appFlavorProvider);
  switch (flavor) {
    case AppFlavor.prod:
      return AppConfig(
        baseUrl: 'https://api.hanko-field.app/api/v1',
        displayName: 'Hanko Field',
      );
    case AppFlavor.stg:
      return AppConfig(
        baseUrl: 'https://stg-api.hanko-field.app/api/v1',
        displayName: 'Hanko Field Staging',
      );
    case AppFlavor.dev:
    default:
      return AppConfig(
        baseUrl: 'http://10.0.2.2:8080/api/v1',
        displayName: 'Hanko Field Dev',
      );
  }
});

