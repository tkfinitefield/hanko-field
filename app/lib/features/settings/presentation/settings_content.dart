import 'dart:convert';

import 'package:flutter/services.dart';

class SettingsContentBundle {
  const SettingsContentBundle({
    required this.about,
    required this.howItWorks,
    required this.faq,
    required this.privacy,
    required this.terms,
    required this.contact,
  });

  final SettingsAboutContent about;
  final SettingsHowItWorksContent howItWorks;
  final SettingsFaqContent faq;
  final SettingsLegalContent privacy;
  final SettingsLegalContent terms;
  final SettingsContactContent contact;

  static const _assetRoot = 'assets/i18n/settings';
  static final Map<AssetBundle, Map<String, Future<SettingsContentBundle>>>
  _cache =
      Map<AssetBundle, Map<String, Future<SettingsContentBundle>>>.identity();

  static Future<SettingsContentBundle> forLanguage(
    String routeCode, {
    AssetBundle? bundle,
  }) async {
    final normalized = routeCode.trim().toLowerCase();
    final assetBundle = bundle ?? rootBundle;
    final assetName = _supportedAssetNames.contains(normalized)
        ? normalized
        : 'en';
    final bundleCache = _cache.putIfAbsent(assetBundle, () => {});
    final cached = bundleCache[assetName];
    if (cached != null) {
      return cached;
    }

    final assetPath = '$_assetRoot/$assetName.json';
    final pending = () async {
      final source = await assetBundle.loadString(assetPath);
      return SettingsContentBundle.fromJson(_decodeObject(source, assetPath));
    }();
    bundleCache[assetName] = pending;

    try {
      return await pending;
    } catch (_) {
      if (identical(bundleCache[assetName], pending)) {
        bundleCache.remove(assetName);
      }
      rethrow;
    }
  }

  factory SettingsContentBundle.fromJson(Map<String, Object?> json) {
    return SettingsContentBundle(
      about: SettingsAboutContent.fromJson(_object(json, 'about')),
      howItWorks: SettingsHowItWorksContent.fromJson(
        _object(json, 'howItWorks'),
      ),
      faq: SettingsFaqContent.fromJson(_object(json, 'faq')),
      privacy: SettingsLegalContent.fromJson(_object(json, 'privacy')),
      terms: SettingsLegalContent.fromJson(_object(json, 'terms')),
      contact: SettingsContactContent.fromJson(_object(json, 'contact')),
    );
  }
}

const _supportedAssetNames = {'ar', 'en', 'ja', 'zh', 'zhtw'};

class SettingsAboutContent {
  const SettingsAboutContent({
    required this.heading,
    required this.body,
    required this.points,
    required this.tagline,
  });

  final String heading;
  final String body;
  final List<SettingsTextSection> points;
  final String tagline;

  factory SettingsAboutContent.fromJson(Map<String, Object?> json) {
    return SettingsAboutContent(
      heading: _string(json, 'heading'),
      body: _string(json, 'body'),
      points: _list(
        json,
        'points',
        (value) => SettingsTextSection.fromJson(_objectValue(value)),
      ),
      tagline: _string(json, 'tagline'),
    );
  }
}

class SettingsFaqContent {
  const SettingsFaqContent({required this.heading, required this.items});

  final String heading;
  final List<SettingsFaqItem> items;

  factory SettingsFaqContent.fromJson(Map<String, Object?> json) {
    return SettingsFaqContent(
      heading: _string(json, 'heading'),
      items: _list(
        json,
        'items',
        (value) => SettingsFaqItem.fromJson(_objectValue(value)),
      ),
    );
  }
}

class SettingsHowItWorksContent {
  const SettingsHowItWorksContent({
    required this.heading,
    required this.intro,
    required this.steps,
    required this.summaryTitle,
    required this.summaryBody,
  });

  final String heading;
  final String intro;
  final List<SettingsTextSection> steps;
  final String summaryTitle;
  final String summaryBody;

  factory SettingsHowItWorksContent.fromJson(Map<String, Object?> json) {
    return SettingsHowItWorksContent(
      heading: _string(json, 'heading'),
      intro: _string(json, 'intro'),
      steps: _list(
        json,
        'steps',
        (value) => SettingsTextSection.fromJson(_objectValue(value)),
      ),
      summaryTitle: _string(json, 'summaryTitle'),
      summaryBody: _string(json, 'summaryBody'),
    );
  }
}

class SettingsFaqItem {
  const SettingsFaqItem({required this.question, required this.answer});

  final String question;
  final String answer;

  factory SettingsFaqItem.fromJson(Map<String, Object?> json) {
    return SettingsFaqItem(
      question: _string(json, 'question'),
      answer: _string(json, 'answer'),
    );
  }
}

class SettingsContactContent {
  const SettingsContactContent({
    required this.heading,
    required this.intro,
    required this.options,
    required this.replyNote,
  });

  final String heading;
  final String intro;
  final List<SettingsContactOption> options;
  final String replyNote;

  factory SettingsContactContent.fromJson(Map<String, Object?> json) {
    return SettingsContactContent(
      heading: _string(json, 'heading'),
      intro: _string(json, 'intro'),
      options: _list(
        json,
        'options',
        (value) => SettingsContactOption.fromJson(_objectValue(value)),
      ),
      replyNote: _string(json, 'replyNote'),
    );
  }
}

class SettingsContactOption {
  const SettingsContactOption({
    required this.title,
    required this.body,
    required this.value,
  });

  final String title;
  final String body;
  final String value;

  factory SettingsContactOption.fromJson(Map<String, Object?> json) {
    return SettingsContactOption(
      title: _string(json, 'title'),
      body: _string(json, 'body'),
      value: _string(json, 'value'),
    );
  }
}

class SettingsLegalContent {
  const SettingsLegalContent({
    required this.updated,
    required this.intro,
    required this.officialLinkLabel,
    required this.officialUrl,
    required this.sections,
  });

  final String updated;
  final String intro;
  final String officialLinkLabel;
  final String officialUrl;
  final List<SettingsTextSection> sections;

  factory SettingsLegalContent.fromJson(Map<String, Object?> json) {
    return SettingsLegalContent(
      updated: _string(json, 'updated'),
      intro: _string(json, 'intro'),
      officialLinkLabel: _string(json, 'officialLinkLabel'),
      officialUrl: _string(json, 'officialUrl'),
      sections: _list(
        json,
        'sections',
        (value) => SettingsTextSection.fromJson(_objectValue(value)),
      ),
    );
  }
}

class SettingsTextSection {
  const SettingsTextSection({required this.title, required this.body});

  final String title;
  final String body;

  factory SettingsTextSection.fromJson(Map<String, Object?> json) {
    return SettingsTextSection(
      title: _string(json, 'title'),
      body: _string(json, 'body'),
    );
  }
}

Map<String, Object?> _decodeObject(String source, String path) {
  final decoded = jsonDecode(source);
  if (decoded is Map<String, Object?>) {
    return decoded;
  }
  throw FormatException('Settings content root must be an object.', path);
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

List<T> _list<T>(
  Map<String, Object?> json,
  String key,
  T Function(Object? value) parse,
) {
  final value = _required(json, key);
  if (value is List<Object?>) {
    return List<T>.unmodifiable(value.map(parse));
  }
  throw FormatException('Expected list for "$key".');
}

Object? _required(Map<String, Object?> json, String key) {
  if (json.containsKey(key)) {
    return json[key];
  }
  throw FormatException('Missing required settings content key "$key".');
}
