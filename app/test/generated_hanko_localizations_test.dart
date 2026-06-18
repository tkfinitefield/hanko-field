import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/l10n/generated/generated_hanko_localizations.dart';

void main() {
  test('generated localizations support English and Japanese', () {
    expect(GeneratedHankoLocalizations.supportedLocales, const [
      Locale('en'),
      Locale('ja'),
    ]);
  });

  testWidgets('generated delegate loads the app title', (tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        locale: Locale('ja'),
        supportedLocales: GeneratedHankoLocalizations.supportedLocales,
        localizationsDelegates:
            GeneratedHankoLocalizations.localizationsDelegates,
        home: _GeneratedTitleProbe(),
      ),
    );

    expect(find.text('STONE SIGNATURE'), findsOneWidget);
  });
}

class _GeneratedTitleProbe extends StatelessWidget {
  const _GeneratedTitleProbe();

  @override
  Widget build(BuildContext context) {
    return Text(GeneratedHankoLocalizations.of(context).appTitle);
  }
}
