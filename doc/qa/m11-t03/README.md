# M11-T03 High-Priority Translation Patch Review

Date: 2026-06-18 JST

Scope: patch high-priority translation issues found during M11 support triage
with small content-only changes, then verify the localization set with
`i18n-check`.

## Result

PASS for the current release state.

- Source triage evidence: `doc/qa/m11-t02/support-feedback-triage.json`.
- High-priority translation issues found in M11-T02: 0.
- Content-only translation patches applied: 0.
- Store-release-enabled locales: none.
- `make i18n-translation-patches-check` validates that future high-priority
  translation issues are covered by content-only patches with `i18n-check`
  evidence.

No translation content, registry flags, store release flags, credentials,
production support exports, polling, streaming, SSE, or WebSocket behavior was
changed in this task.

## Patch Contract

When M11-T02 records a `critical` or `high` severity `translation` issue,
`doc/qa/m11-t03/translation-patch-review.json` must add a matching patch entry.

Each patch entry must include:

- `source_issue_id`: the M11-T02 support issue id.
- `locale`: a route code from `config/languages.json`.
- `owner`: the person or role responsible for the patch.
- `status`: `pass`.
- `files`: one or more localization content files.
- `validation`: passing `make i18n-check` evidence.

Allowed patch file roots are limited to localization content paths:

- `app/lib/l10n/`
- `app/assets/i18n/`
- `api/content/i18n/`
- `web/content/i18n/`
- `release/store_metadata/source/`

## Validation

```sh
node --check scripts/i18n/translation_patches.mjs
make i18n-translation-patches-check
make i18n-translation-patches-test
jq empty doc/qa/m11-t03/translation-patch-review.json
make i18n-check
make i18n-ci
git diff --check
git diff --cached --check
```
