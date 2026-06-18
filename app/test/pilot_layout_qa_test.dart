import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:hankofield/app/localization/app_localization.dart';
import 'package:hankofield/app/theme/app_theme.dart';
import 'package:hankofield/features/design/design.dart';
import 'package:hankofield/features/order/order.dart';
import 'package:hankofield/features/settings/settings.dart';

void main() {
  const viewport = Size(390, 844);

  testWidgets('M7-T05 pilot app surfaces render without overflow', (
    tester,
  ) async {
    await _withFlutterErrorCapture(tester, () async {
      await _pumpPilotSurface(
        tester,
        locale: const Locale('zh'),
        viewport: viewport,
        child: const SettingsHomeScreen(),
      );
      expect(find.text('Settings'), findsOneWidget);

      await _pumpPilotSurface(
        tester,
        locale: const Locale.fromSubtags(
          languageCode: 'zh',
          scriptCode: 'Hant',
        ),
        viewport: viewport,
        child: const DesignHomeScreen(),
      );
      expect(find.text('Design'), findsWidgets);

      await _pumpPilotSurface(
        tester,
        locale: const Locale('ar'),
        viewport: viewport,
        child: CheckoutInputScreen(input: _checkoutInput()),
      );
      expect(find.text('Checkout Information'), findsOneWidget);
      expect(
        Directionality.of(tester.element(find.text('Checkout Information'))),
        TextDirection.rtl,
      );
    });
  });
}

Future<void> _pumpPilotSurface(
  WidgetTester tester, {
  required Locale locale,
  required Size viewport,
  required Widget child,
}) async {
  tester.view.physicalSize = viewport;
  tester.view.devicePixelRatio = 1;
  addTearDown(tester.view.resetPhysicalSize);
  addTearDown(tester.view.resetDevicePixelRatio);

  await tester.pumpWidget(
    MaterialApp(
      locale: locale,
      supportedLocales: GeneratedHankoLocalizations.supportedLocales,
      localizationsDelegates:
          GeneratedHankoLocalizations.localizationsDelegates,
      theme: HankoTheme.light(),
      home: MediaQuery(
        data: MediaQueryData(size: viewport),
        child: child,
      ),
    ),
  );
  await tester.pump();
  await tester.pump(const Duration(milliseconds: 250));
  await tester.pump(const Duration(milliseconds: 250));
}

Future<void> _withFlutterErrorCapture(
  WidgetTester tester,
  Future<void> Function() run,
) async {
  final previousOnError = FlutterError.onError;
  final details = <FlutterErrorDetails>[];
  FlutterError.onError = details.add;
  addTearDown(() => FlutterError.onError = previousOnError);

  try {
    await run();
  } finally {
    FlutterError.onError = previousOnError;
  }

  expect(
    details,
    isEmpty,
    reason: details.map((detail) => detail.exceptionAsString()).join('\n'),
  );
  expect(tester.takeException(), isNull);
}

OrderDraftInput _checkoutInput() {
  return const OrderDraftInput(
    contact: OrderDraftContactInput(
      email: 'customer@example.test',
      preferredLocale: 'ar',
    ),
    shipping: OrderDraftShippingInput(
      countryCode: 'US',
      recipientName: 'Alexandria Montgomery',
      phone: '+1 555 0000 0000',
      postalCode: '94103',
      state: 'California',
      city: 'San Francisco',
      addressLine1: '1 Market Street',
      addressLine2: 'Suite 1200',
    ),
    orderNote: 'Please keep the gift packaging intact.',
    termsAgreed: true,
    customerConfirmation: OrderDraftCustomerConfirmationInput(
      kanjiAndDesign: true,
      customMadePolicy: true,
    ),
  );
}
