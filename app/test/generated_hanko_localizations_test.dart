import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/app/localization/hanko_localizations.dart';
import 'package:hankofield/l10n/generated/generated_hanko_localizations.dart';

void main() {
  test('generated localizations support registry route locales', () {
    expect(GeneratedHankoLocalizations.supportedLocales, _registryLocales());
  });

  test('generated lookup resolves registry Simplified Chinese locale', () {
    final l10n = lookupGeneratedHankoLocalizations(
      const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hans'),
    );

    expect(l10n.designKanjiStyleChinese, '中国风格');
    expect(l10n.designKanjiStyleTaiwanese, '台湾风格');
  });

  test('HankoLocalizations compatibility API uses generated localization', () {
    expect(hankoSupportedLocales, const [
      Locale('en'),
      Locale('ja'),
      Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hans'),
      Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
      Locale('ar'),
    ]);

    final l10n = lookupGeneratedHankoLocalizations(const Locale('ja'));
    expect(l10n.locale, const Locale('ja'));
    expect(l10n.designTipPrefix, 'ヒント: ');
    expect(l10n.orderStatusPaid, '支払い済み');
    expect(l10n.settingsVersionMessage('1.2.3'), 'バージョン 1.2.3');
  });

  test('generated lookup resolves Arabic pilot locale', () {
    final l10n = lookupGeneratedHankoLocalizations(const Locale('ar'));

    expect(l10n.localeName, 'ar');
    expect(l10n.appTitle, 'STONE SIGNATURE');
    expect(l10n.settingsLanguageTitle, 'App language');
  });

  testWidgets('generated delegate loads Chinese migration assets', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        locale: Locale('zh'),
        supportedLocales: GeneratedHankoLocalizations.supportedLocales,
        localizationsDelegates:
            GeneratedHankoLocalizations.localizationsDelegates,
        home: _GeneratedStyleProbe(),
      ),
    );

    expect(find.text('STONE SIGNATURE'), findsOneWidget);
    expect(find.text('中国风格'), findsOneWidget);
    expect(find.text('台湾风格'), findsOneWidget);
  });

  testWidgets('generated delegate distinguishes Traditional Chinese assets', (
    tester,
  ) async {
    await tester.pumpWidget(
      const MaterialApp(
        locale: Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
        supportedLocales: GeneratedHankoLocalizations.supportedLocales,
        localizationsDelegates:
            GeneratedHankoLocalizations.localizationsDelegates,
        home: _GeneratedStyleProbe(),
      ),
    );

    expect(find.text('STONE SIGNATURE'), findsOneWidget);
    expect(find.text('中國風格'), findsOneWidget);
    expect(find.text('台灣風格'), findsOneWidget);
  });
}

List<Locale> _registryLocales() {
  final source = File('../config/languages.json').readAsStringSync();
  final entries = (jsonDecode(source) as List<Object?>)
      .cast<Map<String, Object?>>();
  return entries
      .map((entry) {
        final flutter = (entry['flutter'] as Map<String, Object?>);
        if (entry['route_code'] == 'zh') {
          return const Locale('zh');
        }
        return Locale.fromSubtags(
          languageCode: flutter['languageCode'] as String,
          scriptCode: flutter['scriptCode'] as String?,
          countryCode: flutter['countryCode'] as String?,
        );
      })
      .toList(growable: false);
}

class _GeneratedStyleProbe extends StatelessWidget {
  const _GeneratedStyleProbe();

  @override
  Widget build(BuildContext context) {
    final l10n = GeneratedHankoLocalizations.of(context);
    return Column(
      children: [
        Text(l10n.appTitle),
        Text(l10n.designKanjiStyleChinese),
        Text(l10n.designKanjiStyleTaiwanese),
      ],
    );
  }
}
