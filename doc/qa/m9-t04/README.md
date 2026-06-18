# M9-T04 Tiered Layout QA

Date: 2026-06-18 JST

Scope: localization layout readiness before adding public indexing or release
enablement for the 68-language registry.

## Result

PASS for the current registry state.

- Tier 1 full QA currently applies to `ja`, because Japanese is the only
  non-English locale with `web.indexed=true`.
- Tier 2 screenshot QA currently applies to `ar`, `zh`, and `zhtw`, because
  they remain app-selectable/web-enabled but not indexed or release-enabled.
- Tier 3 mechanical QA currently applies to the remaining non-English route
  languages, because their app, web, and release flags remain disabled.

## Evidence Source

The machine-readable evidence is `layout-qa.json`. It is validated by:

```sh
make i18n-layout-qa-check
```

## Current Guardrail

Before a locale is made `web.indexed=true` or `release.enabled=true`, it must
move into Tier 1 and have passing full layout evidence in `layout-qa.json`.

Before a locale is made app-selectable or web-enabled but not public-indexed,
it must have Tier 2 screenshot/layout evidence in `layout-qa.json`.

Tier 3 locales remain blocked by mechanical translation checks until their
translation batch is ready.
