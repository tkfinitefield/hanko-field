import 'dart:async';
import 'dart:ui';

import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/firebase/firebase_providers.dart';
import 'package:app/core/monitoring/analytics_controller.dart';
import 'package:app/core/monitoring/analytics_events.dart';
import 'package:app/core/monitoring/crash_reporting_controller.dart';
import 'package:app/core/routing/app_route_information_parser.dart';
import 'package:app/core/routing/app_router_delegate.dart';
import 'package:app/core/storage/storage_providers.dart';
import 'package:app/core/theme/app_theme.dart';
import 'package:app/l10n/gen/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final container = ProviderContainer();
  await container.read(localCacheStoreInitializedProvider.future);
  await container.read(sharedPreferencesProvider.future);
  await container.read(firebaseInitializedProvider.future);
  await container.read(crashReportingControllerProvider.future);
  await container.read(analyticsControllerProvider.future);
  final crashController = container.read(
    crashReportingControllerProvider.notifier,
  );

  FlutterError.onError = (details) {
    FlutterError.presentError(details);
    crashController.recordFlutterError(details);
  };

  PlatformDispatcher.instance.onError = (error, stack) {
    crashController.recordError(error, stack, fatal: true);
    return false;
  };

  runZonedGuarded(
    () {
      runApp(
        UncontrolledProviderScope(container: container, child: const App()),
      );
    },
    (error, stack) {
      crashController.recordError(error, stack, fatal: true);
    },
  );
}

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final config = ref.watch(appConfigProvider);
    final routerDelegate = ref.watch(appRouterDelegateProvider);
    // Kick off Firebase initialization once. UI does not block on it.
    ref.listen<AsyncValue<void>>(firebaseInitializedProvider, (_, __) {});
    ref.listen<AsyncValue<AnalyticsState>>(analyticsControllerProvider, (
      previous,
      next,
    ) {
      final prevAllowed = previous?.asData?.value.consentGranted ?? false;
      final nextAllowed = next.asData?.value.consentGranted ?? false;
      if (!prevAllowed && nextAllowed) {
        final analytics = ref.read(analyticsControllerProvider.notifier);
        unawaited(
          analytics.logEvent(const AppLaunchedEvent(fromNotification: false)),
        );
        unawaited(
          analytics.logScreenView(
            const ScreenViewAnalyticsEvent(
              screenName: 'home',
              screenClass: 'HomeScaffold',
            ),
          ),
        );
      }
    });
    return MaterialApp.router(
      routerDelegate: routerDelegate,
      routeInformationParser: const AppRouteInformationParser(),
      title: config.displayName,
      onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
    );
  }
}
