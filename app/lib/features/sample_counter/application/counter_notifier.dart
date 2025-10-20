import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../domain/counter_repository.dart';
import 'providers.dart';

class CounterNotifier extends AsyncNotifier<int> {
  late final CounterRepository _repo;

  @override
  Future<int> build() async {
    _repo = ref.read(counterRepositoryProvider);
    return _repo.load();
  }

  Future<void> increment() async {
    final current = state.value ?? 0;
    final next = current + 1;
    state = const AsyncLoading();
    try {
      await _repo.save(next);
      if (!ref.mounted) return;
      state = AsyncData(next);
    } catch (e, st) {
      if (!ref.mounted) return;
      state = AsyncError(e, st);
    }
  }
}

