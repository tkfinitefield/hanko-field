import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'core/app/app_flavor.dart';

void main() {
  runApp(const ProviderScope(child: App()));
}

class App extends ConsumerWidget {
  const App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final config = ref.watch(appConfigProvider);
    return MaterialApp(
      title: config.displayName,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF2A2A2A)),
        useMaterial3: true,
      ),
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
