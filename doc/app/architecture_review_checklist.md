# Architecture Code Review Checklist (Flutter + Riverpod 3)

Use this checklist to enforce architecture rules during reviews.

- Directory structure
  - Feature-first under `lib/features/<feature>/{presentation,application,domain,data}`.
  - Shared components live in `lib/shared/`; app plumbing in `lib/core/`.
  - Tests mirror `lib` structure under `test/`.
- Riverpod usage
  - Uses `Notifier`/`AsyncNotifier` (no `StateProvider`).
  - Providers declared in module `application/providers.dart`.
  - Provider names end with `Provider`; notifier classes end with `Notifier`.
  - Parameterized state uses `.family`; temporary providers use `.autoDispose`.
- ViewModel responsibilities
  - No direct HTTP/Firestore/platform calls from UI; only via ViewModels → Repositories.
  - Errors are surfaced as `AsyncError` or mapped domain errors; no raw exceptions to Widgets.
  - Avoids business logic in Widgets; Widgets react to state and dispatch intents.
- Repository boundaries
  - Interfaces in `domain/`; implementations in `data/`.
  - Data mapping (DTO ↔ entity) in `data/`; no DTOs in `presentation`/`application`.
  - Injectable via providers; tests override with fakes.
- State modeling
  - IO flows use `AsyncValue` patterns in UI.
  - Complex flows use small sealed classes instead of nested booleans.
- Testing
  - ViewModel unit tests with `ProviderContainer` and overrides.
  - Repository tests validate mapping and error translation.
  - UI tests verify `AsyncValue` rendering and primary interactions.
- Naming & hygiene
  - File names snake_case; classes PascalCase; private helpers prefixed with `_`.
  - One public type per file; keep widgets small and focused.
  - No dead code, debug prints, or TODOs without owner.

Reference implementation: `lib/features/sample_counter/`.
