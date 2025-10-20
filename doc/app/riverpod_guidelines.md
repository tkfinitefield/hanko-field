# Riverpod 3 Usage Guidelines & DI Strategy

Scope: Flutter app uses Riverpod 3 without code generation. Do not use `StateProvider`. Prefer `Notifier`/`AsyncNotifier`, `Provider`, `FutureProvider`, and families.

## Provider Categories & Lifecycle
- App-level (long-lived): Global app state, routing, feature flags. Location: `lib/core/`.
- Feature-level (scoped): Business logic for a domain feature. Location: `lib/features/<feature>/application/providers.dart`.
- Ephemeral UI (short-lived): Screen-specific caches/derived state. Use `.autoDispose` or widget-local `State` if there is no business logic.

Rules
- One module owns its providers. Do not import `presentation/` from other modules. Cross-feature interactions go through repositories/services or `core`.
- Use `.family` for parameterized state (IDs, filters) and `.autoDispose` for detail views to avoid leaks.
- Avoid global singletons; use providers as the DI container.

## Naming & File Layout
- Providers: `fooProvider`, `fooRepositoryProvider`. Notifiers: `FooNotifier`.
- Feature providers live in `application/providers.dart` and re-exported as needed.
- One public type per file; group tiny widgets only when tightly coupled.

## Dependency Injection (No Codegen)
Pattern
```dart
// domain
abstract class UserRepository { Future<User> get(String id); }

// data
class ApiUserRepository implements UserRepository { /* ... */ }

// application/providers.dart
final userRepositoryProvider = Provider<UserRepository>((ref) {
  return ApiUserRepository(/* deps from other providers */);
});

final userProvider = AsyncNotifierProvider.family<UserNotifier, User, String>(
  UserNotifier.new,
);

class UserNotifier extends AsyncNotifier<User> {
  UserNotifier(this.id);
  final String id; // family argument
  @override
  Future<User> build() async {
    final repo = ref.watch(userRepositoryProvider);
    return repo.get(id);
  }
}
```

Overrides
- App bootstrap uses `ProviderScope(overrides: [...])` to swap implementations (dev/stg/prod) or to inject fakes in tests.
```dart
ProviderScope(
  overrides: [
    userRepositoryProvider.overrideWith((ref) => ApiUserRepository(/* prod deps */)),
  ],
  child: const App(),
);
```

Testing
```dart
final c = ProviderContainer(overrides: [
  userRepositoryProvider.overrideWithValue(FakeUserRepository()),
]);
addTearDown(c.dispose);
```

## Async State & Errors
- IO flows use `AsyncNotifier<T>` and `AsyncValue<T>` in UI via `when`/`maybeWhen`.
- Set `state = const AsyncLoading()` before long ops; on success `state = AsyncData(value)`; on failure `state = AsyncError(e, st)`.
- Guard async callbacks with `if (!ref.mounted) return;` before mutating state after awaits.
- Map transport/platform errors to domain errors in repositories. Widgets should not handle raw exceptions.

Retry & Refresh
- Use `ref.invalidate(someProvider)` to refetch; for families, invalidate the specific instance.
- For periodic refresh, use `ref.onDispose` to clean timers; prefer repository-level caching with TTL.

## Listening Patterns
- `ref.watch(provider)`: rebuild UI on change.
- `ref.listen(provider, (prev, next) { ... })`: perform side-effects (toasts, navigation) without rebuilds.
- `ref.listenManual(...)`: advanced pattern for non-widget classes (e.g., router delegate) where manual lifecycle control is needed.

## Mutations (Side-Effects)
- Use Riverpod 3 `Mutation` for one-shot actions (form submit) to expose loading/success/error to UI and keep providers alive during work.
- UI observes mutation state and triggers `mutation.run(ref, (tsx) async { ... })` to call into notifiers.

## When to Use Families vs Arguments
- Families: key the provider by an ID/query. Prefer this to passing arguments through constructors.
- Arguments inside methods: use for transient actions where persistent state is not needed.

## Anti-Patterns
- Do not use `StateProvider` for business logic or IO.
- Do not instantiate repositories/services directly in widgets; always go through providers.
- Do not import `data/` from `presentation/`; depend on `application/` or `domain/` abstractions only.
- Avoid global singletons or service locators; Riverpod is the DI container.

## References & Examples
- Sample module: `lib/features/sample_counter/` demonstrates repository DI, `AsyncNotifier`, overrides in tests.
- App-level routing state: `lib/core/routing/` shows `NotifierProvider` for app state and manual listen in router delegate.

