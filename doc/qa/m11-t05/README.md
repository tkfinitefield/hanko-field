# M11-T05 Release Runbook Update

Date: 2026-06-18 JST

Scope: finalize the localized release runbook so the next language addition,
store metadata update, fastlane release, and post-release cleanup can proceed
without rediscovering the workflow.

## Result

PASS for the current release state.

- Added `doc/localized-release-runbook.md`.
- Documented future language addition steps from `config/languages.json`
  through translation content, staged flags, and QA gates.
- Documented store metadata source and generated platform output paths.
- Documented Android and iOS fastlane metadata, internal/TestFlight, and
  production lanes.
- Documented required secret inputs and production signoff variables.
- Documented M11 post-release diagnostics, support triage, translation patch,
  and migration cleanup checks.
- Added `make i18n-release-runbook-check` to keep the runbook actionable.

No translation content, registry flags, store release flags, credentials,
production support exports, polling, streaming, SSE, or WebSocket behavior was
changed in this task.

## Validation

```sh
node --check scripts/i18n/release_runbook.mjs
make i18n-release-runbook-check
make i18n-release-runbook-test
jq empty doc/qa/m11-t05/release-runbook-review.json
make i18n-ci
git diff --check
git diff --cached --check
```
