import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:hankofield/core/api/core_api.dart';
import 'package:hankofield/core/errors/core_errors.dart';
import 'package:hankofield/features/design/design.dart';
import 'package:hankofield/features/order/order.dart';
import 'package:hankofield/features/order_lookup/order_lookup.dart';
import 'package:hankofield/features/settings/settings.dart';
import 'package:hankofield/features/stones/stones.dart';

void main() {
  test(
    'KanjiCandidatesRepository posts snake_case and maps candidates',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 200,
          body: jsonEncode({
            'real_name': 'Michael',
            'reason_language': 'en',
            'gender': 'unspecified',
            'kanji_style': 'japanese',
            'candidates': [
              {
                'kanji': '美空',
                'reading': 'Misora',
                'meaning': 'Beautiful sky',
                'impression': ['Elegant', 'Gentle'],
                'reason': 'A graceful two-character option.',
                'character_count': 2,
                'stroke_complexity': 'medium',
                'engraving_suitability': 'high',
              },
            ],
          }),
        ),
      ]);
      final repo = KanjiCandidatesRepository(_client(transport));

      final result = await repo.generateCandidates(
        const KanjiCandidatesRequest(
          realName: 'Michael',
          reasonLanguage: 'en',
          count: 3,
        ),
      );

      expect(transport.singleRequest.method, 'POST');
      expect(transport.singleRequest.uri.path, '/v1/kanji-candidates');
      expect(transport.singleRequest.body?['real_name'], 'Michael');
      expect(transport.singleRequest.body?['kanji_style'], 'japanese');
      expect(result.candidates.single.kanji, '美空');
      expect(result.candidates.single.meaning, 'Beautiful sky');
      expect(result.candidates.single.characterCount, 2);
    },
  );

  test(
    'KanjiCandidatesRepository propagates API validation failures',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 422,
          body: jsonEncode({
            'error': {
              'code': 'unsupported_name',
              'message': 'real_name cannot be converted to kanji',
            },
          }),
        ),
      ]);
      final repo = KanjiCandidatesRepository(_client(transport));

      await expectLater(
        repo.generateCandidates(
          const KanjiCandidatesRequest(realName: '???', reasonLanguage: 'en'),
        ),
        throwsA(
          isA<HankoApiException>()
              .having((error) => error.statusCode, 'statusCode', 422)
              .having((error) => error.code, 'code', 'unsupported_name')
              .having(
                (error) => error.message,
                'message',
                'real_name cannot be converted to kanji',
              ),
        ),
      );
      expect(transport.singleRequest.uri.path, '/v1/kanji-candidates');
    },
  );

  test('SealGenerationRepository posts style and maps variants', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 200,
        body: jsonEncode({
          'request_id': 'seal_request_001',
          'variants': [
            {
              'id': 'seal_variant_001',
              'storage_path':
                  'seal_designs/seal_request_001/seal_variant_001.png',
              'download_url':
                  'https://storage.googleapis.com/hanko-assets/seal_designs/seal_request_001/seal_variant_001.png',
              'label': 'Elegant and balanced',
              'width': 1024,
              'height': 1024,
            },
            {
              'id': 'seal_variant_002',
              'storage_path':
                  'seal_designs/seal_request_001/seal_variant_002.png',
              'download_url':
                  'https://storage.googleapis.com/hanko-assets/seal_designs/seal_request_001/seal_variant_002.png',
              'label': 'Soft spacing',
              'width': 1024,
              'height': 1024,
            },
            {
              'id': 'seal_variant_003',
              'storage_path':
                  'seal_designs/seal_request_001/seal_variant_003.png',
              'download_url':
                  'https://storage.googleapis.com/hanko-assets/seal_designs/seal_request_001/seal_variant_003.png',
              'label': 'Bold readable seal',
              'width': 1024,
              'height': 1024,
            },
          ],
        }),
      ),
    ]);
    final repo = SealGenerationRepository(_client(transport));

    final result = await repo.generateSealDesigns(
      const SealGenerationRequest(
        inputName: 'Michael',
        candidate: KanjiCandidate(
          kanji: '美空',
          reading: 'Misora',
          reason: 'A graceful two-character option.',
        ),
        style: SealStyleSelection(),
      ),
    );

    expect(transport.singleRequest.method, 'POST');
    expect(transport.singleRequest.uri.path, '/v1/seal-designs/generate');
    expect(transport.singleRequest.body?['input_name'], 'Michael');
    expect(transport.singleRequest.body?['kanji'], '美空');
    expect(transport.singleRequest.body?['shape'], 'square');
    expect(transport.singleRequest.body?['style'], 'elegant');
    expect(transport.singleRequest.body?['stroke_weight'], 'standard');
    expect(transport.singleRequest.body?['balance'], 'balanced');
    expect(transport.singleRequest.body?['variant_count'], 3);
    expect(transport.singleRequest.body?['attempt_number'], 1);
    final rules = transport.singleRequest.body?['generation_rules'];
    expect(rules, isA<Map>());
    expect((rules! as Map)['plain_background'], isTrue);
    expect(result.requestId, 'seal_request_001');
    expect(result.variants, hasLength(3));
    expect(result.variants[1].id, 'seal_variant_002');
    expect(result.variants[1].storagePath, contains('seal_variant_002.png'));
    expect(result.variants[1].width, 1024);
    expect(result.variants[1].recipe, isNull);
  });

  test(
    'SealGenerationRepository maps optional recipe on generated variants',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 200,
          body: jsonEncode({
            'request_id': 'seal_request_001',
            'variants': [
              {
                'id': 'seal_variant_001',
                'storage_path':
                    'seal_designs/seal_request_001/seal_variant_001.png',
                'download_url':
                    'https://storage.googleapis.com/hanko-assets/seal_designs/seal_request_001/seal_variant_001.png',
                'label': 'Formal balanced',
                'recipe': {
                  'font_profile': 'formal_serif',
                  'impression': 'elegant',
                  'weight': 'standard',
                  'spacing': 'balanced',
                  'texture': 'none',
                  'frame': 'square_standard',
                },
                'width': 1024,
                'height': 1024,
              },
            ],
          }),
        ),
      ]);
      final repo = SealGenerationRepository(_client(transport));

      final result = await repo.generateSealDesigns(
        const SealGenerationRequest(
          inputName: 'Michael',
          candidate: KanjiCandidate(
            kanji: '美空',
            reading: 'Misora',
            reason: 'A graceful two-character option.',
          ),
          style: SealStyleSelection(),
        ),
      );

      final recipe = result.variants.single.recipe;
      expect(recipe, isNotNull);
      expect(recipe!.fontProfile, 'formal_serif');
      expect(recipe.impression, 'elegant');
      expect(recipe.weight, 'standard');
      expect(recipe.spacing, 'balanced');
      expect(recipe.texture, 'none');
      expect(recipe.frame, 'square_standard');
    },
  );

  test(
    'SealGenerationRepository posts previous recipes for regeneration',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 200,
          body: jsonEncode({'request_id': 'seal_request_002', 'variants': []}),
        ),
      ]);
      final repo = SealGenerationRepository(_client(transport));

      await repo.generateSealDesigns(
        const SealGenerationRequest(
          inputName: 'Michael',
          candidate: KanjiCandidate(
            kanji: '美空',
            reading: 'Misora',
            reason: 'A graceful two-character option.',
          ),
          style: SealStyleSelection(),
          attemptNumber: 2,
          previousRecipes: [
            SealDesignRecipe(
              fontProfile: 'formal_serif',
              impression: 'elegant',
              weight: 'standard',
              spacing: 'balanced',
              texture: 'none',
              frame: 'square_standard',
            ),
          ],
        ),
      );

      expect(transport.singleRequest.body?['attempt_number'], 2);
      final previousRecipes =
          transport.singleRequest.body?['previous_recipes'] as List<Object?>;
      expect(previousRecipes, hasLength(1));
      final previousRecipe = previousRecipes.single as Map<Object?, Object?>;
      expect(previousRecipe['font_profile'], 'formal_serif');
      expect(previousRecipe['impression'], 'elegant');
      expect(previousRecipe['weight'], 'standard');
      expect(previousRecipe['spacing'], 'balanced');
      expect(previousRecipe['texture'], 'none');
      expect(previousRecipe['frame'], 'square_standard');
    },
  );

  test('SealGenerationRepository propagates storage save failures', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 500,
        body: jsonEncode({
          'error': {
            'code': 'storage_save_failed',
            'message': 'failed to persist generated seal image',
          },
        }),
      ),
    ]);
    final repo = SealGenerationRepository(_client(transport));

    await expectLater(
      repo.generateSealDesigns(
        const SealGenerationRequest(
          inputName: 'Michael',
          candidate: KanjiCandidate(
            kanji: '美空',
            reading: 'Misora',
            reason: 'A graceful two-character option.',
          ),
          style: SealStyleSelection(
            shape: SealShape.round,
            style: SealStyleName.bold,
            strokeWeight: SealStrokeWeight.bold,
            balance: SealBalance.dense,
          ),
        ),
      ),
      throwsA(
        isA<HankoApiException>()
            .having((error) => error.statusCode, 'statusCode', 500)
            .having((error) => error.code, 'code', 'storage_save_failed'),
      ),
    );
    expect(transport.singleRequest.uri.path, '/v1/seal-designs/generate');
    expect(transport.singleRequest.body?['shape'], 'round');
    expect(transport.singleRequest.body?['style'], 'bold');
    expect(transport.singleRequest.body?['stroke_weight'], 'bold');
    expect(transport.singleRequest.body?['balance'], 'dense');
  });

  test(
    'StoneListingsRepository maps list response into domain model',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 200,
          body: jsonEncode({
            'locale': 'en',
            'currency': 'JPY',
            'stone_listings': [
              {
                'key': 'stone_listing_001',
                'listing_code': 'RQZ-0001',
                'material_key': 'rose_quartz',
                'material_label': 'Rose Quartz',
                'size': '24x24x60 mm',
                'title': 'Soft Pink Rose Quartz Seal Stone',
                'description': 'A soft pink rose quartz seal stone.',
                'story': 'A one-of-a-kind piece.',
                'facets': {
                  'color_family': 'pink',
                  'color_tags': ['soft'],
                  'pattern_primary': 'plain',
                  'pattern_tags': ['clear'],
                  'stone_shape': 'square',
                  'translucency': 'semi_translucent',
                },
                'price': 18000,
                'status': 'published',
                'is_active': true,
                'is_orderable': true,
                'sort_order': 7,
                'photos': [
                  {
                    'asset_id': 'asset_001',
                    'asset_url': 'https://example.test/stone.png',
                    'alt': 'Rose quartz',
                    'sort_order': 1,
                    'is_primary': true,
                    'width': 1200,
                    'height': 900,
                  },
                ],
              },
            ],
          }),
        ),
      ]);
      final repo = StoneListingsRepository(_client(transport));

      final result = await repo.listStoneListings(
        const StoneListingsQuery(locale: 'en', colorFamily: 'pink'),
      );

      expect(transport.singleRequest.method, 'GET');
      expect(transport.singleRequest.uri.path, '/v1/stone-listings');
      expect(transport.singleRequest.uri.queryParameters['locale'], 'en');
      expect(
        transport.singleRequest.uri.queryParameters['color_family'],
        'pink',
      );
      expect(result.listings.single.id, 'stone_listing_001');
      expect(result.listings.single.materialLabel, 'Rose Quartz');
      expect(result.listings.single.price.amount, 18000);
      expect(result.listings.single.price.currency, 'JPY');
      expect(result.listings.single.isOrderable, isTrue);
      expect(result.listings.single.sortOrder, 7);
    },
  );

  test('StoneListingsRepository gets a stone listing detail', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 200,
        body: jsonEncode({
          'id': 'stone_listing_001',
          'code': 'RQZ-0001',
          'material_key': 'rose_quartz',
          'material_label': 'Rose Quartz',
          'size': {'width_mm': 24, 'height_mm': 24, 'depth_mm': 60},
          'title': 'Soft Pink Rose Quartz Seal Stone',
          'description': 'A soft pink rose quartz seal stone.',
          'story': 'A one-of-a-kind piece.',
          'facets': {
            'color_family': 'pink',
            'color_tags': ['soft'],
            'pattern_primary': 'plain',
            'pattern_tags': ['clear'],
            'stone_shape': 'square',
            'translucency': 'semi_translucent',
          },
          'price': {
            'amount': 18000,
            'currency': 'JPY',
            'display': 'JPY 18,000',
          },
          'status': 'published',
          'is_active': true,
          'is_orderable': true,
          'sort_order': 7,
          'photos': [
            {
              'asset_id': 'asset_001',
              'asset_url': 'https://example.test/stone.png',
              'alt': 'Rose quartz',
              'sort_order': 1,
              'is_primary': true,
            },
          ],
        }),
      ),
    ]);
    final repo = StoneListingsRepository(_client(transport));

    final result = await repo.getStoneListingDetail(
      const StoneListingDetailQuery(
        listingId: 'stone_listing_001',
        locale: 'en',
      ),
    );

    expect(transport.singleRequest.method, 'GET');
    expect(
      transport.singleRequest.uri.path,
      '/v1/stone-listings/stone_listing_001',
    );
    expect(transport.singleRequest.uri.queryParameters['locale'], 'en');
    expect(result.id, 'stone_listing_001');
    expect(result.code, 'RQZ-0001');
    expect(result.sizeLabel, '24x24x60 mm');
    expect(result.price.display, 'JPY 18,000');
    expect(result.photos.single.assetUrl, 'https://example.test/stone.png');
  });

  test('OrderRepository maps order and checkout responses', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 201,
        body: jsonEncode({
          'order_id': 'ord_001',
          'order_no': 'HF-0001',
          'status': 'pending_payment',
          'payment_status': 'unpaid',
          'fulfillment_status': 'not_started',
          'pricing': {'total': 22000, 'currency': 'JPY'},
          'idempotent_replay': false,
        }),
      ),
      HankoApiResponse(
        statusCode: 201,
        body: jsonEncode({
          'order_id': 'ord_001',
          'session_id': 'cs_test_001',
          'checkout_url': 'https://checkout.stripe.test/session',
          'payment_intent_id': 'pi_test_001',
        }),
      ),
    ]);
    final repo = OrderRepository(_client(transport));

    final created = await repo.createOrder(_draft());
    final checkout = await repo.createCheckoutSession(
      const CheckoutSessionRequest(orderId: 'ord_001', returnToApp: true),
    );

    expect(transport.requests.first.body?['idempotency_key'], 'idem_001');
    expect(transport.requests.first.body?['terms_agreed'], isTrue);
    expect(transport.requests.first.body?['order_note'], 'Please pack safely.');
    final seal = transport.requests.first.body?['seal'] as Map;
    expect(seal['ai_generation_id'], 'seal_request_001');
    expect(seal['ai_variant_id'], 'seal_variant_001');
    expect(
      (seal['preview_image'] as Map)['storage_path'],
      'seal_designs/seal_request_001/seal_variant_001.png',
    );
    expect((seal['style'] as Map)['name'], 'elegant');
    final confirmation =
        transport.requests.first.body?['customer_confirmation'] as Map;
    expect(confirmation['kanji_and_design'], isTrue);
    expect(confirmation['custom_made_policy'], isTrue);
    expect(confirmation['confirmed_at'], '2026-05-21T02:00:00.000Z');
    expect(confirmation['confirmed_seal_text'], '美空');
    expect(created.orderNo, 'HF-0001');
    expect(created.pricing.amount, 22000);
    expect(
      transport.requests.last.uri.path,
      '/v1/payments/stripe/checkout-session',
    );
    expect(transport.requests.last.body?['return_to_app'], isTrue);
    expect(checkout.sessionId, 'cs_test_001');
  });

  test('OrderLookupRepository maps nested order status response', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 200,
        body: jsonEncode({
          'order_id': 'ord_001',
          'order_no': 'HF-20260521-0001',
          'created_at': '2026-05-21T11:00:00Z',
          'status': 'paid',
          'payment': {
            'status': 'paid',
            'checkout_session_id': 'cs_test_001',
            'payment_intent_id': 'pi_test_001',
          },
          'fulfillment': {
            'status': 'pending',
            'carrier': null,
            'tracking_no': 'TRACK123',
            'shipped_at': '2026-05-22T03:00:00Z',
          },
          'pricing': {'total': 18600, 'currency': 'JPY'},
          'seal': {
            'confirmed_seal_text': '美空',
            'preview_image_url': 'https://example.test/seal.png',
          },
          'listing': {
            'id': 'stone_listing_001',
            'title': 'Soft Pink Rose Quartz Seal Stone',
          },
          'updated_at': '2026-05-21T11:15:00Z',
        }),
      ),
    ]);
    final repo = OrderLookupRepository(_client(transport));

    final status = await repo.fetchOrderStatus('ord_001');

    expect(transport.singleRequest.method, 'GET');
    expect(transport.singleRequest.uri.path, '/v1/orders/ord_001/status');
    expect(status.orderStatus, 'paid');
    expect(
      status.createdAt?.toUtc().toIso8601String(),
      '2026-05-21T11:00:00.000Z',
    );
    expect(status.paymentStatus, 'paid');
    expect(status.fulfillmentStatus, 'pending');
    expect(status.trackingNumber, 'TRACK123');
    expect(
      status.shippedAt?.toUtc().toIso8601String(),
      '2026-05-22T03:00:00.000Z',
    );
    expect(status.sealText, '美空');
    expect(status.sealPreviewImageUrl, 'https://example.test/seal.png');
    expect(status.listingId, 'stone_listing_001');
    expect(status.listingTitle, 'Soft Pink Rose Quartz Seal Stone');
    expect(status.pricing.amount, 18600);
    expect(status.pricing.currency, 'JPY');
    expect(
      status.updatedAt?.toUtc().toIso8601String(),
      '2026-05-21T11:15:00.000Z',
    );
  });

  test(
    'OrderLookupRepository encodes checkout status ids and maps failures',
    () async {
      final transport = FakeTransport([
        HankoApiResponse(
          statusCode: 200,
          body: jsonEncode({
            'order_id': 'ord/needs encoding',
            'order_no': 'HF-FAILED-0001',
            'status': 'payment_failed',
            'payment': {'status': 'failed'},
            'fulfillment': {'status': 'canceled'},
            'pricing': {'total': 18600, 'currency': 'JPY'},
            'production_status': 'not_started',
            'shipping_status': 'not_shipped',
          }),
        ),
      ]);
      final repo = OrderLookupRepository(_client(transport));

      final status = await repo.fetchOrderStatus('ord/needs encoding');

      expect(
        transport.singleRequest.uri.toString(),
        'https://api.example.test/v1/orders/ord%2Fneeds%20encoding/status',
      );
      expect(status.orderStatus, 'payment_failed');
      expect(status.paymentStatus, 'failed');
      expect(status.fulfillmentStatus, 'canceled');
      expect(status.productionStatus, 'not_started');
      expect(status.shippingStatus, 'not_shipped');
      expect(status.pricing.amount, 18600);
    },
  );

  test('OrderLookupRepository posts lookup request body', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 200,
        body: jsonEncode({
          'order_id': 'ord_lookup_001',
          'order_no': 'HF-20260521-0001',
          'status': 'paid',
          'payment_status': 'paid',
          'fulfillment_status': 'pending',
          'pricing': {'total': 18600, 'currency': 'JPY'},
        }),
      ),
    ]);
    final repo = OrderLookupRepository(_client(transport));

    final status = await repo.lookupOrder(
      const OrderLookupRequest(
        orderNo: 'HF-20260521-0001',
        email: 'customer@example.test',
      ),
    );

    expect(transport.singleRequest.method, 'POST');
    expect(transport.singleRequest.uri.path, '/v1/orders/lookup');
    expect(transport.singleRequest.body?['order_no'], 'HF-20260521-0001');
    expect(transport.singleRequest.body?['email'], 'customer@example.test');
    expect(status.orderId, 'ord_lookup_001');
    expect(status.paymentStatus, 'paid');
  });

  test('PublicConfigDto maps public config response', () {
    final config = PublicConfigDto.fromJson({
      'supported_locales': ['en', 'ja'],
      'default_locale': 'en',
      'default_currency': 'USD',
      'currency_by_locale': {'ja': 'JPY'},
    }).toDomain();

    expect(config.defaultLocale, 'en');
    expect(config.currencyForLocale('ja'), 'JPY');
    expect(config.currencyForLocale('en'), 'USD');
  });

  test('HankoApiClient maps API error envelope', () async {
    final transport = FakeTransport([
      HankoApiResponse(
        statusCode: 400,
        body: jsonEncode({
          'error': {
            'code': 'validation_error',
            'message': 'real_name is required',
          },
        }),
      ),
    ]);
    final client = _client(transport);

    await expectLater(
      client.postJson('/v1/kanji-candidates', const {}),
      throwsA(
        isA<HankoApiException>()
            .having((error) => error.statusCode, 'statusCode', 400)
            .having((error) => error.code, 'code', 'validation_error'),
      ),
    );
  });

  test('HankoApiClient maps non-JSON HTTP errors', () async {
    final transport = FakeTransport([
      const HankoApiResponse(statusCode: 404, body: '<html>Not Found</html>'),
    ]);
    final client = _client(transport);

    await expectLater(
      client.postJson('/v1/seal-designs/generate', const {}),
      throwsA(
        isA<HankoApiException>()
            .having((error) => error.statusCode, 'statusCode', 404)
            .having((error) => error.code, 'code', 'http_404'),
      ),
    );
  });

  test('default API base URL targets production in release builds', () {
    expect(
      resolveDefaultHankoApiBaseUrl(isReleaseMode: true),
      productionHankoApiBaseUrl,
    );
  });

  test(
    'default API base URL targets the host from mobile simulators in debug',
    () {
      expect(
        resolveDefaultHankoApiBaseUrl(isAndroid: false, isReleaseMode: false),
        'http://127.0.0.1:3050',
      );
      expect(
        resolveDefaultHankoApiBaseUrl(isAndroid: true, isReleaseMode: false),
        'http://10.0.2.2:3050',
      );
    },
  );

  test('HankoAppError classifies network server and generic errors', () {
    expect(
      HankoAppError.fromObject(const SocketException('offline')).kind,
      HankoAppErrorKind.network,
    );
    expect(
      HankoAppError.fromObject(
        const FileSystemException('permission denied'),
      ).kind,
      HankoAppErrorKind.storage,
    );
    expect(
      HankoAppError.fromObject(
        const HankoDeepLinkException('hankofield://checkout/success'),
      ).kind,
      HankoAppErrorKind.deepLink,
    );
    expect(
      HankoAppError.fromObject(
        const HankoApiException(
          statusCode: 503,
          code: 'internal',
          message: 'temporary failure',
          payload: {},
        ),
      ).kind,
      HankoAppErrorKind.server,
    );
    expect(
      HankoAppError.fromObject(
        const HankoApiException(
          statusCode: 409,
          code: 'idempotency_conflict',
          message: 'conflict',
          payload: {},
        ),
      ).kind,
      HankoAppErrorKind.generic,
    );
  });
}

HankoApiClient _client(FakeTransport transport) {
  return HankoApiClient(
    baseUri: Uri.parse('https://api.example.test'),
    transport: transport,
  );
}

SealOrderDraft _draft() {
  return SealOrderDraft(
    channel: 'app',
    locale: 'en',
    idempotencyKey: 'idem_001',
    termsAgreed: true,
    listingId: 'stone_listing_001',
    seal: SealOrderSeal(
      line1: '美空',
      line2: '',
      shape: 'square',
      fontKey: 'ai_generated_seal',
      aiGenerationId: 'seal_request_001',
      aiVariantId: 'seal_variant_001',
      previewImage: SealOrderPreviewImage(
        storagePath: 'seal_designs/seal_request_001/seal_variant_001.png',
        downloadUrl: 'https://storage.example.test/seal_variant_001.png',
      ),
      style: SealOrderStyle(
        name: 'elegant',
        strokeWeight: 'standard',
        balance: 'balanced',
      ),
    ),
    shipping: SealOrderShipping(
      countryCode: 'JP',
      recipientName: 'Michael Smith',
      phone: '+81-90-0000-0000',
      postalCode: '100-0001',
      state: 'Tokyo',
      city: 'Chiyoda',
      addressLine1: '1-1',
      addressLine2: '',
    ),
    contact: SealOrderContact(
      email: 'michael@example.test',
      preferredLocale: 'en',
    ),
    customerConfirmation: SealOrderCustomerConfirmation(
      kanjiAndDesign: true,
      customMadePolicy: true,
      confirmedAt: DateTime.utc(2026, 5, 21, 2),
      confirmedSealText: '美空',
    ),
    orderNote: 'Please pack safely.',
  );
}

class FakeTransport implements HankoApiTransport {
  FakeTransport(this._responses);

  final List<HankoApiResponse> _responses;
  final List<HankoApiRequest> requests = [];

  HankoApiRequest get singleRequest {
    expect(requests, hasLength(1));
    return requests.single;
  }

  @override
  Future<HankoApiResponse> send(HankoApiRequest request) async {
    requests.add(request);
    if (_responses.isEmpty) {
      fail('unexpected API request to ${request.uri}');
    }
    return _responses.removeAt(0);
  }
}
