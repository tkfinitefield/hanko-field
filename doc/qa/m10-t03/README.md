# M10-T03 Deep Links and Payment Return Paths

Date: 2026-06-18 JST

Scope: verify that checkout return routes preserve route codes across the
current pilot languages and that platform deep-link declarations include the
localized payment paths needed for release QA.

## Result

PASS for static and unit-level smoke coverage.

- Custom scheme: `hankofield://checkout/*` preserves `lang` for `en`, `ja`,
  `zh`, `zhtw`, and `ar`.
- Android App Links: `finitefield.org` and `www.finitefield.org` declare
  `/payment`, `/en/payment`, `/ja/payment`, `/zh/payment`, `/zhtw/payment`,
  and `/ar/payment`.
- iOS Universal Links: the app entitlement continues to include
  `applinks:finitefield.org` and `applinks:www.finitefield.org`.
- Web payment paths: localized success URLs preserve `return_to=app` and route
  code for `zh`, `zhtw`, and `ar`.
- API Stripe return URLs: normalized order locales generate custom-scheme
  return URLs with `lang=zh`, `lang=zhtw`, or `lang=ar`.

No live device install, hosted `assetlinks.json`, hosted AASA, Stripe network
checkout, Google Play upload, or TestFlight upload was performed in this task.
Those checks remain part of the later release/upload tasks.

## Validation

```sh
jq empty doc/qa/m10-t03/deep-link-payment-return-smoke.json
cd app && flutter test test/checkout_return_test.dart test/platform_deep_link_config_test.dart
cargo test --manifest-path api/Cargo.toml stripe_checkout_return_urls_preserve_normalized_route_code -- --nocapture
cargo test --manifest-path api/Cargo.toml pilot_checkout_return_urls_preserve_arabic_route_code -- --nocapture
cargo test --manifest-path web/Cargo.toml payment_result_urls_preserve_app_return_marker -- --nocapture
cargo test --manifest-path web/Cargo.toml pilot_payment_result_urls_preserve_app_return_route_codes -- --nocapture
cargo test --manifest-path web/Cargo.toml app_checkout_success_page_does_not_auto_redirect_to_custom_scheme -- --nocapture
cargo test --manifest-path web/Cargo.toml pilot_payment_routes_render_localized_copy -- --nocapture
make i18n-ci
git diff --check
git diff --cached --check
```

## Notes

- The app parser now infers the pilot route code from localized Universal
  Link/App Link paths when `lang` or `locale` is absent.
- Existing custom-scheme query behavior remains the primary Stripe app-return
  path.
- Hosted association files are documented in
  `doc/app-release-deep-link-config.md`; this repository does not currently
  commit the production-hosted `assetlinks.json` or AASA files.
- No registry flags, translation values, release enablement, credentials,
  polling, streaming, SSE, or WebSocket behavior changed.
