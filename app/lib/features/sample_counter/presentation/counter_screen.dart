import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:app/l10n/gen/app_localizations.dart';

import 'package:app/features/sample_counter/application/providers.dart';

class CounterScreen extends ConsumerWidget {
  const CounterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final counter = ref.watch(counterProvider);
    final l10n = AppLocalizations.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(l10n.counterScreenTitle)),
      body: Center(
        child: switch (counter) {
          AsyncData(:final value) => Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(l10n.countLabel(value)),
              const SizedBox(height: 16),
              FilledButton(
                onPressed: () => ref.read(counterProvider.notifier).increment(),
                child: Text(l10n.increment),
              ),
            ],
          ),
          AsyncLoading() => const CircularProgressIndicator(),
          AsyncError(:final error) => Text('Error: $error'),
        },
      ),
    );
  }
}
