import 'dart:convert';

import 'package:flutter/widgets.dart';
import 'package:flutter/services.dart';

class AppLanguageRegistry {
  const AppLanguageRegistry({
    required this.languages,
    required this.selectableLanguages,
  });

  final List<AppLanguageOption> languages;
  final List<AppLanguageOption> selectableLanguages;

  List<Locale> get enabledLocales =>
      _defaultFirstLocales(languages.where((language) => language.appEnabled));

  List<Locale> get selectableLocales =>
      _defaultFirstLocales(selectableLanguages);

  static const assetPath = '../config/languages.json';

  static Future<AppLanguageRegistry> load({AssetBundle? bundle}) async {
    final assetBundle = bundle ?? rootBundle;
    final source = await assetBundle.loadString(assetPath);
    return AppLanguageRegistry.fromJson(_decodeList(source, assetPath));
  }

  factory AppLanguageRegistry.fromJson(List<Object?> json) {
    final languages = List<AppLanguageOption>.unmodifiable(
      json.map((value) => AppLanguageOption.fromJson(_objectValue(value))),
    );
    final selectableLanguages = languages
        .where((language) => language.appEnabled && language.appSelectable)
        .toList(growable: false);

    return AppLanguageRegistry(
      languages: languages,
      selectableLanguages: List<AppLanguageOption>.unmodifiable(
        selectableLanguages,
      ),
    );
  }

  AppLanguageOption? enabledLanguageForRouteCode(String? routeCode) {
    final normalized = _normalizeRouteCode(routeCode);
    if (normalized == null) {
      return null;
    }
    for (final language in languages) {
      if (language.appEnabled && language.routeCode == normalized) {
        return language;
      }
    }
    return null;
  }

  AppLanguageOption? enabledLanguageForLocale(Locale? locale) {
    if (locale == null) {
      return null;
    }
    for (final language in languages) {
      if (language.appEnabled && language.matchesLocale(locale)) {
        return language;
      }
    }
    return null;
  }

  AppLanguageOption? selectableLanguageForRouteCode(String? routeCode) {
    final normalized = _normalizeRouteCode(routeCode);
    if (normalized == null) {
      return null;
    }
    for (final language in selectableLanguages) {
      if (language.routeCode == normalized) {
        return language;
      }
    }
    return null;
  }

  AppLanguageOption? selectableLanguageForLocale(Locale? locale) {
    if (locale == null) {
      return null;
    }
    for (final language in selectableLanguages) {
      if (language.matchesLocale(locale)) {
        return language;
      }
    }
    return null;
  }

  AppLanguageOption? languageForLocale(Locale? locale) {
    if (locale == null) {
      return null;
    }
    for (final language in languages) {
      if (language.matchesLocale(locale)) {
        return language;
      }
    }
    return null;
  }

  String? routeCodeForLocale(Locale? locale) {
    return languageForLocale(locale)?.routeCode;
  }
}

class AppLanguageOption {
  const AppLanguageOption({
    required this.routeCode,
    required this.locale,
    required this.nativeName,
    required this.englishName,
    required this.textDirection,
    required this.appEnabled,
    required this.appSelectable,
  });

  final String routeCode;
  final Locale locale;
  final String nativeName;
  final String englishName;
  final TextDirection textDirection;
  final bool appEnabled;
  final bool appSelectable;

  String? get englishNameLabel {
    if (englishName == nativeName) {
      return null;
    }
    return englishName;
  }

  bool matchesLocale(Locale currentLocale) {
    return locale.languageCode == currentLocale.languageCode &&
        locale.scriptCode == currentLocale.scriptCode &&
        locale.countryCode == currentLocale.countryCode;
  }

  factory AppLanguageOption.fromJson(Map<String, Object?> json) {
    final flutter = _object(json, 'flutter');
    final app = _object(json, 'app');

    return AppLanguageOption(
      routeCode: _string(json, 'route_code'),
      locale: Locale.fromSubtags(
        languageCode: _string(flutter, 'languageCode'),
        scriptCode: _optionalString(flutter, 'scriptCode'),
        countryCode: _optionalString(flutter, 'countryCode'),
      ),
      nativeName: _string(json, 'native_name'),
      englishName: _string(json, 'english_name'),
      textDirection: _textDirection(json, 'text_direction'),
      appEnabled: _bool(app, 'enabled'),
      appSelectable: _bool(app, 'selectable'),
    );
  }
}

String fallbackRouteCodeForLocale(Locale locale) {
  final languageCode = locale.languageCode.trim().toLowerCase();
  final scriptCode = locale.scriptCode?.trim().toLowerCase();
  final countryCode = locale.countryCode?.trim().toLowerCase();

  if (languageCode == 'zh') {
    if (scriptCode == 'hant' ||
        countryCode == 'tw' ||
        countryCode == 'hk' ||
        countryCode == 'mo') {
      return 'zhtw';
    }
    return 'zh';
  }

  return languageCode.isEmpty ? 'en' : languageCode;
}

Locale resolveAutomaticLocale(
  List<Locale>? preferredLocales,
  Iterable<Locale> selectableLocales, {
  Locale fallbackLocale = const Locale('en'),
}) {
  final selectable = selectableLocales.toList(growable: false);
  for (final preferred in preferredLocales ?? const <Locale>[]) {
    for (final candidate in selectable) {
      if (_sameLocale(candidate, preferred)) {
        return candidate;
      }
    }
    for (final candidate in selectable) {
      if (candidate.languageCode == preferred.languageCode &&
          (candidate.scriptCode == null ||
              candidate.scriptCode == preferred.scriptCode)) {
        return candidate;
      }
    }
  }
  for (final candidate in selectable) {
    if (_sameLocale(candidate, fallbackLocale)) {
      return candidate;
    }
  }
  return selectable.isNotEmpty ? selectable.first : fallbackLocale;
}

List<Locale> _defaultFirstLocales(Iterable<AppLanguageOption> languages) {
  final locales = languages.map((language) => language.locale).toList();
  locales.sort((left, right) {
    if (left.languageCode == 'en') {
      return -1;
    }
    if (right.languageCode == 'en') {
      return 1;
    }
    return fallbackRouteCodeForLocale(
      left,
    ).compareTo(fallbackRouteCodeForLocale(right));
  });
  return List<Locale>.unmodifiable(locales);
}

bool _sameLocale(Locale left, Locale right) {
  return left.languageCode == right.languageCode &&
      left.scriptCode == right.scriptCode &&
      left.countryCode == right.countryCode;
}

List<Object?> _decodeList(String source, String path) {
  final decoded = jsonDecode(source);
  if (decoded is List<Object?>) {
    return decoded;
  }
  throw FormatException('Language registry root must be an array.', path);
}

Map<String, Object?> _object(Map<String, Object?> json, String key) {
  return _objectValue(_required(json, key));
}

Map<String, Object?> _objectValue(Object? value) {
  if (value is Map<String, Object?>) {
    return value;
  }
  throw const FormatException('Expected a JSON object.');
}

String _string(Map<String, Object?> json, String key) {
  final value = _required(json, key);
  if (value is String) {
    return value;
  }
  throw FormatException('Expected string for "$key".');
}

String? _optionalString(Map<String, Object?> json, String key) {
  final value = _required(json, key);
  if (value == null) {
    return null;
  }
  if (value is String) {
    return value;
  }
  throw FormatException('Expected string or null for "$key".');
}

bool _bool(Map<String, Object?> json, String key) {
  final value = _required(json, key);
  if (value is bool) {
    return value;
  }
  throw FormatException('Expected boolean for "$key".');
}

TextDirection _textDirection(Map<String, Object?> json, String key) {
  return switch (_string(json, key)) {
    'ltr' => TextDirection.ltr,
    'rtl' => TextDirection.rtl,
    final value => throw FormatException(
      'Expected "ltr" or "rtl" for "$key", got "$value".',
    ),
  };
}

Object? _required(Map<String, Object?> json, String key) {
  if (json.containsKey(key)) {
    return json[key];
  }
  throw FormatException('Missing required language registry key "$key".');
}

String? _normalizeRouteCode(String? routeCode) {
  final normalized = routeCode?.trim().toLowerCase();
  if (normalized == null || normalized.isEmpty) {
    return null;
  }
  return normalized;
}
