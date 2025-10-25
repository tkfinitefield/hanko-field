enum CacheState { miss, fresh, stale, expired }

/// Defines how long cached content is considered fresh and how long it may be
/// served while a background refresh is happening (stale-while-revalidate).
class CachePolicy {
  const CachePolicy({
    required this.timeToLive,
    this.staleGrace = Duration.zero,
  });

  final Duration timeToLive;
  final Duration staleGrace;

  CacheState evaluate(DateTime cachedAt, DateTime now) {
    final age = now.difference(cachedAt);
    if (age <= timeToLive) {
      return CacheState.fresh;
    }
    if (staleGrace > Duration.zero && age <= timeToLive + staleGrace) {
      return CacheState.stale;
    }
    return CacheState.expired;
  }
}

class CacheReadResult<T> {
  const CacheReadResult._({this.value, required this.state, this.lastUpdated});

  factory CacheReadResult.value({
    required T value,
    required CacheState state,
    required DateTime lastUpdated,
  }) {
    return CacheReadResult._(
      value: value,
      state: state,
      lastUpdated: lastUpdated,
    );
  }

  const CacheReadResult.miss()
    : value = null,
      state = CacheState.miss,
      lastUpdated = null;

  final T? value;
  final CacheState state;
  final DateTime? lastUpdated;

  bool get hasValue => value != null;
  bool get isFresh => state == CacheState.fresh;
  bool get isStale => state == CacheState.stale;
  bool get isMiss => state == CacheState.miss;

  CacheReadResult<T> copyWith({T? value, CacheState? state}) {
    return CacheReadResult._(
      value: value ?? this.value,
      state: state ?? this.state,
      lastUpdated: lastUpdated,
    );
  }
}
