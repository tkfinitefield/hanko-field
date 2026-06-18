# M11-T02 Support Feedback Triage

Date: 2026-06-18 JST

Scope: triage multilingual release feedback by language, platform, and screen,
then make translation and layout follow-up ownership explicit.

## Result

PASS for the current release state.

- Production rollout has not started.
- Support feedback records reviewed in this pre-rollout gate: 0.
- Translation issues requiring owners: 0.
- Layout issues requiring owners: 0.
- Missing owners: 0.
- `make i18n-support-triage-check` validates the triage evidence shape and
  owner requirements.

No support mailbox export, support form export, Google Play review export, App
Store Connect review export, production release, credentials, polling,
streaming, SSE, or WebSocket behavior was performed in this task.

## Reviewed Sources

- `support_email`: support mailbox search for locale feedback.
- `support_form`: support form or customer contact export.
- `google_play_reviews`: Google Play review export by locale.
- `app_store_reviews`: App Store Connect review export by locale.

The current release has no live production feedback because `M10-T07` is still
blocked. Future M11 runs must replace the zero-count local review with real
support and store-review exports after staged rollout starts.

## Triage Contract

Each support feedback group must record:

- `locale`: a route code from `config/languages.json`.
- `platform`: one of `android`, `ios`, `web`, `api`, `google_play`,
  `app_store`, or `unknown`.
- `screen`: the affected screen or release surface.
- `issues`: feedback items for that language, platform, and screen.

Every `translation` or `layout` issue must have an `owner`. Empty groups are
allowed before rollout, but the evidence must still include the owner policy and
support source review state.

## Validation

```sh
node --check scripts/i18n/support_triage.mjs
make i18n-support-triage-check
make i18n-support-triage-test
jq empty doc/qa/m11-t02/support-feedback-triage.json
make i18n-ci
git diff --check
git diff --cached --check
```
