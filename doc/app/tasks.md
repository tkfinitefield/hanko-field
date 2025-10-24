# Mobile App Implementation Task List

## 0. Planning & Architecture
- [x] [Validate Flutter app scope, personas, supported platforms (iOS/Android), and release milestones from `doc/app/app_design.md`.](doc/app/tasks/001-validate-flutter-app-scope-personas-supported-platforms-ios-android-and-release-milestones.md)
- [x] [Define MVVM architecture conventions (view/widget, view-model, repository layers) and directory structure.](doc/app/tasks/002-define-mvvm-architecture-conventions-view-widget-view-model-repository-layers-and-director.md)
- [x] [Establish Riverpod usage guidelines (Notifier/AsyncNotifier, providers scoping) and dependency injection strategy without code generation.](doc/app/tasks/003-establish-riverpod-usage-guidelines-notifier-asyncnotifier-providers-scoping-and-dependenc.md)
- [x] [Document navigation map (routes, tabs, nested flows) and deep link handling for app sections.](doc/app/tasks/004-document-navigation-map-routes-tabs-nested-flows-and-deep-link-handling-for-app-sections.md)
- [x] [Create API contract checklist aligning mobile payloads with backend endpoints for all flows.](doc/app/tasks/005-create-api-contract-checklist-aligning-mobile-payloads-with-backend-endpoints-for-all-flow.md)

## 1. Project Setup & Tooling
- [x] [Initialize Flutter project with flavors (dev/stg/prod), app icons, splash screens, and build configurations.](doc/app/tasks/006-initialize-flutter-project-with-flavors-dev-stg-prod-app-icons-splash-screens-and-build-co.md)
- [x] [Configure linting/formatting (analysis_options), CI pipeline (flutter analyze/test), and code coverage thresholds.](doc/app/tasks/007-configure-linting-formatting-analysis-options-ci-pipeline-flutter-analyze-test-and-code-co.md)
- [x] [Set up localization tooling (ARB files), theming, typography, and design tokens shared across screens.](doc/app/tasks/008-set-up-localization-tooling-arb-files-theming-typography-and-design-tokens-shared-across-s.md)
- [x] [Integrate Firebase services (Auth, Messaging, Remote Config) and configure environment-specific options.](doc/app/tasks/009-integrate-firebase-services-auth-messaging-remote-config-and-configure-environment-specifi.md)
- [x] [Implement secure storage, crash reporting (Sentry/Firebase Crashlytics), and analytics instrumentation.](doc/app/tasks/010-implement-secure-storage-crash-reporting-sentry-firebase-crashlytics-and-analytics-instrum.md)

## 2. Core Infrastructure & Shared Components
- [x] [Implement networking layer with HTTP client, interceptors (auth, logging), retries, and response parsing.](doc/app/tasks/011-implement-networking-layer-with-http-client-interceptors-auth-logging-retries-and-response.md)
- [ ] [Build API data models, DTOs, and repository interfaces for users, designs, catalog, orders, promotions, content.](doc/app/tasks/012-build-api-data-models-dtos-and-repository-interfaces-for-users-designs-catalog-orders-prom.md)
- [ ] [Implement local persistence (Hive/Isar/shared_preferences) for caching, offline screen data, and onboarding state.](doc/app/tasks/013-implement-local-persistence-hive-isar-shared-preferences-for-caching-offline-screen-data-a.md)
- [ ] [Create shared widgets (buttons, form fields, modals, cards, list skeletons) following design system.](doc/app/tasks/014-create-shared-widgets-buttons-form-fields-modals-cards-list-skeletons-following-design-sys.md)
- [ ] [Develop navigation shell with bottom tabs (`作成/ショップ/注文/マイ印鑑/プロフィール`), nested navigators, and deep link support.](doc/app/tasks/015-develop-navigation-shell-with-bottom-tabs-nested-navigators-and-deep-link-support.md)
- [ ] [Implement global app state providers (user session, locale, feature flags) with Riverpod.](doc/app/tasks/016-implement-global-app-state-providers-user-session-locale-feature-flags-with-riverpod.md)
- [ ] [Create notification bell UI, search entry points, and help overlays accessible from top app bar.](doc/app/tasks/017-create-notification-bell-ui-search-entry-points-and-help-overlays-accessible-from-top-app-.md)

## 3. Onboarding & Auth Flow
- [ ] [Implement splash screen logic (`/splash`) checking auth state, app version, feature flags.](doc/app/tasks/018-implement-splash-screen-logic-splash-checking-auth-state-app-version-feature-flags.md)
- [ ] [Build onboarding/tutorial screens (`/onboarding`) with skipping, progress indicator, and analytics events.](doc/app/tasks/019-build-onboarding-tutorial-screens-onboarding-with-skipping-progress-indicator-and-analytic.md)
- [ ] [Implement locale selection (`/locale`) and persona selection (`/persona`) storing preferences locally and server-side.](doc/app/tasks/020-implement-locale-selection-locale-and-persona-selection-persona-storing-preferences-locall.md)
- [ ] [Build authentication flow (`/auth`) supporting Apple Sign-In, Google, Email, and guest mode; handle link with Firebase Auth.](doc/app/tasks/021-build-authentication-flow-auth-supporting-apple-sign-in-google-email-and-guest-mode-handle.md)
- [ ] [Implement language/region and persona gating to drive downstream UI states.](doc/app/tasks/022-implement-language-region-and-persona-gating-to-drive-downstream-ui-states.md)

## 4. Home & Discovery
- [ ] [Implement home screen (`/home`) showing featured items, recent designs, and recommended templates using async providers.](doc/app/tasks/023-implement-home-screen-home-showing-featured-items-recent-designs-and-recommended-templates.md)
- [ ] [Build search screen (`/search`) with global search bar, segmented results (templates/materials/articles/FAQ), and search history.](doc/app/tasks/024-build-search-screen-search-with-global-search-bar-segmented-results-templates-materials-ar.md)
- [ ] [Implement notifications list (`/notifications`) with pagination, read/unread state, and push navigation handling.](doc/app/tasks/025-implement-notifications-list-notifications-with-pagination-read-unread-state-and-push-navi.md)

## 5. Design Creation Flow (作成タブ)
- [ ] [Implement design type selection (`/design/new`) with entry points for text/upload/logo flows.](doc/app/tasks/026-implement-design-type-selection-design-new-with-entry-points-for-text-upload-logo-flows.md)
- [ ] [Build name input screen (`/design/input`) with validation, localization, and optional kanji mapping entry point.](doc/app/tasks/027-build-name-input-screen-design-input-with-validation-localization-and-optional-kanji-mappi.md)
- [ ] [Implement kanji mapping flow (`/design/input/kanji-map`) displaying candidate list, meanings, and selection persistence.](doc/app/tasks/028-implement-kanji-mapping-flow-design-input-kanji-map-displaying-candidate-list-meanings-and.md)
- [ ] [Build style selection (`/design/style`) with preview carousel, filtering by script/shape, and template fetching.](doc/app/tasks/029-build-style-selection-design-style-with-preview-carousel-filtering-by-script-shape-and-tem.md)
- [ ] [Implement design editor (`/design/editor`) with canvas controls (layout, stroke, margins, rotation, grid) using custom painter widgets.](doc/app/tasks/030-implement-design-editor-design-editor-with-canvas-controls-layout-stroke-margins-rotation-.md)
- [ ] [Integrate AI suggestions (`/design/ai`) showing queued/completed proposals, comparison, and accept/reject actions.](doc/app/tasks/031-integrate-ai-suggestions-design-ai-showing-queued-completed-proposals-comparison-and-accep.md)
- [ ] [Implement registrability check (`/design/check`) calling external service, showing result badges.](doc/app/tasks/032-implement-registrability-check-design-check-calling-external-service-showing-result-badges.md)
- [ ] [Build preview screen (`/design/preview`) with actual size view, backgrounds, zoom, and share triggers.](doc/app/tasks/033-build-preview-screen-design-preview-with-actual-size-view-backgrounds-zoom-and-share-trigg.md)
- [ ] [Implement digital export (`/design/export`) generating PNG/SVG, handling permissions and download/share sheets.](doc/app/tasks/034-implement-digital-export-design-export-generating-png-svg-handling-permissions-and-downloa.md)
- [ ] [Implement version history (`/design/versions`) with diff view and rollback actions.](doc/app/tasks/035-implement-version-history-design-versions-with-diff-view-and-rollback-actions.md)
- [ ] [Build share screen (`/design/share`) generating mocked social posts and watermarked images.](doc/app/tasks/036-build-share-screen-design-share-generating-mocked-social-posts-and-watermarked-images.md)

## 6. Shop & Product Browsing
- [ ] [Implement shop home (`/shop`) with category tiles, promotions, and recommended materials.](doc/app/tasks/037-implement-shop-home-shop-with-category-tiles-promotions-and-recommended-materials.md)
- [ ] [Build material detail screen (`/materials/:materialId`) showing specs, gallery, availability.](doc/app/tasks/038-build-material-detail-screen-materials-materialid-showing-specs-gallery-availability.md)
- [ ] [Implement product detail (`/products/:productId`) with variant selector, pricing tiers, stock indicators.](doc/app/tasks/039-implement-product-detail-products-productid-with-variant-selector-pricing-tiers-stock-indi.md)
- [ ] [Build add-ons screen (`/products/:productId/addons`) for optional accessories with upsell logic.](doc/app/tasks/040-build-add-ons-screen-products-productid-addons-for-optional-accessories-with-upsell-logic.md)

## 7. Cart & Checkout
- [ ] [Implement cart screen (`/cart`) with line editing, quantity adjustments, promo code entry, and estimate summary.](doc/app/tasks/041-implement-cart-screen-cart-with-line-editing-quantity-adjustments-promo-code-entry-and-est.md)
- [ ] [Build checkout address screen (`/checkout/address`) with address list, add/edit forms, and validation (JP/international formats).](doc/app/tasks/042-build-checkout-address-screen-checkout-address-with-address-list-add-edit-forms-and-valida.md)
- [ ] [Implement shipping selection (`/checkout/shipping`) supporting domestic/international options and delivery estimates.](doc/app/tasks/043-implement-shipping-selection-checkout-shipping-supporting-domestic-international-options-a.md)
- [ ] [Build payment method screen (`/checkout/payment`) integrating tokenized payment refs and adding new methods if allowed.](doc/app/tasks/044-build-payment-method-screen-checkout-payment-integrating-tokenized-payment-refs-and-adding.md)
- [ ] [Implement review screen (`/checkout/review`) showing order summary, design snapshot, totals, and terms acknowledgement.](doc/app/tasks/045-implement-review-screen-checkout-review-showing-order-summary-design-snapshot-totals-and-t.md)
- [ ] [Build order completion screen (`/checkout/complete`) displaying confirmation, next steps, and share options.](doc/app/tasks/046-build-order-completion-screen-checkout-complete-displaying-confirmation-next-steps-and-sha.md)

## 8. Orders & Tracking
- [ ] [Implement orders list (`/orders`) with filters, status chips, and infinite scroll.](doc/app/tasks/047-implement-orders-list-orders-with-filters-status-chips-and-infinite-scroll.md)
- [ ] [Build order detail (`/orders/:orderId`) showing line items, totals, addresses, and design snapshot gallery.](doc/app/tasks/048-build-order-detail-orders-orderid-showing-line-items-totals-addresses-and-design-snapshot-.md)
- [ ] [Implement production timeline (`/orders/:orderId/production`) visualizing stages and timestamps.](doc/app/tasks/049-implement-production-timeline-orders-orderid-production-visualizing-stages-and-timestamps.md)
- [ ] [Build shipment tracking (`/orders/:orderId/tracking`) with event timeline and carrier integration.](doc/app/tasks/050-build-shipment-tracking-orders-orderid-tracking-with-event-timeline-and-carrier-integratio.md)
- [ ] [Implement invoice viewer (`/orders/:orderId/invoice`) with PDF download/share support.](doc/app/tasks/051-implement-invoice-viewer-orders-orderid-invoice-with-pdf-download-share-support.md)
- [ ] [Build reorder flow (`/orders/:orderId/reorder`) cloning cart data and redirecting to checkout.](doc/app/tasks/052-build-reorder-flow-orders-orderid-reorder-cloning-cart-data-and-redirecting-to-checkout.md)

## 9. My Hanko Library
- [ ] [Implement library list (`/library`) with sorting, filtering (status, date, AI score), and grid/list toggle.](doc/app/tasks/053-implement-library-list-library-with-sorting-filtering-status-date-ai-score-and-grid-list-t.md)
- [ ] [Build design detail (`/library/:designId`) showing metadata, AI score, registrability status, usage history, and quick actions.](doc/app/tasks/054-build-design-detail-library-designid-showing-metadata-ai-score-registrability-status-usage.md)
- [ ] [Implement versions view (`/library/:designId/versions`) reusing diff/rollback components.](doc/app/tasks/055-implement-versions-view-library-designid-versions-reusing-diff-rollback-components.md)
- [ ] [Build duplicate flow (`/library/:designId/duplicate`) creating new design entry and navigating to editor.](doc/app/tasks/056-build-duplicate-flow-library-designid-duplicate-creating-new-design-entry-and-navigating-t.md)
- [ ] [Implement digital export screen (`/library/:designId/export`) with formats and permissions.](doc/app/tasks/057-implement-digital-export-screen-library-designid-export-with-formats-and-permissions.md)
- [ ] [Build share link management (`/library/:designId/shares`) showing issued links, expiry, revoke.](doc/app/tasks/058-build-share-link-management-library-designid-shares-showing-issued-links-expiry-revoke.md)

## 10. Guides & Cultural Content
- [ ] [Implement guides list (`/guides`) with localization filters and recommended content for personas.](doc/app/tasks/059-implement-guides-list-guides-with-localization-filters-and-recommended-content-for-persona.md)
- [ ] [Build guide detail (`/guides/:slug`) rendering CMS content with markdown/HTML and offline caching.](doc/app/tasks/060-build-guide-detail-guides-slug-rendering-cms-content-with-markdown-html-and-offline-cachin.md)
- [ ] [Implement kanji dictionary (`/kanji/dictionary`) with search, favorites, and integration with design input.](doc/app/tasks/061-implement-kanji-dictionary-kanji-dictionary-with-search-favorites-and-integration-with-des.md)
- [ ] [Build how-to screen (`/howto`) aggregating tutorials/videos with embedded players.](doc/app/tasks/062-build-how-to-screen-howto-aggregating-tutorials-videos-with-embedded-players.md)

## 11. Profile & Settings
- [ ] [Implement profile home (`/profile`) showing avatar, display name, persona toggle, quick links.](doc/app/tasks/063-implement-profile-home-profile-showing-avatar-display-name-persona-toggle-quick-links.md)
- [ ] [Build addresses management (`/profile/addresses`) with CRUD, defaults, and shipping sync.](doc/app/tasks/064-build-addresses-management-profile-addresses-with-crud-defaults-and-shipping-sync.md)
- [ ] [Implement payment methods management (`/profile/payments`) referencing PSP tokens, default selection, and removal.](doc/app/tasks/065-implement-payment-methods-management-profile-payments-referencing-psp-tokens-default-selec.md)
- [ ] [Build notifications settings (`/profile/notifications`) for push/email categories and scheduling.](doc/app/tasks/066-build-notifications-settings-profile-notifications-for-push-email-categories-and-schedulin.md)
- [ ] [Implement locale settings (`/profile/locale`) for language/currency overrides.](doc/app/tasks/067-implement-locale-settings-profile-locale-for-language-currency-overrides.md)
- [ ] [Build legal documents screen (`/profile/legal`) rendering static content with offline availability.](doc/app/tasks/068-build-legal-documents-screen-profile-legal-rendering-static-content-with-offline-availabil.md)
- [ ] [Implement support screen (`/profile/support`) linking to FAQ, chat, contact forms.](doc/app/tasks/069-implement-support-screen-profile-support-linking-to-faq-chat-contact-forms.md)
- [ ] [Build linked accounts screen (`/profile/linked-accounts`) showing social auth connections and unlink flow.](doc/app/tasks/070-build-linked-accounts-screen-profile-linked-accounts-showing-social-auth-connections-and-u.md)
- [ ] [Implement data export (`/profile/export`) generating archive and downloading securely.](doc/app/tasks/071-implement-data-export-profile-export-generating-archive-and-downloading-securely.md)
- [ ] [Build account delete flow (`/profile/delete`) with confirmation steps and backend call.](doc/app/tasks/072-build-account-delete-flow-profile-delete-with-confirmation-steps-and-backend-call.md)

## 12. Support & Status
- [ ] [Implement FAQ screen (`/support/faq`) with categories, search, and offline caching.](doc/app/tasks/073-implement-faq-screen-support-faq-with-categories-search-and-offline-caching.md)
- [ ] [Build contact form (`/support/contact`) with ticket creation and file attachment uploading.](doc/app/tasks/074-build-contact-form-support-contact-with-ticket-creation-and-file-attachment-uploading.md)
- [ ] [Implement chat support (`/support/chat`) integrating bot handoff to live agent with push notifications.](doc/app/tasks/075-implement-chat-support-support-chat-integrating-bot-handoff-to-live-agent-with-push-notifi.md)
- [ ] [Build system status screen (`/status`) showing current incidents and historical uptime.](doc/app/tasks/076-build-system-status-screen-status-showing-current-incidents-and-historical-uptime.md)

## 13. System Utilities
- [ ] [Implement permissions onboarding (`/permissions`) prompting for photo/storage/notification access with rationale.](doc/app/tasks/077-implement-permissions-onboarding-permissions-prompting-for-photo-storage-notification-acce.md)
- [ ] [Build changelog screen (`/updates/changelog`) with version history and feature highlights.](doc/app/tasks/078-build-changelog-screen-updates-changelog-with-version-history-and-feature-highlights.md)
- [ ] [Implement forced app update flow (`/app-update`) checking version constraints and gating access.](doc/app/tasks/079-implement-forced-app-update-flow-app-update-checking-version-constraints-and-gating-access.md)
- [ ] [Build offline screen (`/offline`) with retry and cached content access.](doc/app/tasks/080-build-offline-screen-offline-with-retry-and-cached-content-access.md)
- [ ] [Implement generic error screen (`/error`) with diagnostics and support links.](doc/app/tasks/081-implement-generic-error-screen-error-with-diagnostics-and-support-links.md)

## 14. Notifications & Messaging
- [ ] [Integrate push notification handling (background/foreground) and routing to relevant screens.](doc/app/tasks/082-integrate-push-notification-handling-background-foreground-and-routing-to-relevant-screens.md)
- [ ] [Implement in-app messaging/toast system for success, warnings, alerts tied to Riverpod providers.](doc/app/tasks/083-implement-in-app-messaging-toast-system-for-success-warnings-alerts-tied-to-riverpod-provi.md)
- [ ] [Provide notification inbox sync and badge counts shared between app bar and tabs.](doc/app/tasks/084-provide-notification-inbox-sync-and-badge-counts-shared-between-app-bar-and-tabs.md)

## 15. Analytics, Telemetry, and Monitoring
- [ ] [Define analytics events for key flows (design creation, checkout, share) and instrument across view models.](doc/app/tasks/085-define-analytics-events-for-key-flows-design-creation-checkout-share-and-instrument-across.md)
- [ ] [Configure performance monitoring (Firebase Performance) and custom metrics for screen load times.](doc/app/tasks/086-configure-performance-monitoring-firebase-performance-and-custom-metrics-for-screen-load-t.md)
- [ ] [Implement remote config/feature flag handling for gradual rollout of features.](doc/app/tasks/087-implement-remote-config-feature-flag-handling-for-gradual-rollout-of-features.md)
- [ ] [Set up logging/trace pipeline for client errors and attach device context.](doc/app/tasks/088-set-up-logging-trace-pipeline-for-client-errors-and-attach-device-context.md)

## 16. Accessibility, Localization, and QA
- [ ] [Ensure accessibility compliance (semantics, focus order, color contrast, screen reader labels) across screens.](doc/app/tasks/089-ensure-accessibility-compliance-semantics-focus-order-color-contrast-screen-reader-labels-.md)
- [ ] [Complete full localization pass (copy extraction, pluralization, RTL readiness if needed).](doc/app/tasks/090-complete-full-localization-pass-copy-extraction-pluralization-rtl-readiness-if-needed.md)
- [ ] [Implement automated widget tests, integration tests (golden tests, end-to-end flows) covering core journeys.](doc/app/tasks/091-implement-automated-widget-tests-integration-tests-golden-tests-end-to-end-flows-covering-.md)
- [ ] [Prepare manual QA checklist and device matrix for release certification.](doc/app/tasks/092-prepare-manual-qa-checklist-and-device-matrix-for-release-certification.md)

## 17. Release Management & Distribution
- [ ] [Configure App Store / Google Play metadata, screenshots, privacy manifests, and release notes workflow.](doc/app/tasks/093-configure-app-store-google-play-metadata-screenshots-privacy-manifests-and-release-notes-w.md)
- [ ] [Set up build automation (Fastlane/Codemagic) for beta and production releases with environment variables.](doc/app/tasks/094-set-up-build-automation-fastlane-codemagic-for-beta-and-production-releases-with-environme.md)
- [ ] [Establish beta testing program (TestFlight/Play Console) and feedback loop ingestion.](doc/app/tasks/095-establish-beta-testing-program-testflight-play-console-and-feedback-loop-ingestion.md)
- [ ] [Document release checklist including rollback plan, monitoring, and post-release analytics review.](doc/app/tasks/096-document-release-checklist-including-rollback-plan-monitoring-and-post-release-analytics-r.md)
