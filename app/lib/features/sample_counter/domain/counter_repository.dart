abstract class CounterRepository {
  Future<int> load();
  Future<void> save(int value);
}

