# Implement secure storage, crash reporting (Sentry/Firebase Crashlytics), and analytics instrumentation.

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 010

## Goal
Configure secure storage, crash reporting, and analytics instrumentation.

## Tasks
- Integrate `flutter_secure_storage` for tokens, ensuring keychain/keystore configuration.
- Set up Crashlytics or Sentry with user consent gating and environment tagging.
- Add analytics wrapper exposing typed events and parameter validation.
- Document privacy considerations and opt-out flows.

## Implementation Notes
- Added `flutter_secure_storage` with Android encrypted shared preferences and iOS/macOS keychain accessibility tied to first device unlock. Tokens are handled through `AuthTokenStorage` (`app/lib/core/storage/secure_storage_service.dart`).
- Crash reporting uses Firebase Crashlytics via `CrashReportingController`. Collection defaults to disabled until the stored consent flag is flipped through `updateConsent`. Environment and consent state are attached as custom keys.
- Firebase Analytics is wrapped by `AnalyticsController` and a set of strongly-typed events (`analytics_events.dart`). Collection mirrors the privacy consent flag and validates parameters before logging.
- Privacy choices (`privacy.crash_reporting_allowed`, `privacy.analytics_allowed`) are persisted in secure storage so that opt-in/out survives restarts and is respected before any telemetry starts.
- Opt-out is handled by calling `updateConsent(false)` on the crash or analytics controllers. UI workflows can inject the controllers and toggle consent, which immediately disables collection and updates persisted state.
- When adding surfaces that need telemetry, depend on the controllers/providers rather than calling Firebase APIs directly to ensure validation and consent gating stay centralized.
