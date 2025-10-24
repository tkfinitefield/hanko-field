import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../data/local_counter_repository.dart';
import '../domain/counter_repository.dart';
import 'counter_notifier.dart';

final counterRepositoryProvider = Provider<CounterRepository>((ref) {
  return LocalCounterRepository();
});

final counterProvider = AsyncNotifierProvider<CounterNotifier, int>(
  CounterNotifier.new,
);
