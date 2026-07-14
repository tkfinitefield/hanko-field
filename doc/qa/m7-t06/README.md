# M7-T06 Pilot Public Readiness

Date: 2026-06-18 JST

Scope: decide public readiness for pilot languages `zh`, `zhtw`, and `ar`
after M7 content, checkout, and screenshot QA.

## Decision

Keep all three pilot languages render-only and defer app selection, public web
indexing, and store release.

The app ARBs still contain deferred English fallback entries, so the pilot
languages must not be user-selectable yet. Web indexing also remains blocked
because checked-in blog article bodies and metadata are only localized for
English and Japanese.

Store release remains disabled for all three languages because store metadata,
fastlane, signing guardrails, and broader release-language batches are still
planned in M8 and M9.

## Registry Outcome

| Locale | App enabled | App selectable | Web enabled | Web indexed | Release enabled |
| --- | --- | --- | --- | --- | --- |
| `zh` | true | false | true | false | false |
| `zhtw` | true | false | true | false | false |
| `ar` | true | false | true | false | false |

## Evidence

| Locale | Evidence |
| --- | --- |
| `zh` | Forced app/web rendering and layout probes pass. App selection waits for deferred ARB entries to be translated; web indexing waits for localized blog bodies and metadata. |
| `zhtw` | Forced app/web rendering and layout probes pass. App selection waits for deferred ARB entries to be translated; web indexing waits for localized blog bodies and metadata. |
| `ar` | Forced app/web rendering and RTL probes pass. App selection waits for deferred ARB entries to be translated; web indexing waits for localized blog bodies and metadata. |

## Guardrails

- Public pilot pages remain `noindex,follow` until web indexing can exclude or
  localize blog article surfaces.
- App language settings expose only locales without deferred translation copy.
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
