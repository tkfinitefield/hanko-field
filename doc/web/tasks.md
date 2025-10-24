# Web Frontend Implementation Task List

## 0. Planning & Architecture
- [x] [Validate web scope, personas, and release milestones from `doc/web/web_design.md`.](doc/web/tasks/001-validate-web-scope-personas-and-release-milestones-from-doc-web-web-design-md.md)
- [x] [Define Go + htmx architecture (router layout, template structure, partials naming) and directory conventions.](doc/web/tasks/002-define-go-htmx-architecture-router-layout-template-structure-partials-naming-and-directory.md)
- [x] [Establish shared components and Tailwind design tokens; document CSS/JS guidelines.](doc/web/tasks/003-establish-shared-components-and-tailwind-design-tokens-document-css-js-guidelines.md)
- [x] [Map full site navigation (public, design creation, checkout, account) with routes, breadcrumbs, and SEO considerations.](doc/web/tasks/004-map-full-site-navigation-public-design-creation-checkout-account-with-routes-breadcrumbs-a.md)
- [x] [Produce integration checklist aligning web fragments with API endpoints and payload requirements.](doc/web/tasks/005-produce-integration-checklist-aligning-web-fragments-with-api-endpoints-and-payload-requir.md)

## 1. Project Setup & Tooling
- [x] [Scaffold Go web module with chi router, template engine, Tailwind asset pipeline, and dev tooling.](doc/web/tasks/006-scaffold-go-web-module-with-chi-echo-router-template-engine-asset-pipeline-tailwind-alpine.md)
- [x] [Configure build/release pipeline for Cloud Run deployment (Dockerfile, cloudbuild.yaml, env configs).](doc/web/tasks/007-configure-build-release-pipeline-for-cloud-run-deployment-dockerfile-cloudbuild-yaml-env-c.md)
- [x] [Implement middleware stack (auth, CSRF, session, logging, caching) shared across SSR and htmx fragments.](doc/web/tasks/008-implement-middleware-stack-auth-csrf-session-logging-caching-shared-across-ssr-and-htmx-fr.md)
- [x] [Set up localization (i18n dictionaries), formatting helpers, and SEO metadata utilities.](doc/web/tasks/009-set-up-localization-i18n-dictionaries-formatting-helpers-and-seo-metadata-utilities.md)
- [x] [Establish integration/unit testing harness (httptest, HTML assertions, htmx interactions).](doc/web/tasks/010-establish-integration-unit-testing-harness-httptest-html-assertions-htmx-interactions.md)

## 2. Shared Layout & Components
- [x] [Implement base layout (`/layouts/_base.html`) with header, footer, responsive nav, and modal container.](doc/web/tasks/011-implement-base-layout-layouts-base-html-with-header-footer-responsive-nav-and-modal-contai.md)
- [x] [Build navigation menu, breadcrumbs, and active state handling based on current route.](doc/web/tasks/012-build-navigation-menu-breadcrumbs-and-active-state-handling-based-on-current-route.md)
- [x] [Create reusable components (hero, cards, tables, forms, button sets, skeleton loaders) with Tailwind variants.](doc/web/tasks/013-create-reusable-components-hero-cards-tables-forms-button-sets-skeleton-loaders-with-tailw.md)
- [x] [Implement shared modals system with htmx targets and close triggers (ESC, overlay click).](doc/web/tasks/014-implement-shared-modals-system-with-htmx-targets-and-close-triggers-esc-overlay-click.md)
- [x] [Provide SEO/OGP tags, structured data helpers, and analytics instrumentation hooks.](doc/web/tasks/015-provide-seo-ogp-tags-structured-data-helpers-and-analytics-instrumentation-hooks.md)

## 3. Landing & Exploration
- [x] [Build landing page (`/`) with hero, comparison table fragment (`/frags/compare/sku-table`), and latest guides fragment.](doc/web/tasks/016-build-landing-page-with-hero-comparison-table-fragment-frags-compare-sku-table-and-latest-.md)
- [x] [Implement shop listing (`/shop`) with filters form, results fragment (`/shop/table`), and pagination.](doc/web/tasks/017-implement-shop-listing-shop-with-filters-form-results-fragment-shop-table-and-pagination.md)
- [x] [Build product detail (`/products/{productId}`) with gallery fragment, review snippets, and add-to-cart form.](doc/web/tasks/018-build-product-detail-products-productid-with-gallery-fragment-review-snippets-and-add-to-c.md)
- [x] [Implement templates listing/detail (`/templates`, `/templates/{templateId}`) with filter fragment and preview.](doc/web/tasks/019-implement-templates-listing-detail-templates-templates-templateid-with-filter-fragment-and.md)
- [x] [Build guides list/detail (`/guides`, `/guides/{slug}`) with CMS integration and SEO metadata.](doc/web/tasks/020-build-guides-list-detail-guides-guides-slug-with-cms-integration-and-seo-metadata.md)
- [x] [Implement static content pages (`/content/{slug}`, `/legal/{slug}`, `/status`) with caching and markdown rendering.](doc/web/tasks/021-implement-static-content-pages-content-slug-legal-slug-status-with-caching-and-markdown-re.md)

## 4. Design Creation Flow
- [x] [Implement design type selection page (`/design/new`) with CTA routing to editor.](doc/web/tasks/022-implement-design-type-selection-page-design-new-with-cta-routing-to-editor.md)
- [x] [Build design editor (`/design/editor`) two-pane layout with form fragment (`/design/editor/form`) and live preview fragment (`/design/editor/preview`).](doc/web/tasks/023-build-design-editor-design-editor-two-pane-layout-with-form-fragment-design-editor-form-an.md)
- [x] [Implement modal pickers for fonts/templates/kanji mapping (`/modal/pick/font`, `/modal/pick/template`, `/modal/kanji-map`).](doc/web/tasks/024-implement-modal-pickers-for-fonts-templates-kanji-mapping-modal-pick-font-modal-pick-templ.md)
- [x] [Build AI suggestions page (`/design/ai`) with table fragment, accept/reject actions, and polling.](doc/web/tasks/025-build-ai-suggestions-page-design-ai-with-table-fragment-accept-reject-actions-and-polling.md)
- [x] [Implement design preview page (`/design/preview`) with background options fragment (`/design/preview/image`).](doc/web/tasks/026-implement-design-preview-page-design-preview-with-background-options-fragment-design-previ.md)
- [ ] [Build versions page (`/design/versions`) with table fragment and rollback modal.](doc/web/tasks/027-build-versions-page-design-versions-with-table-fragment-and-rollback-modal.md)
- [ ] [Implement design share/download functionality with signed URL workflow.](doc/web/tasks/028-implement-design-share-download-functionality-with-signed-url-workflow.md)

## 5. Cart & Checkout
- [ ] [Implement cart page (`/cart`) with table fragment, promo modal, and estimate refresh fragment.](doc/web/tasks/029-implement-cart-page-cart-with-table-fragment-promo-modal-and-estimate-refresh-fragment.md)
- [ ] [Build checkout address page (`/checkout/address`) with address selection fragment and forms.](doc/web/tasks/030-build-checkout-address-page-checkout-address-with-address-selection-fragment-and-forms.md)
- [ ] [Implement shipping selection (`/checkout/shipping`) with comparison fragment and integration with estimate API.](doc/web/tasks/031-implement-shipping-selection-checkout-shipping-with-comparison-fragment-and-integration-wi.md)
- [ ] [Build payment page (`/checkout/payment`) with PSP session initiation and confirmation handling.](doc/web/tasks/032-build-payment-page-checkout-payment-with-psp-session-initiation-and-confirmation-handling.md)
- [ ] [Implement review page (`/checkout/review`) summarizing order and linking to confirmation action.](doc/web/tasks/033-implement-review-page-checkout-review-summarizing-order-and-linking-to-confirmation-action.md)
- [ ] [Build completion page (`/checkout/complete`) displaying order number, next steps, and share options.](doc/web/tasks/034-build-completion-page-checkout-complete-displaying-order-number-next-steps-and-share-optio.md)

## 6. Account & Library
- [ ] [Implement account profile page (`/account`) with profile form fragment and update flow.](doc/web/tasks/035-implement-account-profile-page-account-with-profile-form-fragment-and-update-flow.md)
- [ ] [Build addresses management (`/account/addresses`) with table fragment and edit modal.](doc/web/tasks/036-build-addresses-management-account-addresses-with-table-fragment-and-edit-modal.md)
- [ ] [Implement orders list (`/account/orders`) with filterable table fragment and pagination.](doc/web/tasks/037-implement-orders-list-account-orders-with-filterable-table-fragment-and-pagination.md)
- [ ] [Build order detail (`/account/orders/{orderId}`) with tabbed fragments for summary, payments, production, tracking, invoice.](doc/web/tasks/038-build-order-detail-account-orders-orderid-with-tabbed-fragments-for-summary-payments-produ.md)
- [ ] [Implement library (`/account/library`) with design list fragment, filters, and actions (duplicate/export/share).](doc/web/tasks/039-implement-library-account-library-with-design-list-fragment-filters-and-actions-duplicate-.md)
- [ ] [Build security/linked accounts page (`/account/security`) covering auth providers and 2FA prompts.](doc/web/tasks/040-build-security-linked-accounts-page-account-security-covering-auth-providers-and-2fa-promp.md)

## 7. Support & Legal
- [ ] [Build support page (`/support`) with contact form, FAQ links, and response handling.](doc/web/tasks/041-build-support-page-support-with-contact-form-faq-links-and-response-handling.md)
- [ ] [Implement legal content pages (`/legal/{slug}`) with markdown rendering, localization, and version tracking.](doc/web/tasks/042-implement-legal-content-pages-legal-slug-with-markdown-rendering-localization-and-version-.md)
- [ ] [Build status page (`/status`) displaying system health and incident history.](doc/web/tasks/043-build-status-page-status-displaying-system-health-and-incident-history.md)

## 8. Notifications, Search, and Utilities
- [ ] [Implement notification dropdown/list accessible from header with htmx refresh.](doc/web/tasks/044-implement-notification-dropdown-list-accessible-from-header-with-htmx-refresh.md)
- [ ] [Build global search overlay integrating products, templates, guides (fragments for results).](doc/web/tasks/045-build-global-search-overlay-integrating-products-templates-guides-fragments-for-results.md)
- [ ] [Implement cookie consent, feature flags, and A/B testing hooks if required.](doc/web/tasks/046-implement-cookie-consent-feature-flags-and-a-b-testing-hooks-if-required.md)
- [ ] [Provide offline/error pages and progressive enhancement fallbacks.](doc/web/tasks/047-provide-offline-error-pages-and-progressive-enhancement-fallbacks.md)

## 9. Performance, Accessibility, and QA
- [ ] [Optimize asset pipeline (Tailwind purge, lazy loading, responsive images) and configure CDN headers.](doc/web/tasks/048-optimize-asset-pipeline-tailwind-purge-lazy-loading-responsive-images-and-configure-cdn-he.md)
- [ ] [Ensure accessibility compliance (ARIA roles, keyboard navigation, focus management) across pages and modals.](doc/web/tasks/049-ensure-accessibility-compliance-aria-roles-keyboard-navigation-focus-management-across-pag.md)
- [ ] [Implement automated tests for fragments/modals (htmx interactions) and critical flows.](doc/web/tasks/050-implement-automated-tests-for-fragments-modals-htmx-interactions-and-critical-flows.md)
- [ ] [Set up monitoring (logging, metrics, uptime checks) and real user metrics reporting.](doc/web/tasks/051-set-up-monitoring-logging-metrics-uptime-checks-and-real-user-metrics-reporting.md)
- [ ] [Document QA checklist covering regression flows prior to release.](doc/web/tasks/052-document-qa-checklist-covering-regression-flows-prior-to-release.md)

## 10. Deployment & Maintenance
- [ ] [Configure Cloud Run service, environment variables, and Secret Manager integration.](doc/web/tasks/053-configure-cloud-run-service-environment-variables-and-secret-manager-integration.md)
- [ ] [Set up CI/CD pipeline with staging/production deployments and smoke tests.](doc/web/tasks/054-set-up-ci-cd-pipeline-with-staging-production-deployments-and-smoke-tests.md)
- [ ] [Implement feature flag rollout strategy and rollback procedures.](doc/web/tasks/055-implement-feature-flag-rollout-strategy-and-rollback-procedures.md)
- [ ] [Document operational runbooks (incident response, cache purge, SEO updates).](doc/web/tasks/056-document-operational-runbooks-incident-response-cache-purge-seo-updates.md)
