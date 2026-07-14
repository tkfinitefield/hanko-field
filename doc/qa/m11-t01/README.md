# M11-T01 Locale Diagnostics Monitoring

Date: 2026-06-18 JST

Scope: review locale diagnostics for unsupported locale requests, fallback
usage, missing content, checkout locale handling, and malformed translation
events after the multilingual release gate work.

## Result

PASS for the current release state.

- Current release-enabled locales: none.
- Unexpected fallback spikes for release-enabled locales: none.
- Required diagnostic streams are represented in
  `doc/qa/m11-t01/locale-diagnostics-review.json`.
- `make i18n-diagnostics-check` now validates this evidence against
  `config/languages.json`.

No production log export, Cloud Logging query, Google Play rollout, TestFlight
rollout, polling, streaming, SSE, or WebSocket behavior was performed in this
task.

## Reviewed Streams

- `unsupported_locale`: unsupported or unknown locale requests.
- `fallback_locale`: locale fallback decisions and fallback reasons.
- `missing_content`: missing app, web, API, or store content files.
- `checkout_locale`: checkout locale, preferred locale, and route code.
- `malformed_translation`: malformed translation or catalog content.

The current release has no `release.enabled=true` languages, so the release
fallback-spike condition is satisfied by an empty release-enabled set. Future
M11 runs must replace the zero-count local review with real production log
queries after a staged rollout starts.

## Validation

```sh
node --check scripts/i18n/diagnostics.mjs
make i18n-diagnostics-check
make i18n-diagnostics-test
jq empty doc/qa/m11-t01/locale-diagnostics-review.json
make i18n-ci
git diff --check
git diff --cached --check
```

## Follow-up Criteria

Open follow-up issues during `M11-T02` when any of these become true:

- A release-enabled locale records fallback events outside approved holdouts.
- Missing content appears for a release-enabled route, app locale, checkout
  locale, or store locale.
- Checkout locale differs from the expected route code after normalization.
- Malformed translation events appear after the freeze manifest has passed.
- Unsupported locale traffic becomes high enough to justify adding aliases or
  redirect rules.
