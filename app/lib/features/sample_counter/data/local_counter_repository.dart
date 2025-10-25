import 'dart:async';

import 'package:app/core/storage/cache_bucket.dart';
import 'package:app/core/storage/local_cache_store.dart';

import '../domain/counter_repository.dart';

class LocalCounterRepository implements CounterRepository {
  LocalCounterRepository(this._store);

  static const _key = 'sample_counter.value';
  final LocalCacheStore _store;

  @override
  Future<int> load() async {
    final cached = await _store.read<int>(
      bucket: CacheBucket.sandbox,
      key: _key,
      decoder: (data) => data as int,
    );
    return cached.value ?? 0;
  }

  @override
  Future<void> save(int value) async {
    await _store.write<int>(
      bucket: CacheBucket.sandbox,
      key: _key,
      encoder: (value) => value,
      value: value,
    );
  }
}
