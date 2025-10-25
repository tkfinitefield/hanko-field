import 'package:app/core/data/dtos/design_dto.dart';
import 'package:app/core/storage/cache_bucket.dart';
import 'package:app/core/storage/cache_policy.dart';
import 'package:app/core/storage/local_cache_store.dart';

class OfflineCacheRepository {
  OfflineCacheRepository(this._store);

  final LocalCacheStore _store;

  Future<CacheReadResult<CachedDesignList>> readDesignList({
    String key = LocalCacheStore.defaultEntryKey,
  }) {
    return _store.read(
      bucket: CacheBucket.designs,
      key: key,
      decoder: (data) => CachedDesignList.fromJson(_asJson(data)),
    );
  }

  Future<void> writeDesignList(
    CachedDesignList payload, {
    String key = LocalCacheStore.defaultEntryKey,
  }) {
    return _store.write(
      bucket: CacheBucket.designs,
      key: key,
      encoder: (value) => value.toJson(),
      value: payload,
    );
  }

  Future<CacheReadResult<CachedCartSnapshot>> readCart() {
    return _store.read(
      bucket: CacheBucket.cart,
      decoder: (data) => CachedCartSnapshot.fromJson(_asJson(data)),
    );
  }

  Future<void> writeCart(CachedCartSnapshot payload) {
    return _store.write(
      bucket: CacheBucket.cart,
      encoder: (value) => value.toJson(),
      value: payload,
    );
  }

  Future<CacheReadResult<CachedGuideList>> readGuides({
    String key = LocalCacheStore.defaultEntryKey,
  }) {
    return _store.read(
      bucket: CacheBucket.guides,
      key: key,
      decoder: (data) => CachedGuideList.fromJson(_asJson(data)),
    );
  }

  Future<void> writeGuides(
    CachedGuideList payload, {
    String key = LocalCacheStore.defaultEntryKey,
  }) {
    return _store.write(
      bucket: CacheBucket.guides,
      key: key,
      encoder: (value) => value.toJson(),
      value: payload,
    );
  }

  Future<CacheReadResult<CachedNotificationsSnapshot>> readNotifications() {
    return _store.read(
      bucket: CacheBucket.notifications,
      decoder: (data) => CachedNotificationsSnapshot.fromJson(_asJson(data)),
    );
  }

  Future<void> writeNotifications(CachedNotificationsSnapshot payload) {
    return _store.write(
      bucket: CacheBucket.notifications,
      encoder: (value) => value.toJson(),
      value: payload,
    );
  }

  Future<CacheReadResult<OnboardingFlags>> readOnboardingFlags() {
    return _store.read(
      bucket: CacheBucket.onboarding,
      decoder: (data) => OnboardingFlags.fromJson(_asJson(data)),
    );
  }

  Future<void> writeOnboardingFlags(OnboardingFlags flags) {
    return _store.write(
      bucket: CacheBucket.onboarding,
      encoder: (value) => value.toJson(),
      value: flags,
    );
  }
}

class CachedDesignList {
  CachedDesignList({required this.items, this.nextPageToken});

  factory CachedDesignList.fromJson(Map<String, dynamic> json) {
    final list = (json['items'] as List<dynamic>? ?? <dynamic>[])
        .map(
          (item) => DesignDto.fromJson(Map<String, dynamic>.from(item as Map)),
        )
        .toList();
    return CachedDesignList(
      items: list,
      nextPageToken: json['nextPageToken'] as String?,
    );
  }

  final List<DesignDto> items;
  final String? nextPageToken;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'items': items.map((dto) => dto.toJson()).toList(),
      'nextPageToken': nextPageToken,
    };
  }
}

class CachedCartSnapshot {
  CachedCartSnapshot({
    required this.lines,
    this.currency,
    this.subtotal,
    this.total,
    this.updatedAt,
  });

  factory CachedCartSnapshot.fromJson(Map<String, dynamic> json) {
    return CachedCartSnapshot(
      lines: (json['lines'] as List<dynamic>? ?? <dynamic>[])
          .map(
            (line) =>
                CartLineCache.fromJson(Map<String, dynamic>.from(line as Map)),
          )
          .toList(),
      currency: json['currency'] as String?,
      subtotal: (json['subtotal'] as num?)?.toDouble(),
      total: (json['total'] as num?)?.toDouble(),
      updatedAt: json['updatedAt'] == null
          ? null
          : DateTime.parse(json['updatedAt'] as String),
    );
  }

  final List<CartLineCache> lines;
  final String? currency;
  final double? subtotal;
  final double? total;
  final DateTime? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'lines': lines.map((line) => line.toJson()).toList(),
      'currency': currency,
      'subtotal': subtotal,
      'total': total,
      'updatedAt': updatedAt?.toIso8601String(),
    };
  }
}

class CartLineCache {
  CartLineCache({
    required this.lineId,
    required this.productId,
    required this.quantity,
    this.designSnapshot,
    this.price,
    this.currency,
    this.addons,
  });

  factory CartLineCache.fromJson(Map<String, dynamic> json) {
    return CartLineCache(
      lineId: json['lineId'] as String,
      productId: json['productId'] as String,
      quantity: json['quantity'] as int,
      designSnapshot: json['designSnapshot'] == null
          ? null
          : DesignDto.fromJson(
              Map<String, dynamic>.from(json['designSnapshot'] as Map),
            ),
      price: (json['price'] as num?)?.toDouble(),
      currency: json['currency'] as String?,
      addons: json['addons'] == null
          ? null
          : Map<String, dynamic>.from(json['addons'] as Map),
    );
  }

  final String lineId;
  final String productId;
  final int quantity;
  final DesignDto? designSnapshot;
  final double? price;
  final String? currency;
  final Map<String, dynamic>? addons;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'lineId': lineId,
      'productId': productId,
      'quantity': quantity,
      'designSnapshot': designSnapshot?.toJson(),
      'price': price,
      'currency': currency,
      'addons': addons,
    };
  }
}

class CachedGuideList {
  CachedGuideList({required this.guides, this.locale, this.updatedAt});

  factory CachedGuideList.fromJson(Map<String, dynamic> json) {
    final guides = (json['guides'] as List<dynamic>? ?? <dynamic>[])
        .map(
          (guide) =>
              GuideCacheItem.fromJson(Map<String, dynamic>.from(guide as Map)),
        )
        .toList();
    return CachedGuideList(
      guides: guides,
      locale: json['locale'] as String?,
      updatedAt: json['updatedAt'] == null
          ? null
          : DateTime.parse(json['updatedAt'] as String),
    );
  }

  final List<GuideCacheItem> guides;
  final String? locale;
  final DateTime? updatedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'guides': guides.map((guide) => guide.toJson()).toList(),
      'locale': locale,
      'updatedAt': updatedAt?.toIso8601String(),
    };
  }
}

class GuideCacheItem {
  GuideCacheItem({
    required this.slug,
    required this.title,
    required this.summary,
    required this.featured,
    this.heroImage,
    List<String>? tags,
  }) : tags = tags ?? <String>[];

  factory GuideCacheItem.fromJson(Map<String, dynamic> json) {
    return GuideCacheItem(
      slug: json['slug'] as String,
      title: json['title'] as String,
      summary: json['summary'] as String,
      featured: json['featured'] as bool? ?? false,
      heroImage: json['heroImage'] as String?,
      tags: (json['tags'] as List<dynamic>? ?? <dynamic>[])
          .map((tag) => tag as String)
          .toList(),
    );
  }

  final String slug;
  final String title;
  final String summary;
  final bool featured;
  final String? heroImage;
  final List<String> tags;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'slug': slug,
      'title': title,
      'summary': summary,
      'featured': featured,
      'heroImage': heroImage,
      'tags': tags,
    };
  }
}

class CachedNotificationsSnapshot {
  CachedNotificationsSnapshot({
    required this.items,
    required this.unreadCount,
    this.lastSyncedAt,
  });

  factory CachedNotificationsSnapshot.fromJson(Map<String, dynamic> json) {
    final items = (json['items'] as List<dynamic>? ?? <dynamic>[])
        .map(
          (item) => NotificationCacheItem.fromJson(
            Map<String, dynamic>.from(item as Map),
          ),
        )
        .toList();
    return CachedNotificationsSnapshot(
      items: items,
      unreadCount: json['unreadCount'] as int? ?? 0,
      lastSyncedAt: json['lastSyncedAt'] == null
          ? null
          : DateTime.parse(json['lastSyncedAt'] as String),
    );
  }

  final List<NotificationCacheItem> items;
  final int unreadCount;
  final DateTime? lastSyncedAt;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'items': items.map((item) => item.toJson()).toList(),
      'unreadCount': unreadCount,
      'lastSyncedAt': lastSyncedAt?.toIso8601String(),
    };
  }
}

class NotificationCacheItem {
  NotificationCacheItem({
    required this.id,
    required this.title,
    required this.body,
    required this.timestamp,
    this.read = false,
    this.deepLink,
  });

  factory NotificationCacheItem.fromJson(Map<String, dynamic> json) {
    return NotificationCacheItem(
      id: json['id'] as String,
      title: json['title'] as String,
      body: json['body'] as String,
      timestamp: DateTime.parse(json['timestamp'] as String),
      read: json['read'] as bool? ?? false,
      deepLink: json['deepLink'] as String?,
    );
  }

  final String id;
  final String title;
  final String body;
  final DateTime timestamp;
  final bool read;
  final String? deepLink;

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'id': id,
      'title': title,
      'body': body,
      'timestamp': timestamp.toIso8601String(),
      'read': read,
      'deepLink': deepLink,
    };
  }
}

enum OnboardingStep { tutorial, locale, persona, notifications }

class OnboardingFlags {
  OnboardingFlags({
    required Map<OnboardingStep, bool> steps,
    DateTime? updatedAt,
  }) : stepCompletion = Map.unmodifiable(steps),
       updatedAt = updatedAt ?? DateTime.now();

  factory OnboardingFlags.initial() {
    return OnboardingFlags(
      steps: {for (final step in OnboardingStep.values) step: false},
      updatedAt: DateTime.fromMillisecondsSinceEpoch(0),
    );
  }

  factory OnboardingFlags.fromJson(Map<String, dynamic> json) {
    final rawSteps = Map<String, dynamic>.from(
      json['steps'] as Map? ?? <String, dynamic>{},
    );
    final steps = {
      for (final entry in rawSteps.entries)
        _parseStep(entry.key): entry.value as bool? ?? false,
    };
    final updatedAtRaw = json['updatedAt'] as String?;
    final updatedAt = updatedAtRaw == null
        ? DateTime.fromMillisecondsSinceEpoch(0)
        : DateTime.tryParse(updatedAtRaw) ??
              DateTime.fromMillisecondsSinceEpoch(0);
    return OnboardingFlags(steps: steps, updatedAt: updatedAt);
  }

  final Map<OnboardingStep, bool> stepCompletion;
  final DateTime updatedAt;

  bool get isCompleted => stepCompletion.values.every((done) => done);

  OnboardingFlags markStep(OnboardingStep step, bool completed) {
    final next = Map<OnboardingStep, bool>.from(stepCompletion)
      ..[step] = completed;
    return OnboardingFlags(steps: next, updatedAt: DateTime.now());
  }

  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'steps': {
        for (final entry in stepCompletion.entries)
          _stepKey(entry.key): entry.value,
      },
      'updatedAt': updatedAt.toIso8601String(),
    };
  }

  static String _stepKey(OnboardingStep step) {
    switch (step) {
      case OnboardingStep.tutorial:
        return 'tutorial';
      case OnboardingStep.locale:
        return 'locale';
      case OnboardingStep.persona:
        return 'persona';
      case OnboardingStep.notifications:
        return 'notifications';
    }
  }

  static OnboardingStep _parseStep(String value) {
    switch (value) {
      case 'tutorial':
        return OnboardingStep.tutorial;
      case 'locale':
        return OnboardingStep.locale;
      case 'persona':
        return OnboardingStep.persona;
      case 'notifications':
        return OnboardingStep.notifications;
    }
    throw ArgumentError.value(value, 'value', 'Unknown onboarding step');
  }
}

Map<String, dynamic> _asJson(Object? data) {
  return Map<String, dynamic>.from((data as Map?) ?? <String, dynamic>{});
}
