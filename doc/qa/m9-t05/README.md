# M9-T05 Staged Language Flags

Date: 2026-06-18 JST

Scope: preserve staged language-flag promotion before public indexing or store
release enablement.

## Result

PASS for the current registry state.

- `ja` remains the only non-English `web_indexed` locale.
- `ar`, `zh`, and `zhtw` are `render_only`: app/web rendering stays available
  for forced QA, while app selection, web indexing, and release remain disabled.
- The remaining 63 non-English route languages remain `disabled`.
- The three pilot locales are demoted from app-selectable until their deferred
  translation entries are resolved.

## Transition Order

1. `disabled`
2. `render_only`
3. `app_selectable`
4. `web_indexed`
5. `store_release_enabled`

Each transition kind must be made in a separate PR or commit with fresh
validation evidence. The current M9-T05 baseline records the existing state and
adds a check that fails when `config/languages.json` changes without matching
stage evidence.

## Evidence Source

The machine-readable evidence is `flag-stages.json`. It is validated by:

```sh
make i18n-flag-stages-check
```

## Guardrails

- `app.selectable=true` requires `app.enabled=true`.
- Deferred translation entries block app selection, web indexing, and store
  release.
- `web.indexed=true` requires `web.enabled=true`.
- `release.enabled=true` requires app selectable, web indexed, Android and iOS
  store locale mappings, and store metadata source.
- Store-release-enabled locales must also have store metadata, fastlane config,
  secret guardrail, holdout, layout, and i18n evidence.
