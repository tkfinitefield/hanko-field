import 'dart:async';

import '../domain/counter_repository.dart';

class LocalCounterRepository implements CounterRepository {
  int _value = 0;

  @override
  Future<int> load() async {
    // Simulate IO
    await Future<void>.delayed(const Duration(milliseconds: 50));
    return _value;
  }

  @override
  Future<void> save(int value) async {
    await Future<void>.delayed(const Duration(milliseconds: 50));
    _value = value;
  }
}
