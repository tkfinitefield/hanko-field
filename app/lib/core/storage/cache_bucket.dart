import 'package:app/core/storage/cache_policy.dart';

enum CacheBucket {
  designs(
    boxName: 'cache.designs',
    policy: CachePolicy(
      timeToLive: Duration(minutes: 10),
      staleGrace: Duration(hours: 12),
    ),
  ),
  cart(
    boxName: 'cache.cart',
    policy: CachePolicy(
      timeToLive: Duration(hours: 1),
      staleGrace: Duration(days: 7),
    ),
    encrypted: true,
  ),
  guides(
    boxName: 'cache.guides',
    policy: CachePolicy(
      timeToLive: Duration(hours: 12),
      staleGrace: Duration(days: 3),
    ),
  ),
  notifications(
    boxName: 'cache.notifications',
    policy: CachePolicy(
      timeToLive: Duration(minutes: 5),
      staleGrace: Duration(hours: 1),
    ),
    encrypted: true,
  ),
  onboarding(
    boxName: 'state.onboarding',
    policy: CachePolicy(
      timeToLive: Duration(days: 365),
      staleGrace: Duration(days: 365),
    ),
  ),

  /// Developer/sandbox cache bucket useful for demos and unit tests.
  sandbox(
    boxName: 'cache.sandbox',
    policy: CachePolicy(
      timeToLive: Duration(days: 30),
      staleGrace: Duration(days: 30),
    ),
  );

  const CacheBucket({
    required this.boxName,
    required this.policy,
    this.encrypted = false,
  });

  final String boxName;
  final CachePolicy policy;
  final bool encrypted;
}
