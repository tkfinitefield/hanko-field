import 'dart:async';

import 'package:declarative_nav/declarative_nav.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import '../core/api/core_api.dart';
import '../core/errors/core_errors.dart';
import '../features/common/common.dart';
import '../features/design/design.dart';
import '../features/my_seals/my_seals.dart';
import '../features/order/order.dart';
import '../features/order_lookup/order_lookup.dart';
import '../features/settings/settings.dart';
import '../features/stones/stones.dart';
import 'localization/app_localization.dart';
import 'localization/language_registry.dart';
import 'navigation/app_navigation_shell.dart';
import 'theme/app_theme.dart';

class HankoApp extends StatefulWidget {
  const HankoApp({
    super.key,
    this.locale,
    this.loadPreferredLocale = _defaultLoadPreferredLocale,
    this.savePreferredLocale = _defaultSavePreferredLocale,
    this.hasSeenOnboardingResolver = _defaultHasSeenOnboardingResolver,
    this.markOnboardingSeen = _defaultMarkOnboardingSeen,
    this.splashMinimumDuration = const Duration(milliseconds: 700),
    this.generateKanjiCandidates = generateKanjiCandidatesWithDefaultApi,
    this.generateSealDesigns = generateSealDesignsWithDefaultApi,
    this.listStoneListings = listStoneListingsWithDefaultApi,
    this.getStoneListingDetail = getStoneListingDetailWithDefaultApi,
    this.createOrder = createOrderWithDefaultApi,
    this.createCheckoutSession = createCheckoutSessionWithDefaultApi,
    this.openCheckoutUrl = openCheckoutUrlWithDefaultLauncher,
    this.fetchOrderStatus = fetchOrderStatusWithDefaultApi,
    this.lookupOrder = lookupOrderWithDefaultApi,
    this.paymentStatusRetryDelay = const Duration(seconds: 2),
    this.initialCheckoutRoute,
    this.localSealDesignRepository,
    this.localOrderDraftRepository,
  });

  final Locale? locale;
  final PreferredLocaleLoader loadPreferredLocale;
  final PreferredLocaleWriter savePreferredLocale;
  final HasSeenOnboardingResolver hasSeenOnboardingResolver;
  final OnboardingCompletionWriter markOnboardingSeen;
  final Duration splashMinimumDuration;
  final KanjiCandidatesGenerator generateKanjiCandidates;
  final SealDesignsGenerator generateSealDesigns;
  final StoneListingsLoader listStoneListings;
  final StoneListingDetailLoader getStoneListingDetail;
  final OrderCreator createOrder;
  final CheckoutSessionCreator createCheckoutSession;
  final CheckoutUrlLauncher openCheckoutUrl;
  final OrderStatusFetcher fetchOrderStatus;
  final OrderLookupFetcher lookupOrder;
  final Duration paymentStatusRetryDelay;
  final String? initialCheckoutRoute;
  final LocalSealDesignRepository? localSealDesignRepository;
  final LocalOrderDraftRepository? localOrderDraftRepository;

  @override
  State<HankoApp> createState() => _HankoAppState();
}

class _HankoAppState extends State<HankoApp> with WidgetsBindingObserver {
  final _checkoutRouteNotifier = ValueNotifier<String?>(null);
  Locale? _preferredLocale;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    unawaited(_loadPreferredLocale());
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      final route =
          widget.initialCheckoutRoute ??
          WidgetsBinding.instance.platformDispatcher.defaultRouteName;
      _publishCheckoutReturnRoute(route);
    });
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _checkoutRouteNotifier.dispose();
    super.dispose();
  }

  @override
  Future<bool> didPushRouteInformation(
    RouteInformation routeInformation,
  ) async {
    return _publishCheckoutReturnRoute(routeInformation.uri.toString());
  }

  bool _publishCheckoutReturnRoute(String route) {
    if (parseCheckoutReturnRoute(route) == null &&
        !isMalformedCheckoutReturnRoute(route)) {
      return false;
    }
    _checkoutRouteNotifier.value = route;
    return true;
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      onGenerateTitle: (context) => context.l10n.appTitle,
      debugShowCheckedModeBanner: false,
      locale: widget.locale ?? _preferredLocale,
      supportedLocales: hankoSupportedLocales,
      localizationsDelegates: hankoLocalizationsDelegates,
      theme: HankoTheme.light(),
      home: _AppLaunchGate(
        hasSeenOnboardingResolver: widget.hasSeenOnboardingResolver,
        markOnboardingSeen: widget.markOnboardingSeen,
        splashMinimumDuration: widget.splashMinimumDuration,
        generateKanjiCandidates: widget.generateKanjiCandidates,
        generateSealDesigns: widget.generateSealDesigns,
        listStoneListings: widget.listStoneListings,
        getStoneListingDetail: widget.getStoneListingDetail,
        createOrder: widget.createOrder,
        createCheckoutSession: widget.createCheckoutSession,
        openCheckoutUrl: widget.openCheckoutUrl,
        fetchOrderStatus: widget.fetchOrderStatus,
        lookupOrder: widget.lookupOrder,
        paymentStatusRetryDelay: widget.paymentStatusRetryDelay,
        initialCheckoutRoute: widget.initialCheckoutRoute,
        checkoutRouteListenable: _checkoutRouteNotifier,
        localSealDesignRepository: widget.localSealDesignRepository,
        localOrderDraftRepository: widget.localOrderDraftRepository,
        onLocaleSelected: _selectPreferredLocale,
      ),
    );
  }

  Future<void> _loadPreferredLocale() async {
    try {
      final locale = await widget.loadPreferredLocale();
      if (!mounted || widget.locale != null) {
        return;
      }
      setState(() => _preferredLocale = _supportedLocale(locale));
    } catch (error) {
      debugPrint('failed to load preferred locale: $error');
    }
  }

  void _selectPreferredLocale(Locale locale) {
    final supportedLocale = _supportedLocale(locale);
    if (supportedLocale == null) {
      return;
    }
    if (widget.locale == null) {
      setState(() => _preferredLocale = supportedLocale);
    }
    unawaited(_savePreferredLocale(supportedLocale));
  }

  Future<void> _savePreferredLocale(Locale locale) async {
    try {
      await widget.savePreferredLocale(locale);
    } catch (error) {
      debugPrint('failed to save preferred locale: $error');
    }
  }
}

typedef PreferredLocaleLoader = Future<Locale?> Function();
typedef PreferredLocaleWriter = Future<void> Function(Locale locale);

Future<bool> _defaultHasSeenOnboardingResolver() async {
  return const AppLaunchStore().hasSeenOnboarding();
}

Future<void> _defaultMarkOnboardingSeen() {
  return const AppLaunchStore().setHasSeenOnboarding(true);
}

Future<Locale?> _defaultLoadPreferredLocale() async {
  final routeCode = await const AppLaunchStore().preferredRouteCode();
  if (routeCode == null) {
    return null;
  }
  final registry = await AppLanguageRegistry.load();
  return _supportedLocale(
    registry.enabledLanguageForRouteCode(routeCode)?.locale,
  );
}

Future<void> _defaultSavePreferredLocale(Locale locale) async {
  final supportedLocale = _supportedLocale(locale);
  if (supportedLocale == null) {
    return;
  }
  final registry = await AppLanguageRegistry.load();
  final language = registry.enabledLanguageForLocale(supportedLocale);
  if (language == null) {
    return;
  }
  await const AppLaunchStore().setPreferredRouteCode(language.routeCode);
}

Future<String> _checkoutRouteCodeForLocale(Locale locale) async {
  try {
    final registry = await AppLanguageRegistry.load();
    return registry.routeCodeForLocale(locale) ??
        _fallbackCheckoutRouteCodeForLocale(locale);
  } catch (error) {
    debugPrint('failed to load checkout locale registry: $error');
    return _fallbackCheckoutRouteCodeForLocale(locale);
  }
}

String _fallbackCheckoutRouteCodeForLocale(Locale locale) {
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

Locale? _supportedLocale(Locale? locale) {
  if (locale == null) {
    return null;
  }
  for (final supportedLocale in hankoSupportedLocales) {
    if (supportedLocale.languageCode == locale.languageCode &&
        supportedLocale.scriptCode == locale.scriptCode &&
        supportedLocale.countryCode == locale.countryCode) {
      return supportedLocale;
    }
  }
  return null;
}

enum _AppLaunchStage { splash, onboarding, shell }

class _AppLaunchGate extends StatefulWidget {
  const _AppLaunchGate({
    required this.hasSeenOnboardingResolver,
    required this.markOnboardingSeen,
    required this.splashMinimumDuration,
    required this.generateKanjiCandidates,
    required this.generateSealDesigns,
    required this.listStoneListings,
    required this.getStoneListingDetail,
    required this.createOrder,
    required this.createCheckoutSession,
    required this.openCheckoutUrl,
    required this.fetchOrderStatus,
    required this.lookupOrder,
    required this.paymentStatusRetryDelay,
    required this.initialCheckoutRoute,
    required this.checkoutRouteListenable,
    required this.localSealDesignRepository,
    required this.localOrderDraftRepository,
    required this.onLocaleSelected,
  });

  final HasSeenOnboardingResolver hasSeenOnboardingResolver;
  final OnboardingCompletionWriter markOnboardingSeen;
  final Duration splashMinimumDuration;
  final KanjiCandidatesGenerator generateKanjiCandidates;
  final SealDesignsGenerator generateSealDesigns;
  final StoneListingsLoader listStoneListings;
  final StoneListingDetailLoader getStoneListingDetail;
  final OrderCreator createOrder;
  final CheckoutSessionCreator createCheckoutSession;
  final CheckoutUrlLauncher openCheckoutUrl;
  final OrderStatusFetcher fetchOrderStatus;
  final OrderLookupFetcher lookupOrder;
  final Duration paymentStatusRetryDelay;
  final String? initialCheckoutRoute;
  final ValueListenable<String?> checkoutRouteListenable;
  final LocalSealDesignRepository? localSealDesignRepository;
  final LocalOrderDraftRepository? localOrderDraftRepository;
  final ValueChanged<Locale> onLocaleSelected;

  @override
  State<_AppLaunchGate> createState() => _AppLaunchGateState();
}

class _AppLaunchGateState extends State<_AppLaunchGate> {
  var _stage = _AppLaunchStage.splash;

  @override
  Widget build(BuildContext context) {
    return switch (_stage) {
      _AppLaunchStage.splash => SplashScreen(
        hasSeenOnboardingResolver: widget.hasSeenOnboardingResolver,
        minimumDisplayDuration: widget.splashMinimumDuration,
        onLaunchResolved: _showLaunchDestination,
      ),
      _AppLaunchStage.onboarding => OnboardingScreen(
        onComplete: _completeOnboarding,
      ),
      _AppLaunchStage.shell => BottomNavigationShell(
        generateKanjiCandidates: widget.generateKanjiCandidates,
        generateSealDesigns: widget.generateSealDesigns,
        listStoneListings: widget.listStoneListings,
        getStoneListingDetail: widget.getStoneListingDetail,
        createOrder: widget.createOrder,
        createCheckoutSession: widget.createCheckoutSession,
        openCheckoutUrl: widget.openCheckoutUrl,
        fetchOrderStatus: widget.fetchOrderStatus,
        lookupOrder: widget.lookupOrder,
        paymentStatusRetryDelay: widget.paymentStatusRetryDelay,
        initialCheckoutRoute: widget.initialCheckoutRoute,
        checkoutRouteListenable: widget.checkoutRouteListenable,
        localSealDesignRepository: widget.localSealDesignRepository,
        localOrderDraftRepository: widget.localOrderDraftRepository,
        onLocaleSelected: widget.onLocaleSelected,
      ),
    };
  }

  Future<void> _completeOnboarding() async {
    await widget.markOnboardingSeen();
    if (!mounted) {
      return;
    }
    setState(() => _stage = _AppLaunchStage.shell);
  }

  void _showLaunchDestination(AppLaunchDestination destination) {
    setState(() {
      _stage = switch (destination) {
        AppLaunchDestination.onboarding => _AppLaunchStage.onboarding,
        AppLaunchDestination.shell => _AppLaunchStage.shell,
      };
    });
  }
}

class BottomNavigationShell extends StatefulWidget {
  const BottomNavigationShell({
    super.key,
    this.generateKanjiCandidates = generateKanjiCandidatesWithDefaultApi,
    this.generateSealDesigns = generateSealDesignsWithDefaultApi,
    this.listStoneListings = listStoneListingsWithDefaultApi,
    this.getStoneListingDetail = getStoneListingDetailWithDefaultApi,
    this.createOrder = createOrderWithDefaultApi,
    this.createCheckoutSession = createCheckoutSessionWithDefaultApi,
    this.openCheckoutUrl = openCheckoutUrlWithDefaultLauncher,
    this.fetchOrderStatus = fetchOrderStatusWithDefaultApi,
    this.lookupOrder = lookupOrderWithDefaultApi,
    this.paymentStatusRetryDelay = const Duration(seconds: 2),
    this.initialCheckoutRoute,
    this.checkoutRouteListenable,
    this.localSealDesignRepository,
    this.localOrderDraftRepository,
    required this.onLocaleSelected,
  });

  final KanjiCandidatesGenerator generateKanjiCandidates;
  final SealDesignsGenerator generateSealDesigns;
  final StoneListingsLoader listStoneListings;
  final StoneListingDetailLoader getStoneListingDetail;
  final OrderCreator createOrder;
  final CheckoutSessionCreator createCheckoutSession;
  final CheckoutUrlLauncher openCheckoutUrl;
  final OrderStatusFetcher fetchOrderStatus;
  final OrderLookupFetcher lookupOrder;
  final Duration paymentStatusRetryDelay;
  final String? initialCheckoutRoute;
  final ValueListenable<String?>? checkoutRouteListenable;
  final LocalSealDesignRepository? localSealDesignRepository;
  final LocalOrderDraftRepository? localOrderDraftRepository;
  final ValueChanged<Locale> onLocaleSelected;

  @override
  State<BottomNavigationShell> createState() => _BottomNavigationShellState();
}

class _KanjiSuggestionFailure {
  const _KanjiSuggestionFailure({required this.request, required this.error});

  final KanjiCandidatesRequest request;
  final Object error;
}

class _KanjiCandidateSelection {
  const _KanjiCandidateSelection({
    required this.result,
    required this.candidate,
    this.initialStyleSelection = const SealStyleSelection(),
  });

  final KanjiCandidatesResult result;
  final KanjiCandidate candidate;
  final SealStyleSelection initialStyleSelection;
}

class _SealGenerationFailure {
  const _SealGenerationFailure({required this.request, required this.error});

  final SealGenerationRequest request;
  final Object error;
}

class _SealPreviewSelection {
  const _SealPreviewSelection({required this.result, required this.variant});

  final SealGenerationResult result;
  final SealDesignVariant variant;
}

class _SealSaveFailure {
  const _SealSaveFailure({
    required this.result,
    required this.variant,
    required this.error,
  });

  final SealGenerationResult result;
  final SealDesignVariant variant;
  final Object error;
}

class _StoneImageGallerySelection {
  const _StoneImageGallerySelection({
    required this.listing,
    required this.initialPhotoIndex,
  });

  final StoneListing listing;
  final int initialPhotoIndex;
}

class _BottomNavigationShellState extends State<BottomNavigationShell>
    with WidgetsBindingObserver {
  static const _checkoutInputRetentionDuration = Duration(hours: 24);

  static const _shellPage = PageEntry(
    key: 'COM-003-bottom-navigation-shell',
    name: '/shell',
  );
  static const _settingsPage = PageEntry(
    key: 'COM-004-settings',
    name: '/settings',
  );
  static const _orderReviewPage = PageEntry(
    key: 'CMB-001-order-combination-review',
    name: '/order/review',
  );
  static const _checkoutInputPage = PageEntry(
    key: 'CHK-001-checkout-input',
    name: '/checkout/input',
  );
  static const _orderConfirmationPage = PageEntry(
    key: 'CHK-006-order-confirmation',
    name: '/checkout/confirmation',
  );
  static const _checkoutProcessingPage = PageEntry(
    key: 'CHK-008-checkout-processing',
    name: '/checkout/processing',
  );
  static const _stripeCheckoutTransitionPage = PageEntry(
    key: 'CHK-009-stripe-checkout-transition',
    name: '/checkout/stripe',
  );
  static const _checkoutDeepLinkErrorPage = PageEntry(
    key: 'ERR-006-checkout-deep-link-error',
    name: '/checkout/deep-link-error',
  );
  static const _stripeCheckoutCanceledPage = PageEntry(
    key: 'CHK-012-payment-canceled',
    name: '/checkout/payment-canceled',
  );
  static const _stripeCheckoutFailedPage = PageEntry(
    key: 'CHK-013-payment-failed',
    name: '/checkout/payment-failed',
  );
  static const _paymentStatusPage = PageEntry(
    key: 'CHK-011-payment-status',
    name: '/checkout/payment-status',
  );
  static const _orderCompletePage = PageEntry(
    key: 'CHK-015-order-complete',
    name: '/checkout/complete',
  );
  static const _orderLookupPage = PageEntry(
    key: 'LKP-001-order-lookup-input',
    name: '/order-lookup',
  );
  static const _checkoutProcessingPages = [
    _shellPage,
    _orderReviewPage,
    _checkoutInputPage,
    _orderConfirmationPage,
    _checkoutProcessingPage,
  ];
  static const _stripeCheckoutPages = [
    _shellPage,
    _orderReviewPage,
    _checkoutInputPage,
    _orderConfirmationPage,
    _checkoutProcessingPage,
    _stripeCheckoutTransitionPage,
  ];
  static const _navigationTabs = [
    HankoTabDefinition(
      tab: HankoAppTab.design,
      rootKey: 'COM-003-design-root',
      rootName: '/design',
    ),
    HankoTabDefinition(
      tab: HankoAppTab.mySeals,
      rootKey: 'COM-003-my-seals-root',
      rootName: '/my-seals',
    ),
    HankoTabDefinition(
      tab: HankoAppTab.stones,
      rootKey: 'COM-003-stones-root',
      rootName: '/stones',
    ),
  ];
  static const _designNameInputPage = PageEntry(
    key: 'DES-002-name-input',
    name: '/design/name',
  );
  static const _designKanjiLoadingPageKey = 'DES-003-kanji-suggestion-loading';
  static const _designKanjiSuggestionsPageKey = 'DES-004-kanji-suggestions';
  static const _designKanjiCandidateDetailPageKey =
      'DES-005-kanji-candidate-detail';
  static const _designSealStyleSelectionPageKey =
      'DES-006-seal-style-selection';
  static const _designSealGenerationLoadingPageKey =
      'DES-007-seal-generation-loading';
  static const _designSealVariantSelectionPageKey =
      'DES-008-seal-variant-selection';
  static const _designSealPreviewDetailPageKey = 'DES-009-seal-preview-detail';
  static const _designSealSaveConfirmationPageKey =
      'DES-010-seal-save-confirmation';
  static const _designKanjiErrorPageKey = 'DES-011-kanji-suggestion-error';
  static const _designSealGenerationErrorPageKey =
      'DES-012-seal-generation-error';
  static const _designSealSaveErrorPageKey = 'ERR-005-seal-save-error';
  static const _designUnsupportedKanjiPageKey =
      'DES-014-unsupported-kanji-result';
  static const _designSealGenerationLimitPageKey =
      'DES-015-seal-generation-limit';
  static const _mySealsDetailPageKey = 'MYS-003-seal-detail';
  static const _stoneDetailPageKey = 'STN-007-stone-detail';
  static const _stoneImageGalleryPageKey = 'STN-008-stone-image-gallery';

  late final LocalSealDesignRepository _localSealDesignRepository;
  late final LocalOrderDraftRepository _localOrderDraftRepository;
  var _localSealDesigns = const <LocalSealDesign>[];
  var _localSealDesignsLoaded = false;
  Object? _localSealDesignsLoadError;
  var _orderDraft = OrderDraft.empty();
  OrderDraft? _completedOrderDraft;
  var _orderDraftHasLocalChanges = false;
  StoneListingsResult? _stoneListingsResult;
  var _stoneListingsLoaded = false;
  var _stoneListingsLoading = false;
  Object? _stoneListingsLoadError;
  String? _stoneListingsLocale;
  var _checkoutProcessingStep = CheckoutProcessingStep.creatingOrder;
  var _checkoutProcessingInFlight = false;
  Object? _checkoutProcessingError;
  CreatedOrder? _checkoutCreatedOrder;
  CheckoutSession? _checkoutSession;
  String? _checkoutRouteCode;
  var _stripeCheckoutStep = StripeCheckoutExternalStep.opening;
  Object? _stripeCheckoutLaunchError;
  CheckoutReturnResult? _checkoutReturnResult;
  var _paymentStatusStep = PaymentStatusStep.checking;
  OrderStatus? _paymentStatus;
  Object? _paymentStatusError;
  Object? _checkoutDeepLinkError;
  int _paymentStatusRequestId = 0;
  String? _lastHandledCheckoutRoute;
  final _savingLocalSealKeys = <String>{};
  final _deletingLocalSealIds = <String>{};
  final _updatingLocalSealFavoriteIds = <String>{};
  HankoAppTab? _requestedTab;
  List<PageEntry>? _requestedTabPages;
  var _pages = const <PageEntry>[_shellPage];

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _localSealDesignRepository =
        widget.localSealDesignRepository ?? InMemoryLocalSealDesignRepository();
    _localOrderDraftRepository =
        widget.localOrderDraftRepository ?? InMemoryLocalOrderDraftRepository();
    widget.checkoutRouteListenable?.addListener(
      _handleCheckoutRouteNotification,
    );
    unawaited(_loadLocalSealDesigns());
    unawaited(_loadOrderDraft());
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) {
        return;
      }
      final route =
          widget.initialCheckoutRoute ??
          WidgetsBinding.instance.platformDispatcher.defaultRouteName;
      unawaited(_handleIncomingCheckoutRoute(route));
    });
    _handleCheckoutRouteNotification();
  }

  @override
  void dispose() {
    widget.checkoutRouteListenable?.removeListener(
      _handleCheckoutRouteNotification,
    );
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant BottomNavigationShell oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.checkoutRouteListenable == widget.checkoutRouteListenable) {
      return;
    }
    oldWidget.checkoutRouteListenable?.removeListener(
      _handleCheckoutRouteNotification,
    );
    widget.checkoutRouteListenable?.addListener(
      _handleCheckoutRouteNotification,
    );
    _handleCheckoutRouteNotification();
  }

  @override
  Future<bool> didPushRouteInformation(RouteInformation routeInformation) {
    return _handleIncomingCheckoutRoute(routeInformation.uri.toString());
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state != AppLifecycleState.resumed) {
      return;
    }
    _beginCheckoutResumeStatusCheck();
  }

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    final locale = Localizations.localeOf(context).languageCode;
    if (_stoneListingsLocale == locale) {
      return;
    }
    _stoneListingsLocale = locale;
    unawaited(_loadStoneListings(locale: locale));
  }

  @override
  Widget build(BuildContext context) {
    return DeclarativePagesNavigator(
      pages: _pages,
      buildPage: (context, page) {
        if (page.key == _paymentStatusPage.key) {
          return _buildPaymentStatusPage();
        }
        if (page.key == _orderCompletePage.key) {
          return _buildOrderCompletePage();
        }
        if (page.key == _orderLookupPage.key) {
          return _buildOrderLookupPage();
        }
        if (page.key == _stripeCheckoutTransitionPage.key ||
            page.key == _stripeCheckoutCanceledPage.key ||
            page.key == _stripeCheckoutFailedPage.key) {
          return _buildStripeCheckoutTransitionPage();
        }
        if (page.key == _checkoutDeepLinkErrorPage.key) {
          return _buildCheckoutDeepLinkErrorPage();
        }
        if (page.key == _checkoutProcessingPage.key) {
          return _buildCheckoutProcessingPage();
        }
        if (page.key == _orderConfirmationPage.key) {
          return _buildOrderConfirmationPage();
        }
        if (page.key == _checkoutInputPage.key) {
          return _buildCheckoutInputPage();
        }
        if (page.key == _orderReviewPage.key) {
          return _buildOrderReviewPage();
        }
        if (page.key == _settingsPage.key) {
          return _buildSettingsPage(page);
        }
        return _buildShellPage(context);
      },
      onPopTop: _closeTopPage,
    );
  }

  Widget _buildShellPage(BuildContext context) {
    final l10n = context.l10n;
    final tabItems = [
      _TabItem(l10n.design, _TabIcon.design),
      _TabItem(l10n.mySeals, _TabIcon.mySeals),
      _TabItem(l10n.stones, _TabIcon.stones),
    ];

    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: HankoTabNavigationShell(
            tabs: _navigationTabs,
            selectedTab: _requestedTab,
            selectedTabPages: _requestedTabPages,
            buildPage: _buildTabPage,
            buildBottomNavigation: (context, selectedIndex, onSelected) {
              return _BottomTabs(
                selectedIndex: selectedIndex,
                tabs: tabItems,
                onSelected: onSelected,
              );
            },
          ),
        ),
      ),
    );
  }

  Widget _buildSettingsPage(PageEntry page) {
    final initialDestination = page.data is SettingsInitialDestination
        ? page.data! as SettingsInitialDestination
        : null;

    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: SettingsScreen(
            onClose: _closeTopPage,
            initialDestination: initialDestination,
            onLocaleSelected: widget.onLocaleSelected,
          ),
        ),
      ),
    );
  }

  Widget _buildOrderReviewPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: OrderFlowEntryScreen(
            draft: _orderDraft,
            onBack: _closeTopPage,
            onChooseSeal: () => _showTabFromOrder(HankoAppTab.mySeals),
            onChooseStone: () => _showTabFromOrder(HankoAppTab.stones),
            onContinueToShipping: _openCheckoutInput,
          ),
        ),
      ),
    );
  }

  Widget _buildCheckoutInputPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: CheckoutInputScreen(
            input: _orderDraft.input,
            onBack: _closeTopPage,
            onSave: _saveCheckoutInput,
            onContinueToReview: _openOrderConfirmation,
          ),
        ),
      ),
    );
  }

  Widget _buildOrderConfirmationPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: OrderConfirmationScreen(
            draft: _orderDraft,
            onBack: _closeTopPage,
            onEditCheckout: _openCheckoutInput,
            onConfirmAgreements: _confirmOrderAgreements,
          ),
        ),
      ),
    );
  }

  Widget _buildCheckoutProcessingPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: CheckoutProcessingScreen(
            step: _checkoutProcessingStep,
            createdOrder: _checkoutCreatedOrder,
            checkoutSession: _checkoutSession,
            error: _checkoutProcessingError,
            onBack: _checkoutProcessingInFlight ? null : _closeTopPage,
          ),
        ),
      ),
    );
  }

  Widget _buildStripeCheckoutTransitionPage() {
    final isOpening = _stripeCheckoutStep == StripeCheckoutExternalStep.opening;
    final canOpenCheckout = _checkoutSession != null && !isOpening;
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: StripeCheckoutTransitionScreen(
            draft: _orderDraft,
            step: _stripeCheckoutStep,
            createdOrder: _checkoutCreatedOrder,
            checkoutSession: _checkoutSession,
            returnResult: _checkoutReturnResult,
            error: _stripeCheckoutLaunchError,
            onBack: isOpening ? null : _closeTopPage,
            onOpenCheckout: canOpenCheckout
                ? () => unawaited(_openStripeCheckout(_checkoutSession!))
                : null,
          ),
        ),
      ),
    );
  }

  Widget _buildCheckoutDeepLinkErrorPage() {
    final canOpenCheckout = _checkoutSession != null;
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: CheckoutDeepLinkErrorScreen(
            error: _checkoutDeepLinkError,
            onBack: _closeTopPage,
            onOpenCheckout: canOpenCheckout
                ? () => unawaited(_openStripeCheckout(_checkoutSession!))
                : null,
          ),
        ),
      ),
    );
  }

  Widget _buildPaymentStatusPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: PaymentStatusScreen(
            draft: _orderDraft,
            step: _paymentStatusStep,
            status: _paymentStatus,
            createdOrder: _checkoutCreatedOrder,
            returnResult: _checkoutReturnResult,
            error: _paymentStatusError,
            onBack: _closeTopPage,
          ),
        ),
      ),
    );
  }

  Widget _buildOrderCompletePage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: OrderCompleteScreen(
            draft: _completedOrderDraft ?? _orderDraft,
            status: _paymentStatus,
            createdOrder: _checkoutCreatedOrder,
            onOpenOrderLookup: _openOrderLookup,
            onContactSupport: _openContactSupport,
            onBackToDesign: () => _showTabFromOrder(HankoAppTab.design),
          ),
        ),
      ),
    );
  }

  Widget _buildOrderLookupPage() {
    return Scaffold(
      backgroundColor: HankoColors.background,
      body: Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 432),
          child: OrderLookupEntryScreen(
            onBack: _closeTopPage,
            initialOrderNo:
                _paymentStatus?.orderNo ?? _checkoutCreatedOrder?.orderNo,
            initialEmail: _orderDraft.input.contact.email,
            lookupOrder: widget.lookupOrder,
          ),
        ),
      ),
    );
  }

  Widget _buildTabPage(
    BuildContext context,
    HankoAppTab tab,
    PageEntry page,
    HankoTabStackController stack,
  ) {
    return switch (tab) {
      HankoAppTab.design => _buildDesignPage(page, stack),
      HankoAppTab.mySeals => _buildMySealsPage(page, stack),
      HankoAppTab.stones => _buildStonesPage(page, stack),
    };
  }

  Widget _buildStonesPage(PageEntry page, HankoTabStackController stack) {
    final pageData = page.data;
    if (page.key == _stoneDetailPageKey && pageData is StoneListing) {
      return StoneDetailScreen(
        listing: pageData,
        locale: _stoneListingsLocale ?? _stoneListingsResult?.locale,
        loadStoneListing: widget.getStoneListingDetail,
        onOpenImageGallery: (listing, initialPhotoIndex) => stack.push(
          _stoneImageGalleryPage(
            listing: listing,
            initialPhotoIndex: initialPhotoIndex,
          ),
        ),
        isSelectedForOrder:
            _orderDraft.stoneSelection?.listingId == pageData.id,
        onSelectStone: _chooseStoneForOrder,
        onBack: stack.pop,
      );
    }
    if (page.key == _stoneImageGalleryPageKey &&
        pageData is _StoneImageGallerySelection) {
      return StoneImageGalleryScreen(
        listing: pageData.listing,
        initialPhotoIndex: pageData.initialPhotoIndex,
        onBack: stack.pop,
      );
    }

    return StonesHomeScreen(
      result: _stoneListingsResult,
      isLoading: !_stoneListingsLoaded || _stoneListingsLoading,
      loadError: _stoneListingsLoadError,
      onRetry: _retryStoneListings,
      onOpenStoneDetail: (listing) => stack.push(_stoneDetailPage(listing)),
      selectedStoneId: _orderDraft.stoneSelection?.listingId,
      onSelectStone: _chooseStoneForOrder,
    );
  }

  Widget _buildMySealsPage(PageEntry page, HankoTabStackController stack) {
    final pageData = page.data;
    if (page.key == _mySealsDetailPageKey && pageData is LocalSealDesign) {
      return SealDetailScreen(
        design: pageData,
        isSelectedForOrder:
            _orderDraft.sealSelection?.localSealDesignId == pageData.id,
        onChooseForOrder: _chooseLocalSealForOrder,
        onEditRegenerate: _openRegenerateLocalSealDesign,
        onDelete: (design) => _deleteLocalSealDesign(design, stack),
        onBack: stack.pop,
      );
    }

    return MySealsHomeScreen(
      designs: _localSealDesigns,
      isLoading: !_localSealDesignsLoaded,
      loadError: _localSealDesignsLoadError,
      onStartDesigning: () => stack.selectTab(HankoAppTab.design),
      onExploreStones: () => stack.selectTab(HankoAppTab.stones),
      onChooseSeal: (design) => stack.push(_mySealsDetailPage(design)),
      onToggleFavorite: _toggleLocalSealFavorite,
    );
  }

  Widget _buildDesignPage(PageEntry page, HankoTabStackController stack) {
    final pageData = page.data;
    if (page.key == _designNameInputPage.key) {
      return NameInputScreen(
        onBack: stack.pop,
        onSubmit: (request) {
          stack.push(_kanjiLoadingPage(request));
        },
      );
    }

    if (page.key == _designKanjiLoadingPageKey &&
        pageData is KanjiCandidatesRequest) {
      return KanjiSuggestionLoadingScreen(
        request: pageData,
        generateCandidates: widget.generateKanjiCandidates,
        onLoaded: (result) {
          if (result.candidates.isEmpty) {
            stack.replaceTop(_unsupportedKanjiPage(pageData));
            return;
          }
          stack.replaceTop(_kanjiSuggestionsPage(result));
        },
        onError: (error) {
          stack.replaceTop(_kanjiSuggestionErrorPage(pageData, error));
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designKanjiSuggestionsPageKey &&
        pageData is KanjiCandidatesResult) {
      return KanjiSuggestionsScreen(
        result: pageData,
        onOpenCandidate: (candidate) {
          stack.push(_kanjiCandidateDetailPage(pageData, candidate));
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designKanjiCandidateDetailPageKey &&
        pageData is _KanjiCandidateSelection) {
      return KanjiCandidateDetailScreen(
        candidate: pageData.candidate,
        onSelected: (candidate) {
          stack.push(_sealStyleSelectionPage(pageData));
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealStyleSelectionPageKey &&
        pageData is _KanjiCandidateSelection) {
      return SealStyleSelectionScreen(
        candidate: pageData.candidate,
        onBack: stack.pop,
        initialSelection: pageData.initialStyleSelection,
        onGenerate: (selection) {
          stack.push(
            _sealGenerationLoadingPage(
              SealGenerationRequest(
                inputName: pageData.result.realName,
                candidate: pageData.candidate,
                style: selection,
              ),
            ),
          );
        },
      );
    }

    if (page.key == _designSealGenerationLoadingPageKey &&
        pageData is SealGenerationRequest) {
      return SealGenerationLoadingScreen(
        request: pageData,
        generateSealDesigns: widget.generateSealDesigns,
        onGenerated: (result) {
          stack.replaceTop(_sealVariantSelectionPage(result));
        },
        onError: (error) {
          if (pageData.hasReachedLimit) {
            stack.replaceTop(_sealGenerationLimitPage(pageData));
            return;
          }
          stack.replaceTop(_sealGenerationErrorPage(pageData, error));
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealVariantSelectionPageKey &&
        pageData is SealGenerationResult) {
      return SealVariantSelectionScreen(
        result: pageData,
        onSelected: (variant) {
          stack.push(_sealPreviewDetailPage(pageData, variant));
        },
        onRegenerate: pageData.request.hasReachedLimit
            ? null
            : () {
                final previousRecipes = pageData.variants
                    .map((variant) => variant.recipe)
                    .whereType<SealDesignRecipe>();
                stack.replaceTop(
                  _sealGenerationLoadingPage(
                    pageData.request.nextAttempt(avoidRecipes: previousRecipes),
                  ),
                );
              },
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealPreviewDetailPageKey &&
        pageData is _SealPreviewSelection) {
      return SealPreviewDetailScreen(
        result: pageData.result,
        variant: pageData.variant,
        onSave: () {
          unawaited(
            _saveLocalSealDesign(
              result: pageData.result,
              variant: pageData.variant,
              stack: stack,
            ),
          );
        },
        onChooseStone: () => _chooseGeneratedSealForOrder(
          result: pageData.result,
          variant: pageData.variant,
          stack: stack,
        ),
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealSaveConfirmationPageKey &&
        pageData is _SealPreviewSelection) {
      return SealSaveConfirmationScreen(
        result: pageData.result,
        variant: pageData.variant,
        onOpenMySeals: () => stack.selectTab(HankoAppTab.mySeals),
        onChooseStone: () => stack.selectTab(HankoAppTab.stones),
        onCreateAnother: stack.popToRoot,
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealSaveErrorPageKey &&
        pageData is _SealSaveFailure) {
      return SealSaveErrorScreen(
        result: pageData.result,
        variant: pageData.variant,
        error: pageData.error,
        onRetry: () {
          unawaited(
            _saveLocalSealDesign(
              result: pageData.result,
              variant: pageData.variant,
              stack: stack,
              replaceTopOnComplete: true,
            ),
          );
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designKanjiErrorPageKey &&
        pageData is _KanjiSuggestionFailure) {
      return KanjiSuggestionErrorScreen(
        request: pageData.request,
        error: pageData.error,
        onRetry: () => stack.replaceTop(_kanjiLoadingPage(pageData.request)),
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealGenerationErrorPageKey &&
        pageData is _SealGenerationFailure) {
      return SealGenerationErrorScreen(
        request: pageData.request,
        error: pageData.error,
        onRetry: () {
          stack.replaceTop(
            _sealGenerationLoadingPage(pageData.request.nextAttempt()),
          );
        },
        onBack: stack.pop,
      );
    }

    if (page.key == _designUnsupportedKanjiPageKey &&
        pageData is KanjiCandidatesRequest) {
      return UnsupportedKanjiResultScreen(
        request: pageData,
        onRetry: () => stack.replaceTop(_kanjiLoadingPage(pageData)),
        onEditName: stack.pop,
        onBack: stack.pop,
      );
    }

    if (page.key == _designSealGenerationLimitPageKey &&
        pageData is SealGenerationRequest) {
      return SealGenerationLimitScreen(
        request: pageData,
        onAdjustStyle: stack.pop,
        onBack: stack.pop,
      );
    }

    return DesignHomeScreen(
      onOpenSettings: _openSettings,
      onStartDesign: () => stack.push(_designNameInputPage),
      onOpenMySeals: () => stack.selectTab(HankoAppTab.mySeals),
      onOpenStones: () => stack.selectTab(HankoAppTab.stones),
    );
  }

  Future<void> _loadLocalSealDesigns() async {
    try {
      final designs = await _localSealDesignRepository.listLocalSealDesigns();
      if (!mounted) {
        return;
      }
      setState(() {
        _localSealDesigns = List.unmodifiable(designs);
        _localSealDesignsLoaded = true;
        _localSealDesignsLoadError = null;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _localSealDesignsLoaded = true;
        _localSealDesignsLoadError = error;
      });
    }
  }

  Future<void> _loadOrderDraft() async {
    try {
      final draft = await _localOrderDraftRepository.loadOrderDraft();
      if (!mounted) {
        return;
      }
      if (_orderDraftHasLocalChanges) {
        return;
      }
      final freshDraft = _discardExpiredCheckoutInput(
        draft,
        now: DateTime.now(),
      );
      final nextDraft = _validateOrderDraftStoneSelection(
        freshDraft,
        _stoneListingsResult,
      );
      final shouldPersistDraft =
          !identical(freshDraft, draft) || !identical(nextDraft, freshDraft);
      setState(() {
        _orderDraft = nextDraft;
        if (shouldPersistDraft) {
          _orderDraftHasLocalChanges = true;
        }
      });
      if (shouldPersistDraft) {
        unawaited(_saveReconciledOrderDraft(nextDraft));
      }
    } catch (_) {
      if (!mounted) {
        return;
      }
      if (_orderDraftHasLocalChanges) {
        return;
      }
      setState(() => _orderDraft = OrderDraft.empty());
    }
  }

  Future<void> _loadStoneListings({required String locale}) async {
    if (_stoneListingsLoading) {
      return;
    }
    setState(() {
      _stoneListingsLoading = true;
      _stoneListingsLoadError = null;
    });
    try {
      final result = await widget.listStoneListings(
        StoneListingsQuery(locale: locale),
      );
      if (!mounted) {
        return;
      }
      final nextDraft = _validateOrderDraftStoneSelection(_orderDraft, result);
      final shouldPersistDraft = !identical(nextDraft, _orderDraft);
      setState(() {
        _orderDraft = nextDraft;
        if (shouldPersistDraft) {
          _orderDraftHasLocalChanges = true;
        }
        _stoneListingsResult = result;
        _stoneListingsLoaded = true;
        _stoneListingsLoading = false;
        _stoneListingsLoadError = null;
      });
      if (shouldPersistDraft) {
        unawaited(_saveReconciledOrderDraft(nextDraft));
      }
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _stoneListingsLoaded = true;
        _stoneListingsLoading = false;
        _stoneListingsLoadError = error;
      });
    }
  }

  void _retryStoneListings() {
    final locale =
        _stoneListingsLocale ?? Localizations.localeOf(context).languageCode;
    unawaited(_loadStoneListings(locale: locale));
  }

  OrderDraft _validateOrderDraftStoneSelection(
    OrderDraft draft,
    StoneListingsResult? result,
  ) {
    final stoneSelection = draft.stoneSelection;
    if (stoneSelection == null || result == null) {
      return draft;
    }

    for (final listing in result.listings) {
      if (listing.id == stoneSelection.listingId) {
        return listing.isOrderable ? draft : draft.withoutStoneSelection();
      }
    }

    return draft.withoutStoneSelection();
  }

  OrderDraft _discardExpiredCheckoutInput(
    OrderDraft draft, {
    required DateTime now,
  }) {
    if (draft.input.isEmpty) {
      return draft;
    }
    final inputUpdatedAt = draft.inputUpdatedAt ?? draft.updatedAt;
    if (inputUpdatedAt.isAfter(now)) {
      return draft;
    }
    if (now.difference(inputUpdatedAt) < _checkoutInputRetentionDuration) {
      return draft;
    }
    return draft.withoutInput(updatedAt: now);
  }

  Future<void> _saveReconciledOrderDraft(OrderDraft draft) async {
    try {
      await _localOrderDraftRepository.saveOrderDraft(draft);
    } catch (_) {
      // Keep the in-memory draft reconciled even if persistence is temporarily unavailable.
    }
  }

  Future<void> _saveLocalSealDesign({
    required SealGenerationResult result,
    required SealDesignVariant variant,
    required HankoTabStackController stack,
    bool replaceTopOnComplete = false,
  }) async {
    final saveKey = '${result.requestId}:${variant.id}';
    if (_savingLocalSealKeys.contains(saveKey)) {
      return;
    }
    _savingLocalSealKeys.add(saveKey);
    final design = _localSealDesignFromSelection(result, variant);
    try {
      await _localSealDesignRepository.saveLocalSealDesign(design);
    } catch (error) {
      if (!mounted) {
        _savingLocalSealKeys.remove(saveKey);
        return;
      }
      _savingLocalSealKeys.remove(saveKey);
      final errorPage = _sealSaveErrorPage(result, variant, error);
      if (replaceTopOnComplete) {
        stack.replaceTop(errorPage);
      } else {
        stack.push(errorPage);
      }
      return;
    }
    _savingLocalSealKeys.remove(saveKey);
    if (!mounted) {
      return;
    }

    setState(() {
      final remaining = _localSealDesigns
          .where((savedDesign) => savedDesign.id != design.id)
          .toList(growable: false);
      _localSealDesigns = List.unmodifiable([design, ...remaining]);
      _localSealDesignsLoaded = true;
      _localSealDesignsLoadError = null;
    });
    await _applyOrderDraft(
      _orderDraft.withSealSelection(
        _orderDraftSealSelectionFromLocalSealDesign(design),
      ),
    );
    if (!mounted) {
      return;
    }
    final confirmationPage = _sealSaveConfirmationPage(result, variant);
    if (replaceTopOnComplete) {
      stack.replaceTop(confirmationPage);
    } else {
      stack.push(confirmationPage);
    }
  }

  void _chooseLocalSealForOrder(LocalSealDesign design) {
    final nextDraft = _orderDraft.withSealSelection(
      _orderDraftSealSelectionFromLocalSealDesign(design),
    );
    unawaited(_applyOrderDraft(nextDraft));
    _openOrderReview();
  }

  void _chooseGeneratedSealForOrder({
    required SealGenerationResult result,
    required SealDesignVariant variant,
    required HankoTabStackController stack,
  }) {
    final nextDraft = _orderDraft.withSealSelection(
      _orderDraftSealSelectionFromGeneratedSeal(result, variant),
    );
    unawaited(_applyOrderDraft(nextDraft));
    stack.selectTab(HankoAppTab.stones);
  }

  Future<void> _toggleLocalSealFavorite(LocalSealDesign design) async {
    if (_updatingLocalSealFavoriteIds.contains(design.id)) {
      return;
    }
    _updatingLocalSealFavoriteIds.add(design.id);

    final updatedDesign = design.copyWith(isFavorite: !design.isFavorite);
    try {
      await _localSealDesignRepository.saveLocalSealDesign(updatedDesign);
    } catch (error) {
      _updatingLocalSealFavoriteIds.remove(design.id);
      if (!mounted) {
        return;
      }
      setState(() {
        _localSealDesignsLoaded = true;
        _localSealDesignsLoadError = error;
      });
      return;
    }

    _updatingLocalSealFavoriteIds.remove(design.id);
    if (!mounted) {
      return;
    }

    setState(() {
      _localSealDesigns = List.unmodifiable(
        _localSealDesigns.map((savedDesign) {
          return savedDesign.id == updatedDesign.id
              ? updatedDesign
              : savedDesign;
        }),
      );
      _localSealDesignsLoaded = true;
      _localSealDesignsLoadError = null;
    });
  }

  void _chooseStoneForOrder(StoneListing listing) {
    if (!listing.isOrderable) {
      return;
    }
    final nextDraft = _orderDraft.withStoneSelection(
      _orderDraftStoneSelectionFromStoneListing(listing),
    );
    unawaited(_applyOrderDraft(nextDraft));
    _openOrderReview();
  }

  Future<void> _applyOrderDraft(OrderDraft draft) async {
    final now = DateTime.now();
    final nextDraft = OrderDraft(
      sealSelection: draft.sealSelection,
      stoneSelection: draft.stoneSelection,
      input: draft.input,
      updatedAt: now,
      inputUpdatedAt: draft.input.isEmpty ? null : draft.inputUpdatedAt ?? now,
    );
    if (mounted) {
      setState(() {
        _orderDraft = nextDraft;
        _completedOrderDraft = null;
        _orderDraftHasLocalChanges = true;
      });
    }
    try {
      await _localOrderDraftRepository.saveOrderDraft(nextDraft);
    } catch (_) {
      // Keep the in-memory draft so tab-to-tab flow remains usable.
    }
  }

  Future<void> _deleteLocalSealDesign(
    LocalSealDesign design,
    HankoTabStackController stack,
  ) async {
    if (_deletingLocalSealIds.contains(design.id)) {
      return;
    }
    _deletingLocalSealIds.add(design.id);
    try {
      await _localSealDesignRepository.deleteLocalSealDesign(design.id);
    } catch (error) {
      if (!mounted) {
        _deletingLocalSealIds.remove(design.id);
        return;
      }
      setState(() {
        _localSealDesignsLoaded = true;
        _localSealDesignsLoadError = error;
      });
      _deletingLocalSealIds.remove(design.id);
      return;
    }
    _deletingLocalSealIds.remove(design.id);
    if (!mounted) {
      return;
    }

    setState(() {
      _localSealDesigns = List.unmodifiable(
        _localSealDesigns.where((savedDesign) => savedDesign.id != design.id),
      );
      _localSealDesignsLoaded = true;
      _localSealDesignsLoadError = null;
    });
    if (_orderDraft.sealSelection?.localSealDesignId == design.id) {
      unawaited(_applyOrderDraft(_orderDraft.withoutSealSelection()));
    }
    stack.pop();
  }

  PageEntry _kanjiLoadingPage(KanjiCandidatesRequest request) {
    return PageEntry(
      key: _designKanjiLoadingPageKey,
      name: '/design/kanji/loading',
      data: request,
    );
  }

  PageEntry _kanjiSuggestionsPage(KanjiCandidatesResult result) {
    return PageEntry(
      key: _designKanjiSuggestionsPageKey,
      name: '/design/kanji/suggestions',
      data: result,
    );
  }

  PageEntry _kanjiCandidateDetailPage(
    KanjiCandidatesResult result,
    KanjiCandidate candidate,
  ) {
    return PageEntry(
      key: _designKanjiCandidateDetailPageKey,
      name: '/design/kanji/candidate',
      data: _KanjiCandidateSelection(result: result, candidate: candidate),
    );
  }

  PageEntry _sealStyleSelectionPage(_KanjiCandidateSelection selection) {
    return PageEntry(
      key: _designSealStyleSelectionPageKey,
      name: '/design/seal/style',
      data: selection,
    );
  }

  PageEntry _sealGenerationLoadingPage(SealGenerationRequest request) {
    return PageEntry(
      key: _designSealGenerationLoadingPageKey,
      name: '/design/seal/generating',
      data: request,
    );
  }

  PageEntry _sealVariantSelectionPage(SealGenerationResult result) {
    return PageEntry(
      key: _designSealVariantSelectionPageKey,
      name: '/design/seal/variants',
      data: result,
    );
  }

  PageEntry _sealPreviewDetailPage(
    SealGenerationResult result,
    SealDesignVariant variant,
  ) {
    return PageEntry(
      key: _designSealPreviewDetailPageKey,
      name: '/design/seal/preview',
      data: _SealPreviewSelection(result: result, variant: variant),
    );
  }

  PageEntry _sealSaveConfirmationPage(
    SealGenerationResult result,
    SealDesignVariant variant,
  ) {
    return PageEntry(
      key: _designSealSaveConfirmationPageKey,
      name: '/design/seal/saved',
      data: _SealPreviewSelection(result: result, variant: variant),
    );
  }

  PageEntry _sealSaveErrorPage(
    SealGenerationResult result,
    SealDesignVariant variant,
    Object error,
  ) {
    return PageEntry(
      key: _designSealSaveErrorPageKey,
      name: '/design/seal/save-error',
      data: _SealSaveFailure(result: result, variant: variant, error: error),
    );
  }

  PageEntry _kanjiSuggestionErrorPage(
    KanjiCandidatesRequest request,
    Object error,
  ) {
    return PageEntry(
      key: _designKanjiErrorPageKey,
      name: '/design/kanji/error',
      data: _KanjiSuggestionFailure(request: request, error: error),
    );
  }

  PageEntry _sealGenerationErrorPage(
    SealGenerationRequest request,
    Object error,
  ) {
    return PageEntry(
      key: _designSealGenerationErrorPageKey,
      name: '/design/seal/error',
      data: _SealGenerationFailure(request: request, error: error),
    );
  }

  PageEntry _unsupportedKanjiPage(KanjiCandidatesRequest request) {
    return PageEntry(
      key: _designUnsupportedKanjiPageKey,
      name: '/design/kanji/empty',
      data: request,
    );
  }

  PageEntry _sealGenerationLimitPage(SealGenerationRequest request) {
    return PageEntry(
      key: _designSealGenerationLimitPageKey,
      name: '/design/seal/limit',
      data: request,
    );
  }

  PageEntry _mySealsDetailPage(LocalSealDesign design) {
    return PageEntry(
      key: _mySealsDetailPageKey,
      name: '/my-seals/detail',
      data: design,
    );
  }

  PageEntry _stoneDetailPage(StoneListing listing) {
    return PageEntry(
      key: _stoneDetailPageKey,
      name: '/stones/detail',
      data: listing,
    );
  }

  PageEntry _stoneImageGalleryPage({
    required StoneListing listing,
    required int initialPhotoIndex,
  }) {
    return PageEntry(
      key: _stoneImageGalleryPageKey,
      name: '/stones/detail/gallery',
      data: _StoneImageGallerySelection(
        listing: listing,
        initialPhotoIndex: initialPhotoIndex,
      ),
    );
  }

  void _openSettings() {
    if (_pages.last.key == _settingsPage.key) {
      return;
    }
    setState(() => _pages = const [_shellPage, _settingsPage]);
  }

  void _openOrderReview() {
    if (_pages.last.key == _orderReviewPage.key) {
      return;
    }
    setState(() => _pages = const [_shellPage, _orderReviewPage]);
  }

  void _openCheckoutInput() {
    final freshDraft = _discardExpiredCheckoutInput(
      _orderDraft,
      now: DateTime.now(),
    );
    final shouldPersistDraft = !identical(freshDraft, _orderDraft);
    if (_pages.last.key == _checkoutInputPage.key) {
      if (shouldPersistDraft) {
        setState(() {
          _orderDraft = freshDraft;
          _orderDraftHasLocalChanges = true;
        });
        unawaited(_saveReconciledOrderDraft(freshDraft));
      }
      return;
    }
    setState(() {
      _orderDraft = freshDraft;
      if (shouldPersistDraft) {
        _orderDraftHasLocalChanges = true;
      }
      _pages = const [_shellPage, _orderReviewPage, _checkoutInputPage];
    });
    if (shouldPersistDraft) {
      unawaited(_saveReconciledOrderDraft(freshDraft));
    }
  }

  Future<void> _saveCheckoutInput(OrderDraftInput input) {
    return _applyOrderDraft(_orderDraft.withInput(input));
  }

  void _openOrderConfirmation() {
    if (_pages.last.key == _orderConfirmationPage.key) {
      return;
    }
    setState(
      () => _pages = const [
        _shellPage,
        _orderReviewPage,
        _checkoutInputPage,
        _orderConfirmationPage,
      ],
    );
  }

  Future<void> _confirmOrderAgreements(
    OrderDraftCustomerConfirmationInput confirmation,
  ) async {
    final input = _orderDraft.input.copyWith(
      termsAgreed: confirmation.isComplete,
      customerConfirmation: confirmation,
    );
    final nextDraft = _orderDraft.withInput(input);
    await _applyOrderDraft(nextDraft);
    if (!mounted) {
      return;
    }
    unawaited(_startCheckoutProcessing(nextDraft));
  }

  Future<void> _startCheckoutProcessing(OrderDraft draft) async {
    if (_checkoutProcessingInFlight) {
      return;
    }

    final idempotencyKey = _checkoutIdempotencyKey(DateTime.now());
    final confirmedAt = DateTime.now().toUtc();
    setState(() {
      _checkoutProcessingStep = CheckoutProcessingStep.creatingOrder;
      _checkoutProcessingInFlight = true;
      _checkoutProcessingError = null;
      _checkoutCreatedOrder = null;
      _checkoutSession = null;
      _checkoutRouteCode = null;
      _stripeCheckoutStep = StripeCheckoutExternalStep.opening;
      _stripeCheckoutLaunchError = null;
      _checkoutDeepLinkError = null;
      _checkoutReturnResult = null;
      _completedOrderDraft = null;
      _paymentStatusStep = PaymentStatusStep.checking;
      _paymentStatus = null;
      _paymentStatusError = null;
      _paymentStatusRequestId++;
      _pages = _checkoutProcessingPages;
    });

    try {
      final checkoutRouteCode = await _checkoutRouteCodeForLocale(
        Localizations.localeOf(context),
      );
      if (!mounted) {
        return;
      }
      _checkoutRouteCode = checkoutRouteCode;
      final orderDraft = _sealOrderDraftFromOrderDraft(
        draft,
        locale: checkoutRouteCode,
        idempotencyKey: idempotencyKey,
        confirmedAt: confirmedAt,
      );
      final order = await widget.createOrder(orderDraft);
      if (!mounted) {
        return;
      }
      setState(() {
        _checkoutCreatedOrder = order;
        _checkoutProcessingStep =
            CheckoutProcessingStep.creatingCheckoutSession;
      });

      final session = await widget.createCheckoutSession(
        CheckoutSessionRequest(
          orderId: order.orderId,
          customerEmail: draft.input.contact.email.trim(),
          returnToApp: true,
        ),
      );
      if (!mounted) {
        return;
      }
      setState(() {
        _checkoutSession = session;
        _checkoutProcessingStep = CheckoutProcessingStep.ready;
      });
      await _openStripeCheckout(session);
    } catch (error) {
      if (!mounted) {
        return;
      }
      setState(() {
        _checkoutProcessingError = error;
        if (_isStripeCheckoutPreparationError(error)) {
          _stripeCheckoutStep = StripeCheckoutExternalStep.failed;
          _stripeCheckoutLaunchError = error;
          _checkoutReturnResult = null;
          _paymentStatusRequestId++;
          _pages = const [_shellPage, _stripeCheckoutFailedPage];
        }
      });
    } finally {
      if (mounted) {
        setState(() => _checkoutProcessingInFlight = false);
      }
    }
  }

  Future<void> _openStripeCheckout(CheckoutSession session) async {
    setState(() {
      _stripeCheckoutStep = StripeCheckoutExternalStep.opening;
      _stripeCheckoutLaunchError = null;
      _checkoutDeepLinkError = null;
      _checkoutReturnResult = null;
      _paymentStatusStep = PaymentStatusStep.checking;
      _paymentStatus = null;
      _paymentStatusError = null;
      _pages = _stripeCheckoutPages;
    });

    try {
      await widget.openCheckoutUrl(session);
      if (!mounted) {
        return;
      }
      if (_checkoutReturnResult != null) {
        return;
      }
      setState(() {
        _stripeCheckoutStep = StripeCheckoutExternalStep.waitingForReturn;
      });
    } catch (error) {
      if (!mounted) {
        return;
      }
      if (_checkoutReturnResult != null) {
        return;
      }
      setState(() {
        _stripeCheckoutStep = StripeCheckoutExternalStep.failed;
        _stripeCheckoutLaunchError = error;
        _paymentStatusRequestId++;
        _pages = const [_shellPage, _stripeCheckoutFailedPage];
      });
    }
  }

  void _beginCheckoutResumeStatusCheck() {
    final order = _checkoutCreatedOrder;
    final session = _checkoutSession;
    if (order == null || session == null || _checkoutReturnResult != null) {
      return;
    }
    if (_stripeCheckoutStep != StripeCheckoutExternalStep.opening &&
        _stripeCheckoutStep != StripeCheckoutExternalStep.waitingForReturn) {
      return;
    }
    if (!_pages.any((page) => page.key == _stripeCheckoutTransitionPage.key)) {
      return;
    }

    final checkoutRouteCode =
        _checkoutRouteCode ??
        _fallbackCheckoutRouteCodeForLocale(Localizations.localeOf(context));
    final sourceUri = Uri(
      scheme: 'hankofield',
      host: 'checkout',
      path: '/success',
      queryParameters: {
        'order_id': order.orderId,
        'session_id': session.sessionId,
        'lang': checkoutRouteCode,
      },
    );
    _beginPaymentStatusCheck(
      CheckoutReturnResult(
        outcome: CheckoutReturnOutcome.success,
        sourceUri: sourceUri,
        orderId: order.orderId,
        sessionId: session.sessionId,
        locale: checkoutRouteCode,
      ),
    );
  }

  Future<bool> _handleIncomingCheckoutRoute(String route) async {
    if (route == _lastHandledCheckoutRoute) {
      return true;
    }
    final result = parseCheckoutReturnRoute(route);
    if (result == null) {
      if (isMalformedCheckoutReturnRoute(route)) {
        _lastHandledCheckoutRoute = route;
        if (!mounted) {
          return true;
        }
        _showCheckoutDeepLinkError(HankoDeepLinkException(route));
        return true;
      }
      return false;
    }
    _lastHandledCheckoutRoute = route;
    if (!mounted) {
      return true;
    }
    if (result.outcome == CheckoutReturnOutcome.success) {
      _beginPaymentStatusCheck(result);
      return true;
    }
    setState(() {
      _checkoutReturnResult = result;
      _stripeCheckoutLaunchError = null;
      _checkoutDeepLinkError = null;
      _stripeCheckoutStep = StripeCheckoutExternalStep.returned;
      _paymentStatusRequestId++;
      _pages = [
        _shellPage,
        switch (result.outcome) {
          CheckoutReturnOutcome.canceled => _stripeCheckoutCanceledPage,
          CheckoutReturnOutcome.failed => _stripeCheckoutFailedPage,
          CheckoutReturnOutcome.success => _stripeCheckoutTransitionPage,
        },
      ];
    });
    return true;
  }

  void _beginPaymentStatusCheck(CheckoutReturnResult result) {
    final requestId = _paymentStatusRequestId + 1;
    setState(() {
      _paymentStatusRequestId = requestId;
      _checkoutReturnResult = result;
      _stripeCheckoutLaunchError = null;
      _checkoutDeepLinkError = null;
      _stripeCheckoutStep = StripeCheckoutExternalStep.returned;
      _paymentStatusStep = PaymentStatusStep.checking;
      _paymentStatus = null;
      _paymentStatusError = null;
      _pages = const [_shellPage, _paymentStatusPage];
    });
    unawaited(_verifyPaymentStatus(result, requestId));
  }

  void _showCheckoutDeepLinkError(Object error) {
    setState(() {
      _checkoutDeepLinkError = error;
      _checkoutReturnResult = null;
      _stripeCheckoutLaunchError = null;
      _paymentStatusRequestId++;
      _paymentStatusStep = PaymentStatusStep.failed;
      _paymentStatusError = error;
      _pages = const [_shellPage, _checkoutDeepLinkErrorPage];
    });
  }

  Future<void> _verifyPaymentStatus(
    CheckoutReturnResult result,
    int requestId,
  ) async {
    final orderId = (result.orderId ?? _checkoutCreatedOrder?.orderId ?? '')
        .trim();
    if (orderId.isEmpty) {
      if (!mounted || _paymentStatusRequestId != requestId) {
        return;
      }
      _showCheckoutDeepLinkError(
        HankoDeepLinkException(
          result.sourceUri.toString(),
          StateError('checkout return is missing order_id'),
        ),
      );
      return;
    }

    const maxAttempts = 3;
    Object? lastError;
    OrderStatus? lastStatus;
    for (var attempt = 0; attempt < maxAttempts; attempt++) {
      try {
        final status = await widget.fetchOrderStatus(orderId);
        if (!mounted || _paymentStatusRequestId != requestId) {
          return;
        }
        lastStatus = status;
        lastError = null;
        setState(() => _paymentStatus = status);
        if (_isPaidOrderStatus(status)) {
          final completedDraft = _orderDraft;
          final clearedDraft = _orderDraft.withoutInput();
          setState(() {
            _completedOrderDraft = completedDraft;
            _orderDraft = clearedDraft;
            _paymentStatusStep = PaymentStatusStep.paid;
            _paymentStatusError = null;
            _pages = const [_shellPage, _orderCompletePage];
          });
          unawaited(_saveReconciledOrderDraft(clearedDraft));
          return;
        }
      } catch (error) {
        lastError = error;
      }

      if (attempt < maxAttempts - 1) {
        await Future<void>.delayed(widget.paymentStatusRetryDelay);
        if (!mounted || _paymentStatusRequestId != requestId) {
          return;
        }
      }
    }

    if (!mounted || _paymentStatusRequestId != requestId) {
      return;
    }
    setState(() {
      _paymentStatus = lastStatus;
      _paymentStatusError = lastStatus == null ? lastError : null;
      _paymentStatusStep = lastStatus == null
          ? PaymentStatusStep.failed
          : PaymentStatusStep.pending;
    });
  }

  void _handleCheckoutRouteNotification() {
    final route = widget.checkoutRouteListenable?.value;
    if (route == null || route == _lastHandledCheckoutRoute) {
      return;
    }
    unawaited(_handleIncomingCheckoutRoute(route));
  }

  void _showTabFromOrder(HankoAppTab tab) {
    setState(() {
      _requestedTab = tab;
      _requestedTabPages = null;
      _pages = const [_shellPage];
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _requestedTab != tab) {
        return;
      }
      setState(() {
        _requestedTab = null;
        _requestedTabPages = null;
      });
    });
  }

  void _openRegenerateLocalSealDesign(LocalSealDesign design) {
    final designTab = _navigationTabs.firstWhere(
      (tab) => tab.tab == HankoAppTab.design,
    );
    final selection = _kanjiCandidateSelectionFromLocalSealDesign(
      design,
      reasonLanguage: _reasonLanguageForCurrentLocale(context),
    );
    setState(() {
      _requestedTab = HankoAppTab.design;
      _requestedTabPages = [
        designTab.rootPage,
        _sealStyleSelectionPage(selection),
      ];
      _pages = const [_shellPage];
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || _requestedTab != HankoAppTab.design) {
        return;
      }
      setState(() {
        _requestedTab = null;
        _requestedTabPages = null;
      });
    });
  }

  void _openOrderLookup() {
    setState(
      () => _pages = const [_shellPage, _orderCompletePage, _orderLookupPage],
    );
  }

  void _openContactSupport() {
    setState(
      () => _pages = [
        _shellPage,
        PageEntry(
          key: _settingsPage.key,
          name: '/settings/contact',
          data: SettingsInitialDestination.contact,
        ),
      ],
    );
  }

  void _closeTopPage() {
    if (_pages.length <= 1) {
      return;
    }
    setState(() => _pages = List.unmodifiable(_pages.take(_pages.length - 1)));
  }
}

_KanjiCandidateSelection _kanjiCandidateSelectionFromLocalSealDesign(
  LocalSealDesign design, {
  required String reasonLanguage,
}) {
  final meaning = design.meaning?.trim();
  final reason = meaning != null && meaning.isNotEmpty
      ? meaning
      : design.impression.join(', ');
  final candidate = KanjiCandidate(
    kanji: design.selectedKanji,
    reading: design.reading,
    meaning: meaning,
    impression: design.impression,
    characterCount: design.characterCount,
    strokeComplexity: design.strokeComplexity,
    engravingSuitability: design.engravingSuitability,
    reason: reason,
  );
  return _KanjiCandidateSelection(
    result: KanjiCandidatesResult(
      realName: design.inputName,
      reasonLanguage: reasonLanguage,
      gender: KanjiCandidateGender.unspecified,
      kanjiStyle: KanjiNameStyle.japanese,
      candidates: [candidate],
    ),
    candidate: candidate,
    initialStyleSelection: _sealStyleSelectionFromLocalSealDesign(design),
  );
}

SealStyleSelection _sealStyleSelectionFromLocalSealDesign(
  LocalSealDesign design,
) {
  return SealStyleSelection(
    shape: _sealShapeFromApiValue(design.shape),
    style: _sealStyleNameFromApiValue(design.style),
    strokeWeight: _sealStrokeWeightFromApiValue(design.strokeWeight),
    balance: _sealBalanceFromApiValue(design.balance),
  );
}

String _reasonLanguageForCurrentLocale(BuildContext context) {
  return Localizations.localeOf(context).languageCode == 'ja' ? 'ja' : 'en';
}

SealShape _sealShapeFromApiValue(String value) {
  return switch (value.trim().toLowerCase()) {
    'round' => SealShape.round,
    _ => SealShape.square,
  };
}

SealStyleName _sealStyleNameFromApiValue(String value) {
  return switch (value.trim().toLowerCase()) {
    'traditional' => SealStyleName.traditional,
    'soft' => SealStyleName.soft,
    'bold' => SealStyleName.bold,
    _ => SealStyleName.elegant,
  };
}

SealStrokeWeight _sealStrokeWeightFromApiValue(String value) {
  return switch (value.trim().toLowerCase()) {
    'bold' => SealStrokeWeight.bold,
    _ => SealStrokeWeight.standard,
  };
}

SealBalance _sealBalanceFromApiValue(String value) {
  return switch (value.trim().toLowerCase()) {
    'airy' => SealBalance.airy,
    'dense' => SealBalance.dense,
    _ => SealBalance.balanced,
  };
}

LocalSealDesign _localSealDesignFromSelection(
  SealGenerationResult result,
  SealDesignVariant variant,
) {
  final now = DateTime.now();
  final candidate = result.request.candidate;
  final style = result.request.style;

  return LocalSealDesign(
    id: _localSealDesignId(result.requestId, variant.id, now),
    inputName: result.request.inputName,
    selectedKanji: candidate.kanji,
    reading: candidate.reading,
    meaning: candidate.meaning,
    impression: candidate.impression,
    characterCount: candidate.characterCount ?? candidate.kanji.runes.length,
    strokeComplexity: candidate.strokeComplexity,
    engravingSuitability: candidate.engravingSuitability,
    shape: style.shape.apiValue,
    style: style.style.apiValue,
    strokeWeight: style.strokeWeight.apiValue,
    balance: style.balance.apiValue,
    aiGenerationId: result.requestId,
    aiVariantId: variant.id,
    previewImageStoragePath: variant.storagePath,
    previewImageDownloadUrl: variant.downloadUrl,
    localImagePath: '',
    isFavorite: false,
    createdAt: now,
    updatedAt: now,
  );
}

String _localSealDesignId(String requestId, String variantId, DateTime now) {
  final rawId = 'local_${requestId}_${variantId}_${now.microsecondsSinceEpoch}';
  return rawId.replaceAll(RegExp(r'[^A-Za-z0-9_-]'), '_');
}

String _checkoutIdempotencyKey(DateTime now) {
  return 'app_${now.microsecondsSinceEpoch}';
}

bool _isPaidOrderStatus(OrderStatus status) {
  return status.paymentStatus.trim().toLowerCase() == 'paid' ||
      status.orderStatus.trim().toLowerCase() == 'paid';
}

bool _isStripeCheckoutPreparationError(Object error) {
  return error is HankoApiException &&
      (error.code == 'stripe_not_configured' ||
          error.code == 'stripe_checkout_failed');
}

SealOrderDraft _sealOrderDraftFromOrderDraft(
  OrderDraft draft, {
  required String locale,
  required String idempotencyKey,
  required DateTime confirmedAt,
}) {
  final seal = draft.sealSelection;
  final stone = draft.stoneSelection;
  if (seal == null || stone == null) {
    throw StateError('Order draft requires seal and stone selections');
  }

  final normalizedLocale = draft.input.contact.preferredLocale.trim().isEmpty
      ? locale
      : draft.input.contact.preferredLocale.trim();
  final sealText = seal.selectedKanji.trim();

  return SealOrderDraft(
    channel: 'app',
    locale: normalizedLocale,
    idempotencyKey: idempotencyKey,
    termsAgreed: draft.input.termsAgreed,
    listingId: stone.listingId,
    seal: SealOrderSeal(
      line1: sealText,
      line2: '',
      shape: seal.shape,
      fontKey: 'ai_generated_seal',
      aiGenerationId: seal.aiGenerationId,
      aiVariantId: seal.aiVariantId,
      previewImage: SealOrderPreviewImage(
        storagePath: seal.previewImageStoragePath,
        downloadUrl: seal.previewImageDownloadUrl,
      ),
      style: SealOrderStyle(
        name: seal.style,
        strokeWeight: seal.strokeWeight,
        balance: seal.balance,
      ),
    ),
    shipping: SealOrderShipping(
      countryCode: draft.input.shipping.countryCode,
      recipientName: draft.input.shipping.recipientName,
      phone: draft.input.shipping.phone,
      postalCode: draft.input.shipping.postalCode,
      state: draft.input.shipping.state,
      city: draft.input.shipping.city,
      addressLine1: draft.input.shipping.addressLine1,
      addressLine2: draft.input.shipping.addressLine2,
    ),
    contact: SealOrderContact(
      email: draft.input.contact.email,
      preferredLocale: normalizedLocale,
    ),
    customerConfirmation: SealOrderCustomerConfirmation(
      kanjiAndDesign: draft.input.customerConfirmation.kanjiAndDesign,
      customMadePolicy: draft.input.customerConfirmation.customMadePolicy,
      confirmedAt: confirmedAt,
      confirmedSealText: sealText,
    ),
    orderNote: draft.input.orderNote,
  );
}

OrderDraftSealSelection _orderDraftSealSelectionFromLocalSealDesign(
  LocalSealDesign design,
) {
  return OrderDraftSealSelection(
    localSealDesignId: design.id,
    selectedKanji: design.selectedKanji,
    reading: design.reading,
    shape: design.shape,
    style: design.style,
    strokeWeight: design.strokeWeight,
    balance: design.balance,
    aiGenerationId: design.aiGenerationId,
    aiVariantId: design.aiVariantId,
    previewImageStoragePath: design.previewImageStoragePath,
    previewImageDownloadUrl: design.previewImageDownloadUrl,
    localImagePath: design.localImagePath,
  );
}

OrderDraftSealSelection _orderDraftSealSelectionFromGeneratedSeal(
  SealGenerationResult result,
  SealDesignVariant variant,
) {
  final candidate = result.request.candidate;
  final style = result.request.style;

  return OrderDraftSealSelection(
    localSealDesignId: _generatedSealDraftId(result.requestId, variant.id),
    selectedKanji: candidate.kanji,
    reading: candidate.reading,
    shape: style.shape.apiValue,
    style: style.style.apiValue,
    strokeWeight: style.strokeWeight.apiValue,
    balance: style.balance.apiValue,
    aiGenerationId: result.requestId,
    aiVariantId: variant.id,
    previewImageStoragePath: variant.storagePath,
    previewImageDownloadUrl: variant.downloadUrl,
    localImagePath: '',
  );
}

String _generatedSealDraftId(String requestId, String variantId) {
  final rawId = 'generated_${requestId}_$variantId';
  return rawId.replaceAll(RegExp(r'[^A-Za-z0-9_-]'), '_');
}

OrderDraftStoneSelection _orderDraftStoneSelectionFromStoneListing(
  StoneListing listing,
) {
  return OrderDraftStoneSelection(
    listingId: listing.id,
    code: listing.code,
    materialKey: listing.materialKey,
    materialLabel: listing.materialLabel,
    sizeLabel: listing.sizeLabel,
    title: listing.title,
    price: listing.price,
    status: listing.status,
    isOrderable: listing.isOrderable,
    primaryPhotoUrl: _primaryStonePhotoUrl(listing.photos),
  );
}

String _primaryStonePhotoUrl(List<StoneListingPhoto> photos) {
  if (photos.isEmpty) {
    return '';
  }
  for (final photo in photos) {
    if (photo.isPrimary) {
      return photo.assetUrl;
    }
  }
  final sorted = [...photos]
    ..sort((left, right) {
      final order = left.sortOrder.compareTo(right.sortOrder);
      return order == 0 ? left.assetId.compareTo(right.assetId) : order;
    });
  return sorted.first.assetUrl;
}

class _BottomTabs extends StatelessWidget {
  const _BottomTabs({
    required this.selectedIndex,
    required this.tabs,
    required this.onSelected,
  });

  final int selectedIndex;
  final List<_TabItem> tabs;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 100,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final tabWidth = constraints.maxWidth / tabs.length;
          return DecoratedBox(
            decoration: const BoxDecoration(
              color: HankoColors.background,
              border: Border(
                top: BorderSide(color: HankoColors.navBorder, width: 0.7),
              ),
            ),
            child: Stack(
              children: [
                AnimatedPositioned(
                  duration: const Duration(milliseconds: 180),
                  curve: Curves.easeOutCubic,
                  top: 0,
                  left: tabWidth * selectedIndex + (tabWidth / 2) - 25,
                  width: 50,
                  height: 4,
                  child: DecoratedBox(
                    decoration: BoxDecoration(
                      color: HankoColors.red,
                      borderRadius: BorderRadius.circular(3),
                    ),
                  ),
                ),
                Row(
                  children: [
                    for (var index = 0; index < tabs.length; index++)
                      Expanded(
                        child: _BottomTabButton(
                          item: tabs[index],
                          isSelected: selectedIndex == index,
                          onTap: () => onSelected(index),
                        ),
                      ),
                  ],
                ),
                const Positioned(
                  left: 0,
                  right: 0,
                  bottom: 8,
                  child: Center(child: _HomeIndicator()),
                ),
              ],
            ),
          );
        },
      ),
    );
  }
}

class _BottomTabButton extends StatelessWidget {
  const _BottomTabButton({
    required this.item,
    required this.isSelected,
    required this.onTap,
  });

  final _TabItem item;
  final bool isSelected;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final color = isSelected ? HankoColors.red : HankoColors.ink;
    return Semantics(
      button: true,
      selected: isSelected,
      label: item.label,
      onTap: onTap,
      child: ExcludeSemantics(
        child: InkResponse(
          onTap: onTap,
          containedInkWell: true,
          highlightShape: BoxShape.rectangle,
          child: Padding(
            padding: const EdgeInsets.only(top: 21, bottom: 20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                CustomPaint(
                  size: const Size.square(31),
                  painter: _TabIconPainter(item.icon, color),
                ),
                const SizedBox(height: 7),
                Text(
                  item.label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: color,
                    fontSize: 13.5,
                    fontWeight: FontWeight.w500,
                    height: 1,
                    letterSpacing: 0,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _HomeIndicator extends StatelessWidget {
  const _HomeIndicator();

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 136,
      height: 5,
      decoration: BoxDecoration(
        color: Colors.black,
        borderRadius: BorderRadius.circular(10),
      ),
    );
  }
}

class _TabIconPainter extends CustomPainter {
  const _TabIconPainter(this.icon, this.color);

  final _TabIcon icon;
  final Color color;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint()
      ..color = color
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2
      ..strokeCap = StrokeCap.round
      ..strokeJoin = StrokeJoin.round;
    final fillPaint = Paint()
      ..color = color
      ..style = PaintingStyle.fill;

    switch (icon) {
      case _TabIcon.design:
        final stem = Path()
          ..moveTo(size.width * 0.61, size.height * 0.04)
          ..lineTo(size.width * 0.32, size.height * 0.61);
        canvas.drawPath(stem, paint);
        final brush = Path()
          ..moveTo(size.width * 0.65, size.height * 0.03)
          ..quadraticBezierTo(
            size.width * 0.79,
            size.height * 0.16,
            size.width * 0.68,
            size.height * 0.30,
          )
          ..lineTo(size.width * 0.42, size.height * 0.66)
          ..quadraticBezierTo(
            size.width * 0.33,
            size.height * 0.59,
            size.width * 0.31,
            size.height * 0.54,
          )
          ..lineTo(size.width * 0.56, size.height * 0.20)
          ..quadraticBezierTo(
            size.width * 0.59,
            size.height * 0.10,
            size.width * 0.65,
            size.height * 0.03,
          );
        canvas.drawPath(brush, paint);
        final plate = Path()
          ..moveTo(size.width * 0.18, size.height * 0.61)
          ..quadraticBezierTo(
            size.width * 0.34,
            size.height * 0.80,
            size.width * 0.17,
            size.height * 0.84,
          )
          ..quadraticBezierTo(
            size.width * 0.42,
            size.height * 1.02,
            size.width * 0.72,
            size.height * 0.76,
          );
        canvas.drawPath(plate, paint);
        canvas.drawOval(
          Rect.fromLTWH(
            size.width * 0.03,
            size.height * 0.69,
            size.width * 0.65,
            size.height * 0.25,
          ),
          paint,
        );
        break;
      case _TabIcon.mySeals:
        final shield = Path()
          ..moveTo(size.width * 0.50, size.height * 0.07)
          ..lineTo(size.width * 0.84, size.height * 0.20)
          ..lineTo(size.width * 0.79, size.height * 0.61)
          ..quadraticBezierTo(
            size.width * 0.74,
            size.height * 0.80,
            size.width * 0.50,
            size.height * 0.93,
          )
          ..quadraticBezierTo(
            size.width * 0.26,
            size.height * 0.80,
            size.width * 0.21,
            size.height * 0.61,
          )
          ..lineTo(size.width * 0.16, size.height * 0.20)
          ..close();
        canvas.drawPath(shield, paint);
        canvas.drawCircle(
          Offset(size.width * 0.50, size.height * 0.51),
          size.width * 0.045,
          fillPaint,
        );
        final sparkle = Path()
          ..moveTo(size.width * 0.50, size.height * 0.34)
          ..lineTo(size.width * 0.54, size.height * 0.47)
          ..lineTo(size.width * 0.67, size.height * 0.51)
          ..lineTo(size.width * 0.54, size.height * 0.55)
          ..lineTo(size.width * 0.50, size.height * 0.68)
          ..lineTo(size.width * 0.46, size.height * 0.55)
          ..lineTo(size.width * 0.33, size.height * 0.51)
          ..lineTo(size.width * 0.46, size.height * 0.47)
          ..close();
        canvas.drawPath(sparkle, fillPaint);
        break;
      case _TabIcon.stones:
        final path = Path()
          ..moveTo(size.width * 0.50, size.height * 0.94)
          ..lineTo(size.width * 0.09, size.height * 0.35)
          ..lineTo(size.width * 0.27, size.height * 0.12)
          ..lineTo(size.width * 0.73, size.height * 0.12)
          ..lineTo(size.width * 0.91, size.height * 0.35)
          ..close();
        canvas.drawPath(path, paint);
        canvas.drawLine(
          Offset(size.width * 0.09, size.height * 0.35),
          Offset(size.width * 0.91, size.height * 0.35),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.27, size.height * 0.12),
          Offset(size.width * 0.50, size.height * 0.94),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.73, size.height * 0.12),
          Offset(size.width * 0.50, size.height * 0.94),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.40, size.height * 0.12),
          Offset(size.width * 0.31, size.height * 0.35),
          paint,
        );
        canvas.drawLine(
          Offset(size.width * 0.60, size.height * 0.12),
          Offset(size.width * 0.69, size.height * 0.35),
          paint,
        );
        break;
    }
  }

  @override
  bool shouldRepaint(covariant _TabIconPainter oldDelegate) {
    return oldDelegate.icon != icon || oldDelegate.color != color;
  }
}

class _TabItem {
  const _TabItem(this.label, this.icon);

  final String label;
  final _TabIcon icon;
}

enum _TabIcon { design, mySeals, stones }
