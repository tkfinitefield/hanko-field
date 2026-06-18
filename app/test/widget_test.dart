import 'dart:async';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:miniriverpod/miniriverpod.dart';

import 'package:hankofield/app/app.dart';
import 'package:hankofield/app/localization/app_localization.dart';
import 'package:hankofield/app/theme/app_theme.dart';
import 'package:hankofield/core/api/core_api.dart';
import 'package:hankofield/core/widgets/core_widgets.dart';
import 'package:hankofield/features/common/common.dart';
import 'package:hankofield/features/design/design.dart';
import 'package:hankofield/features/my_seals/my_seals.dart';
import 'package:hankofield/features/order/order.dart';
import 'package:hankofield/features/order_lookup/order_lookup.dart';
import 'package:hankofield/features/settings/settings.dart';
import 'package:hankofield/features/stones/stones.dart';

const _sealStyleAdjustmentControlKeys = <Key>[
  Key('DES-006-seal-shape-options'),
  Key('DES-006-seal-style-options'),
  Key('DES-006-seal-stroke-options'),
  Key('DES-006-seal-balance-options'),
];

void _expectSealStyleAdjustmentControlsPresent() {
  for (final key in _sealStyleAdjustmentControlKeys) {
    expect(find.byKey(key), findsOneWidget);
  }
}

void _expectSealStyleAdjustmentControlsAbsent() {
  for (final key in _sealStyleAdjustmentControlKeys) {
    expect(find.byKey(key), findsNothing);
  }
  expect(find.byType(ChoiceChip), findsNothing);
}

void _expectNetworkImageUrl(WidgetTester tester, String url) {
  expect(
    tester.widgetList<Image>(find.byType(Image)).any((image) {
      final provider = image.image;
      return provider is NetworkImage && provider.url == url;
    }),
    isTrue,
    reason: 'Expected a NetworkImage backed by $url.',
  );
}

void main() {
  Future<void> pumpLaunchedApp(
    WidgetTester tester, {
    Locale? locale,
    bool hasSeenOnboarding = true,
    KanjiCandidatesGenerator? generateKanjiCandidates,
    SealDesignsGenerator? generateSealDesigns,
    StoneListingsLoader? listStoneListings,
    StoneListingDetailLoader? getStoneListingDetail,
    OrderCreator? createOrder,
    CheckoutSessionCreator? createCheckoutSession,
    CheckoutUrlLauncher? openCheckoutUrl,
    OrderStatusFetcher? fetchOrderStatus,
    OrderLookupFetcher? lookupOrder,
    PreferredLocaleLoader? loadPreferredLocale,
    PreferredLocaleWriter? savePreferredLocale,
    Duration paymentStatusRetryDelay = Duration.zero,
    String? initialCheckoutRoute,
    LocalSealDesignRepository? localSealDesignRepository,
    LocalOrderDraftRepository? localOrderDraftRepository,
  }) async {
    await tester.pumpWidget(
      ProviderScope(
        child: HankoApp(
          locale: locale,
          loadPreferredLocale: loadPreferredLocale ?? () async => null,
          savePreferredLocale: savePreferredLocale ?? (_) async {},
          hasSeenOnboardingResolver: () async => hasSeenOnboarding,
          markOnboardingSeen: () async {},
          splashMinimumDuration: Duration.zero,
          generateKanjiCandidates:
              generateKanjiCandidates ?? _successfulKanjiGenerator,
          generateSealDesigns:
              generateSealDesigns ?? generateSealDesignsWithDefaultApi,
          listStoneListings: listStoneListings ?? _emptyStoneListingsLoader,
          getStoneListingDetail:
              getStoneListingDetail ?? _successfulStoneDetailLoader,
          createOrder: createOrder ?? _successfulCreateOrder,
          createCheckoutSession:
              createCheckoutSession ?? _successfulCreateCheckoutSession,
          openCheckoutUrl: openCheckoutUrl ?? _successfulOpenCheckoutUrl,
          fetchOrderStatus: fetchOrderStatus ?? _successfulFetchOrderStatus,
          lookupOrder: lookupOrder ?? _successfulLookupOrder,
          paymentStatusRetryDelay: paymentStatusRetryDelay,
          initialCheckoutRoute: initialCheckoutRoute,
          localSealDesignRepository: localSealDesignRepository,
          localOrderDraftRepository: localOrderDraftRepository,
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();
  }

  testWidgets('COM-001 routes returning users to the shell', (tester) async {
    final launchCheck = Completer<bool>();

    await tester.pumpWidget(
      ProviderScope(
        child: HankoApp(
          hasSeenOnboardingResolver: () => launchCheck.future,
          markOnboardingSeen: () async {},
          splashMinimumDuration: Duration.zero,
          listStoneListings: _emptyStoneListingsLoader,
        ),
      ),
    );

    expect(find.byType(SplashScreen), findsOneWidget);
    expect(find.text('Preparing your design experience.'), findsOneWidget);

    launchCheck.complete(true);
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();

    expect(find.byType(SplashScreen), findsNothing);
    expect(find.byType(BottomNavigationShell), findsOneWidget);
    expect(find.byType(DesignHomeScreen, skipOffstage: false), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-001 routes first-time users to onboarding', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var savedOnboardingState = false;
    final saveCompleter = Completer<void>();

    await tester.pumpWidget(
      ProviderScope(
        child: HankoApp(
          hasSeenOnboardingResolver: () async => false,
          markOnboardingSeen: () {
            savedOnboardingState = true;
            return saveCompleter.future;
          },
          splashMinimumDuration: Duration.zero,
          listStoneListings: _emptyStoneListingsLoader,
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();

    expect(find.byType(SplashScreen), findsNothing);
    expect(find.byType(OnboardingScreen), findsOneWidget);
    expect(find.text('Create your\nseal in minutes'), findsOneWidget);
    expect(find.text('Choose kanji from your name'), findsOneWidget);
    expect(find.text('Generate a seal design with AI'), findsOneWidget);

    await tester.ensureVisible(find.text('Get Started'));
    await tester.pump();
    await tester.tap(find.text('Get Started'));
    await tester.pump();

    expect(savedOnboardingState, isTrue);
    expect(find.byType(BottomNavigationShell), findsNothing);

    saveCompleter.complete();
    await tester.pump();

    expect(find.byType(BottomNavigationShell), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-001 treats launch read failures as first run', (
    tester,
  ) async {
    await tester.pumpWidget(
      ProviderScope(
        child: HankoApp(
          hasSeenOnboardingResolver: () async => throw StateError('no storage'),
          markOnboardingSeen: () async {},
          splashMinimumDuration: Duration.zero,
          listStoneListings: _emptyStoneListingsLoader,
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();

    expect(find.byType(SplashScreen), findsNothing);
    expect(find.byType(OnboardingScreen), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('boots the COM-003 bottom navigation shell', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(tester);

    expect(find.byType(MaterialApp), findsOneWidget);
    expect(find.byType(DesignHomeScreen, skipOffstage: false), findsOneWidget);
    expect(find.byType(MySealsHomeScreen, skipOffstage: false), findsOneWidget);
    expect(find.byType(StonesHomeScreen, skipOffstage: false), findsOneWidget);
    expect(find.byType(HankoSurfaceCard, skipOffstage: false), findsWidgets);
    expect(find.byType(HankoPrimaryButton, skipOffstage: false), findsWidgets);
    expect(find.byType(HankoStateView, skipOffstage: false), findsWidgets);
    expect(find.text('Design'), findsNWidgets(2));
    expect(find.text('Create your\ncustom seal'), findsOneWidget);
    expect(find.text('Start Designing'), findsOneWidget);
    expect(find.text('Saved Seals'), findsOneWidget);
    expect(find.text('Browse Stones'), findsOneWidget);
    expect(find.text('My Seals'), findsOneWidget);
    expect(find.text('Stones'), findsOneWidget);
    expect(find.byType(Navigator, skipOffstage: false), findsNWidgets(5));

    await tester.tap(find.text('Saved Seals'));
    await tester.pumpAndSettle();

    expect(find.text('My Seals'), findsNWidgets(2));

    await tester.tap(find.text('Design').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('Browse Stones'));
    await tester.pumpAndSettle();

    expect(find.text('Stones'), findsNWidgets(2));

    await tester.tap(find.text('Design').last);
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();

    expect(find.text('Stones'), findsNWidgets(2));

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();

    expect(find.text('My Seals'), findsNWidgets(2));
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-004 calls kanji API and displays candidates', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final apiCall = Completer<KanjiCandidatesResult>();
    final sealCall = Completer<SealGenerationResult>();
    KanjiCandidatesRequest? capturedRequest;
    SealGenerationRequest? capturedSealRequest;

    await pumpLaunchedApp(
      tester,
      generateKanjiCandidates: (request) {
        capturedRequest = request;
        return apiCall.future;
      },
      generateSealDesigns: (request) {
        capturedSealRequest = request;
        return sealCall.future;
      },
    );

    await tester.tap(find.text('Start Designing'));
    await tester.pumpAndSettle();

    expect(find.byType(NameInputScreen), findsOneWidget);
    expect(find.text('Enter Your Name'), findsOneWidget);
    expect(
      find.text("We'll suggest kanji based on your preferences."),
      findsOneWidget,
    );
    expect(find.text('Your name'), findsOneWidget);
    expect(find.text('Gender preference'), findsOneWidget);
    expect(find.text('Kanji style'), findsOneWidget);

    final submitButton = find.widgetWithText(TextButton, 'Suggest Kanji');
    expect(submitButton, findsOneWidget);

    await tester.ensureVisible(find.text('Suggest Kanji'));
    await tester.pump();
    await tester.tap(find.text('Suggest Kanji'));
    await tester.pump();

    expect(find.text('Enter your name to continue.'), findsOneWidget);
    expect(
      find.text('Please enter a valid first name or short name.'),
      findsOneWidget,
    );

    await tester.enterText(find.byType(TextFormField).first, 'Michael Smith');
    await tester.pump();

    expect(tester.widget<TextButton>(submitButton).onPressed, isNotNull);

    await tester.ensureVisible(find.text('Suggest Kanji'));
    await tester.pump();
    await tester.tap(find.text('Suggest Kanji'));
    await tester.pump();

    expect(find.byType(KanjiSuggestionLoadingScreen), findsOneWidget);
    expect(find.text('Finding Kanji'), findsOneWidget);
    expect(find.text('Creating kanji suggestions...'), findsOneWidget);
    expect(find.textContaining('sound, meaning'), findsNothing);
    expect(find.text('Michael Smith'), findsWidgets);
    expect(find.text('Japanese style'), findsWidgets);
    expect(capturedRequest?.realName, 'Michael Smith');
    expect(capturedRequest?.reasonLanguage, 'en');
    expect(capturedRequest?.kanjiStyle, KanjiNameStyle.japanese);

    apiCall.complete(_kanjiResult(capturedRequest!));
    await tester.pump();
    await tester.pumpAndSettle();

    expect(find.byType(KanjiSuggestionsScreen), findsOneWidget);
    expect(find.text('Kanji Suggestions'), findsOneWidget);
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Misora'), findsOneWidget);
    expect(
      find.text('Meaning: Beautiful sky', findRichText: true),
      findsOneWidget,
    );
    expect(find.text('Elegant'), findsOneWidget);
    expect(find.text('Gentle'), findsOneWidget);
    expect(find.text('A graceful two-character option.'), findsOneWidget);
    expect(find.text('Characters'), findsNothing);
    expect(find.text('Stroke complexity'), findsNothing);
    expect(find.text('Engraving suitability'), findsNothing);

    await tester.tap(find.text('美空'));
    await tester.pumpAndSettle();

    expect(find.byType(KanjiCandidateDetailScreen), findsOneWidget);
    expect(find.text('Kanji Detail'), findsOneWidget);
    expect(
      find.text(
        'Review the meaning and engraving fit before choosing this kanji.',
      ),
      findsNothing,
    );
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Reading: Misora', findRichText: true), findsOneWidget);
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('Characters'), findsNothing);
    expect(find.text('Stroke complexity'), findsNothing);
    expect(find.text('Engraving suitability'), findsNothing);

    await tester.ensureVisible(find.text('Select Kanji'));
    await tester.pump();
    await tester.tap(find.text('Select Kanji'));
    await tester.pumpAndSettle();

    expect(find.byType(SealStyleSelectionScreen), findsOneWidget);
    expect(find.text('Seal Style'), findsOneWidget);
    expect(find.text('Customize your seal style.'), findsOneWidget);
    expect(find.text('Selected kanji'), findsOneWidget);
    _expectSealStyleAdjustmentControlsPresent();
    expect(find.text('Shape'), findsWidgets);
    expect(find.text('Square'), findsWidgets);
    expect(find.text('Round'), findsOneWidget);
    expect(find.text('Style'), findsWidgets);
    expect(find.text('Traditional'), findsOneWidget);
    expect(find.text('Elegant'), findsWidgets);
    expect(find.text('Soft'), findsOneWidget);
    expect(find.text('Stroke Weight'), findsWidgets);
    expect(find.text('Standard'), findsWidgets);
    expect(find.text('Balance'), findsWidgets);
    expect(find.text('Balanced'), findsWidgets);

    await tester.tap(find.text('Round'));
    await tester.pump();
    await tester.tap(find.text('Traditional'));
    await tester.pump();
    await tester.ensureVisible(find.text('Airy'));
    await tester.pump();
    await tester.tap(find.text('Airy'));
    await tester.pump();
    await tester.ensureVisible(find.text('Confirm Style'));
    await tester.pump();
    await tester.tap(find.text('Confirm Style'));
    await tester.pumpAndSettle();

    expect(find.text('Style selected'), findsOneWidget);
    expect(
      find.text('These style choices are ready for AI seal generation.'),
      findsOneWidget,
    );
    expect(find.text('Generate Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Generate Seal'));
    await tester.pump();
    await tester.tap(find.text('Generate Seal'));
    await tester.pump();

    expect(find.byType(SealGenerationLoadingScreen), findsOneWidget);
    expect(capturedSealRequest?.inputName, 'Michael Smith');
    expect(capturedSealRequest?.candidate.kanji, '美空');
    expect(capturedSealRequest?.style.shape, SealShape.round);
    expect(capturedSealRequest?.style.style, SealStyleName.traditional);
    expect(capturedSealRequest?.style.balance, SealBalance.airy);

    sealCall.complete(_sealGenerationResult(request: capturedSealRequest!));
    await tester.pump();
    await tester.pumpAndSettle();

    expect(find.byType(SealVariantSelectionScreen), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('Seal Options'), findsOneWidget);
    expect(find.text('Elegant and balanced'), findsOneWidget);
    expect(find.text('Soft spacing'), findsOneWidget);
    expect(find.text('Bold readable seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Soft spacing'));
    await tester.pump();
    await tester.tap(find.text('Soft spacing'));
    await tester.pumpAndSettle();

    expect(find.byType(SealPreviewDetailScreen), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('Seal Preview'), findsOneWidget);
    expect(
      find.text('Review your selected seal design before saving.'),
      findsOneWidget,
    );
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('AI Variant'), findsOneWidget);
    expect(find.text('Soft spacing'), findsOneWidget);
    expect(find.text('Save Seal'), findsOneWidget);
    expect(find.text('Choose a Stone'), findsOneWidget);

    await tester.ensureVisible(find.text('Save Seal'));
    await tester.pump();
    await tester.tap(find.text('Save Seal'));
    await tester.pumpAndSettle();

    expect(find.byType(SealSaveConfirmationScreen), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('Seal Saved'), findsOneWidget);
    expect(find.text('Seal saved to My Seals'), findsOneWidget);
    expect(find.text('Go to My Seals'), findsOneWidget);
    expect(find.text('Create Another Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Choose a Stone'));
    await tester.pump();
    await tester.tap(find.text('Choose a Stone'));
    await tester.pumpAndSettle();

    expect(find.text('No stones loaded'), findsOneWidget);

    await tester.tap(find.text('Design').last);
    await tester.pumpAndSettle();

    expect(find.byType(SealSaveConfirmationScreen), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();

    await tester.ensureVisible(find.text('Go to My Seals'));
    await tester.pump();
    await tester.tap(find.text('Go to My Seals'));
    await tester.pumpAndSettle();

    expect(find.text('Saved on this device'), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('美空'), findsWidgets);
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('View Details'), findsOneWidget);

    await tester.tap(find.text('Design').last);
    await tester.pumpAndSettle();

    expect(find.byType(SealSaveConfirmationScreen), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();

    await tester.ensureVisible(find.text('Create Another Seal'));
    await tester.pump();
    await tester.tap(find.text('Create Another Seal'));
    await tester.pumpAndSettle();

    expect(find.byType(DesignHomeScreen), findsOneWidget);
    expect(find.text('Start Designing'), findsOneWidget);

    expect(tester.takeException(), isNull);
  });

  testWidgets('M12-T05 shows storage save errors in the design flow', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      generateSealDesigns: (request) async =>
          _sealGenerationResult(request: request),
      localSealDesignRepository: _FailingSaveLocalSealDesignRepository(),
    );
    await tester.pumpAndSettle();

    await _openGeneratedSealPreview(tester);
    expect(find.byType(SealPreviewDetailScreen), findsOneWidget);

    await tester.ensureVisible(find.text('Save Seal'));
    await tester.pump();
    await tester.tap(find.text('Save Seal'));
    await tester.pumpAndSettle();

    expect(find.byType(SealSaveErrorScreen), findsOneWidget);
    expect(find.text("Couldn't Save Seal"), findsOneWidget);
    expect(
      find.text(
        "The seal image couldn't be saved on this device. Check storage permissions and available space, then try again.",
      ),
      findsOneWidget,
    );
    expect(find.text('Try Again'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-007 shows seal generation progress details', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final generation = Completer<SealGenerationResult>();
    var started = false;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealGenerationLoadingScreen(
          request: _sealGenerationRequest(),
          generateSealDesigns: (request) {
            started = true;
            expect(request.attemptNumber, 1);
            return generation.future;
          },
          onGenerated: (_) {},
          onError: (_) {},
          onBack: () {},
        ),
      ),
    );
    await tester.pump();

    expect(started, isTrue);
    expect(find.text('Generating Seal'), findsOneWidget);
    expect(
      find.text('Creating three AI seal design directions...'),
      findsOneWidget,
    );
    expect(
      find.text('We are checking the kanji and style before saving previews.'),
      findsOneWidget,
    );
    expect(find.textContaining('engraving safety'), findsNothing);
    expect(find.text('Generation details'), findsOneWidget);
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Attempts'), findsOneWidget);
    expect(find.text('1/3'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsOneWidget);

    generation.complete(_sealGenerationResult());
    await tester.pump();
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-008 lets the user select one generated seal variant', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    SealDesignVariant? selected;
    var regenerateCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealVariantSelectionScreen(
          result: _sealGenerationResult(),
          onSelected: (variant) => selected = variant,
          onBack: () {},
          onRegenerate: () => regenerateCount += 1,
        ),
      ),
    );

    expect(find.text('Seal Options'), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('Choose one AI seal design.'), findsOneWidget);
    expect(find.text('Elegant and balanced'), findsOneWidget);
    expect(find.text('Soft spacing'), findsOneWidget);
    expect(find.text('Bold readable seal'), findsOneWidget);
    expect(find.text('Selected'), findsNothing);
    expect(find.text('Regenerate Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Regenerate Seal'));
    await tester.pump();
    await tester.tap(find.text('Regenerate Seal'));
    await tester.pumpAndSettle();

    expect(regenerateCount, 1);

    await tester.ensureVisible(find.text('Soft spacing'));
    await tester.pump();
    await tester.tap(find.text('Soft spacing'));
    await tester.pumpAndSettle();

    expect(selected?.id, 'seal_variant_002');
    expect(find.text('Selected'), findsOneWidget);
    expect(find.text('Seal design selected'), findsOneWidget);
    expect(
      find.text('This AI seal design is ready for preview and saving.'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-009 previews selected seal and exposes next actions', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var saveCount = 0;
    var chooseStoneCount = 0;
    final result = _sealGenerationResult();
    final variant = result.variants[1];

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealPreviewDetailScreen(
          result: result,
          variant: variant,
          onSave: () => saveCount += 1,
          onChooseStone: () => chooseStoneCount += 1,
          onBack: () {},
        ),
      ),
    );

    expect(find.text('Seal Preview'), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(
      find.text('Review your selected seal design before saving.'),
      findsOneWidget,
    );
    expect(
      find.text('Created within engraving-friendly design rules.'),
      findsNothing,
    );
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('AI Variant'), findsOneWidget);
    expect(find.text('Soft spacing'), findsOneWidget);
    expect(
      find.text('seal_designs/seal_request_001/seal_variant_002.png'),
      findsNothing,
    );

    await tester.ensureVisible(find.text('Save Seal'));
    await tester.pump();
    await tester.tap(find.text('Save Seal'));
    await tester.pump();
    await tester.ensureVisible(find.text('Choose a Stone'));
    await tester.pump();
    await tester.tap(find.text('Choose a Stone'));
    await tester.pump();

    expect(saveCount, 1);
    expect(chooseStoneCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-009 hides internal seal preview metadata in Japanese', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final result = _sealGenerationResult();
    final variant = result.variants.first;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('ja'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealPreviewDetailScreen(
          result: result,
          variant: variant,
          onSave: () {},
          onChooseStone: () {},
          onBack: () {},
        ),
      ),
    );

    expect(find.text('印影プレビュー'), findsOneWidget);
    expect(find.textContaining('Storageパス'), findsNothing);
    expect(find.textContaining('seal_designs/'), findsNothing);
    expect(find.textContaining('彫刻しやすいデザインルール'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-010 lets the user choose the next saved seal action', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var openMySealsCount = 0;
    var chooseStoneCount = 0;
    var createAnotherCount = 0;
    var backCount = 0;
    final result = _sealGenerationResult();

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealSaveConfirmationScreen(
          result: result,
          variant: result.variants[1],
          onOpenMySeals: () => openMySealsCount += 1,
          onChooseStone: () => chooseStoneCount += 1,
          onCreateAnother: () => createAnotherCount += 1,
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text('Seal Saved'), findsOneWidget);
    _expectSealStyleAdjustmentControlsAbsent();
    expect(find.text('Seal saved to My Seals'), findsOneWidget);
    expect(
      find.text(
        'Your custom seal design is ready for comparison and ordering.',
      ),
      findsOneWidget,
    );
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Soft spacing'), findsOneWidget);
    expect(find.text('Choose a Stone'), findsOneWidget);
    expect(find.text('Go to My Seals'), findsOneWidget);
    expect(find.text('Create Another Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Choose a Stone'));
    await tester.pump();
    await tester.tap(find.text('Choose a Stone'));
    await tester.pump();
    await tester.ensureVisible(find.text('Go to My Seals'));
    await tester.pump();
    await tester.tap(find.text('Go to My Seals'));
    await tester.pump();
    await tester.ensureVisible(find.text('Create Another Seal'));
    await tester.pump();
    await tester.tap(find.text('Create Another Seal'));
    await tester.pump();
    await tester.tap(find.byTooltip('Back'));
    await tester.pump();

    expect(chooseStoneCount, 1);
    expect(openMySealsCount, 1);
    expect(createAnotherCount, 1);
    expect(backCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'M15-T03 keeps generated storage image through save and checkout draft',
    (tester) async {
      tester.view.physicalSize = const Size(432, 912);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      const selectedVariantId = 'seal_variant_002';
      final selectedStoragePath = _sealVariantStoragePath(selectedVariantId);
      final selectedDownloadUrl = _sealVariantDownloadUrl(selectedVariantId);
      final sealRepository = InMemoryLocalSealDesignRepository();
      final draftRepository = InMemoryLocalOrderDraftRepository();
      SealOrderDraft? submittedOrderDraft;

      await pumpLaunchedApp(
        tester,
        generateSealDesigns: (request) async =>
            _sealGenerationResult(request: request, includeDownloadUrls: true),
        listStoneListings: (query) async => _stoneListingsResult(),
        createOrder: (draft) async {
          submittedOrderDraft = draft;
          return _successfulCreateOrder(draft);
        },
        localSealDesignRepository: sealRepository,
        localOrderDraftRepository: draftRepository,
      );
      await tester.pumpAndSettle();

      await _openGeneratedSealVariantSelection(tester);

      expect(find.byType(SealVariantSelectionScreen), findsOneWidget);
      expect(find.textContaining('Storage path'), findsNothing);
      for (final variantId in const [
        'seal_variant_001',
        'seal_variant_002',
        'seal_variant_003',
      ]) {
        _expectNetworkImageUrl(tester, _sealVariantDownloadUrl(variantId));
      }

      await tester.ensureVisible(find.text('Soft spacing'));
      await tester.pump();
      await tester.tap(find.text('Soft spacing'));
      await tester.pumpAndSettle();

      expect(find.byType(SealPreviewDetailScreen), findsOneWidget);
      expect(find.text('Storage path'), findsNothing);
      expect(find.text(selectedStoragePath), findsNothing);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      await tester.ensureVisible(find.text('Save Seal'));
      await tester.pump();
      await tester.tap(find.text('Save Seal'));
      await tester.pumpAndSettle();

      expect(find.byType(SealSaveConfirmationScreen), findsOneWidget);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      final savedDesigns = await sealRepository.listLocalSealDesigns();
      expect(savedDesigns, hasLength(1));
      expect(savedDesigns.single.previewImageStoragePath, selectedStoragePath);
      expect(savedDesigns.single.previewImageDownloadUrl, selectedDownloadUrl);

      final draftAfterSave = await draftRepository.loadOrderDraft();
      expect(draftAfterSave.sealSelection?.aiVariantId, selectedVariantId);
      expect(
        draftAfterSave.sealSelection?.previewImageStoragePath,
        selectedStoragePath,
      );
      expect(
        draftAfterSave.sealSelection?.previewImageDownloadUrl,
        selectedDownloadUrl,
      );

      await tester.ensureVisible(find.text('Choose a Stone'));
      await tester.pump();
      await tester.tap(find.text('Choose a Stone'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Select Stone'));
      await tester.pump();
      await tester.tap(find.text('Select Stone'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('stone-selection-confirm')));
      await tester.pumpAndSettle();

      expect(find.byType(OrderCombinationReviewScreen), findsOneWidget);
      expect(find.text('Stone missing'), findsNothing);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      await tester.ensureVisible(find.text('Continue to Shipping'));
      await tester.pump();
      await tester.tap(find.text('Continue to Shipping'));
      await tester.pumpAndSettle();

      Future<void> enterCheckoutField(String key, String text) async {
        final field = find.byKey(Key(key));
        await tester.ensureVisible(field);
        await tester.pump();
        await tester.enterText(
          find.descendant(of: field, matching: find.byType(EditableText)),
          text,
        );
        await tester.pump();
      }

      await enterCheckoutField('checkout-email-field', 'customer@example.test');
      await enterCheckoutField('checkout-full-name-field', 'Michael Smith');
      await enterCheckoutField('checkout-phone-field', '+1 555 0100');
      await enterCheckoutField('checkout-postal-code-field', '10001');
      await enterCheckoutField(
        'checkout-address-line1-field',
        '123 Example Street',
      );
      await enterCheckoutField('checkout-city-field', 'New York');
      await enterCheckoutField('checkout-state-field', 'NY');

      await tester.ensureVisible(find.text('Save Checkout Information'));
      await tester.pump();
      await tester.tap(find.text('Save Checkout Information'));
      await tester.pumpAndSettle();

      expect(find.byType(OrderConfirmationScreen), findsOneWidget);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      await tester.ensureVisible(
        find.byKey(const Key('order-confirm-kanji-design-checkbox')),
      );
      await tester.pump();
      await tester.tap(
        find.byKey(const Key('order-confirm-kanji-design-checkbox')),
      );
      await tester.pumpAndSettle();
      await tester.ensureVisible(
        find.byKey(const Key('order-confirm-custom-made-checkbox')),
      );
      await tester.pump();
      await tester.tap(
        find.byKey(const Key('order-confirm-custom-made-checkbox')),
      );
      await tester.pumpAndSettle();

      await tester.ensureVisible(find.text('Proceed to Secure Payment'));
      await tester.pump();
      await tester.tap(find.text('Proceed to Secure Payment'));
      await tester.pumpAndSettle();

      expect(find.text('Complete payment in Stripe Checkout'), findsOneWidget);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);
      expect(
        submittedOrderDraft?.seal.previewImage?.storagePath,
        selectedStoragePath,
      );
      expect(
        submittedOrderDraft?.seal.previewImage?.downloadUrl,
        selectedDownloadUrl,
      );
      expect(submittedOrderDraft?.seal.aiVariantId, selectedVariantId);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets(
    'DES-009 uses current unsaved generated seal when choosing a stone',
    (tester) async {
      tester.view.physicalSize = const Size(432, 912);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      const selectedVariantId = 'seal_variant_002';
      final selectedStoragePath = _sealVariantStoragePath(selectedVariantId);
      final selectedDownloadUrl = _sealVariantDownloadUrl(selectedVariantId);
      const pastSealSelection = OrderDraftSealSelection(
        localSealDesignId: 'local_past_seal',
        selectedKanji: '過去',
        reading: 'Kako',
        shape: 'square',
        style: 'traditional',
        strokeWeight: 'standard',
        balance: 'balanced',
        aiGenerationId: 'old_seal_request',
        aiVariantId: 'old_seal_variant',
        previewImageStoragePath: 'seal_designs/old_request/old_variant.png',
        previewImageDownloadUrl: 'https://storage.example.test/old.png',
        localImagePath: '',
      );
      final sealRepository = InMemoryLocalSealDesignRepository();
      final draftRepository = InMemoryLocalOrderDraftRepository(
        OrderDraft.empty().withSealSelection(pastSealSelection),
      );

      await pumpLaunchedApp(
        tester,
        generateSealDesigns: (request) async =>
            _sealGenerationResult(request: request, includeDownloadUrls: true),
        listStoneListings: (query) async => _stoneListingsResult(),
        localSealDesignRepository: sealRepository,
        localOrderDraftRepository: draftRepository,
      );
      await tester.pumpAndSettle();

      await _openGeneratedSealPreview(tester);
      expect(find.byType(SealPreviewDetailScreen), findsOneWidget);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      await tester.ensureVisible(find.text('Choose a Stone'));
      await tester.pump();
      await tester.tap(find.text('Choose a Stone'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Select Stone'));
      await tester.pump();
      await tester.tap(find.text('Select Stone'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('stone-selection-confirm')));
      await tester.pumpAndSettle();

      expect(find.byType(OrderCombinationReviewScreen), findsOneWidget);
      expect(find.text('過去'), findsNothing);
      _expectNetworkImageUrl(tester, selectedDownloadUrl);

      final savedDraft = await draftRepository.loadOrderDraft();
      expect(savedDraft.sealSelection?.selectedKanji, '美空');
      expect(savedDraft.sealSelection?.aiGenerationId, 'seal_request_001');
      expect(savedDraft.sealSelection?.aiVariantId, selectedVariantId);
      expect(
        savedDraft.sealSelection?.previewImageStoragePath,
        selectedStoragePath,
      );
      expect(
        savedDraft.sealSelection?.previewImageDownloadUrl,
        selectedDownloadUrl,
      );
      expect(savedDraft.stoneSelection?.listingId, 'stone_listing_001');
      expect(await sealRepository.listLocalSealDesigns(), isEmpty);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('MYS-001 displays saved seal cards', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    LocalSealDesign? opened;
    LocalSealDesign? favoriteToggled;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: MySealsHomeScreen(
          designs: [
            _localSealDesign(),
            _localSealDesign(
              id: 'local_seal_002',
              selectedKanji: '永愛',
              meaning: 'Eternal love',
              style: 'soft',
              isFavorite: true,
            ),
          ],
          onChooseSeal: (design) => opened = design,
          onToggleFavorite: (design) => favoriteToggled = design,
        ),
      ),
    );

    expect(find.text('My Seals'), findsOneWidget);
    expect(find.text('Saved on this device'), findsOneWidget);
    expect(find.text('美空'), findsWidgets);
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('永愛'), findsWidgets);
    expect(find.text('Eternal love'), findsOneWidget);
    expect(find.text('Elegant'), findsOneWidget);
    expect(find.text('Soft'), findsOneWidget);
    expect(find.text('Standard'), findsWidgets);
    expect(find.text('Balanced'), findsWidgets);
    expect(find.text('Compare Seals'), findsOneWidget);
    expect(
      tester
          .getTopLeft(
            find.byKey(
              const ValueKey('MYS-001-saved-seal-card-local_seal_002'),
            ),
          )
          .dy,
      lessThan(
        tester
            .getTopLeft(
              find.byKey(
                const ValueKey('MYS-001-saved-seal-card-local_seal_001'),
              ),
            )
            .dy,
      ),
    );

    await tester.ensureVisible(find.text('Compare Seals'));
    await tester.pump();
    await tester.tap(find.text('Compare Seals'));
    await tester.pumpAndSettle();

    expect(find.text('Compare saved seals'), findsOneWidget);
    expect(
      find.text(
        'Review saved seal previews, kanji meanings, and style choices side by side.',
      ),
      findsOneWidget,
    );
    expect(find.textContaining('will be added later'), findsNothing);
    expect(find.text('Reading'), findsWidgets);
    expect(find.text('Shape'), findsWidgets);
    expect(find.text('永愛'), findsWidgets);
    expect(
      tester
          .getTopLeft(
            find.byKey(
              const ValueKey('MYS-002-comparison-card-local_seal_002'),
            ),
          )
          .dx,
      lessThan(
        tester
            .getTopLeft(
              find.byKey(
                const ValueKey('MYS-002-comparison-card-local_seal_001'),
              ),
            )
            .dx,
      ),
    );

    await tester.tap(find.text('Close'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Favorite seal').first);
    await tester.pump();

    expect(favoriteToggled?.id, 'local_seal_001');

    await tester.ensureVisible(find.text('View Details').first);
    await tester.pump();
    await tester.tap(find.text('View Details').first);
    await tester.pump();

    expect(opened?.id, 'local_seal_002');
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-002 shows empty saved seal actions', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var startDesignCount = 0;
    var exploreStonesCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: MySealsHomeScreen(
          onStartDesigning: () => startDesignCount += 1,
          onExploreStones: () => exploreStonesCount += 1,
        ),
      ),
    );

    expect(find.text('No saved seals'), findsOneWidget);
    expect(
      find.text('Saved seal designs will appear here after you create one.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Start Designing'));
    await tester.pump();
    await tester.tap(find.text('Browse Stones'));
    await tester.pump();

    expect(startDesignCount, 1);
    expect(exploreStonesCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-003 displays saved seal detail fields', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var chooseCount = 0;
    var editCount = 0;
    var deleteCount = 0;
    var backCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealDetailScreen(
          design: _localSealDesign(),
          onChooseForOrder: (_) => chooseCount += 1,
          onEditRegenerate: (_) => editCount += 1,
          onDelete: (_) async {
            deleteCount += 1;
          },
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text('Seal Detail'), findsOneWidget);
    expect(find.text('Kanji'), findsOneWidget);
    expect(find.text('美空'), findsWidgets);
    expect(find.text('Reading'), findsOneWidget);
    expect(find.text('Misora'), findsOneWidget);
    expect(find.text('Meaning'), findsOneWidget);
    expect(find.text('Beautiful sky'), findsOneWidget);
    expect(find.text('Shape'), findsOneWidget);
    expect(find.text('Square'), findsOneWidget);
    expect(find.text('Style'), findsOneWidget);
    expect(find.text('Elegant'), findsOneWidget);
    expect(find.text('Stroke Weight'), findsOneWidget);
    expect(find.text('Standard'), findsOneWidget);
    expect(find.text('Balance'), findsOneWidget);
    expect(find.text('Balanced'), findsOneWidget);
    expect(find.text('Created'), findsOneWidget);
    expect(find.text('2026-05-21 11:00'), findsOneWidget);
    expect(find.text('Choose for Order'), findsOneWidget);
    expect(find.text('Edit / Regenerate'), findsOneWidget);
    expect(find.text('Delete Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Edit / Regenerate'));
    await tester.pump();
    await tester.tap(find.text('Edit / Regenerate'));
    await tester.pumpAndSettle();

    expect(editCount, 1);
    expect(find.text('Create a new version from Design'), findsNothing);

    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pump();

    expect(chooseCount, 1);

    await tester.ensureVisible(find.text('Delete Seal'));
    await tester.pump();
    await tester.tap(find.text('Delete Seal'));
    await tester.pumpAndSettle();

    expect(find.text('Delete saved seal?'), findsOneWidget);
    expect(
      find.text(
        'This removes the seal design from this device. This action cannot be undone.',
      ),
      findsOneWidget,
    );

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(deleteCount, 0);

    await tester.ensureVisible(find.text('Delete Seal'));
    await tester.pump();
    await tester.tap(find.text('Delete Seal'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete').last);
    await tester.pumpAndSettle();

    expect(deleteCount, 1);

    await tester.ensureVisible(find.byTooltip('Back'));
    await tester.pump();
    await tester.tap(find.byTooltip('Back'));
    await tester.pump();

    expect(backCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-005 regenerates a saved seal through the design flow', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final generation = Completer<SealGenerationResult>();
    SealGenerationRequest? capturedRequest;

    await pumpLaunchedApp(
      tester,
      localSealDesignRepository: InMemoryLocalSealDesignRepository([
        _localSealDesign(
          selectedKanji: '雄護',
          meaning: 'Strong guardian',
          shape: 'round',
          style: 'bold',
          strokeWeight: 'bold',
          balance: 'dense',
        ),
      ]),
      generateSealDesigns: (request) {
        capturedRequest = request;
        return generation.future;
      },
    );

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Edit / Regenerate'));
    await tester.pump();
    await tester.tap(find.text('Edit / Regenerate'));
    await tester.pumpAndSettle();

    expect(find.byType(SealStyleSelectionScreen), findsOneWidget);
    expect(find.text('Seal Style'), findsOneWidget);
    expect(find.text('Selected kanji'), findsOneWidget);
    expect(find.text('雄護'), findsWidgets);
    expect(find.text('Strong guardian'), findsOneWidget);
    _expectSealStyleAdjustmentControlsPresent();

    await tester.ensureVisible(find.text('Confirm Style'));
    await tester.pump();
    await tester.tap(find.text('Confirm Style'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Generate Seal'));
    await tester.pump();
    await tester.tap(find.text('Generate Seal'));
    await tester.pump();

    expect(find.byType(SealGenerationLoadingScreen), findsOneWidget);
    expect(capturedRequest?.inputName, 'Michael Smith');
    expect(capturedRequest?.candidate.kanji, '雄護');
    expect(capturedRequest?.candidate.reading, 'Misora');
    expect(capturedRequest?.candidate.meaning, 'Strong guardian');
    expect(capturedRequest?.style.shape, SealShape.round);
    expect(capturedRequest?.style.style, SealStyleName.bold);
    expect(capturedRequest?.style.strokeWeight, SealStrokeWeight.bold);
    expect(capturedRequest?.style.balance, SealBalance.dense);

    generation.complete(_sealGenerationResult(request: capturedRequest!));
    await tester.pump();
    await tester.pumpAndSettle();

    expect(find.byType(SealVariantSelectionScreen), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-003 opens from the My Seals stack', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      localSealDesignRepository: InMemoryLocalSealDesignRepository([
        _localSealDesign(),
      ]),
    );

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    expect(find.byType(SealDetailScreen), findsOneWidget);
    expect(find.text('Seal Detail'), findsOneWidget);
    expect(find.text('2026-05-21 11:00'), findsOneWidget);

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(find.byType(MySealsHomeScreen), findsOneWidget);
    expect(find.text('Saved on this device'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-004 toggles a saved seal favorite from the card', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final repository = InMemoryLocalSealDesignRepository([_localSealDesign()]);

    await pumpLaunchedApp(tester, localSealDesignRepository: repository);

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();

    expect(find.byTooltip('Favorite seal'), findsOneWidget);

    await tester.tap(find.byTooltip('Favorite seal'));
    await tester.pumpAndSettle();

    expect(
      (await repository.getLocalSealDesign('local_seal_001'))?.isFavorite,
      isTrue,
    );
    expect(find.byTooltip('Remove favorite'), findsOneWidget);
    expect(
      find.descendant(
        of: find.byTooltip('Remove favorite'),
        matching: find.byIcon(Icons.favorite),
      ),
      findsOneWidget,
    );
    expect(
      find.descendant(
        of: find.byTooltip('Remove favorite'),
        matching: find.byIcon(Icons.star),
      ),
      findsNothing,
    );

    await tester.tap(find.byTooltip('Remove favorite'));
    await tester.pumpAndSettle();

    expect(
      (await repository.getLocalSealDesign('local_seal_001'))?.isFavorite,
      isFalse,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-008 keeps a saved seal selected for order draft', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      localSealDesignRepository: InMemoryLocalSealDesignRepository([
        _localSealDesign(),
      ]),
    );

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Stone missing'), findsOneWidget);
    expect(find.text('Choose a Stone'), findsWidgets);

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(find.text('Selected for order'), findsOneWidget);
    expect(
      find.text('This seal is now saved in the order draft.'),
      findsOneWidget,
    );
    expect(find.text('Selected for Order'), findsOneWidget);

    await tester.ensureVisible(find.byTooltip('Back'));
    await tester.pump();
    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    expect(find.text('Selected for order'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-008 prioritizes favorite saved seals for order selection', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      localSealDesignRepository: InMemoryLocalSealDesignRepository([
        _localSealDesign(),
        _localSealDesign(
          id: 'local_seal_002',
          selectedKanji: '永愛',
          meaning: 'Eternal love',
          style: 'soft',
          isFavorite: true,
        ),
      ]),
    );

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();

    expect(
      tester
          .getTopLeft(
            find.byKey(
              const ValueKey('MYS-001-saved-seal-card-local_seal_002'),
            ),
          )
          .dy,
      lessThan(
        tester
            .getTopLeft(
              find.byKey(
                const ValueKey('MYS-001-saved-seal-card-local_seal_001'),
              ),
            )
            .dy,
      ),
    );

    await tester.tap(find.text('View Details').first);
    await tester.pumpAndSettle();

    expect(find.text('永愛'), findsWidgets);
    expect(find.text('Eternal love'), findsOneWidget);

    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Stone missing'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('MYS-007 confirms and deletes a saved seal', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final repository = InMemoryLocalSealDesignRepository([_localSealDesign()]);

    await pumpLaunchedApp(tester, localSealDesignRepository: repository);

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Delete Seal'));
    await tester.pump();
    await tester.tap(find.text('Delete Seal'));
    await tester.pumpAndSettle();

    expect(find.text('Delete saved seal?'), findsOneWidget);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(find.byType(SealDetailScreen), findsOneWidget);
    expect(await repository.listLocalSealDesigns(), hasLength(1));

    await tester.tap(find.text('Delete Seal'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete').last);
    await tester.pumpAndSettle();

    expect(find.byType(SealDetailScreen), findsNothing);
    expect(find.text('No saved seals'), findsOneWidget);
    expect(await repository.listLocalSealDesigns(), isEmpty);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-003 displays the stones loading state', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: const StonesHomeScreen(isLoading: true),
      ),
    );

    expect(find.text('Stones'), findsOneWidget);
    expect(find.text('Loading stones'), findsOneWidget);
    expect(
      find.text('Checking available one-of-a-kind seal stones.'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-004 displays stones load errors and retry', (tester) async {
    var retryCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StonesHomeScreen(
          loadError: const HankoApiException(
            statusCode: 503,
            code: 'internal',
            message: 'temporary failure',
            payload: {},
          ),
          onRetry: () => retryCount += 1,
        ),
      ),
    );

    expect(find.text('Server Error'), findsOneWidget);
    expect(
      find.text(
        "We're experiencing a temporary issue on our end. Please wait a moment and try again.",
      ),
      findsOneWidget,
    );

    await tester.tap(find.text('Try Again'));
    await tester.pump();

    expect(retryCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-001 displays stone listing cards', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    StoneListing? selectedStone;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StonesHomeScreen(
          result: _stoneListingsResult(),
          onSelectStone: (listing) => selectedStone = listing,
        ),
      ),
    );

    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Rose Quartz'), findsWidgets);
    expect(find.text('¥18,000'), findsOneWidget);
    expect(find.text('Pink'), findsWidgets);
    expect(find.text('Plain'), findsWidgets);
    expect(find.text('24x24x60 mm'), findsOneWidget);
    expect(find.text('Available'), findsWidgets);
    expect(find.text('Select Stone'), findsOneWidget);

    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();

    expect(find.text('Select this stone?'), findsOneWidget);
    expect(selectedStone, isNull);

    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pump();

    expect(selectedStone?.id, 'stone_listing_001');
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-001 does not mark unavailable stale stone as selected', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    StoneListing? selectedStone;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StonesHomeScreen(
          result: _stoneListingsResult(
            listings: [_stoneListing(status: 'reserved', isOrderable: false)],
          ),
          selectedStoneId: 'stone_listing_001',
          onSelectStone: (listing) => selectedStone = listing,
        ),
      ),
    );

    expect(find.text('Unavailable'), findsWidgets);
    expect(find.text('Selected for Order'), findsNothing);
    expect(find.text('Select Stone'), findsOneWidget);

    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'), warnIfMissed: false);
    await tester.pumpAndSettle();

    expect(find.text('Select this stone?'), findsNothing);
    expect(selectedStone, isNull);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-005 filters stones by material color pattern and stock', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StonesHomeScreen(
          result: _stoneListingsResult(
            listings: [
              _stoneListing(),
              _stoneListing(
                id: 'stone_listing_002',
                title: 'Green Jade Seal Stone',
                materialKey: 'jade',
                materialLabel: 'Jade',
                colorFamily: 'green',
                patternPrimary: 'cloudy',
              ),
              _stoneListing(
                id: 'stone_listing_003',
                title: 'Black Onyx Seal Stone',
                materialKey: 'black_onyx',
                materialLabel: 'Black Onyx',
                colorFamily: 'black',
                patternPrimary: 'banded',
                status: 'sold',
                isOrderable: false,
              ),
            ],
          ),
        ),
      ),
    );

    expect(find.text('Filters'), findsOneWidget);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Green Jade Seal Stone'), findsOneWidget);
    expect(find.text('Black Onyx Seal Stone'), findsOneWidget);

    await tester.tap(find.byKey(const Key('stone-filter-material-jade')));
    await tester.pump();

    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsNothing);
    expect(find.text('Green Jade Seal Stone'), findsOneWidget);
    expect(find.text('Black Onyx Seal Stone'), findsNothing);

    await tester.tap(find.byKey(const Key('stone-filters-reset')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('stone-filter-color-pink')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('stone-filter-pattern-plain')));
    await tester.pump();

    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Green Jade Seal Stone'), findsNothing);
    expect(find.text('Black Onyx Seal Stone'), findsNothing);

    await tester.tap(find.byKey(const Key('stone-filters-reset')));
    await tester.pump();
    await tester.tap(
      find.byKey(const Key('stone-filter-availability-unavailable')),
    );
    await tester.pump();

    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsNothing);
    expect(find.text('Green Jade Seal Stone'), findsNothing);
    expect(find.text('Black Onyx Seal Stone'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-006 sorts stones by newest and price', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    const highPriceTitle = 'High Price Stone';
    const lowPriceTitle = 'Low Price Stone';
    const newestTitle = 'Newest Stone';

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StonesHomeScreen(
          result: _stoneListingsResult(
            listings: [
              _stoneListing(
                id: 'stone_listing_high_price',
                title: highPriceTitle,
                priceAmount: 32000,
                sortOrder: 10,
              ),
              _stoneListing(
                id: 'stone_listing_low_price',
                title: lowPriceTitle,
                priceAmount: 12000,
                sortOrder: 20,
              ),
              _stoneListing(
                id: 'stone_listing_newest',
                title: newestTitle,
                priceAmount: 22000,
                sortOrder: 30,
              ),
            ],
          ),
        ),
      ),
    );

    final titles = [highPriceTitle, lowPriceTitle, newestTitle];

    expect(_stoneTitleOrder(tester, titles), [
      highPriceTitle,
      lowPriceTitle,
      newestTitle,
    ]);

    await tester.tap(find.byKey(const Key('stone-sort-open')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-sort-price-low-to-high')));
    await tester.pumpAndSettle();

    expect(_stoneTitleOrder(tester, titles), [
      lowPriceTitle,
      newestTitle,
      highPriceTitle,
    ]);

    await tester.tap(find.byKey(const Key('stone-sort-open')));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-sort-newest')));
    await tester.pumpAndSettle();

    expect(_stoneTitleOrder(tester, titles), [
      newestTitle,
      lowPriceTitle,
      highPriceTitle,
    ]);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-007 displays stone detail fields and notes', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var backCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: StoneDetailScreen(
          listing: _stoneListing(
            description: 'A soft pink rose quartz seal stone.',
            story: 'A one-of-a-kind piece with delicate translucency.',
          ),
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text('Stone Detail'), findsOneWidget);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Rose Quartz'), findsWidgets);
    expect(find.text('¥18,000'), findsOneWidget);
    expect(find.text('Description'), findsOneWidget);
    expect(find.text('A soft pink rose quartz seal stone.'), findsOneWidget);
    expect(find.text('Story'), findsOneWidget);
    expect(
      find.text('A one-of-a-kind piece with delicate translucency.'),
      findsOneWidget,
    );
    expect(find.text('Details'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
    expect(find.text('24x24x60 mm'), findsOneWidget);
    expect(find.text('Color'), findsOneWidget);
    expect(find.text('Pattern'), findsOneWidget);
    expect(find.text('Texture'), findsOneWidget);
    expect(find.text('Available'), findsWidgets);
    expect(find.text('Notes'), findsOneWidget);
    expect(
      find.textContaining('Natural stone color, pattern, and translucency'),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('Back'));
    await tester.pump();

    expect(backCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-007 opens from stones list and refreshes detail', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    StoneListingDetailQuery? capturedQuery;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      getStoneListingDetail: (query) async {
        capturedQuery = query;
        return _stoneListing(
          id: query.listingId,
          title: 'Detailed Rose Quartz Seal Stone',
          description: 'Detailed description from the API.',
          story: 'Detailed story from the API.',
        );
      },
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('View Details'));
    await tester.pump();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    expect(capturedQuery?.listingId, 'stone_listing_001');
    expect(capturedQuery?.locale, 'en');
    expect(find.text('Stone Detail'), findsOneWidget);
    expect(find.text('Detailed Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Detailed description from the API.'), findsOneWidget);
    expect(find.text('Detailed story from the API.'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-008 opens image gallery from stone detail', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      getStoneListingDetail: (query) async {
        return _stoneListing(id: query.listingId, photos: _stonePhotos());
      },
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('View Details'));
    await tester.pump();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    await tester.tap(find.byKey(const Key('stone-detail-gallery-thumbnail-1')));
    await tester.pumpAndSettle();

    expect(find.byType(StoneImageGalleryScreen), findsOneWidget);
    expect(find.text('2 / 3'), findsOneWidget);

    await tester.tap(find.byKey(const Key('stone-gallery-next')));
    await tester.pumpAndSettle();

    expect(find.text('3 / 3'), findsOneWidget);

    await tester.tap(find.byKey(const Key('stone-gallery-thumbnail-0')));
    await tester.pumpAndSettle();

    expect(find.text('1 / 3'), findsOneWidget);

    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();

    expect(find.byType(StoneImageGalleryScreen), findsNothing);
    expect(find.text('Stone Detail'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-009 confirms stone selection in the app shell', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();

    expect(find.text('Select this stone?'), findsOneWidget);
    expect(find.text('Confirm Selection'), findsOneWidget);

    await tester.tap(find.text('Cancel'));
    await tester.pumpAndSettle();

    expect(find.text('Selected for Order'), findsNothing);

    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    expect(find.text('Seal design missing'), findsOneWidget);
    expect(find.text('Choose a Seal'), findsWidgets);

    await tester.tap(find.byTooltip('Back'));
    await tester.pumpAndSettle();

    expect(find.text('Selected for Order'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-010 blocks sold out stone selection from detail', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      getStoneListingDetail: (query) async {
        return _stoneListing(
          id: query.listingId,
          status: 'sold',
          isOrderable: false,
        );
      },
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('View Details'));
    await tester.pump();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    expect(find.byKey(const Key('stone-sold-out-state')), findsOneWidget);
    expect(find.text('Stone unavailable'), findsOneWidget);
    expect(find.text('Unavailable'), findsWidgets);

    await tester.ensureVisible(find.byKey(const Key('stone-sold-out-state')));
    await tester.pump();
    await tester.tap(find.text('Select Stone').last, warnIfMissed: false);
    await tester.pumpAndSettle();

    expect(find.text('Select this stone?'), findsNothing);
    expect(find.text('Selected for Order'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M08-T01 keeps order draft across tabs and app shell reload', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Stone missing'), findsOneWidget);
    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.sealSelection?.localSealDesignId, 'local_seal_001');
    expect(savedDraft.stoneSelection?.listingId, 'stone_listing_001');

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();

    expect(find.text('Selected for order'), findsOneWidget);

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();

    expect(find.text('Selected for Order'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M08-T02 opens combination review with pricing summary', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Stone missing'), findsOneWidget);
    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    expect(find.text('Order Review'), findsOneWidget);
    expect(find.text('美空'), findsWidgets);
    expect(find.text('Elegant'), findsOneWidget);
    expect(find.text('Square'), findsOneWidget);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Rose Quartz / 24x24x60 mm'), findsOneWidget);
    expect(find.text('Item price'), findsOneWidget);
    expect(find.text('Shipping'), findsOneWidget);
    expect(find.text('Total'), findsOneWidget);
    expect(find.text('JPY 18,000'), findsWidgets);
    expect(find.text('JPY 600'), findsOneWidget);
    expect(find.text('JPY 18,600'), findsOneWidget);
    expect(find.text('Continue to Shipping'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M08-T02 displays USD order pricing with currency code', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderFlowEntryScreen(
          draft: OrderDraft.empty()
              .withSealSelection(_orderDraftSealSelection())
              .withStoneSelection(
                _orderDraftStoneSelection(
                  price: const Money(amount: 28000, currency: 'USD'),
                ),
              )
              .withInput(
                const OrderDraftInput.empty().copyWith(
                  shipping: const OrderDraftShippingInput.empty().copyWith(
                    countryCode: 'US',
                  ),
                ),
              ),
        ),
      ),
    );

    expect(find.text('USD 280.00'), findsWidgets);
    expect(find.text('USD 18.00'), findsOneWidget);
    expect(find.text('USD 298.00'), findsOneWidget);
    expect(find.textContaining(r'$'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M08-T03 shows missing seal and stone next actions', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    expect(find.text('Seal design missing'), findsOneWidget);
    expect(
      find.text('Choose a saved seal design before continuing to checkout.'),
      findsOneWidget,
    );

    await tester.tap(find.text('Choose a Seal').last);
    await tester.pumpAndSettle();

    expect(find.text('Saved on this device'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();

    final secondDraftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: secondDraftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Stone missing'), findsOneWidget);
    expect(
      find.text('Choose a gemstone seal stone before continuing to checkout.'),
      findsOneWidget,
    );

    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();

    expect(find.text('Stones'), findsWidgets);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M08-T04 returns to My Seals and Stones to change choices', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
      _localSealDesign(id: 'local_seal_002', selectedKanji: '光', style: 'bold'),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    final stoneListings = [
      _stoneListing(),
      _stoneListing(
        id: 'stone_listing_002',
        title: 'Blue Lapis Seal Stone',
        materialKey: 'lapis_lazuli',
        materialLabel: 'Lapis Lazuli',
        colorFamily: 'blue',
        patternPrimary: 'flecked',
        priceAmount: 24000,
        sortOrder: 1,
      ),
    ];

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async =>
          _stoneListingsResult(listings: stoneListings),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details').first);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone').first);
    await tester.pump();
    await tester.tap(find.text('Select Stone').first);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    expect(find.text('Order Review'), findsOneWidget);
    expect(find.text('Change Seal'), findsOneWidget);
    expect(find.text('Change Stone'), findsOneWidget);

    await tester.ensureVisible(find.text('Change Seal'));
    await tester.pump();
    await tester.tap(find.text('Change Seal'));
    await tester.pumpAndSettle();

    expect(find.text('Saved on this device'), findsOneWidget);

    await tester.tap(find.text('View Details').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    expect(find.text('Order Review'), findsOneWidget);
    expect(find.text('光'), findsWidgets);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);

    await tester.ensureVisible(find.text('Change Stone'));
    await tester.pump();
    await tester.tap(find.text('Change Stone'));
    await tester.pumpAndSettle();

    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Blue Lapis Seal Stone'), findsOneWidget);

    await tester.tap(find.text('Select Stone').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    expect(find.text('Order Review'), findsOneWidget);
    expect(find.text('光'), findsWidgets);
    expect(find.text('Blue Lapis Seal Stone'), findsOneWidget);
    expect(find.text('JPY 24,000'), findsWidgets);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T01 saves checkout contact shipping and note input', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Continue to Shipping'));
    await tester.pump();
    await tester.tap(find.text('Continue to Shipping'));
    await tester.pumpAndSettle();

    expect(find.byType(CheckoutInputScreen), findsOneWidget);
    expect(find.text('Checkout Information'), findsOneWidget);
    expect(find.text('Contact'), findsOneWidget);
    expect(find.text('Shipping address'), findsOneWidget);
    expect(find.text('Order note'), findsWidgets);
    expect(find.text('Country / Region'), findsOneWidget);

    Future<void> enterCheckoutField(String key, String text) async {
      final field = find.byKey(Key(key));
      await tester.ensureVisible(field);
      await tester.pump();
      await tester.enterText(
        find.descendant(of: field, matching: find.byType(EditableText)),
        text,
      );
      await tester.pump();
    }

    await enterCheckoutField('checkout-email-field', 'customer@example.test');
    await enterCheckoutField('checkout-full-name-field', 'Michael Smith');
    await enterCheckoutField('checkout-phone-field', '+1 555 0100');

    await tester.ensureVisible(find.byKey(const Key('checkout-country-field')));
    await tester.pump();
    await tester.tap(find.byKey(const Key('checkout-country-field')));
    await tester.pumpAndSettle();
    await tester.tap(find.text('US - United States').last);
    await tester.pumpAndSettle();

    await enterCheckoutField('checkout-postal-code-field', '10001');
    await enterCheckoutField(
      'checkout-address-line1-field',
      '123 Example Street',
    );
    await enterCheckoutField('checkout-address-line2-field', 'Apt 1');
    await enterCheckoutField('checkout-city-field', 'New York');
    await enterCheckoutField('checkout-state-field', 'NY');
    await enterCheckoutField(
      'checkout-order-note-field',
      'Please ship on a weekday.',
    );

    await tester.ensureVisible(find.text('Save Checkout Information'));
    await tester.pump();
    await tester.tap(find.text('Save Checkout Information'));
    await tester.pumpAndSettle();

    expect(find.byType(OrderConfirmationScreen), findsOneWidget);
    expect(find.text('Order Confirmation'), findsOneWidget);
    expect(find.text('Proceed to Secure Payment'), findsOneWidget);

    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.input.contact.email, 'customer@example.test');
    expect(savedDraft.inputUpdatedAt, isNotNull);
    expect(savedDraft.input.contact.preferredLocale, 'en');
    expect(savedDraft.input.shipping.countryCode, 'US');
    expect(savedDraft.input.shipping.recipientName, 'Michael Smith');
    expect(savedDraft.input.shipping.phone, '+1 555 0100');
    expect(savedDraft.input.shipping.postalCode, '10001');
    expect(savedDraft.input.shipping.addressLine1, '123 Example Street');
    expect(savedDraft.input.shipping.addressLine2, 'Apt 1');
    expect(savedDraft.input.shipping.city, 'New York');
    expect(savedDraft.input.shipping.state, 'NY');
    expect(savedDraft.input.orderNote, 'Please ship on a weekday.');
    expect(savedDraft.hasCombinationSelections, isTrue);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T01 discards checkout input older than 24 hours', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final expiredAt = DateTime.now().subtract(const Duration(hours: 25));
    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository(
      OrderDraft.empty(updatedAt: expiredAt)
          .withStoneSelection(_orderDraftStoneSelection(), updatedAt: expiredAt)
          .withInput(_checkoutInput(), updatedAt: expiredAt),
    );

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Continue to Shipping'));
    await tester.pump();
    await tester.tap(find.text('Continue to Shipping'));
    await tester.pumpAndSettle();

    expect(find.byType(CheckoutInputScreen), findsOneWidget);
    expect(find.text('old@example.test'), findsNothing);
    expect(find.text('Old Recipient'), findsNothing);
    expect(find.text('999 Expired Street'), findsNothing);

    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.hasCombinationSelections, isTrue);
    expect(savedDraft.input.isEmpty, isTrue);
    expect(savedDraft.inputUpdatedAt, isNull);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'M09-T02 summarizes invalid checkout input and returns to fields',
    (tester) async {
      tester.view.physicalSize = const Size(432, 912);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);

      final sealRepository = InMemoryLocalSealDesignRepository([
        _localSealDesign(),
      ]);
      final draftRepository = InMemoryLocalOrderDraftRepository();

      await pumpLaunchedApp(
        tester,
        listStoneListings: (query) async => _stoneListingsResult(),
        localSealDesignRepository: sealRepository,
        localOrderDraftRepository: draftRepository,
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('My Seals').last);
      await tester.pumpAndSettle();
      await tester.tap(find.text('View Details'));
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Choose for Order'));
      await tester.pump();
      await tester.tap(find.text('Choose for Order'));
      await tester.pumpAndSettle();

      await tester.ensureVisible(find.text('Choose a Stone').last);
      await tester.pump();
      await tester.tap(find.text('Choose a Stone').last);
      await tester.pumpAndSettle();
      await tester.ensureVisible(find.text('Select Stone'));
      await tester.pump();
      await tester.tap(find.text('Select Stone'));
      await tester.pumpAndSettle();
      await tester.tap(find.byKey(const Key('stone-selection-confirm')));
      await tester.pumpAndSettle();

      await tester.ensureVisible(find.text('Continue to Shipping'));
      await tester.pump();
      await tester.tap(find.text('Continue to Shipping'));
      await tester.pumpAndSettle();

      Future<void> enterCheckoutField(String key, String text) async {
        final field = find.byKey(Key(key));
        await tester.ensureVisible(field);
        await tester.pump();
        await tester.enterText(
          find.descendant(of: field, matching: find.byType(EditableText)),
          text,
        );
        await tester.pump();
      }

      await enterCheckoutField('checkout-email-field', 'not-an-email');
      await enterCheckoutField('checkout-full-name-field', 'Michael Smith');

      await tester.ensureVisible(find.text('Save Checkout Information'));
      await tester.pump();
      await tester.tap(find.text('Save Checkout Information'));
      await tester.pumpAndSettle();

      expect(
        find.text('Please review the highlighted fields.'),
        findsOneWidget,
      );
      expect(
        find.text('Some information is missing or invalid.'),
        findsOneWidget,
      );
      expect(find.text('Please enter a valid email address.'), findsWidgets);
      expect(find.text('Please enter a valid phone number.'), findsWidgets);
      expect(find.text('Postal code is required.'), findsWidgets);
      expect(find.text('Address line 1 is required.'), findsWidgets);
      expect(find.text('City is required.'), findsWidgets);
      expect(find.text('State / Province is required.'), findsWidgets);
      expect(
        find.text('State / Province: State / Province is required.'),
        findsOneWidget,
      );
      expect(
        find.text('Checkout information was saved to this order draft.'),
        findsNothing,
      );

      final invalidDraft = await draftRepository.loadOrderDraft();
      expect(invalidDraft.input.contact.email, isEmpty);
      expect(invalidDraft.input.shipping.recipientName, isEmpty);

      await tester.tap(
        find.text('State / Province: State / Province is required.'),
      );
      await tester.pumpAndSettle();

      final stateFieldTop = tester
          .getTopLeft(find.byKey(const Key('checkout-state-field')))
          .dy;
      expect(stateFieldTop, greaterThanOrEqualTo(0));
      expect(stateFieldTop, lessThan(tester.view.physicalSize.height));

      await enterCheckoutField('checkout-email-field', 'customer@example.test');
      await enterCheckoutField('checkout-phone-field', '+1 555 0100');
      await enterCheckoutField('checkout-postal-code-field', '10001');
      await enterCheckoutField(
        'checkout-address-line1-field',
        '123 Example Street',
      );
      await enterCheckoutField('checkout-city-field', 'New York');
      await enterCheckoutField('checkout-state-field', 'NY');

      await tester.ensureVisible(find.text('Save Checkout Information'));
      await tester.pump();
      await tester.tap(find.text('Save Checkout Information'));
      await tester.pumpAndSettle();

      expect(find.text('Please review the highlighted fields.'), findsNothing);
      expect(find.byType(OrderConfirmationScreen), findsOneWidget);
      expect(find.text('Order Confirmation'), findsOneWidget);

      final savedDraft = await draftRepository.loadOrderDraft();
      expect(savedDraft.input.contact.email, 'customer@example.test');
      expect(savedDraft.input.shipping.recipientName, 'Michael Smith');
      expect(savedDraft.input.shipping.phone, '+1 555 0100');
      expect(savedDraft.input.shipping.postalCode, '10001');
      expect(savedDraft.input.shipping.addressLine1, '123 Example Street');
      expect(savedDraft.input.shipping.city, 'New York');
      expect(savedDraft.input.shipping.state, 'NY');
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('M09-T03 requires order confirmation agreement checks', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Continue to Shipping'));
    await tester.pump();
    await tester.tap(find.text('Continue to Shipping'));
    await tester.pumpAndSettle();

    Future<void> enterCheckoutField(String key, String text) async {
      final field = find.byKey(Key(key));
      await tester.ensureVisible(field);
      await tester.pump();
      await tester.enterText(
        find.descendant(of: field, matching: find.byType(EditableText)),
        text,
      );
      await tester.pump();
    }

    await enterCheckoutField('checkout-email-field', 'customer@example.test');
    await enterCheckoutField('checkout-full-name-field', 'Michael Smith');
    await enterCheckoutField('checkout-phone-field', '+1 555 0100');
    await enterCheckoutField('checkout-postal-code-field', '10001');
    await enterCheckoutField(
      'checkout-address-line1-field',
      '123 Example Street',
    );
    await enterCheckoutField('checkout-city-field', 'New York');
    await enterCheckoutField('checkout-state-field', 'NY');

    await tester.ensureVisible(find.text('Save Checkout Information'));
    await tester.pump();
    await tester.tap(find.text('Save Checkout Information'));
    await tester.pumpAndSettle();

    expect(find.byType(OrderConfirmationScreen), findsOneWidget);
    expect(find.text('Order Confirmation'), findsOneWidget);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Michael Smith'), findsOneWidget);
    expect(find.text('customer@example.test'), findsOneWidget);
    expect(find.text('JPY 18,600'), findsOneWidget);
    expect(
      find.text(
        'I confirm that the selected kanji and seal design are correct.',
      ),
      findsOneWidget,
    );
    expect(
      find.text(
        'I understand that this is a custom-made item and cannot be changed after production begins.',
      ),
      findsOneWidget,
    );

    final proceedButton = find.widgetWithText(
      TextButton,
      'Proceed to Secure Payment',
    );
    expect(tester.widget<TextButton>(proceedButton).onPressed, isNull);

    await tester.ensureVisible(
      find.byKey(const Key('order-confirm-kanji-design-checkbox')),
    );
    await tester.pump();
    await tester.tap(
      find.byKey(const Key('order-confirm-kanji-design-checkbox')),
    );
    await tester.pumpAndSettle();

    expect(tester.widget<TextButton>(proceedButton).onPressed, isNull);

    await tester.ensureVisible(
      find.byKey(const Key('order-confirm-custom-made-checkbox')),
    );
    await tester.pump();
    await tester.tap(
      find.byKey(const Key('order-confirm-custom-made-checkbox')),
    );
    await tester.pumpAndSettle();

    expect(tester.widget<TextButton>(proceedButton).onPressed, isNotNull);

    await tester.ensureVisible(find.text('Proceed to Secure Payment'));
    await tester.pump();
    await tester.tap(find.text('Proceed to Secure Payment'));
    await tester.pumpAndSettle();

    expect(find.text('Secure Payment'), findsOneWidget);
    expect(find.text('Complete payment in Stripe Checkout'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);

    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.input.customerConfirmation.kanjiAndDesign, isTrue);
    expect(savedDraft.input.customerConfirmation.customMadePolicy, isTrue);
    expect(savedDraft.input.customerConfirmation.isComplete, isTrue);
    expect(savedDraft.input.termsAgreed, isTrue);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T07 shows order and checkout session creation progress', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    final orderCompleter = Completer<CreatedOrder>();
    final sessionCompleter = Completer<CheckoutSession>();
    SealOrderDraft? submittedOrderDraft;
    CheckoutSessionRequest? submittedCheckoutRequest;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      createOrder: (draft) {
        submittedOrderDraft = draft;
        return orderCompleter.future;
      },
      createCheckoutSession: (request) {
        submittedCheckoutRequest = request;
        return sessionCompleter.future;
      },
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('View Details'));
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Choose for Order'));
    await tester.pump();
    await tester.tap(find.text('Choose for Order'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Choose a Stone').last);
    await tester.pump();
    await tester.tap(find.text('Choose a Stone').last);
    await tester.pumpAndSettle();
    await tester.ensureVisible(find.text('Select Stone'));
    await tester.pump();
    await tester.tap(find.text('Select Stone'));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const Key('stone-selection-confirm')));
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Continue to Shipping'));
    await tester.pump();
    await tester.tap(find.text('Continue to Shipping'));
    await tester.pumpAndSettle();

    Future<void> enterCheckoutField(String key, String text) async {
      final field = find.byKey(Key(key));
      await tester.ensureVisible(field);
      await tester.pump();
      await tester.enterText(
        find.descendant(of: field, matching: find.byType(EditableText)),
        text,
      );
      await tester.pump();
    }

    await enterCheckoutField('checkout-email-field', 'customer@example.test');
    await enterCheckoutField('checkout-full-name-field', 'Michael Smith');
    await enterCheckoutField('checkout-phone-field', '+1 555 0100');
    await enterCheckoutField('checkout-postal-code-field', '10001');
    await enterCheckoutField(
      'checkout-address-line1-field',
      '123 Example Street',
    );
    await enterCheckoutField('checkout-city-field', 'New York');
    await enterCheckoutField('checkout-state-field', 'NY');

    await tester.ensureVisible(find.text('Save Checkout Information'));
    await tester.pump();
    await tester.tap(find.text('Save Checkout Information'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(
      find.byKey(const Key('order-confirm-kanji-design-checkbox')),
    );
    await tester.pump();
    await tester.tap(
      find.byKey(const Key('order-confirm-kanji-design-checkbox')),
    );
    await tester.pumpAndSettle();
    await tester.ensureVisible(
      find.byKey(const Key('order-confirm-custom-made-checkbox')),
    );
    await tester.pump();
    await tester.tap(
      find.byKey(const Key('order-confirm-custom-made-checkbox')),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.text('Proceed to Secure Payment'));
    await tester.pump();
    await tester.tap(find.text('Proceed to Secure Payment'));
    await tester.pump();
    await tester.pump();

    expect(find.text('Preparing Checkout'), findsOneWidget);
    expect(find.text('Creating order'), findsOneWidget);
    expect(find.text('Creating secure payment session'), findsOneWidget);
    expect(submittedOrderDraft?.channel, 'app');
    expect(submittedOrderDraft?.seal.fontKey, 'ai_generated_seal');
    expect(submittedOrderDraft?.seal.aiGenerationId, 'seal_request_001');
    expect(submittedOrderDraft?.seal.aiVariantId, 'seal_variant_001');
    expect(
      submittedOrderDraft?.seal.previewImage?.storagePath,
      'seal_designs/seal_request_001/seal_variant_001.png',
    );
    expect(submittedOrderDraft?.seal.style?.name, 'elegant');
    expect(submittedOrderDraft?.customerConfirmation?.kanjiAndDesign, isTrue);
    expect(submittedOrderDraft?.customerConfirmation?.customMadePolicy, isTrue);
    expect(submittedOrderDraft?.customerConfirmation?.confirmedSealText, '美空');

    orderCompleter.complete(
      const CreatedOrder(
        orderId: 'ord_001',
        orderNo: 'HF-20260521-0001',
        status: 'pending_payment',
        paymentStatus: 'unpaid',
        fulfillmentStatus: 'pending',
        pricing: Money(amount: 18600, currency: 'JPY'),
        idempotentReplay: false,
      ),
    );
    await tester.pump();
    await tester.pump();

    expect(find.text('Creating secure payment session'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(submittedCheckoutRequest?.orderId, 'ord_001');
    expect(submittedCheckoutRequest?.customerEmail, 'customer@example.test');
    expect(submittedCheckoutRequest?.returnToApp, isTrue);

    sessionCompleter.complete(
      const CheckoutSession(
        orderId: 'ord_001',
        sessionId: 'cs_test_001',
        checkoutUrl: 'https://checkout.stripe.test/session',
        paymentIntentId: 'pi_test_001',
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Secure Payment'), findsOneWidget);
    expect(find.text('Complete payment in Stripe Checkout'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T08 opens Stripe Checkout and handles return routes', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    final launchedSessions = <CheckoutSession>[];

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      openCheckoutUrl: (session) async => launchedSessions.add(session),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    expect(launchedSessions, hasLength(1));
    expect(
      launchedSessions.single.checkoutUrl,
      'https://checkout.stripe.test/session',
    );
    expect(find.text('Secure Payment'), findsOneWidget);
    expect(find.text('Complete payment in Stripe Checkout'), findsOneWidget);

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(find.text('Order Complete'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(find.text('customer@example.test'), findsOneWidget);
    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.hasCombinationSelections, isTrue);
    expect(savedDraft.input.isEmpty, isTrue);
    expect(savedDraft.inputUpdatedAt, isNull);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T08 reconciles paid checkout when the app resumes', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    final launchedSessions = <CheckoutSession>[];
    final checkedOrderIds = <String>[];

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      openCheckoutUrl: (session) async => launchedSessions.add(session),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      fetchOrderStatus: (orderId) {
        checkedOrderIds.add(orderId);
        return _successfulFetchOrderStatus(orderId);
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    expect(launchedSessions, hasLength(1));
    expect(find.text('Secure Payment'), findsOneWidget);
    expect(find.text('Complete payment in Stripe Checkout'), findsOneWidget);

    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pumpAndSettle();

    expect(checkedOrderIds, ['ord_001']);
    expect(find.text('Order Complete'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(find.text('Open Stripe Checkout'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M12-T05 shows deep link errors for invalid checkout returns', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var didFetchOrderStatus = false;

    await pumpLaunchedApp(
      tester,
      initialCheckoutRoute:
          'hankofield://checkout/success?session_id=cs_test_001&lang=en',
      fetchOrderStatus: (orderId) async {
        didFetchOrderStatus = true;
        return _successfulFetchOrderStatus(orderId);
      },
    );
    await tester.pumpAndSettle();

    expect(didFetchOrderStatus, isFalse);
    expect(find.byType(CheckoutDeepLinkErrorScreen), findsOneWidget);
    expect(find.text('Checkout Return Link Error'), findsOneWidget);
    expect(
      find.text(
        "The Stripe Checkout return link couldn't be processed. Please open Checkout again or contact support if payment may have completed.",
      ),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T09 confirms paid order status after Stripe return', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    final statusCompleter = Completer<OrderStatus>();
    final checkedOrderIds = <String>[];

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      fetchOrderStatus: (orderId) {
        checkedOrderIds.add(orderId);
        return statusCompleter.future;
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pump();

    expect(handled, isTrue);
    expect(find.text('Checking payment status'), findsOneWidget);
    expect(checkedOrderIds, ['ord_001']);

    statusCompleter.complete(
      const OrderStatus(
        orderId: 'ord_001',
        orderNo: 'HF-20260521-0001',
        orderStatus: 'paid',
        paymentStatus: 'paid',
        fulfillmentStatus: 'pending',
        productionStatus: 'not_started',
        shippingStatus: 'not_shipped',
        pricing: Money(amount: 18600, currency: 'JPY'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Order Complete'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T09 shows pending when webhook status is not reflected', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    var checkCount = 0;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      fetchOrderStatus: (orderId) async {
        checkCount++;
        return const OrderStatus(
          orderId: 'ord_001',
          orderNo: 'HF-20260521-0001',
          orderStatus: 'pending_payment',
          paymentStatus: 'unpaid',
          fulfillmentStatus: 'pending',
          productionStatus: 'not_started',
          shippingStatus: 'not_shipped',
          pricing: Money(amount: 18600, currency: 'JPY'),
        );
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(checkCount, 3);
    expect(find.text('Payment pending'), findsOneWidget);
    expect(
      find.text(
        'Stripe returned successfully, but payment confirmation is still pending.',
      ),
      findsOneWidget,
    );
    expect(
      find.textContaining('Webhook confirmation may take a moment'),
      findsNothing,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T11 shows order complete and order lookup route', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(find.text('Order Complete'), findsOneWidget);
    expect(find.text('Thank you for your order'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(find.text('Payment received'), findsOneWidget);
    expect(find.text('Order summary'), findsOneWidget);
    expect(find.text('Stripe payment email'), findsOneWidget);
    expect(
      find.text(
        'Stripe sends the payment receipt to the email address on the order. Please check your inbox and spam folder.',
      ),
      findsOneWidget,
    );
    expect(find.text("Can't find the Stripe email?"), findsOneWidget);
    expect(find.text('Here are a few quick things to check.'), findsOneWidget);
    expect(find.text('Check your spam or junk folder.'), findsOneWidget);
    expect(
      find.text('Make sure the email address on the order is correct.'),
      findsOneWidget,
    );
    expect(
      find.text('Please allow a few minutes for delivery.'),
      findsOneWidget,
    );
    expect(
      find.text(
        "If you still can't find it, contact support with your order number.",
      ),
      findsOneWidget,
    );
    expect(find.text('Need help?'), findsOneWidget);
    expect(
      find.text(
        'Our support team can help with order, shipping, payment, and email questions. Include your order number for faster support.',
      ),
      findsOneWidget,
    );
    expect(find.text('Contact Support'), findsOneWidget);
    expect(find.text('Open Order Lookup'), findsOneWidget);

    await tester.ensureVisible(find.text('Open Order Lookup'));
    await tester.pump();
    await tester.tap(find.text('Open Order Lookup'));
    await tester.pumpAndSettle();

    expect(find.text('Order Lookup'), findsOneWidget);
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(find.widgetWithText(TextFormField, 'Email'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M12-T03 opens contact support prompt to Contact', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/success?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(find.text('Need help?'), findsOneWidget);

    await tester.ensureVisible(find.text('Contact Support'));
    await tester.pump();
    await tester.tap(find.text('Contact Support'));
    await tester.pumpAndSettle();

    expect(find.text('Contact'), findsOneWidget);
    expect(
      find.textContaining('https://finitefield.org/en/contact/'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T10 separates Stripe cancel returns', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    var statusCheckCount = 0;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      fetchOrderStatus: (orderId) {
        statusCheckCount++;
        return _successfulFetchOrderStatus(orderId);
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/cancel?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(statusCheckCount, 0);
    expect(find.text('Checkout was canceled'), findsOneWidget);
    expect(
      find.text('Stripe returned without completing payment.'),
      findsOneWidget,
    );
    expect(find.text('Payment failed'), findsNothing);
    expect(find.text('Open Stripe Checkout'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T10 separates Stripe failure returns', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    var statusCheckCount = 0;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      fetchOrderStatus: (orderId) {
        statusCheckCount++;
        return _successfulFetchOrderStatus(orderId);
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    final handled = await tester.binding.handlePushRoute(
      'hankofield://checkout/failed?order_id=ord_001&session_id=cs_test_001&lang=en',
    );
    await tester.pumpAndSettle();

    expect(handled, isTrue);
    expect(statusCheckCount, 0);
    expect(find.text('Payment failed'), findsOneWidget);
    expect(
      find.text(
        'Stripe Checkout could not be completed. You can try Checkout again.',
      ),
      findsOneWidget,
    );
    expect(find.text('Checkout was canceled'), findsNothing);
    expect(find.text('Try Again'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M09-T10 maps Stripe checkout session API errors to failed', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final sealRepository = InMemoryLocalSealDesignRepository([
      _localSealDesign(),
    ]);
    final draftRepository = InMemoryLocalOrderDraftRepository();
    var didOpenCheckout = false;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(),
      localSealDesignRepository: sealRepository,
      localOrderDraftRepository: draftRepository,
      createCheckoutSession: (request) async {
        throw const HankoApiException(
          statusCode: 502,
          code: 'stripe_checkout_failed',
          message: 'Stripe checkout session creation failed',
          payload: {},
        );
      },
      openCheckoutUrl: (session) async {
        didOpenCheckout = true;
      },
    );
    await tester.pumpAndSettle();

    await _completeCheckoutConfirmationFromSavedSeal(tester);
    await tester.pumpAndSettle();

    expect(didOpenCheckout, isFalse);
    expect(find.text('Payment failed'), findsOneWidget);
    expect(
      find.text(
        'Stripe Checkout could not be completed. You can try Checkout again.',
      ),
      findsOneWidget,
    );
    expect(find.text('Preparing Checkout'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T03 accepts order lookup input', (tester) async {
    OrderLookupRequest? submittedRequest;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          onLookup: (request) => submittedRequest = request,
        ),
      ),
    );

    await tester.tap(find.text('Lookup Order'));
    await tester.pump();

    expect(submittedRequest, isNull);

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      '  HF-20260521-0001  ',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      '  customer@example.test  ',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pump();

    expect(submittedRequest?.orderNo, 'HF-20260521-0001');
    expect(submittedRequest?.email, 'customer@example.test');
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T04 shows order lookup loading state', (tester) async {
    final lookupCompleter = Completer<OrderStatus>();
    OrderLookupRequest? submittedRequest;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          lookupOrder: (request) {
            submittedRequest = request;
            return lookupCompleter.future;
          },
        ),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-20260521-0001',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'customer@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pump();

    expect(submittedRequest?.orderNo, 'HF-20260521-0001');
    expect(submittedRequest?.email, 'customer@example.test');
    expect(find.text('Looking up your order'), findsOneWidget);
    expect(
      find.text('Checking the order number and email address.'),
      findsOneWidget,
    );

    lookupCompleter.complete(await _successfulLookupOrder(submittedRequest!));
    await tester.pumpAndSettle();

    expect(find.text('Looking up your order'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T04 shows order lookup not found state', (tester) async {
    var lookupCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          lookupOrder: (request) async {
            lookupCount++;
            throw const HankoApiException(
              statusCode: 404,
              code: 'order_not_found',
              message: 'Order not found',
              payload: {},
            );
          },
        ),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-404',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'missing@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(lookupCount, 1);
    expect(find.text('Order not found'), findsOneWidget);
    expect(
      find.text(
        "We couldn't find an order matching that order number and email address.",
      ),
      findsOneWidget,
    );
    expect(find.text('Try Again'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T04 treats lookup validation as not found', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          lookupOrder: (request) async {
            throw const HankoApiException(
              statusCode: 400,
              code: 'validation_error',
              message: 'email must be valid',
              payload: {},
            );
          },
        ),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-20260521-0001',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'invalid-email',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(find.text('Order not found'), findsOneWidget);
    expect(find.text("Couldn't load order"), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T04 shows order lookup error state', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          lookupOrder: (request) async {
            throw StateError('Lookup failed');
          },
        ),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-500',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'customer@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(find.text('Something Went Wrong'), findsOneWidget);
    expect(
      find.text(
        'An unexpected error occurred. Please try again in a few moments.',
      ),
      findsOneWidget,
    );
    expect(find.text('Try Again'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M12-T04 maps server API errors to common error state', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(
          lookupOrder: (request) async {
            throw const HankoApiException(
              statusCode: 500,
              code: 'internal',
              message: 'internal server error',
              payload: {},
            );
          },
        ),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-500',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'customer@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(find.text('Server Error'), findsOneWidget);
    expect(
      find.text(
        "We're experiencing a temporary issue on our end. Please wait a moment and try again.",
      ),
      findsOneWidget,
    );
    expect(find.text('Try Again'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T05 shows order lookup result details', (tester) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(lookupOrder: _successfulLookupOrder),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-20260521-0001',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'customer@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(find.text('Order Status'), findsOneWidget);
    expect(
      find.text("Here's the latest update on your order."),
      findsOneWidget,
    );
    expect(find.text('Stripe payment email'), findsOneWidget);
    expect(
      find.text(
        'Stripe sends the payment receipt to the email address on the order. Please check your inbox and spam folder.',
      ),
      findsOneWidget,
    );
    expect(find.text('HF-20260521-0001'), findsOneWidget);
    expect(find.text('2026-05-21 20:00'), findsOneWidget);
    expect(find.text('Paid'), findsNWidgets(2));
    expect(find.text('In production'), findsOneWidget);
    expect(find.text('Preparing shipment'), findsWidgets);
    expect(find.text('美空'), findsOneWidget);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('¥18,600'), findsOneWidget);
    expect(find.text('Yamato'), findsOneWidget);
    expect(find.text('TRACK123'), findsOneWidget);
    expect(find.text('Lookup another order'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M10-T06 shows tracking details in the lookup result', (
    tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: OrderLookupEntryScreen(lookupOrder: _successfulLookupOrder),
      ),
    );

    await tester.enterText(
      find.widgetWithText(TextFormField, 'Order No'),
      'HF-20260521-0001',
    );
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Email'),
      'customer@example.test',
    );
    await tester.pump();
    await tester.tap(find.text('Lookup Order'));
    await tester.pumpAndSettle();

    expect(find.text('Tracking details'), findsOneWidget);
    expect(find.text('Shipping status'), findsNWidgets(2));
    expect(find.text('Carrier'), findsOneWidget);
    expect(find.text('Yamato'), findsOneWidget);
    expect(find.text('Tracking number'), findsOneWidget);
    expect(find.text('TRACK123'), findsOneWidget);
    expect(find.text('Shipped at'), findsOneWidget);
    expect(find.text('2026-05-22 12:00'), findsOneWidget);
    expect(find.text('Last updated'), findsOneWidget);
    expect(find.text('2026-05-21 20:15'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-001 loads stone listings from the app shell', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    StoneListingsQuery? capturedQuery;

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async {
        capturedQuery = query;
        return _stoneListingsResult();
      },
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();

    expect(capturedQuery?.locale, 'en');
    expect(capturedQuery?.status, isNull);
    expect(find.text('Soft Pink Rose Quartz Seal Stone'), findsOneWidget);
    expect(find.text('Select Stone'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('STN-001 clears unavailable saved stone from the order draft', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    final draftRepository = InMemoryLocalOrderDraftRepository(
      OrderDraft.empty().withStoneSelection(
        const OrderDraftStoneSelection(
          listingId: 'stone_listing_001',
          code: 'RQZ-0001',
          materialKey: 'rose_quartz',
          materialLabel: 'Rose Quartz',
          sizeLabel: '24x24x60 mm',
          title: 'Soft Pink Rose Quartz Seal Stone',
          price: Money(amount: 18000, currency: 'JPY'),
          status: 'published',
          isOrderable: true,
          primaryPhotoUrl: '',
        ),
      ),
    );

    await pumpLaunchedApp(
      tester,
      listStoneListings: (query) async => _stoneListingsResult(
        listings: [_stoneListing(status: 'reserved', isOrderable: false)],
      ),
      localOrderDraftRepository: draftRepository,
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Stones').last);
    await tester.pumpAndSettle();

    expect(find.text('Unavailable'), findsWidgets);
    expect(find.text('Selected for Order'), findsNothing);
    expect(find.text('Select Stone'), findsOneWidget);

    final savedDraft = await draftRepository.loadOrderDraft();
    expect(savedDraft.stoneSelection, isNull);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-012 and DES-015 expose retry and limit actions', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var retryCount = 0;
    var backCount = 0;
    var adjustCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealGenerationErrorScreen(
          request: _sealGenerationRequest(),
          onRetry: () => retryCount += 1,
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text("We couldn't generate seal designs"), findsOneWidget);
    expect(find.text('Try Again'), findsOneWidget);
    expect(find.text('Back'), findsOneWidget);
    expect(find.text('1/3'), findsOneWidget);

    await tester.ensureVisible(find.text('Try Again'));
    await tester.pump();
    await tester.tap(find.text('Try Again'));
    await tester.pump();
    await tester.ensureVisible(find.text('Back'));
    await tester.pump();
    await tester.tap(find.text('Back'));
    await tester.pump();

    expect(retryCount, 1);
    expect(backCount, 1);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: SealGenerationLimitScreen(
          request: _sealGenerationRequest(attemptNumber: 3),
          onAdjustStyle: () => adjustCount += 1,
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text('Generation limit reached'), findsOneWidget);
    expect(find.text('3/3'), findsOneWidget);
    expect(find.text('Adjust Style'), findsOneWidget);
    expect(find.text('Back'), findsOneWidget);

    await tester.ensureVisible(find.text('Adjust Style'));
    await tester.pump();
    await tester.tap(find.text('Adjust Style'));
    await tester.pump();
    await tester.ensureVisible(find.text('Back'));
    await tester.pump();
    await tester.tap(find.text('Back'));
    await tester.pump();

    expect(adjustCount, 1);
    expect(backCount, 2);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-011 and DES-014 expose retry and edit actions', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    const request = KanjiCandidatesRequest(realName: 'Michael Smith');
    var retryCount = 0;
    var backCount = 0;
    var editCount = 0;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: KanjiSuggestionErrorScreen(
          request: request,
          onRetry: () => retryCount += 1,
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text("We couldn't suggest kanji"), findsOneWidget);
    expect(find.text('Try Again'), findsOneWidget);
    expect(find.text('Back'), findsOneWidget);

    await tester.ensureVisible(find.text('Try Again'));
    await tester.pump();
    await tester.tap(find.text('Try Again'));
    await tester.pump();
    await tester.ensureVisible(find.text('Back'));
    await tester.pump();
    await tester.tap(find.text('Back'));
    await tester.pump();

    expect(retryCount, 1);
    expect(backCount, 1);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: UnsupportedKanjiResultScreen(
          request: request,
          onRetry: () => retryCount += 1,
          onEditName: () => editCount += 1,
          onBack: () => backCount += 1,
        ),
      ),
    );

    expect(find.text("We couldn't find a suitable kanji"), findsOneWidget);
    expect(find.text('1-2 characters only'), findsOneWidget);
    expect(find.text('Simple, common kanji'), findsOneWidget);
    expect(find.text('Edit Name'), findsOneWidget);
    expect(find.text('Try Again'), findsOneWidget);

    await tester.ensureVisible(find.text('Edit Name'));
    await tester.pump();
    await tester.tap(find.text('Edit Name'));
    await tester.pump();
    await tester.ensureVisible(find.text('Try Again'));
    await tester.pump();
    await tester.tap(find.text('Try Again'));
    await tester.pump();

    expect(editCount, 1);
    expect(retryCount, 2);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-004 opens settings from the design header', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(tester);

    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();

    expect(find.byType(SettingsScreen), findsOneWidget);
    expect(find.text('Settings'), findsOneWidget);
    expect(find.text('Language'), findsOneWidget);
    expect(find.text('How It Works'), findsOneWidget);
    expect(find.text('Terms'), findsOneWidget);

    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();

    expect(find.byType(SettingsScreen), findsNothing);
    expect(find.byType(BottomNavigationShell), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-004 switches the app language from settings', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    Locale? savedLocale;

    await pumpLaunchedApp(
      tester,
      loadPreferredLocale: () async => null,
      savePreferredLocale: (locale) async => savedLocale = locale,
    );

    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Language'));
    await tester.pumpAndSettle();

    expect(find.text('App language'), findsOneWidget);
    expect(find.text('English'), findsOneWidget);
    expect(find.text('日本語'), findsOneWidget);
    expect(find.text('Japanese'), findsOneWidget);
    expect(find.text('简体中文'), findsNothing);
    expect(find.text('繁體中文'), findsNothing);

    await tester.tap(find.text('Japanese'));
    await tester.pumpAndSettle();

    expect(savedLocale?.languageCode, 'ja');
    expect(find.text('アプリの言語'), findsOneWidget);
    expect(find.text('English'), findsOneWidget);
    expect(find.text('日本語'), findsOneWidget);

    await tester.tap(find.text('English'));
    await tester.pumpAndSettle();

    expect(savedLocale?.languageCode, 'en');
    expect(find.text('App language'), findsOneWidget);
    expect(find.text('English'), findsOneWidget);
    expect(find.text('日本語'), findsOneWidget);
    expect(find.text('Japanese'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-004 restores saved locale and recovers invalid values', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await pumpLaunchedApp(
      tester,
      loadPreferredLocale: () async => const Locale('ja'),
    );

    expect(find.text('デザイン'), findsWidgets);
    expect(find.text('保存済み印影'), findsOneWidget);

    await tester.pumpWidget(const SizedBox.shrink());
    await tester.pump();

    await pumpLaunchedApp(
      tester,
      loadPreferredLocale: () async => const Locale('fr'),
    );

    expect(find.text('Design'), findsWidgets);
    expect(find.text('Saved Seals'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('switches major labels with the app locale', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      ProviderScope(
        child: HankoApp(
          locale: const Locale('ja'),
          hasSeenOnboardingResolver: () async => true,
          markOnboardingSeen: () async {},
          splashMinimumDuration: Duration.zero,
          listStoneListings: _emptyStoneListingsLoader,
        ),
      ),
    );
    await tester.pump(const Duration(milliseconds: 1));
    await tester.pump();

    expect(find.text('デザイン'), findsNWidgets(2));
    expect(find.text('あなた専用の\n印影を作成'), findsOneWidget);
    expect(find.text('作成をはじめる'), findsOneWidget);
    expect(find.text('保存済み印影'), findsOneWidget);
    expect(find.text('石を探す'), findsOneWidget);
    expect(find.text('マイ印影'), findsOneWidget);
    expect(find.text('石'), findsOneWidget);
    expect(find.text('Design'), findsNothing);

    await tester.tap(find.text('作成をはじめる'));
    await tester.pumpAndSettle();

    expect(find.text('名前を入力'), findsOneWidget);
    expect(find.text('あなたの希望に合わせた漢字を提案します。'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('renders non-tab feature entry screens independently', (
    tester,
  ) async {
    Future<void> expectEntryScreen(
      Widget screen,
      String title,
      Type expectedCommonWidget,
    ) async {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          supportedLocales: hankoSupportedLocales,
          localizationsDelegates: hankoLocalizationsDelegates,
          theme: HankoTheme.light(),
          home: screen,
        ),
      );

      expect(find.text(title), findsOneWidget);
      expect(find.byType(expectedCommonWidget), findsWidgets);
      expect(tester.takeException(), isNull);
    }

    await expectEntryScreen(
      const OrderFlowEntryScreen(),
      'Order',
      HankoStateView,
    );
    await expectEntryScreen(
      const OrderLookupEntryScreen(),
      'Order Lookup',
      HankoTextField,
    );
    expect(find.byType(HankoTextField), findsNWidgets(2));
    await expectEntryScreen(
      const SettingsHomeScreen(),
      'Settings',
      HankoSurfaceCard,
    );
  });

  testWidgets('M12-T06 renders maintenance and app update screens', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    var updateCount = 0;

    Future<void> pumpAvailabilityScreen(Widget screen) async {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          supportedLocales: hankoSupportedLocales,
          localizationsDelegates: hankoLocalizationsDelegates,
          theme: HankoTheme.light(),
          home: screen,
        ),
      );
      await tester.pump();
    }

    await pumpAvailabilityScreen(const MaintenanceScreen());
    expect(find.text('Temporarily Unavailable'), findsOneWidget);
    expect(
      find.text(
        'Stone Signature is currently undergoing maintenance. Please check back in a little while.',
      ),
      findsOneWidget,
    );
    expect(find.byType(HankoSurfaceCard), findsOneWidget);

    await pumpAvailabilityScreen(
      AppUpdateRequiredScreen(onUpdate: () => updateCount += 1),
    );
    expect(find.text('Update Required'), findsOneWidget);
    expect(
      find.text(
        'A newer app version is required to continue. Please update the app, then open Stone Signature again.',
      ),
      findsOneWidget,
    );
    await tester.tap(find.text('Update App'));
    expect(updateCount, 1);
    expect(tester.takeException(), isNull);
  });

  testWidgets('COM-004 settings rows navigate to destination screens', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: const SettingsHomeScreen(),
      ),
    );

    Future<void> openAndReturn(
      String rowLabel,
      Finder expectedFinder, {
      bool useSystemBack = false,
      List<Finder> additionalExpectedFinders = const [],
    }) async {
      await tester.ensureVisible(find.text(rowLabel));
      await tester.pump();
      await tester.tap(find.text(rowLabel));
      await tester.pumpAndSettle();

      expect(expectedFinder, findsOneWidget);
      for (final finder in additionalExpectedFinders) {
        expect(finder, findsOneWidget);
      }

      if (useSystemBack) {
        await tester.binding.handlePopRoute();
      } else {
        await tester.tap(find.byTooltip('Back'));
      }
      await tester.pumpAndSettle();

      expect(find.text('Settings'), findsOneWidget);
    }

    await openAndReturn(
      'Language',
      find.text('App language'),
      useSystemBack: true,
    );
    await openAndReturn('About', find.text('Your seal, made from gemstone'));
    await openAndReturn(
      'How It Works',
      find.text('Choose your name and kanji'),
    );
    await openAndReturn('FAQ', find.text('How is kanji selected?'));
    await openAndReturn(
      'Privacy',
      find.textContaining('https://finitefield.org/en/privacy/'),
    );
    await openAndReturn('Terms', find.text('Orders and contract formation'));
    await openAndReturn(
      'Contact',
      find.textContaining('https://finitefield.org/en/contact/'),
      additionalExpectedFinders: [find.text('dev@finitefield.org')],
    );
    await openAndReturn('Version', find.text('Version 1.0.4+10'));

    expect(tester.takeException(), isNull);
  });

  testWidgets('localizes non-tab feature entry screens', (tester) async {
    Future<void> pumpLocalizedEntry(Widget screen) async {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('ja'),
          supportedLocales: hankoSupportedLocales,
          localizationsDelegates: hankoLocalizationsDelegates,
          theme: HankoTheme.light(),
          home: screen,
        ),
      );
    }

    await pumpLocalizedEntry(const OrderLookupEntryScreen());

    expect(find.text('注文照会'), findsOneWidget);
    expect(find.text('注文番号'), findsOneWidget);
    expect(find.text('メールアドレス'), findsOneWidget);
    expect(find.text('注文を照会'), findsOneWidget);

    await pumpLocalizedEntry(const SettingsHomeScreen());

    expect(find.text('設定'), findsOneWidget);
    expect(find.text('言語'), findsOneWidget);
    expect(find.text('使い方'), findsOneWidget);
    expect(find.text('利用規約'), findsOneWidget);

    await tester.tap(find.text('このアプリについて'));
    await tester.pumpAndSettle();

    expect(find.text('宝石でつくる、あなたの印鑑'), findsOneWidget);

    expect(tester.takeException(), isNull);
  });

  testWidgets('M13-T02 renders DES-006 style selection controls', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    SealStyleSelection? generatedSelection;

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: Scaffold(
          body: SealStyleSelectionScreen(
            candidate: const KanjiCandidate(
              kanji: '美空',
              reading: 'Misora',
              meaning: 'Beautiful sky',
              reason: 'A graceful two-character option.',
            ),
            onBack: () {},
            onGenerate: (selection) {
              generatedSelection = selection;
            },
          ),
        ),
      ),
    );

    expect(find.byType(SealStyleSelectionScreen), findsOneWidget);
    expect(find.text('Seal Style'), findsOneWidget);
    expect(find.text('Customize your seal style.'), findsOneWidget);
    expect(find.text('Selected kanji'), findsOneWidget);
    expect(find.text('Shape'), findsWidgets);
    expect(find.text('Square'), findsWidgets);
    expect(find.text('Round'), findsOneWidget);
    expect(find.text('Style'), findsWidgets);
    expect(find.text('Traditional'), findsOneWidget);
    expect(find.text('Elegant'), findsWidgets);
    expect(find.text('Soft'), findsOneWidget);
    expect(find.text('Stroke Weight'), findsWidgets);
    expect(find.text('Standard'), findsWidgets);
    expect(find.text('Balance'), findsWidgets);
    expect(find.text('Balanced'), findsWidgets);

    await tester.tap(find.text('Round'));
    await tester.pump();
    await tester.tap(find.text('Traditional'));
    await tester.pump();
    await tester.ensureVisible(find.text('Airy'));
    await tester.pump();
    await tester.tap(find.text('Airy'));
    await tester.pump();
    await tester.ensureVisible(find.text('Confirm Style'));
    await tester.pump();
    await tester.tap(find.text('Confirm Style'));
    await tester.pumpAndSettle();

    expect(find.text('Style selected'), findsOneWidget);
    expect(find.text('Generate Seal'), findsOneWidget);

    await tester.ensureVisible(find.text('Generate Seal'));
    await tester.pump();
    await tester.tap(find.text('Generate Seal'));
    await tester.pump();

    expect(generatedSelection?.shape, SealShape.round);
    expect(generatedSelection?.style, SealStyleName.traditional);
    expect(generatedSelection?.strokeWeight, SealStrokeWeight.standard);
    expect(generatedSelection?.balance, SealBalance.airy);
    expect(tester.takeException(), isNull);
  });

  testWidgets('DES-006 localizes style customization message in Japanese', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('ja'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: Scaffold(
          body: SealStyleSelectionScreen(
            candidate: const KanjiCandidate(
              kanji: '美空',
              reading: 'Misora',
              meaning: '美しい空',
              reason: '穏やかな印象です。',
            ),
            onBack: () {},
          ),
        ),
      ),
    );

    expect(find.text('印影スタイル'), findsOneWidget);
    expect(find.text('印影スタイルをカスタマイズしてください。'), findsOneWidget);
    expect(find.textContaining('固定スタイル'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M13-T02 renders required common error states', (tester) async {
    var retryCount = 0;

    Future<void> pumpErrorState(Object error) async {
      await tester.pumpWidget(
        MaterialApp(
          locale: const Locale('en'),
          supportedLocales: hankoSupportedLocales,
          localizationsDelegates: hankoLocalizationsDelegates,
          theme: HankoTheme.light(),
          home: Scaffold(
            body: HankoErrorStateView(
              error: error,
              actionLabel: 'Try Again',
              onAction: () {
                retryCount += 1;
              },
            ),
          ),
        ),
      );
    }

    await pumpErrorState(const SocketException('offline'));

    expect(find.text('Network Error'), findsOneWidget);
    expect(
      find.text(
        "We're unable to connect to the server. Please check your internet connection and try again.",
      ),
      findsOneWidget,
    );
    await tester.tap(find.text('Try Again'));
    expect(retryCount, 1);

    await pumpErrorState(StateError('unexpected'));

    expect(find.text('Something Went Wrong'), findsOneWidget);
    expect(
      find.text(
        'An unexpected error occurred. Please try again in a few moments.',
      ),
      findsOneWidget,
    );
    await tester.tap(find.text('Try Again'));
    expect(retryCount, 2);
    expect(tester.takeException(), isNull);
  });

  testWidgets('M13-T08 exposes navigation and state semantics', (tester) async {
    tester.view.physicalSize = const Size(432, 912);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    void expectSemanticsWidget({
      required String label,
      bool? button,
      bool? selected,
      bool? scopesRoute,
      bool? namesRoute,
    }) {
      expect(
        find.byWidgetPredicate((widget) {
          if (widget is! Semantics) {
            return false;
          }
          final properties = widget.properties;
          return properties.label == label &&
              (button == null || properties.button == button) &&
              (selected == null || properties.selected == selected) &&
              (scopesRoute == null || properties.scopesRoute == scopesRoute) &&
              (namesRoute == null || properties.namesRoute == namesRoute);
        }),
        findsAtLeastNWidgets(1),
      );
    }

    void expectHeaderSemantics(String label) {
      expect(
        find.byWidgetPredicate((widget) {
          if (widget is! Semantics || widget.properties.header != true) {
            return false;
          }
          final child = widget.child;
          return child is Text && child.data == label;
        }),
        findsAtLeastNWidgets(1),
      );
    }

    await pumpLaunchedApp(tester);
    await tester.pumpAndSettle();

    expectSemanticsWidget(label: 'Design', scopesRoute: true, namesRoute: true);
    expectSemanticsWidget(label: 'Design', button: true, selected: true);
    expectSemanticsWidget(label: 'My Seals', button: true);
    expectSemanticsWidget(label: 'Stones', button: true);

    await tester.tap(find.text('My Seals').last);
    await tester.pumpAndSettle();

    expectSemanticsWidget(
      label: 'My Seals',
      scopesRoute: true,
      namesRoute: true,
    );
    expectHeaderSemantics('No saved seals');

    await tester.pumpWidget(
      MaterialApp(
        locale: const Locale('en'),
        supportedLocales: hankoSupportedLocales,
        localizationsDelegates: hankoLocalizationsDelegates,
        theme: HankoTheme.light(),
        home: const Scaffold(
          body: HankoStateView.loading(
            title: 'Loading stones',
            message: 'We are loading available seal stones.',
          ),
        ),
      ),
    );

    expectHeaderSemantics('Loading stones');
    expect(
      find.byWidgetPredicate(
        (widget) => widget is Semantics && widget.properties.liveRegion == true,
      ),
      findsAtLeastNWidgets(1),
    );
  });

  testWidgets(
    'M13-T08 checkout form moves focus from keyboard and error summary',
    (tester) async {
      tester.view.physicalSize = const Size(432, 912);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      final semanticsHandle = tester.ensureSemantics();
      try {
        await tester.pumpWidget(
          MaterialApp(
            locale: const Locale('en'),
            supportedLocales: hankoSupportedLocales,
            localizationsDelegates: hankoLocalizationsDelegates,
            theme: HankoTheme.light(),
            home: Scaffold(body: CheckoutInputScreen(onSave: (_) async {})),
          ),
        );
        await tester.pump();

        Finder editableFor(String key) => find.descendant(
          of: find.byKey(Key(key)),
          matching: find.byType(EditableText),
        );

        bool isFocused(String key) {
          final editable = tester.widget<EditableText>(editableFor(key));
          return editable.focusNode.hasFocus;
        }

        await tester.tap(editableFor('checkout-email-field'));
        await tester.pump();
        expect(isFocused('checkout-email-field'), isTrue);

        await tester.testTextInput.receiveAction(TextInputAction.next);
        await tester.pump();
        expect(isFocused('checkout-full-name-field'), isTrue);

        await tester.testTextInput.receiveAction(TextInputAction.next);
        await tester.pump();
        expect(isFocused('checkout-phone-field'), isTrue);

        await tester.ensureVisible(find.text('Save Checkout Information'));
        await tester.pump();
        await tester.tap(find.text('Save Checkout Information'));
        await tester.pumpAndSettle();

        final emailErrorNode = tester.getSemantics(
          find.text('Email: Please enter a valid email address.'),
        );
        expect(
          emailErrorNode.getSemanticsData().hasAction(SemanticsAction.tap),
          isTrue,
        );

        await tester.tap(
          find.text('Email: Please enter a valid email address.'),
        );
        await tester.pumpAndSettle();

        expect(isFocused('checkout-email-field'), isTrue);
      } finally {
        semanticsHandle.dispose();
      }
    },
  );
}

Future<KanjiCandidatesResult> _successfulKanjiGenerator(
  KanjiCandidatesRequest request,
) async {
  return _kanjiResult(request);
}

Future<StoneListingsResult> _emptyStoneListingsLoader(
  StoneListingsQuery query,
) async {
  return StoneListingsResult(
    locale: query.locale ?? 'en',
    currency: 'JPY',
    listings: const [],
  );
}

Future<StoneListing> _successfulStoneDetailLoader(
  StoneListingDetailQuery query,
) async {
  return _stoneListing(id: query.listingId);
}

Future<CreatedOrder> _successfulCreateOrder(SealOrderDraft draft) async {
  return const CreatedOrder(
    orderId: 'ord_001',
    orderNo: 'HF-20260521-0001',
    status: 'pending_payment',
    paymentStatus: 'unpaid',
    fulfillmentStatus: 'pending',
    pricing: Money(amount: 18600, currency: 'JPY'),
    idempotentReplay: false,
  );
}

Future<CheckoutSession> _successfulCreateCheckoutSession(
  CheckoutSessionRequest request,
) async {
  return CheckoutSession(
    orderId: request.orderId,
    sessionId: 'cs_test_001',
    checkoutUrl: 'https://checkout.stripe.test/session',
    paymentIntentId: 'pi_test_001',
  );
}

Future<void> _successfulOpenCheckoutUrl(CheckoutSession session) async {}

Future<OrderStatus> _successfulFetchOrderStatus(String orderId) async {
  return OrderStatus(
    orderId: orderId,
    orderNo: 'HF-20260521-0001',
    orderStatus: 'paid',
    paymentStatus: 'paid',
    fulfillmentStatus: 'pending',
    productionStatus: 'not_started',
    shippingStatus: 'not_shipped',
    pricing: const Money(amount: 18600, currency: 'JPY'),
  );
}

Future<OrderStatus> _successfulLookupOrder(OrderLookupRequest request) async {
  return OrderStatus(
    orderId: 'ord_lookup_001',
    orderNo: request.orderNo,
    orderStatus: 'paid',
    paymentStatus: 'paid',
    fulfillmentStatus: 'pending',
    productionStatus: 'in_production',
    shippingStatus: 'preparing_shipment',
    pricing: const Money(amount: 18600, currency: 'JPY'),
    createdAt: DateTime(2026, 5, 21, 20),
    updatedAt: DateTime(2026, 5, 21, 20, 15),
    trackingNumber: 'TRACK123',
    fulfillmentCarrier: 'Yamato',
    shippedAt: DateTime(2026, 5, 22, 12),
    sealText: '美空',
    sealPreviewImageUrl: 'https://example.test/seal.png',
    listingId: 'stone_listing_001',
    listingTitle: 'Soft Pink Rose Quartz Seal Stone',
  );
}

class _FailingSaveLocalSealDesignRepository
    implements LocalSealDesignRepository {
  @override
  Future<List<LocalSealDesign>> listLocalSealDesigns() async => const [];

  @override
  Future<LocalSealDesign?> getLocalSealDesign(String id) async => null;

  @override
  Future<void> saveLocalSealDesign(LocalSealDesign design) async {
    throw const FileSystemException('permission denied');
  }

  @override
  Future<void> deleteLocalSealDesign(String id) async {}
}

Future<void> _openGeneratedSealVariantSelection(WidgetTester tester) async {
  await tester.ensureVisible(find.text('Start Designing'));
  await tester.pump();
  await tester.tap(find.text('Start Designing'));
  await tester.pumpAndSettle();

  await tester.enterText(find.byType(TextFormField).first, 'Michael Smith');
  await tester.pump();
  await tester.ensureVisible(find.text('Suggest Kanji'));
  await tester.pump();
  await tester.tap(find.text('Suggest Kanji'));
  await tester.pumpAndSettle();

  await tester.tap(find.text('美空'));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Select Kanji'));
  await tester.pump();
  await tester.tap(find.text('Select Kanji'));
  await tester.pumpAndSettle();

  await tester.ensureVisible(find.text('Confirm Style'));
  await tester.pump();
  await tester.tap(find.text('Confirm Style'));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Generate Seal'));
  await tester.pump();
  await tester.tap(find.text('Generate Seal'));
  await tester.pumpAndSettle();
}

Future<void> _openGeneratedSealPreview(WidgetTester tester) async {
  await _openGeneratedSealVariantSelection(tester);
  await tester.ensureVisible(find.text('Soft spacing'));
  await tester.pump();
  await tester.tap(find.text('Soft spacing'));
  await tester.pumpAndSettle();
}

Future<void> _completeCheckoutConfirmationFromSavedSeal(
  WidgetTester tester,
) async {
  await tester.tap(find.text('My Seals').last);
  await tester.pumpAndSettle();
  await tester.tap(find.text('View Details'));
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Choose for Order'));
  await tester.pump();
  await tester.tap(find.text('Choose for Order'));
  await tester.pumpAndSettle();

  await tester.ensureVisible(find.text('Choose a Stone').last);
  await tester.pump();
  await tester.tap(find.text('Choose a Stone').last);
  await tester.pumpAndSettle();
  await tester.ensureVisible(find.text('Select Stone'));
  await tester.pump();
  await tester.tap(find.text('Select Stone'));
  await tester.pumpAndSettle();
  await tester.tap(find.byKey(const Key('stone-selection-confirm')));
  await tester.pumpAndSettle();

  await tester.ensureVisible(find.text('Continue to Shipping'));
  await tester.pump();
  await tester.tap(find.text('Continue to Shipping'));
  await tester.pumpAndSettle();

  Future<void> enterCheckoutField(String key, String text) async {
    final field = find.byKey(Key(key));
    await tester.ensureVisible(field);
    await tester.pump();
    await tester.enterText(
      find.descendant(of: field, matching: find.byType(EditableText)),
      text,
    );
    await tester.pump();
  }

  await enterCheckoutField('checkout-email-field', 'customer@example.test');
  await enterCheckoutField('checkout-full-name-field', 'Michael Smith');
  await enterCheckoutField('checkout-phone-field', '+1 555 0100');
  await enterCheckoutField('checkout-postal-code-field', '10001');
  await enterCheckoutField(
    'checkout-address-line1-field',
    '123 Example Street',
  );
  await enterCheckoutField('checkout-city-field', 'New York');
  await enterCheckoutField('checkout-state-field', 'NY');

  await tester.ensureVisible(find.text('Save Checkout Information'));
  await tester.pump();
  await tester.tap(find.text('Save Checkout Information'));
  await tester.pumpAndSettle();

  await tester.ensureVisible(
    find.byKey(const Key('order-confirm-kanji-design-checkbox')),
  );
  await tester.pump();
  await tester.tap(
    find.byKey(const Key('order-confirm-kanji-design-checkbox')),
  );
  await tester.pumpAndSettle();
  await tester.ensureVisible(
    find.byKey(const Key('order-confirm-custom-made-checkbox')),
  );
  await tester.pump();
  await tester.tap(find.byKey(const Key('order-confirm-custom-made-checkbox')));
  await tester.pumpAndSettle();

  await tester.ensureVisible(find.text('Proceed to Secure Payment'));
  await tester.pump();
  await tester.tap(find.text('Proceed to Secure Payment'));
}

StoneListingsResult _stoneListingsResult({List<StoneListing>? listings}) {
  return StoneListingsResult(
    locale: 'en',
    currency: 'JPY',
    listings: listings ?? [_stoneListing()],
  );
}

List<String> _stoneTitleOrder(WidgetTester tester, List<String> titles) {
  final titleSet = titles.toSet();
  return tester
      .widgetList<Text>(find.byType(Text))
      .map((widget) => widget.data)
      .whereType<String>()
      .where(titleSet.contains)
      .toList(growable: false);
}

StoneListing _stoneListing({
  String id = 'stone_listing_001',
  String title = 'Soft Pink Rose Quartz Seal Stone',
  String description = 'A soft pink rose quartz seal stone.',
  String story = 'A one-of-a-kind piece.',
  String materialKey = 'rose_quartz',
  String materialLabel = 'Rose Quartz',
  String colorFamily = 'pink',
  String patternPrimary = 'plain',
  String status = 'published',
  bool isActive = true,
  bool? isOrderable,
  int priceAmount = 18000,
  int sortOrder = 0,
  List<StoneListingPhoto> photos = const [],
}) {
  return StoneListing(
    id: id,
    code: 'RQZ-0001',
    materialKey: materialKey,
    materialLabel: materialLabel,
    sizeLabel: '24x24x60 mm',
    title: title,
    description: description,
    story: story,
    facets: StoneListingFacets(
      colorFamily: colorFamily,
      colorTags: [colorFamily],
      patternPrimary: patternPrimary,
      patternTags: [patternPrimary],
      stoneShape: 'square',
      translucency: 'semi_translucent',
    ),
    price: Money(amount: priceAmount, currency: 'JPY'),
    status: status,
    isActive: isActive,
    isOrderable: isOrderable,
    sortOrder: sortOrder,
    photos: photos,
  );
}

List<StoneListingPhoto> _stonePhotos() {
  return const [
    StoneListingPhoto(
      assetId: 'stone_photo_001',
      assetUrl: '',
      alt: 'Front view',
      isPrimary: true,
      sortOrder: 1,
    ),
    StoneListingPhoto(
      assetId: 'stone_photo_002',
      assetUrl: '',
      alt: 'Side view',
      isPrimary: false,
      sortOrder: 2,
    ),
    StoneListingPhoto(
      assetId: 'stone_photo_003',
      assetUrl: '',
      alt: 'Texture detail',
      isPrimary: false,
      sortOrder: 3,
    ),
  ];
}

LocalSealDesign _localSealDesign({
  String id = 'local_seal_001',
  String selectedKanji = '美空',
  String? meaning = 'Beautiful sky',
  String shape = 'square',
  String style = 'elegant',
  String strokeWeight = 'standard',
  String balance = 'balanced',
  bool isFavorite = false,
}) {
  return LocalSealDesign(
    id: id,
    inputName: 'Michael Smith',
    selectedKanji: selectedKanji,
    reading: 'Misora',
    meaning: meaning,
    impression: const ['Elegant', 'Gentle'],
    characterCount: selectedKanji.runes.length,
    strokeComplexity: 'medium',
    engravingSuitability: 'high',
    shape: shape,
    style: style,
    strokeWeight: strokeWeight,
    balance: balance,
    aiGenerationId: 'seal_request_001',
    aiVariantId: 'seal_variant_001',
    previewImageStoragePath:
        'seal_designs/seal_request_001/seal_variant_001.png',
    previewImageDownloadUrl: '',
    localImagePath: '',
    isFavorite: isFavorite,
    createdAt: DateTime(2026, 5, 21, 11),
    updatedAt: DateTime(2026, 5, 21, 11, 10),
  );
}

OrderDraftSealSelection _orderDraftSealSelection() {
  return const OrderDraftSealSelection(
    localSealDesignId: 'local_seal_001',
    selectedKanji: '美空',
    reading: 'Misora',
    shape: 'square',
    style: 'elegant',
    strokeWeight: 'standard',
    balance: 'balanced',
    aiGenerationId: 'seal_request_001',
    aiVariantId: 'seal_variant_001',
    previewImageStoragePath:
        'seal_designs/seal_request_001/seal_variant_001.png',
    previewImageDownloadUrl: '',
    localImagePath: '',
  );
}

OrderDraftStoneSelection _orderDraftStoneSelection({
  Money price = const Money(amount: 18000, currency: 'JPY'),
}) {
  return OrderDraftStoneSelection(
    listingId: 'stone_listing_001',
    code: 'RQZ-0001',
    materialKey: 'rose_quartz',
    materialLabel: 'Rose Quartz',
    sizeLabel: '24x24x60 mm',
    title: 'Soft Pink Rose Quartz Seal Stone',
    price: price,
    status: 'published',
    isOrderable: true,
    primaryPhotoUrl: 'https://example.test/1.png',
  );
}

OrderDraftInput _checkoutInput() {
  return const OrderDraftInput(
    contact: OrderDraftContactInput(
      email: 'old@example.test',
      preferredLocale: 'en',
    ),
    shipping: OrderDraftShippingInput(
      countryCode: 'US',
      recipientName: 'Old Recipient',
      phone: '+1 555 0000',
      postalCode: '99999',
      state: 'CA',
      city: 'Old City',
      addressLine1: '999 Expired Street',
      addressLine2: '',
    ),
    orderNote: 'Old order note',
    termsAgreed: true,
    customerConfirmation: OrderDraftCustomerConfirmationInput(
      kanjiAndDesign: true,
      customMadePolicy: true,
    ),
  );
}

SealGenerationRequest _sealGenerationRequest({int attemptNumber = 1}) {
  return SealGenerationRequest(
    inputName: 'Michael Smith',
    candidate: const KanjiCandidate(
      kanji: '美空',
      reading: 'Misora',
      meaning: 'Beautiful sky',
      reason: 'A graceful two-character option.',
    ),
    style: const SealStyleSelection(),
    attemptNumber: attemptNumber,
  );
}

String _sealVariantStoragePath(String variantId) {
  return 'seal_designs/seal_request_001/$variantId.png';
}

String _sealVariantDownloadUrl(String variantId) {
  return 'https://storage.example.test/seal_request_001/$variantId.png';
}

SealGenerationResult _sealGenerationResult({
  SealGenerationRequest? request,
  bool includeDownloadUrls = false,
}) {
  String downloadUrl(String variantId) {
    return includeDownloadUrls ? _sealVariantDownloadUrl(variantId) : '';
  }

  return SealGenerationResult(
    request: request ?? _sealGenerationRequest(),
    requestId: 'seal_request_001',
    variants: [
      SealDesignVariant(
        id: 'seal_variant_001',
        storagePath: _sealVariantStoragePath('seal_variant_001'),
        downloadUrl: downloadUrl('seal_variant_001'),
        label: 'Elegant and balanced',
        width: 1024,
        height: 1024,
      ),
      SealDesignVariant(
        id: 'seal_variant_002',
        storagePath: _sealVariantStoragePath('seal_variant_002'),
        downloadUrl: downloadUrl('seal_variant_002'),
        label: 'Soft spacing',
        width: 1024,
        height: 1024,
      ),
      SealDesignVariant(
        id: 'seal_variant_003',
        storagePath: _sealVariantStoragePath('seal_variant_003'),
        downloadUrl: downloadUrl('seal_variant_003'),
        label: 'Bold readable seal',
        width: 1024,
        height: 1024,
      ),
    ],
  );
}

KanjiCandidatesResult _kanjiResult(KanjiCandidatesRequest request) {
  return KanjiCandidatesResult(
    realName: request.realName,
    reasonLanguage: request.reasonLanguage,
    gender: request.gender,
    kanjiStyle: request.kanjiStyle,
    candidates: const [
      KanjiCandidate(
        kanji: '美空',
        reading: 'Misora',
        meaning: 'Beautiful sky',
        impression: ['Elegant', 'Gentle'],
        reason: 'A graceful two-character option.',
        characterCount: 2,
        strokeComplexity: 'medium',
        engravingSuitability: 'high',
      ),
    ],
  );
}
