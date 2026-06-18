import 'dart:io';

import 'package:flutter_test/flutter_test.dart';

void main() {
  group('Android deep link config', () {
    late String manifest;

    setUpAll(() {
      manifest = File(
        'android/app/src/main/AndroidManifest.xml',
      ).readAsStringSync();
    });

    test('keeps the custom scheme checkout return route', () {
      expect(manifest, contains('android:scheme="hankofield"'));
      expect(manifest, contains('android:host="checkout"'));
    });

    test('declares verified HTTPS app links for Stripe return routes', () {
      expect(manifest, contains('android:autoVerify="true"'));
      expect(manifest, contains('android:scheme="https"'));

      for (final host in const ['finitefield.org', 'www.finitefield.org']) {
        expect(manifest, contains('android:host="$host"'));
      }
      for (final pathPrefix in const [
        '/payment',
        '/en/payment',
        '/ja/payment',
        '/zh/payment',
        '/zhtw/payment',
        '/ar/payment',
      ]) {
        expect(manifest, contains('android:pathPrefix="$pathPrefix"'));
      }
    });

    test('allows cleartext traffic only for local development API hosts', () {
      final networkConfig = File(
        'android/app/src/main/res/xml/network_security_config.xml',
      ).readAsStringSync();

      expect(
        manifest,
        contains(
          'android:networkSecurityConfig="@xml/network_security_config"',
        ),
      );
      expect(
        networkConfig,
        contains('<domain-config cleartextTrafficPermitted="true">'),
      );
      expect(
        networkConfig,
        contains('<domain includeSubdomains="true">localhost</domain>'),
      );
      expect(
        networkConfig,
        contains('<domain includeSubdomains="true">127.0.0.1</domain>'),
      );
      expect(
        networkConfig,
        contains('<domain includeSubdomains="true">10.0.2.2</domain>'),
      );
    });
  });

  group('iOS deep link config', () {
    test('keeps the custom scheme checkout return route', () {
      final infoPlist = File('ios/Runner/Info.plist').readAsStringSync();

      expect(infoPlist, contains('org.finitefield.hankofield.checkout'));
      expect(infoPlist, contains('<string>hankofield</string>'));
    });

    test('allows local HTTP API calls from the simulator', () {
      final infoPlist = File('ios/Runner/Info.plist').readAsStringSync();

      expect(infoPlist, contains('<key>NSAppTransportSecurity</key>'));
      expect(infoPlist, contains('<key>NSAllowsLocalNetworking</key>'));
    });

    test('declares associated domains for universal links', () {
      final entitlements = File(
        'ios/Runner/Runner.entitlements',
      ).readAsStringSync();

      expect(entitlements, contains('com.apple.developer.associated-domains'));
      expect(
        entitlements,
        contains('<string>applinks:finitefield.org</string>'),
      );
      expect(
        entitlements,
        contains('<string>applinks:www.finitefield.org</string>'),
      );
    });

    test('attaches entitlements to every Runner build configuration', () {
      final project = File(
        'ios/Runner.xcodeproj/project.pbxproj',
      ).readAsStringSync();
      final matches = RegExp(
        r'CODE_SIGN_ENTITLEMENTS = Runner/Runner\.entitlements;',
      ).allMatches(project);

      expect(matches, hasLength(3));
    });
  });

  test('prod env example separates web and app Stripe return URLs', () {
    final envExample = File('../.env.prod.example').readAsStringSync();

    expect(
      envExample,
      contains(
        'API_PSP_STRIPE_CHECKOUT_SUCCESS_URL=https://finitefield.org/payment/success?session_id={CHECKOUT_SESSION_ID}',
      ),
    );
    expect(
      envExample,
      contains(
        'API_PSP_STRIPE_CHECKOUT_CANCEL_URL=https://finitefield.org/payment/failure',
      ),
    );
    expect(
      envExample,
      contains(
        'API_PSP_STRIPE_APP_CHECKOUT_SUCCESS_URL=hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}',
      ),
    );
    expect(
      envExample,
      contains(
        'API_PSP_STRIPE_APP_CHECKOUT_CANCEL_URL=hankofield://checkout/cancel',
      ),
    );
    expect(
      envExample,
      contains('HANKO_WEB_SITE_BASE_URL=https://finitefield.org'),
    );
  });
}
