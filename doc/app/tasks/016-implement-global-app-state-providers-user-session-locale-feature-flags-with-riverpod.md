# Implement global app state providers (user session, locale, feature flags) with Riverpod.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 016

## Goal
Provide Riverpod providers for session, locale, and feature flags.

## Implementation Steps
1. `AsyncNotifier` for user session that listens to Firebase auth state and backend profile.
2. Locale provider synced with device settings and `/profile/locale` screen updates.
3. Feature flag provider using Remote Config with defaults and caching.
4. Document provider overrides for widget tests.
