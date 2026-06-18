import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/app/localization/language_registry.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test('loads selectable app languages from the checked-in registry', () async {
    final registry = await AppLanguageRegistry.load();

    expect(registry.selectableLanguages.map((language) => language.routeCode), [
      'en',
      'ja',
    ]);
    expect(
      registry.selectableLanguages.map((language) => language.nativeName),
      ['English', '日本語'],
    );
    expect(
      registry.selectableLanguages.map((language) => language.englishName),
      ['English', 'Japanese'],
    );
  });

  test('filters out app-disabled and non-selectable languages', () {
    final registry = AppLanguageRegistry.fromJson([
      _language(
        routeCode: 'en',
        nativeName: 'English',
        englishName: 'English',
        appEnabled: true,
        appSelectable: true,
      ),
      _language(
        routeCode: 'zh',
        nativeName: '简体中文',
        englishName: 'Simplified Chinese',
        appEnabled: true,
        appSelectable: false,
      ),
      _language(
        routeCode: 'fr',
        nativeName: 'Français',
        englishName: 'French',
        appEnabled: false,
        appSelectable: true,
      ),
    ]);

    expect(registry.selectableLanguages, hasLength(1));
    expect(registry.selectableLanguages.single.routeCode, 'en');
  });

  test('builds locale and display labels from registry fields', () {
    final registry = AppLanguageRegistry.fromJson([
      _language(
        routeCode: 'zhtw',
        languageCode: 'zh',
        scriptCode: 'Hant',
        nativeName: '繁體中文',
        englishName: 'Traditional Chinese',
        appEnabled: true,
        appSelectable: true,
      ),
    ]);

    final language = registry.selectableLanguages.single;

    expect(
      language.locale,
      const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
    );
    expect(language.nativeName, '繁體中文');
    expect(language.englishNameLabel, 'Traditional Chinese');
    expect(
      language.matchesLocale(
        const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
      ),
      isTrue,
    );
    expect(registry.enabledLanguageForRouteCode(' ZHTW ')?.routeCode, 'zhtw');
    expect(
      registry
          .enabledLanguageForLocale(
            const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
          )
          ?.routeCode,
      'zhtw',
    );
  });

  test('does not resolve app-disabled languages for runtime locale use', () {
    final registry = AppLanguageRegistry.fromJson([
      _language(
        routeCode: 'zh',
        nativeName: '简体中文',
        englishName: 'Simplified Chinese',
        appEnabled: false,
        appSelectable: false,
      ),
    ]);

    expect(registry.enabledLanguageForRouteCode('zh'), isNull);
    expect(registry.enabledLanguageForLocale(const Locale('zh')), isNull);
  });
}

Map<String, Object?> _language({
  required String routeCode,
  String? languageCode,
  String? scriptCode,
  String? countryCode,
  required String nativeName,
  required String englishName,
  required bool appEnabled,
  required bool appSelectable,
}) {
  return {
    'route_code': routeCode,
    'flutter': {
      'languageCode': languageCode ?? routeCode,
      'scriptCode': scriptCode,
      'countryCode': countryCode,
    },
    'native_name': nativeName,
    'english_name': englishName,
    'app': {'enabled': appEnabled, 'selectable': appSelectable},
  };
}
