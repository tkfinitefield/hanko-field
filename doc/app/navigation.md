# Navigation Map & Deep Links

This document defines the mobile app’s navigation map, tab structure, nested flows, and deep link handling. It aligns with the screen list in `doc/app/app_design.md`.

## Tabs & Shell
- Tabs: 作成 (Create) / ショップ (Shop) / 注文 (Orders) / マイ印鑑 (Library) / プロフィール (Profile)
- Default tab: 作成 (Create)
- Back-stack per tab: each tab maintains its own stack. Switching tabs preserves stack state. Android back button pops the current tab’s stack; if empty, returns to default tab; if already on default tab and stack empty, exits app.

Routing architecture
- Router 2.0 with typed routes. `AppRoute` (hierarchical) for primary screens; `IndependentRoute` for modal/sheets/dialog-like pages that can live atop any tab.
- State-driven routing via Riverpod: `appStateProvider` holds `currentRoute`, `currentTab`, and `AppStack`.
- Recommended additions: `AppRouteInformationParser` to translate between URLs and `AppRoute`, and a `BackButtonDispatcher`.

## Route Table (by area)
Startup & Onboarding
- `/splash`
- `/onboarding`
- `/locale`
- `/persona`
- `/auth` (Apple/Google/Email/Guest)

Home & Discovery
- `/home`
- `/search`
- `/notifications`

Create (印影作成)
- `/design/new`
- `/design/input` (+ `/design/input/kanji-map` for foreigner mode)
- `/design/style`
- `/design/editor`
- `/design/ai`
- `/design/check`
- `/design/preview`
- `/design/export`
- `/design/versions`
- `/design/share`

Shop
- `/shop`
- `/materials/:materialId`
- `/products/:productId`
- `/products/:productId/addons`

Cart & Checkout
- `/cart`
- `/checkout/address`
- `/checkout/shipping`
- `/checkout/payment`
- `/checkout/review`
- `/checkout/complete`

Orders
- `/orders`
- `/orders/:orderId`
- `/orders/:orderId/production`
- `/orders/:orderId/tracking`
- `/orders/:orderId/invoice`
- `/orders/:orderId/reorder`

Library (マイ印鑑)
- `/library`
- `/library/:designId`
- `/library/:designId/versions`
- `/library/:designId/duplicate`
- `/library/:designId/export`
- `/library/:designId/shares`

Guides (i18n)
- `/guides`
- `/guides/:slug`
- `/kanji/dictionary`
- `/howto`

Profile & Settings
- `/profile`
- `/profile/addresses`
- `/profile/payments`
- `/profile/notifications`
- `/profile/locale`
- `/profile/legal`
- `/profile/support`
- `/profile/linked-accounts`
- `/profile/export`
- `/profile/delete`

Support & System
- `/support/faq`
- `/support/contact`
- `/support/chat`
- `/status`
- `/permissions`
- `/updates/changelog`
- `/app-update`
- `/offline`
- `/error`

## Typed Routes & Arguments
- Define typed route classes in `lib/core/routing/app_route_configuration.dart` with minimal fields (IDs, enums) and static `key`/`toString` for logging.
- Prefer `String` for IDs (`orderId`, `designId`, `productId`, `materialId`).
- For flows (e.g., editor options), persist state in ViewModels; keep routes lean.

## Navigation Guards
- Auth guard: routes under Orders, Library, Checkout require authenticated user; redirect to `/auth?next=<encoded>`.
- Onboarding guard: if locale/persona not selected, redirect to `/onboarding` or `/locale`/`/persona` as needed.
- Feature flags/maintenance: use Remote Config to disable entry points and show `/status` or inline banners.

## Deep Links
Schemes
- Custom scheme: `hanko://` (app-internal)
- Universal links (preferred): `https://hanko-field.app/…` (placeholder host; set per environment)

Patterns
- Orders: `hanko://orders/{orderId}` or `https://hanko-field.app/orders/{orderId}` → open Orders tab, push Order Detail
- Library: `hanko://library/{designId}` → open Library tab, show Design Detail
- Guides: `hanko://guides/{slug}` → open Guides section
- Products: `hanko://products/{productId}`; Materials: `hanko://materials/{materialId}`
- Checkout: `hanko://checkout/review?cartId=…`
- Auth: `hanko://auth?provider=apple&next=…`

Query params
- `next`: encoded path to continue after auth/onboarding
- `ref`, `utm_*`: analytics only; do not affect routing
- `tab`: explicitly select starting tab (`create|shop|orders|library|profile`)

Handling
- Android: add Intent Filters (custom scheme + https). iOS: add Custom URL Types and Associated Domains.
- Bridge: map incoming `Uri` → `AppRoute` + optional `IndependentRoute` list; set `currentTab` accordingly.
- Fallback: if user lacks permission or resource missing, show `/error` with retry/feedback.

## Back Navigation Rules
- Modal/sheet (`IndependentRoute`) dismisses first.
- If tab stack not empty → pop last independent route.
- Else if not on default tab → switch to default tab.
- Else → exit/minimize (Android) or do nothing (iOS swipe gesture finishes).

## Testing Matrix
- Verify per-platform back behavior (Android hardware, iOS gesture).
- Validate deep links cold/warm-start with and without auth.
- Ensure `next` flow returns to intended page after login/onboarding.

## Implementation Notes
- Add `AppRouteInformationParser` to parse/build URLs.
- Keep parsing pure and covered by unit tests.
- Use Riverpod to wire guards: a derived provider computes allowed target; `AppRouterDelegate` reacts and updates `appStateProvider`.

References
- Screen list: `doc/app/app_design.md`
- Routing core: `app/lib/core/routing/`
