# Mobile App Implementation Task List

## 0. Planning & Architecture
- [ ] Validate Flutter app scope, personas, supported platforms (iOS/Android), and release milestones from `doc/app/app_design.md`.
- [ ] Define MVVM architecture conventions (view/widget, view-model, repository layers) and directory structure.
- [ ] Establish Riverpod usage guidelines (Notifier/AsyncNotifier, providers scoping) and dependency injection strategy without code generation.
- [ ] Document navigation map (routes, tabs, nested flows) and deep link handling for app sections.
- [ ] Create API contract checklist aligning mobile payloads with backend endpoints for all flows.

## 1. Project Setup & Tooling
- [ ] Initialize Flutter project with flavors (dev/stg/prod), app icons, splash screens, and build configurations.
- [ ] Configure linting/formatting (analysis_options), CI pipeline (flutter analyze/test), and code coverage thresholds.
- [ ] Set up localization tooling (ARB files), theming, typography, and design tokens shared across screens.
- [ ] Integrate Firebase services (Auth, Messaging, Remote Config) and configure environment-specific options.
- [ ] Implement secure storage, crash reporting (Sentry/Firebase Crashlytics), and analytics instrumentation.

## 2. Core Infrastructure & Shared Components
- [ ] Implement networking layer with HTTP client, interceptors (auth, logging), retries, and response parsing.
- [ ] Build API data models, DTOs, and repository interfaces for users, designs, catalog, orders, promotions, content.
- [ ] Implement local persistence (Hive/Isar/shared_preferences) for caching, offline screen data, and onboarding state.
- [ ] Create shared widgets (buttons, form fields, modals, cards, list skeletons) following design system.
- [ ] Develop navigation shell with bottom tabs (`作成/ショップ/注文/マイ印鑑/プロフィール`), nested navigators, and deep link support.
- [ ] Implement global app state providers (user session, locale, feature flags) with Riverpod.
- [ ] Create notification bell UI, search entry points, and help overlays accessible from top app bar.

## 3. Onboarding & Auth Flow
- [ ] Implement splash screen logic (`/splash`) checking auth state, app version, feature flags.
- [ ] Build onboarding/tutorial screens (`/onboarding`) with skipping, progress indicator, and analytics events.
- [ ] Implement locale selection (`/locale`) and persona selection (`/persona`) storing preferences locally and server-side.
- [ ] Build authentication flow (`/auth`) supporting Apple Sign-In, Google, Email, and guest mode; handle link with Firebase Auth.
- [ ] Implement language/region and persona gating to drive downstream UI states.

## 4. Home & Discovery
- [ ] Implement home screen (`/home`) showing featured items, recent designs, and recommended templates using async providers.
- [ ] Build search screen (`/search`) with global search bar, segmented results (templates/materials/articles/FAQ), and search history.
- [ ] Implement notifications list (`/notifications`) with pagination, read/unread state, and push navigation handling.

## 5. Design Creation Flow (作成タブ)
- [ ] Implement design type selection (`/design/new`) with entry points for text/upload/logo flows.
- [ ] Build name input screen (`/design/input`) with validation, localization, and optional kanji mapping entry point.
- [ ] Implement kanji mapping flow (`/design/input/kanji-map`) displaying candidate list, meanings, and selection persistence.
- [ ] Build style selection (`/design/style`) with preview carousel, filtering by script/shape, and template fetching.
- [ ] Implement design editor (`/design/editor`) with canvas controls (layout, stroke, margins, rotation, grid) using custom painter widgets.
- [ ] Integrate AI suggestions (`/design/ai`) showing queued/completed proposals, comparison, and accept/reject actions.
- [ ] Implement registrability check (`/design/check`) calling external service, showing result badges.
- [ ] Build preview screen (`/design/preview`) with actual size view, backgrounds, zoom, and share triggers.
- [ ] Implement digital export (`/design/export`) generating PNG/SVG, handling permissions and download/share sheets.
- [ ] Implement version history (`/design/versions`) with diff view and rollback actions.
- [ ] Build share screen (`/design/share`) generating mocked social posts and watermarked images.

## 6. Shop & Product Browsing
- [ ] Implement shop home (`/shop`) with category tiles, promotions, and recommended materials.
- [ ] Build material detail screen (`/materials/:materialId`) showing specs, gallery, availability.
- [ ] Implement product detail (`/products/:productId`) with variant selector, pricing tiers, stock indicators.
- [ ] Build add-ons screen (`/products/:productId/addons`) for optional accessories with upsell logic.

## 7. Cart & Checkout
- [ ] Implement cart screen (`/cart`) with line editing, quantity adjustments, promo code entry, and estimate summary.
- [ ] Build checkout address screen (`/checkout/address`) with address list, add/edit forms, and validation (JP/international formats).
- [ ] Implement shipping selection (`/checkout/shipping`) supporting domestic/international options and delivery estimates.
- [ ] Build payment method screen (`/checkout/payment`) integrating tokenized payment refs and adding new methods if allowed.
- [ ] Implement review screen (`/checkout/review`) showing order summary, design snapshot, totals, and terms acknowledgement.
- [ ] Build order completion screen (`/checkout/complete`) displaying confirmation, next steps, and share options.

## 8. Orders & Tracking
- [ ] Implement orders list (`/orders`) with filters, status chips, and infinite scroll.
- [ ] Build order detail (`/orders/:orderId`) showing line items, totals, addresses, and design snapshot gallery.
- [ ] Implement production timeline (`/orders/:orderId/production`) visualizing stages and timestamps.
- [ ] Build shipment tracking (`/orders/:orderId/tracking`) with event timeline and carrier integration.
- [ ] Implement invoice viewer (`/orders/:orderId/invoice`) with PDF download/share support.
- [ ] Build reorder flow (`/orders/:orderId/reorder`) cloning cart data and redirecting to checkout.

## 9. My Hanko Library
- [ ] Implement library list (`/library`) with sorting, filtering (status, date, AI score), and grid/list toggle.
- [ ] Build design detail (`/library/:designId`) showing metadata, AI score, registrability status, usage history, and quick actions.
- [ ] Implement versions view (`/library/:designId/versions`) reusing diff/rollback components.
- [ ] Build duplicate flow (`/library/:designId/duplicate`) creating new design entry and navigating to editor.
- [ ] Implement digital export screen (`/library/:designId/export`) with formats and permissions.
- [ ] Build share link management (`/library/:designId/shares`) showing issued links, expiry, revoke.

## 10. Guides & Cultural Content
- [ ] Implement guides list (`/guides`) with localization filters and recommended content for personas.
- [ ] Build guide detail (`/guides/:slug`) rendering CMS content with markdown/HTML and offline caching.
- [ ] Implement kanji dictionary (`/kanji/dictionary`) with search, favorites, and integration with design input.
- [ ] Build how-to screen (`/howto`) aggregating tutorials/videos with embedded players.

## 11. Profile & Settings
- [ ] Implement profile home (`/profile`) showing avatar, display name, persona toggle, quick links.
- [ ] Build addresses management (`/profile/addresses`) with CRUD, defaults, and shipping sync.
- [ ] Implement payment methods management (`/profile/payments`) referencing PSP tokens, default selection, and removal.
- [ ] Build notifications settings (`/profile/notifications`) for push/email categories and scheduling.
- [ ] Implement locale settings (`/profile/locale`) for language/currency overrides.
- [ ] Build legal documents screen (`/profile/legal`) rendering static content with offline availability.
- [ ] Implement support screen (`/profile/support`) linking to FAQ, chat, contact forms.
- [ ] Build linked accounts screen (`/profile/linked-accounts`) showing social auth connections and unlink flow.
- [ ] Implement data export (`/profile/export`) generating archive and downloading securely.
- [ ] Build account delete flow (`/profile/delete`) with confirmation steps and backend call.

## 12. Support & Status
- [ ] Implement FAQ screen (`/support/faq`) with categories, search, and offline caching.
- [ ] Build contact form (`/support/contact`) with ticket creation and file attachment uploading.
- [ ] Implement chat support (`/support/chat`) integrating bot handoff to live agent with push notifications.
- [ ] Build system status screen (`/status`) showing current incidents and historical uptime.

## 13. System Utilities
- [ ] Implement permissions onboarding (`/permissions`) prompting for photo/storage/notification access with rationale.
- [ ] Build changelog screen (`/updates/changelog`) with version history and feature highlights.
- [ ] Implement forced app update flow (`/app-update`) checking version constraints and gating access.
- [ ] Build offline screen (`/offline`) with retry and cached content access.
- [ ] Implement generic error screen (`/error`) with diagnostics and support links.

## 14. Notifications & Messaging
- [ ] Integrate push notification handling (background/foreground) and routing to relevant screens.
- [ ] Implement in-app messaging/toast system for success, warnings, alerts tied to Riverpod providers.
- [ ] Provide notification inbox sync and badge counts shared between app bar and tabs.

## 15. Analytics, Telemetry, and Monitoring
- [ ] Define analytics events for key flows (design creation, checkout, share) and instrument across view models.
- [ ] Configure performance monitoring (Firebase Performance) and custom metrics for screen load times.
- [ ] Implement remote config/feature flag handling for gradual rollout of features.
- [ ] Set up logging/trace pipeline for client errors and attach device context.

## 16. Accessibility, Localization, and QA
- [ ] Ensure accessibility compliance (semantics, focus order, color contrast, screen reader labels) across screens.
- [ ] Complete full localization pass (copy extraction, pluralization, RTL readiness if needed).
- [ ] Implement automated widget tests, integration tests (golden tests, end-to-end flows) covering core journeys.
- [ ] Prepare manual QA checklist and device matrix for release certification.

## 17. Release Management & Distribution
- [ ] Configure App Store / Google Play metadata, screenshots, privacy manifests, and release notes workflow.
- [ ] Set up build automation (Fastlane/Codemagic) for beta and production releases with environment variables.
- [ ] Establish beta testing program (TestFlight/Play Console) and feedback loop ingestion.
- [ ] Document release checklist including rollback plan, monitoring, and post-release analytics review.
