import 'dart:async';
import 'dart:ui';

import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/firebase/firebase_providers.dart';
import 'package:app/core/monitoring/analytics_controller.dart';
import 'package:app/core/monitoring/analytics_events.dart';
import 'package:app/core/monitoring/crash_reporting_controller.dart';
import 'package:app/core/theme/app_theme.dart';
import 'package:app/l10n/gen/app_localizations.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();

  final container = ProviderContainer();
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
    return MaterialApp(
      onGenerateTitle: (context) => AppLocalizations.of(context).appTitle,
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      localizationsDelegates: AppLocalizations.localizationsDelegates,
      supportedLocales: AppLocalizations.supportedLocales,
      home: Scaffold(
        appBar: AppBar(title: Text(config.displayName)),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text('Base URL: ${config.baseUrl}'),
              const SizedBox(height: 12),
              Text(
                'Flavor (dart-define FLAVOR): '
                '${const String.fromEnvironment('FLAVOR', defaultValue: 'dev')}',
              ),
              const SizedBox(height: 12),
              // Remote Config sample value (default set in init)
              const Text('RC welcome_title: '),
              Text(
                ref.watch(welcomeTitleProvider),
                style: Theme.of(context).textTheme.titleLarge,
              ),
            ],
          ),
        ),
      ),
    );
  }
}
