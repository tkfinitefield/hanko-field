import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../application/providers.dart';

class CounterScreen extends ConsumerWidget {
  const CounterScreen({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final counter = ref.watch(counterProvider);
    return Scaffold(
      appBar: AppBar(title: const Text('Sample Counter')),
      body: Center(
        child: switch (counter) {
          AsyncData(:final value) => Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Text('Count: $value'),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: () =>
                      ref.read(counterProvider.notifier).increment(),
                  child: const Text('Increment'),
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
