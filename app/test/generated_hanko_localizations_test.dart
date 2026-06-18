import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/l10n/generated/generated_hanko_localizations.dart';

void main() {
  test('generated localizations support migration baseline locales', () {
    expect(GeneratedHankoLocalizations.supportedLocales, const [
      Locale('en'),
      Locale('ja'),
      Locale('zh'),
      Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
    ]);
  });

  test('generated lookup resolves registry Simplified Chinese locale', () {
    final l10n = lookupGeneratedHankoLocalizations(
      const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hans'),
    );

    expect(l10n.designKanjiStyleChinese, '中国风格');
    expect(l10n.designKanjiStyleTaiwanese, '台湾风格');
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
