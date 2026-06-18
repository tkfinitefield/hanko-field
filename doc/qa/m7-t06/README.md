# M7-T06 Pilot Public Readiness

Date: 2026-06-18 JST

Scope: decide public readiness for pilot languages `zh`, `zhtw`, and `ar`
after M7 content, checkout, and screenshot QA.

## Decision

Keep all three pilot languages app-selectable and defer public web indexing.

The blocking reason is that `web.indexed` is currently language-wide: enabling
it would also index blog index and article routes, while checked-in blog article
bodies and metadata are only localized for English and Japanese.

Store release remains disabled for all three languages because store metadata,
fastlane, signing guardrails, and broader release-language batches are still
planned in M8 and M9.

## Registry Outcome

| Locale | App enabled | App selectable | Web enabled | Web indexed | Release enabled |
| --- | --- | --- | --- | --- | --- |
| `zh` | true | true | true | false | false |
| `zhtw` | true | true | true | false | false |
| `ar` | true | true | true | false | false |

## Evidence

| Locale | Evidence |
| --- | --- |
| `zh` | App content, web static/payment content, API catalog, checkout routing, web top screenshot, and app Settings layout probe passed. Web indexing is deferred until localized blog article bodies and metadata exist. |
| `zhtw` | App content, web static/payment content, API catalog, checkout routing, web about screenshot, and app Design layout probe passed. Web indexing is deferred until localized blog article bodies and metadata exist. |
| `ar` | App content, web static/payment content, API catalog, Arabic checkout, RTL web screenshots, and app Checkout RTL layout probe passed. Web indexing is deferred until localized blog article bodies and metadata exist. |

## Guardrails

- Public pilot pages remain `noindex,follow` until web indexing can exclude or
  localize blog article surfaces.
- Payment result pages remain `noindex,follow`.
- `release.enabled` remains `false` until M8/M9 release work is complete.
- No store metadata, fastlane, signing, polling, streaming, SSE, or WebSocket
  behavior changes are included in this task.

## Validation Commands

```sh
jq empty config/languages.json doc/qa/m7-t06/readiness.json
make i18n-check
cargo fmt --manifest-path web/Cargo.toml -- --check
cargo test --manifest-path web/Cargo.toml web_router_resolves_supported_and_unknown_locale_prefixes -- --nocapture
cargo test --manifest-path web/Cargo.toml
make i18n-ci
git diff --check
```
