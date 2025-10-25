import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:app/core/storage/storage_providers.dart';

import '../data/local_counter_repository.dart';
import '../domain/counter_repository.dart';
import 'counter_notifier.dart';

final counterRepositoryProvider = Provider<CounterRepository>((ref) {
  final store = ref.watch(localCacheStoreProvider);
  return LocalCounterRepository(store);
});

final counterProvider = AsyncNotifierProvider<CounterNotifier, int>(
  CounterNotifier.new,
);
