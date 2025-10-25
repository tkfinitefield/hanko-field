# Implement local persistence (Hive/Isar/shared_preferences) for caching, offline screen data, and onboarding state.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 013

## Goal
Support offline caching and storing user preferences.

## Implementation Steps
1. Select persistence engine (Hive/Isar) and set up adapters/migrations.
2. Design cache schemas for designs, cart, guides, notifications, onboarding flags.
3. Implement repository caching policies (stale-while-revalidate, TTL).
4. Provide encryption for sensitive data where required.

## Implementation Notes
- Adopted Hive (with `hive_flutter`) as the primary object store and wrapped it in `LocalCacheStore`, exposing typed `CacheBucket`s for designs, cart, guides, notifications, onboarding flags, and a sandbox bucket for demos/tests.
- Defined cache schemas (`CachedDesignList`, `CachedCartSnapshot`, `CachedGuideList`, `CachedNotificationsSnapshot`, `OnboardingFlags`) inside `OfflineCacheRepository`, each using stale-while-revalidate policies tuned per bucket (e.g., designs: 10m fresh/12h stale, cart: 1h fresh/7d stale).
- Added AES encryption for sensitive buckets (cart + notifications) and source-of-truth onboarding flags mirrored between Hive and `SharedPreferences` via `OnboardingLocalDataSource`.
- Wired initialization into `main.dart`, exposed Riverpod providers (`localCacheStoreProvider`, `offlineCacheRepositoryProvider`, `sharedPreferencesProvider`, etc.), and migrated the sample counter repository to read/write through the sandbox cache bucket.
- Added unit tests covering the cache store TTL/encryption behaviour and onboarding data persistence to ensure offline readiness.
