import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/features/order/order.dart';

void main() {
  test('parses custom scheme checkout success returns', () {
    final result = parseCheckoutReturnRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=ja',
    );

    expect(result?.outcome, CheckoutReturnOutcome.success);
    expect(result?.orderId, 'ord_001');
    expect(result?.sessionId, 'cs_test_001');
    expect(result?.locale, 'ja');
  });

  test('parses localized web checkout cancel returns', () {
    final result = parseCheckoutReturnRoute(
      'https://finitefield.org/ja/payment/cancel?checkout=cancel&order_id=ord_002',
    );

    expect(result?.outcome, CheckoutReturnOutcome.canceled);
    expect(result?.orderId, 'ord_002');
  });

  test('parses universal link checkout success returns', () {
    final result = parseCheckoutReturnRoute(
      'https://www.finitefield.org/en/payment/success?checkout=success&order_id=ord_005&session_id=cs_test_005&lang=en',
    );

    expect(result?.outcome, CheckoutReturnOutcome.success);
    expect(result?.orderId, 'ord_005');
    expect(result?.sessionId, 'cs_test_005');
    expect(result?.locale, 'en');
  });

  test('normalizes lang and locale query values to route codes', () {
    final langResult = parseCheckoutReturnRoute(
      'hankofield://checkout/success?order_id=ord_006&session_id=cs_test_006&lang=zh_Hant',
    );
    final localeResult = parseCheckoutReturnRoute(
      'https://finitefield.org/zhtw/payment/success?checkout=success&order_id=ord_007&locale=zh-TW',
    );

    expect(langResult?.locale, 'zhtw');
    expect(localeResult?.locale, 'zhtw');
  });

  test('preserves pilot checkout return route codes', () {
    final appResult = parseCheckoutReturnRoute(
      'hankofield://checkout/success?order_id=ord_008&session_id=cs_test_008&lang=ar',
    );
    final webResult = parseCheckoutReturnRoute(
      'https://finitefield.org/ar/payment/success?checkout=success&order_id=ord_009&locale=ar',
    );

    expect(appResult?.outcome, CheckoutReturnOutcome.success);
    expect(appResult?.locale, 'ar');
    expect(webResult?.outcome, CheckoutReturnOutcome.success);
    expect(webResult?.orderId, 'ord_009');
    expect(webResult?.locale, 'ar');
  });

  test('infers pilot locale from localized universal link paths', () {
    for (final entry in const [
      ('zh', 'https://finitefield.org/zh/payment/success'),
      ('zhtw', 'https://finitefield.org/zhtw/payment/success'),
      ('ar', 'https://finitefield.org/ar/payment/success'),
    ]) {
      final result = parseCheckoutReturnRoute(
        '${entry.$2}?checkout=success&order_id=ord_${entry.$1}&session_id=cs_test_${entry.$1}&return_to=app',
      );

      expect(result?.outcome, CheckoutReturnOutcome.success);
      expect(result?.orderId, 'ord_${entry.$1}');
      expect(result?.sessionId, 'cs_test_${entry.$1}');
      expect(result?.locale, entry.$1);
    }
  });

  test('uses checkout query to distinguish Stripe cancel from failure path', () {
    final result = parseCheckoutReturnRoute(
      'https://finitefield.org/payment/failure?checkout=cancel&order_id=ord_004',
    );

    expect(result?.outcome, CheckoutReturnOutcome.canceled);
    expect(result?.orderId, 'ord_004');
  });

  test('parses checkout failure returns', () {
    final result = parseCheckoutReturnRoute(
      'hankofield://checkout/failed?order_id=ord_003&checkout_session_id=cs_test_003',
    );

    expect(result?.outcome, CheckoutReturnOutcome.failed);
    expect(result?.orderId, 'ord_003');
    expect(result?.sessionId, 'cs_test_003');
  });

  test('ignores unrelated app routes', () {
    expect(parseCheckoutReturnRoute('/design'), isNull);
    expect(parseCheckoutReturnRoute('/design?checkout=success'), isNull);
  });

  test('detects malformed checkout return routes', () {
    expect(
      isMalformedCheckoutReturnRoute(
        'hankofield://checkout/unknown?session_id=cs_test_001',
      ),
      isTrue,
    );
    expect(isMalformedCheckoutReturnRoute('/design?checkout=success'), isFalse);
  });
}
