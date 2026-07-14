# App Release Deep Link Config

This checklist covers Stripe Checkout return routes for the Flutter app release.

## App Links / Universal Links

- Android package: `org.finitefield.hankofield`
- iOS bundle ID: `org.finitefield.hankofield`
- Primary app link domain: `finitefield.org`
- Secondary app link domain: `www.finitefield.org`
- Checkout return paths: `/payment/*`, `/en/payment/*`, `/ja/payment/*`,
  `/zh/payment/*`, `/zhtw/payment/*`, `/ar/payment/*`
- Custom scheme fallback: `hankofield://checkout/*`

## Hosted Association Files

Before release, host the platform association files on both domains.

- Android: `https://finitefield.org/.well-known/assetlinks.json`
- Android: `https://www.finitefield.org/.well-known/assetlinks.json`
- iOS: `https://finitefield.org/.well-known/apple-app-site-association`
- iOS: `https://www.finitefield.org/.well-known/apple-app-site-association`

`assetlinks.json` must include the release signing certificate SHA-256 fingerprint for `org.finitefield.hankofield`. The Apple App Site Association file must include the App ID `<APPLE_TEAM_ID>.org.finitefield.hankofield` and allow the checkout return paths above.

## Stripe Environment

Production Stripe Checkout web return URLs should use the public web domain. App-originated Checkout Sessions should use the app checkout custom scheme so Stripe returns to the installed Flutter app instead of first loading a public web page:

```env
API_PSP_STRIPE_CHECKOUT_SUCCESS_URL=https://finitefield.org/payment/success?session_id={CHECKOUT_SESSION_ID}
API_PSP_STRIPE_CHECKOUT_CANCEL_URL=https://finitefield.org/payment/failure
API_PSP_STRIPE_APP_CHECKOUT_SUCCESS_URL=hankofield://checkout/success?session_id={CHECKOUT_SESSION_ID}
API_PSP_STRIPE_APP_CHECKOUT_CANCEL_URL=hankofield://checkout/cancel
HANKO_WEB_SITE_BASE_URL=https://finitefield.org
```

The web return URLs keep browser checkout on the web site. The app return URLs are used only when the Flutter app sends `return_to_app=true`; those URLs must match the app's registered `hankofield://checkout/*` custom scheme. Universal Links can remain configured as a compatibility path, but app-originated Stripe Checkout should not use `https://finitefield.org/payment/*` because a hosting or association-file issue can leave the customer on a web 404 before returning to the app.

The API appends `checkout=success` or `checkout=cancel`, `order_id`, `lang`, and `return_to=app` for app-originated Checkout Sessions, so the app receives enough context to show success, pending, cancel, or failure states. App-originated Checkout Sessions also set `origin_context=mobile_app`.

## Smoke Test

1. Install a release-signed app build.
2. Verify the custom scheme routes for `en`, `ja`, `zh`, `zhtw`, and `ar`, for
   example `hankofield://checkout/success?checkout=success&order_id=<order>&session_id=<session>&lang=ar`.
3. Start Checkout from the app, pay with Stripe test mode, and confirm the return opens the app at the payment status check screen without showing a web 404.
4. Cancel Checkout and confirm the app opens the canceled state.
5. Verify the Universal Link routes for `en`, `ja`, `zh`, `zhtw`, and `ar`, for
   example `https://finitefield.org/ar/payment/success?checkout=success&order_id=<order>&session_id=<session>&return_to=app`.
6. Open an invalid checkout route and confirm the app shows the deep link error state.
