import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:app/features/sample_counter/application/providers.dart';

class CounterNotifier extends AsyncNotifier<int> {
  @override
  Future<int> build() async {
    final repo = ref.read(counterRepositoryProvider);
    return repo.load();
  }

  Future<void> increment() async {
    final current = state.value ?? 0;
    final next = current + 1;
    state = const AsyncLoading();
    try {
      final repo = ref.read(counterRepositoryProvider);
      await repo.save(next);
      if (!ref.mounted) return;
      state = AsyncData(next);
    } catch (e, st) {
      if (!ref.mounted) return;
      state = AsyncError(e, st);
    }
  }
}
