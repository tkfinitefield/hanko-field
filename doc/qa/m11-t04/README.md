# M11-T04 Migration Wrapper Cleanup

Date: 2026-06-18 JST

Scope: remove temporary localization migration wrappers so app runtime uses the
generated localization output and the shared language registry as the active
localization mechanisms.

## Result

PASS for the current release state.

- Removed the `HankoLocalizations` typedef.
- Removed the hardcoded `hankoSupportedLocales` constant.
- Removed the `hankoLocalizationsDelegates` constant.
- `MaterialApp` reads delegates from `GeneratedHankoLocalizations` and limits
  runtime supported locales to registry entries with `app.enabled=true`.
- Automatic platform and saved-locale selection is further limited to
  registry entries with `app.selectable=true`.
- App code now types localized helper inputs as `GeneratedHankoLocalizations`.
- `make i18n-migration-cleanup-check` validates that these temporary wrappers
  do not return.

The existing `preferred_language_code` upgrade fallback remains intentionally
retained. It is user-state migration safety, not an active translation lookup
mechanism, and removing it would silently discard a previously saved language
preference for upgraded installs.

The 2026-07-14 review updates the Japanese onboarding copy from `origin/main`
and demotes deferred pilot locales from app-selectable to render-only. Store
release flags, credentials, production support exports, polling, streaming,
SSE, and WebSocket behavior remain unchanged.

## Active Localization Mechanisms

- Flutter generated localization:
  `app/lib/l10n/generated/generated_hanko_localizations.dart`.
- Shared app language registry:
  `config/languages.json` loaded through
  `app/lib/app/localization/language_registry.dart`.

## Validation

```sh
node --check scripts/i18n/migration_cleanup.mjs
make i18n-migration-cleanup-check
make i18n-migration-cleanup-test
jq empty doc/qa/m11-t04/migration-cleanup.json
make i18n-check
make i18n-ci
cd app && flutter test test/generated_hanko_localizations_test.dart
git diff --check
git diff --cached --check
```
