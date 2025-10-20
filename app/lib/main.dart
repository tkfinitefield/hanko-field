import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:app/l10n/gen/app_localizations.dart';

import 'package:app/core/app/app_flavor.dart';
import 'package:app/core/theme/app_theme.dart';

void main() {
  runApp(const ProviderScope(child: App()));
}

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final config = ref.watch(appConfigProvider);
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
              Text('Flavor (dart-define FLAVOR): '
                  '${const String.fromEnvironment('FLAVOR', defaultValue: 'dev')}'),
            ],
          ),
        ),
      ),
    );
  }
}
