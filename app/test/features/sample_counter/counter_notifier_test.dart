import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:app/features/sample_counter/application/providers.dart';
import 'package:app/features/sample_counter/domain/counter_repository.dart';

class _FakeRepo implements CounterRepository {
  _FakeRepo(this._value);
  int _value;
  @override
  Future<int> load() async => _value;
  @override
  Future<void> save(int value) async => _value = value;
}

void main() {
  test('loads initial value and increments', () async {
    final container = ProviderContainer(
      overrides: [counterRepositoryProvider.overrideWithValue(_FakeRepo(1))],
    );
    addTearDown(container.dispose);

    final notifier = container.read(counterProvider.notifier);
    // First build triggers load
    final value = await notifier.build();
    expect(value, 1);
    await notifier.increment();
    expect(container.read(counterProvider).value, 2);
  });
}
