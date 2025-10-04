# Web Frontend Implementation Task List

## 0. Planning & Architecture
- [ ] Validate web scope, personas, and release milestones from `doc/web/web_design.md`.
- [ ] Define Go + htmx architecture (router layout, template structure, partials naming) and directory conventions.
- [ ] Establish shared components and Tailwind design tokens; document CSS/JS guidelines.
- [ ] Map full site navigation (public, design creation, checkout, account) with routes, breadcrumbs, and SEO considerations.
- [ ] Produce integration checklist aligning web fragments with API endpoints and payload requirements.

## 1. Project Setup & Tooling
- [ ] Scaffold Go web module with chi/echo router, template engine, asset pipeline (Tailwind, Alpine), and dev tooling.
- [ ] Configure build/release pipeline for Cloud Run deployment (Dockerfile, cloudbuild.yaml, env configs).
- [ ] Implement middleware stack (auth, CSRF, session, logging, caching) shared across SSR and htmx fragments.
- [ ] Set up localization (i18n dictionaries), formatting helpers, and SEO metadata utilities.
- [ ] Establish integration/unit testing harness (httptest, HTML assertions, htmx interactions).

## 2. Shared Layout & Components
- [ ] Implement base layout (`/layouts/_base.html`) with header, footer, responsive nav, and modal container.
- [ ] Build navigation menu, breadcrumbs, and active state handling based on current route.
- [ ] Create reusable components (hero, cards, tables, forms, button sets, skeleton loaders) with Tailwind variants.
- [ ] Implement shared modals system with htmx targets and close triggers (ESC, overlay click).
- [ ] Provide SEO/OGP tags, structured data helpers, and analytics instrumentation hooks.

## 3. Landing & Exploration
- [ ] Build landing page (`/`) with hero, comparison table fragment (`/frags/compare/sku-table`), and latest guides fragment.
- [ ] Implement shop listing (`/shop`) with filters form, results fragment (`/shop/table`), and pagination.
- [ ] Build product detail (`/products/{productId}`) with gallery fragment, review snippets, and add-to-cart form.
- [ ] Implement templates listing/detail (`/templates`, `/templates/{templateId}`) with filter fragment and preview.
- [ ] Build guides list/detail (`/guides`, `/guides/{slug}`) with CMS integration and SEO metadata.
- [ ] Implement static content pages (`/content/{slug}`, `/legal/{slug}`, `/status`) with caching and markdown rendering.

## 4. Design Creation Flow
- [ ] Implement design type selection page (`/design/new`) with CTA routing to editor.
- [ ] Build design editor (`/design/editor`) two-pane layout with form fragment (`/design/editor/form`) and live preview fragment (`/design/editor/preview`).
- [ ] Implement modal pickers for fonts/templates/kanji mapping (`/modal/pick/font`, `/modal/pick/template`, `/modal/kanji-map`).
- [ ] Build AI suggestions page (`/design/ai`) with table fragment, accept/reject actions, and polling.
- [ ] Implement design preview page (`/design/preview`) with background options fragment (`/design/preview/image`).
- [ ] Build versions page (`/design/versions`) with table fragment and rollback modal.
- [ ] Implement design share/download functionality with signed URL workflow.

## 5. Cart & Checkout
- [ ] Implement cart page (`/cart`) with table fragment, promo modal, and estimate refresh fragment.
- [ ] Build checkout address page (`/checkout/address`) with address selection fragment and forms.
- [ ] Implement shipping selection (`/checkout/shipping`) with comparison fragment and integration with estimate API.
- [ ] Build payment page (`/checkout/payment`) with PSP session initiation and confirmation handling.
- [ ] Implement review page (`/checkout/review`) summarizing order and linking to confirmation action.
- [ ] Build completion page (`/checkout/complete`) displaying order number, next steps, and share options.

## 6. Account & Library
- [ ] Implement account profile page (`/account`) with profile form fragment and update flow.
- [ ] Build addresses management (`/account/addresses`) with table fragment and edit modal.
- [ ] Implement orders list (`/account/orders`) with filterable table fragment and pagination.
- [ ] Build order detail (`/account/orders/{orderId}`) with tabbed fragments for summary, payments, production, tracking, invoice.
- [ ] Implement library (`/account/library`) with design list fragment, filters, and actions (duplicate/export/share).
- [ ] Build security/linked accounts page (`/account/security`) covering auth providers and 2FA prompts.

## 7. Support & Legal
- [ ] Build support page (`/support`) with contact form, FAQ links, and response handling.
- [ ] Implement legal content pages (`/legal/{slug}`) with markdown rendering, localization, and version tracking.
- [ ] Build status page (`/status`) displaying system health and incident history.

## 8. Notifications, Search, and Utilities
- [ ] Implement notification dropdown/list accessible from header with htmx refresh.
- [ ] Build global search overlay integrating products, templates, guides (fragments for results).
- [ ] Implement cookie consent, feature flags, and A/B testing hooks if required.
- [ ] Provide offline/error pages and progressive enhancement fallbacks.

## 9. Performance, Accessibility, and QA
- [ ] Optimize asset pipeline (Tailwind purge, lazy loading, responsive images) and configure CDN headers.
- [ ] Ensure accessibility compliance (ARIA roles, keyboard navigation, focus management) across pages and modals.
- [ ] Implement automated tests for fragments/modals (htmx interactions) and critical flows.
- [ ] Set up monitoring (logging, metrics, uptime checks) and real user metrics reporting.
- [ ] Document QA checklist covering regression flows prior to release.

## 10. Deployment & Maintenance
- [ ] Configure Cloud Run service, environment variables, and Secret Manager integration.
- [ ] Set up CI/CD pipeline with staging/production deployments and smoke tests.
- [ ] Implement feature flag rollout strategy and rollback procedures.
- [ ] Document operational runbooks (incident response, cache purge, SEO updates).
