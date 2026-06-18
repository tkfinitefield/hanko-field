# M7-T05 Pilot Screenshot QA

Date: 2026-06-18 JST

Scope: pilot language QA for `zh`, `zhtw`, and `ar` across app settings,
app design, app checkout, web top, web about, web payment, and RTL rendering.

## Result

PASS with one tracked tooling limitation.

- Web pilot screenshots were captured from a local mock web server.
- Web `ar` pages now render with `dir="rtl"`.
- Web screenshot probes found no horizontal overflow.
- App pilot settings, design, and checkout surfaces passed focused Flutter
  layout probes with no framework overflow errors.
- Arabic app checkout renders with `TextDirection.rtl`.
- App PNG capture from `flutter test` was attempted, but
  `RenderRepaintBoundary.toImage()` did not return in this local tester
  environment. The limitation is tracked here; the app coverage for this task
  is the focused layout probe rather than committed app PNG files.

## Screenshots

| Surface | Locale | File | Result |
| --- | --- | --- | --- |
| Web top | `zh` | `screenshots/web-zh-top-mobile.png` | PASS |
| Web about | `zhtw` | `screenshots/web-zhtw-about-mobile.png` | PASS |
| Web payment success | `ar` | `screenshots/web-ar-payment-success-mobile.png` | PASS |
| Web RTL about | `ar` | `screenshots/web-ar-about-mobile.png` | PASS |

## App Layout QA

| Surface | Locale | Evidence | Result |
| --- | --- | --- | --- |
| Settings | `zh` | `flutter test test/pilot_layout_qa_test.dart` | PASS |
| Design | `zhtw` | `flutter test test/pilot_layout_qa_test.dart` | PASS |
| Checkout | `ar` | `flutter test test/pilot_layout_qa_test.dart` | PASS |
| RTL overflow probes | `ar` | `flutter test test/rtl_overflow_test.dart` | PASS |

## Web DOM Checks

Browser viewport screenshots were captured through Chrome DevTools from:

```text
http://localhost:3054/zh/
http://localhost:3054/zhtw/about
http://localhost:3054/ar/payment/success?checkout=success&order_id=ord_m7_t05&session_id=cs_test_m7_t05
http://localhost:3054/ar/about
```

Observed DOM checks:

| URL | `lang` | `dir` | Horizontal overflow |
| --- | --- | --- | --- |
| `/zh/` | `zh` | `ltr` | no |
| `/zhtw/about` | `zhtw` | `ltr` | no |
| `/ar/payment/success?...` | `ar` | `rtl` | no |
| `/ar/about` | `ar` | `rtl` | no |

## Validation Commands

```sh
HANKO_WEB_PORT=3054 HANKO_WEB_MODE=mock HANKO_WEB_LOCALE=en cargo run --manifest-path web/Cargo.toml
flutter test test/pilot_layout_qa_test.dart test/rtl_overflow_test.dart
cargo test --manifest-path web/Cargo.toml web_router_resolves_supported_and_unknown_locale_prefixes -- --nocapture
cargo test --manifest-path web/Cargo.toml pilot_payment_routes_render_localized_copy -- --nocapture
sips -g pixelWidth -g pixelHeight doc/qa/m7-t05/screenshots/*.png
```

## Follow-Up

- If committed app PNG evidence is required later, capture it with a live
  Flutter device or a dedicated screenshot integration harness instead of the
  local `flutter test` image path.
