# Mobile App Architecture (MVVM + Riverpod 3)

This document defines conventions for Hanko Field’s Flutter app. It aligns with a feature-first structure and uses Riverpod 3 without code generation. StateProvider is not used; prefer Notifier/AsyncNotifier and families.

## Principles
- Feature-first modularization: isolate features to simplify ownership and testing.
- Clean layering: Presentation (UI) → Application (ViewModel) → Domain (interfaces/models) → Data (implementations).
- Dependency inversion: UI depends on abstractions. Repositories are defined in Domain and implemented in Data.
- Predictable state: Use AsyncNotifier + AsyncValue for IO-bound flows. Use small sealed state classes when UI needs more than loading/error/data.
- Testability: Everything injectable via providers; override repositories in tests.

## Directory Layout
```
lib/
  core/                 # App-wide plumbing (routing, app state, misc.)
  shared/               # Reusable widgets/utils
    widgets/
    utils/
  features/             # Feature-first modules
    <feature>/
      presentation/     # UI (screens, widgets)
      application/      # ViewModels (Notifier/AsyncNotifier), use-cases
      domain/           # Entities, value objects, repository interfaces
      data/             # Repository implementations, DTOs, mappers

test/
  features/
    <feature>/          # Mirroring lib/features structure
  shared/
  core/
```

Example: `features/sample_counter` is scaffolded as a reference implementation.

## Riverpod Conventions
- Provider types: use `Notifier` and `AsyncNotifier`. Avoid `StateProvider`.
- Naming: `FooNotifier`, `fooProvider`, `fooRepositoryProvider`.
- Provider location: declare module-scoped providers in `application/providers.dart`.
- Families: prefer `.family` for parameterized state (IDs, filters).
- Disposal: default persistent; add `.autoDispose` for ephemeral screens or detail views.
- Mutations: use Riverpod 3 mutations when the UI must react to side-effects (form submit toasts, etc.).

## ViewModel State
- IO-bound flows: `AsyncNotifier<T>` + `AsyncValue<T>` in UI (`when/data/error`).
- Complex flows: sealed state classes in `application/` (e.g., `CheckoutState`).
- One-shot events: prefer mutations; avoid `StreamController` in widgets.

## Repositories and Data
- Define repository interfaces in `domain/` (e.g., `UserRepository`).
- Implement in `data/` (e.g., `UserRepositoryImpl`) with data sources/DTOs/mappers.
- Map platform/HTTP errors to domain errors; do not throw raw exceptions from ViewModels.
- Inject implementations via providers; tests override with fakes.

## Navigation
- Use the app’s `core/routing` module for shells and tabs. Features expose screens in `presentation/`; route wiring remains in `core`.

## Testing
- Use `ProviderContainer` and `overrideWithValue/overrideWithProvider` to inject fakes.
- Focus tests at Application (ViewModel) and Repository levels.
- UI tests verify rendering of `AsyncValue` and key interactions.

## Example Flow (ASCII Diagram)
```
Presentation (Widget/Screen)
      |
      v
Application (AsyncNotifier/ViewModel)  <-- injects --  Domain (Repository interface)
      |                                                      ^
      v                                                      |
Data (Repository impl → API/Local)  ----> Mappers/DTOs ------+
```

## Naming & Files
- Files: snake_case; Classes: PascalCase; Providers: camelCase + `Provider` suffix.
- One type per file; group simple widgets in a single file if tightly coupled.
- Place `providers.dart` in `application/` to centralize DI for the module.

## Sample Module
See `lib/features/sample_counter/` for a minimal end-to-end example including repository, notifier, and screen.

