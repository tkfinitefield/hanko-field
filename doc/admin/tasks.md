# Admin Console Implementation Task List

## 0. Planning & Architecture
- [x] [Confirm admin console scope, personas (ops, CS, marketing), and rollout priorities based on `doc/admin/admin_design.md`.](doc/admin/tasks/001-confirm-admin-console-scope-personas-ops-cs-marketing-and-rollout-priorities-based-on-doc-.md)
- [x] [Define UI architecture (Go server-side rendering + htmx partials) and routing conventions for full pages vs fragment endpoints.](doc/admin/tasks/002-define-ui-architecture-go-server-side-rendering-htmx-partials-and-routing-conventions-for-.md)
- [x] [Model navigation taxonomy and RBAC visibility rules; document mapping between sidebar groups and user roles.](doc/admin/tasks/003-model-navigation-taxonomy-and-rbac-visibility-rules-document-mapping-between-sidebar-group.md)
- [x] [Produce data contract checklist mapping every admin action to the corresponding API endpoint and required request/response fields.](doc/admin/tasks/004-produce-data-contract-checklist-mapping-every-admin-action-to-the-corresponding-api-endpoi.md)

## 1. Project & Infrastructure Setup
- [x] [Scaffold Go web module (`/admin` or `/web`) with templating pipeline, asset bundler (Tailwind), and dev tooling (hot reload, lint).](doc/admin/tasks/005-scaffold-go-web-module-admin-or-web-with-templating-pipeline-asset-bundler-tailwind-and-de.md)
- [x] [Configure chi/echo router with middleware for auth, CSRF, caching headers, and htmx request detection.](doc/admin/tasks/006-configure-chi-echo-router-with-middleware-for-auth-csrf-caching-headers-and-htmx-request-d.md)
- [x] [Establish HTML template structure (`layouts`, `partials`, `components`) and helper functions (i18n, currency, date formatting).](doc/admin/tasks/007-establish-html-template-structure-layouts-partials-components-and-helper-functions-i18n-cu.md)
- [x] [Implement Tailwind configuration, design tokens, and base components library (buttons, tables, forms, modals, toasts).](doc/admin/tasks/008-implement-tailwind-configuration-design-tokens-and-base-components-library-buttons-tables-.md)
- [x] [Set up integration tests harness (httptest + DOM assertions) and smoke test environment for admin flows.](doc/admin/tasks/009-set-up-integration-tests-harness-httptest-dom-assertions-and-smoke-test-environment-for-ad.md)

## 2. Authentication, Authorization, and Session Management
- [x] [Implement Firebase ID token verification for staff/admin, including refresh and error handling UX.](doc/admin/tasks/010-implement-firebase-id-token-verification-for-staff-admin-including-refresh-and-error-handl.md)
- [x] [Build session middleware storing user info, CSRF tokens, and feature flags; add remember-me support if required.](doc/admin/tasks/011-build-session-middleware-storing-user-info-csrf-tokens-and-feature-flags-add-remember-me-s.md)
- [x] [Implement RBAC guard utilities for template rendering (sidebar filtering, action visibility).](doc/admin/tasks/012-implement-rbac-guard-utilities-for-template-rendering-sidebar-filtering-action-visibility.md)
- [x] [Create login/logout flows (`/admin/login`, `/admin/logout`) with error states and redirect logic.](doc/admin/tasks/013-create-login-logout-flows-admin-login-admin-logout-with-error-states-and-redirect-logic.md)
- [x] [Add MFA/API-key management UI under `/admin/profile`, integrating with backend secrets service.](doc/admin/tasks/014-add-mfa-api-key-management-ui-under-admin-profile-integrating-with-backend-secrets-service.md)

## 3. Layout, Navigation, and Shared UX
- [x] [Implement global layout template (`/admin/_layout.html`) with sidebar, top bar, breadcrumbs, and responsive behaviour.](doc/admin/tasks/015-implement-global-layout-template-admin-layout-html-with-sidebar-top-bar-breadcrumbs-and-re.md)
- [x] [Build dynamic sidebar rendering with route highlighting and permission-based sections.](doc/admin/tasks/016-build-dynamic-sidebar-rendering-with-route-highlighting-and-permission-based-sections.md)
- [x] [Create top bar components (environment badge, search shortcut, notification icon, user menu).](doc/admin/tasks/017-create-top-bar-components-environment-badge-search-shortcut-notification-icon-user-menu.md)
- [x] [Implement modal container (`#modal`) with htmx target wiring, animations, and escape-key close behaviour.](doc/admin/tasks/018-implement-modal-container-modal-with-htmx-target-wiring-animations-and-escape-key-close-be.md)
- [x] [Provide reusable pagination controls, table headers with sort indicators, and bulk action toolbar components.](doc/admin/tasks/019-provide-reusable-pagination-controls-table-headers-with-sort-indicators-and-bulk-action-to.md)
- [x] [Implement toast/alert system triggered by htmx response headers.](doc/admin/tasks/020-implement-toast-alert-system-triggered-by-htmx-response-headers.md)

## 4. Shared Utilities & System Pages
- [x] [Build dashboard page (`/admin`) with KPI fragment endpoints (`/admin/fragments/kpi`, `/admin/fragments/alerts`).](doc/admin/tasks/021-build-dashboard-page-admin-with-kpi-fragment-endpoints-admin-fragments-kpi-admin-fragments.md)
- [x] [Implement global search page (`/admin/search`) and result fragment (`/admin/search/table`) with filtering and keyboard shortcuts.](doc/admin/tasks/022-implement-global-search-page-admin-search-and-result-fragment-admin-search-table-with-filt.md)
- [x] [Create notifications page (`/admin/notifications`) and table fragment for failed jobs, stock alerts, and shipping exceptions.](doc/admin/tasks/023-create-notifications-page-admin-notifications-and-table-fragment-for-failed-jobs-stock-ale.md)
- [x] [Implement profile/account page (`/admin/profile`) with 2FA setup, password/API key management, and session history.](doc/admin/tasks/024-implement-profile-account-page-admin-profile-with-2fa-setup-password-api-key-management-an.md)

## 5. Orders & Operations
### 5.1 Orders List & Detail
- [x] [Implement orders index page (`/admin/orders`) with filter form, table fragment (`/admin/orders/table`), pagination, and bulk actions (status updates, label generation, CSV export).](doc/admin/tasks/025-implement-orders-index-page-admin-orders-with-filter-form-table-fragment-admin-orders-tabl.md)
- [x] [Build order detail page (`/admin/orders/{orderId}`) with tabbed content endpoints (`/admin/orders/{id}/tab/{summary|lines|payments|production|shipments|invoice|audit}`).](doc/admin/tasks/026-build-order-detail-page-admin-orders-orderid-with-tabbed-content-endpoints-admin-orders-id.md)
- [x] [Implement status update modal (`/admin/orders/{id}/modal/status`) posting to `PUT /admin/orders/{id}:status` and updating the UI inline.](doc/admin/tasks/027-implement-status-update-modal-admin-orders-id-modal-status-posting-to-put-admin-orders-id-.md)
- [x] [Implement refund modal (`/admin/orders/{id}/modal/refund`) integrating with `POST /orders/{id}/payments:refund` and showing validation errors.](doc/admin/tasks/028-implement-refund-modal-admin-orders-id-modal-refund-integrating-with-post-orders-id-paymen.md)
- [ ] [Implement invoice request modal (`/admin/orders/{id}/modal/invoice`) calling `POST /admin/invoices:issue` and reflecting result in tabs.](doc/admin/tasks/029-implement-invoice-request-modal-admin-orders-id-modal-invoice-calling-post-admin-invoices-.md)
- [ ] [Add bulk export and print actions (CSV, PDF) with progress feedback.](doc/admin/tasks/030-add-bulk-export-and-print-actions-csv-pdf-with-progress-feedback.md)

### 5.2 Shipments & Tracking
- [ ] [Create shipment batch page (`/admin/shipments/batches`) for label generation workflows and integration with shipment POST endpoints.](doc/admin/tasks/031-create-shipment-batch-page-admin-shipments-batches-for-label-generation-workflows-and-inte.md)
- [ ] [Implement shipment tracking monitor (`/admin/shipments/tracking`) with table fragment (`/admin/shipments/tracking/table`) and filtering by carrier/status.](doc/admin/tasks/032-implement-shipment-tracking-monitor-admin-shipments-tracking-with-table-fragment-admin-shi.md)
- [ ] [Hook carrier webhook data or Firestore views to populate tracking dashboard, including exception badges and SLA indicators.](doc/admin/tasks/033-hook-carrier-webhook-data-or-firestore-views-to-populate-tracking-dashboard-including-exce.md)

### 5.3 Production & Workshop
- [ ] [Implement production kanban page (`/admin/production/queues`) with board fragment (`/admin/production/queues/board`) and drag-and-drop updates posting production events.](doc/admin/tasks/034-implement-production-kanban-page-admin-production-queues-with-board-fragment-admin-product.md)
- [ ] [Create work order view (`/admin/production/workorders/{orderId}`) summarizing design assets, instructions, and tasks.](doc/admin/tasks/035-create-work-order-view-admin-production-workorders-orderid-summarizing-design-assets-instr.md)
- [ ] [Build QC page (`/admin/production/qc`) to record pass/fail events and trigger rework flows.](doc/admin/tasks/036-build-qc-page-admin-production-qc-to-record-pass-fail-events-and-trigger-rework-flows.md)

## 6. Catalog Management
- [ ] [Develop catalog overview page with tabs for templates, fonts, materials, and products (`/admin/catalog/{kind}`).](doc/admin/tasks/037-develop-catalog-overview-page-with-tabs-for-templates-fonts-materials-and-products-admin-c.md)
- [ ] [Implement table fragments (`/admin/catalog/{kind}/table`) with filter/sort controls and pagination.](doc/admin/tasks/038-implement-table-fragments-admin-catalog-kind-table-with-filter-sort-controls-and-paginatio.md)
- [ ] [Build CRUD modals (`/admin/catalog/{kind}/modal/{new|edit|delete}`) connecting to `POST/PUT/DELETE /admin/catalog/{kind}` endpoints.](doc/admin/tasks/039-build-crud-modals-admin-catalog-kind-modal-new-edit-delete-connecting-to-post-put-delete-a.md)
- [ ] [Support asset uploads (preview images, SVGs) via integration with assets signed URL workflow inside modals.](doc/admin/tasks/040-support-asset-uploads-preview-images-svgs-via-integration-with-assets-signed-url-workflow-.md)
- [ ] [Add versioning/publish status indicators and scheduled publish support where applicable.](doc/admin/tasks/041-add-versioning-publish-status-indicators-and-scheduled-publish-support-where-applicable.md)

## 7. CMS (Guides & Pages)
- [ ] [Implement guides page (`/admin/content/guides`) with table fragment, draft/publish toggles, and scheduled publish UI.](doc/admin/tasks/042-implement-guides-page-admin-content-guides-with-table-fragment-draft-publish-toggles-and-s.md)
- [ ] [Build guide preview route (`/admin/content/guides/{id}/preview?lang=`) with localization selector.](doc/admin/tasks/043-build-guide-preview-route-admin-content-guides-id-preview-lang-with-localization-selector.md)
- [ ] [Implement edit modal or two-pane editor with live preview using htmx for partial refresh.](doc/admin/tasks/044-implement-edit-modal-or-two-pane-editor-with-live-preview-using-htmx-for-partial-refresh.md)
- [ ] [Implement pages management (`/admin/content/pages`) with block-based editor, preview, and publish scheduling.](doc/admin/tasks/045-implement-pages-management-admin-content-pages-with-block-based-editor-preview-and-publish.md)
- [ ] [Ensure content editing includes markdown/HTML sanitization and version history display.](doc/admin/tasks/046-ensure-content-editing-includes-markdown-html-sanitization-and-version-history-display.md)

## 8. Promotions & Marketing
- [ ] [Build promotions index page (`/admin/promotions`) with table fragment, filters (status, type, schedule), and mass actions.](doc/admin/tasks/047-build-promotions-index-page-admin-promotions-with-table-fragment-filters-status-type-sched.md)
- [ ] [Implement promotion modals for create/edit (`/admin/promotions/modal/{new|edit}`) with validation and conditional fields.](doc/admin/tasks/048-implement-promotion-modals-for-create-edit-admin-promotions-modal-new-edit-with-validation.md)
- [ ] [Implement promotion usage view (`/admin/promotions/{promoId}/usages`) with pagination and CSV export capability.](doc/admin/tasks/049-implement-promotion-usage-view-admin-promotions-promoid-usages-with-pagination-and-csv-exp.md)
- [ ] [Integrate promotion dry-run validation UI linking to `POST /admin/promotions:validate` with rule breakdown display.](doc/admin/tasks/050-integrate-promotion-dry-run-validation-ui-linking-to-post-admin-promotions-validate-with-r.md)

## 9. Customers, Reviews, and KYC
- [ ] [Implement customers list page (`/admin/customers`) with search filters (name, email, status) and table fragment.](doc/admin/tasks/051-implement-customers-list-page-admin-customers-with-search-filters-name-email-status-and-ta.md)
- [ ] [Build customer detail page (`/admin/customers/{uid}`) showing orders, addresses, payment methods, and support notes.](doc/admin/tasks/052-build-customer-detail-page-admin-customers-uid-showing-orders-addresses-payment-methods-an.md)
- [ ] [Implement deactivate-and-mask modal tied to `POST /users/{uid}:deactivate-and-mask` with confirmation and audit log output.](doc/admin/tasks/053-implement-deactivate-and-mask-modal-tied-to-post-users-uid-deactivate-and-mask-with-confir.md)
- [ ] [Implement review moderation dashboard (`/admin/reviews?moderation=pending`) with table fragment showing review details and filters.](doc/admin/tasks/054-implement-review-moderation-dashboard-admin-reviews-moderation-pending-with-table-fragment.md)
- [ ] [Build moderation modal(s) for approve/reject (`PUT /admin/reviews/{id}:moderate`) and store reply (`POST /admin/reviews/{id}:store-reply`).](doc/admin/tasks/055-build-moderation-modal-s-for-approve-reject-put-admin-reviews-id-moderate-and-store-reply-.md)

## 10. Production Queues & Org Management
- [ ] [Implement production queue settings page (`/admin/production-queues`) with CRUD modals for queue definitions.](doc/admin/tasks/056-implement-production-queue-settings-page-admin-production-queues-with-crud-modals-for-queu.md)
- [ ] [Provide queue WIP summary view and metrics (capacity, SLA) for operations oversight.](doc/admin/tasks/057-provide-queue-wip-summary-view-and-metrics-capacity-sla-for-operations-oversight.md)
- [ ] [Build staff/role management pages (`/admin/org/staff`, `/admin/org/roles`) or placeholder hooking into Firebase Console if deferred; document admin-only access.](doc/admin/tasks/058-build-staff-role-management-pages-admin-org-staff-admin-org-roles-or-placeholder-hooking-i.md)
- [ ] [Implement role assignment UI once supporting APIs available, including invitations and access revocation.](doc/admin/tasks/059-implement-role-assignment-ui-once-supporting-apis-available-including-invitations-and-acce.md)

## 11. Finance & Accounting
- [ ] [Implement payments transactions page (`/admin/payments/transactions`) with filters by provider, status, date, and amount.](doc/admin/tasks/060-implement-payments-transactions-page-admin-payments-transactions-with-filters-by-provider-.md)
- [ ] [Provide manual capture modal linking to `POST /orders/{id}/payments:manual-capture` with PSP response handling.](doc/admin/tasks/061-provide-manual-capture-modal-linking-to-post-orders-id-payments-manual-capture-with-psp-re.md)
- [ ] [Provide refund modal integration (reuse from orders or dedicated UI) ensuring accounting notes.](doc/admin/tasks/062-provide-refund-modal-integration-reuse-from-orders-or-dedicated-ui-ensuring-accounting-not.md)
- [ ] [Implement tax settings page (`/admin/finance/taxes`) if in scope, with country/region rules management.](doc/admin/tasks/063-implement-tax-settings-page-admin-finance-taxes-if-in-scope-with-country-region-rules-mana.md)
- [ ] [Surface reconciliation reports or export links as required by accounting stakeholders.](doc/admin/tasks/064-surface-reconciliation-reports-or-export-links-as-required-by-accounting-stakeholders.md)

## 12. Logs, Counters, and System Operations
- [ ] [Implement audit log viewer (`/admin/audit-logs`) with table fragment, diff collapsible rows, and filters by target/user/date.](doc/admin/tasks/065-implement-audit-log-viewer-admin-audit-logs-with-table-fragment-diff-collapsible-rows-and-.md)
- [ ] [Build system errors dashboard (`/admin/system/errors`) pulling failed webhook/job logs with retry actions when permitted.](doc/admin/tasks/066-build-system-errors-dashboard-admin-system-errors-pulling-failed-webhook-job-logs-with-ret.md)
- [ ] [Build jobs/tasks monitor (`/admin/system/tasks`) showing scheduler runs, job status, and manual retry triggers.](doc/admin/tasks/067-build-jobs-tasks-monitor-admin-system-tasks-showing-scheduler-runs-job-status-and-manual-r.md)
- [ ] [Implement counters management UI (`/admin/system/counters`) allowing admins to view and test `POST /admin/counters/{name}:next`.](doc/admin/tasks/068-implement-counters-management-ui-admin-system-counters-allowing-admins-to-view-and-test-po.md)
- [ ] [Implement settings page for environment configuration toggles or at minimum a read-only configuration summary.](doc/admin/tasks/069-implement-settings-page-for-environment-configuration-toggles-or-at-minimum-a-read-only-co.md)

## 13. Notifications & Real-Time Feedback
- [ ] [Integrate WebSocket or polling mechanism for real-time updates (new orders, failed jobs, queue changes) within notifications center.](doc/admin/tasks/070-integrate-websocket-or-polling-mechanism-for-real-time-updates-new-orders-failed-jobs-queu.md)
- [ ] [Provide badge counts on sidebar/top bar for pending reviews, alerts, and tasks.](doc/admin/tasks/071-provide-badge-counts-on-sidebar-top-bar-for-pending-reviews-alerts-and-tasks.md)
- [ ] [Implement toast/alert patterns for success/failure responses from htmx requests using response headers or JSON payload.](doc/admin/tasks/072-implement-toast-alert-patterns-for-success-failure-responses-from-htmx-requests-using-resp.md)

## 14. Accessibility, Localization, and UX Enhancements
- [ ] [Audit templates for accessibility (semantic HTML, ARIA attributes, focus management for modals and drag/drop).](doc/admin/tasks/073-audit-templates-for-accessibility-semantic-html-aria-attributes-focus-management-for-modal.md)
- [ ] [Ensure internationalization support (i18n helper usage, date/number formatting, locale switch readiness).](doc/admin/tasks/074-ensure-internationalization-support-i18n-helper-usage-date-number-formatting-locale-switch.md)
- [ ] [Implement keyboard shortcuts (`/` search, `f` filter, `j/k` navigation, `o` open detail, `g` tab switch) with hint overlay.](doc/admin/tasks/075-implement-keyboard-shortcuts-search-f-filter-j-k-navigation-o-open-detail-g-tab-switch-wit.md)
- [ ] [Ensure responsive behaviour for tablet view with collapsible sidebar and touch-friendly controls.](doc/admin/tasks/076-ensure-responsive-behaviour-for-tablet-view-with-collapsible-sidebar-and-touch-friendly-co.md)

## 15. Quality Assurance & Documentation
- [ ] [Write end-to-end tests covering critical workflows (order status change, shipment label generation, promotion creation, review moderation).](doc/admin/tasks/077-write-end-to-end-tests-covering-critical-workflows-order-status-change-shipment-label-gene.md)
- [ ] [Document admin console deployment checklist, including environment variables, Firebase auth setup, and CDN configuration.](doc/admin/tasks/078-document-admin-console-deployment-checklist-including-environment-variables-firebase-auth-.md)
- [ ] [Create user guide for staff with screenshots/workflow instructions hosted in `doc/admin/guide.md`.](doc/admin/tasks/079-create-user-guide-for-staff-with-screenshots-workflow-instructions-hosted-in-doc-admin-gui.md)
- [ ] [Establish bug reporting and feedback process linked from the admin UI (e.g., footer link to issue tracker).](doc/admin/tasks/080-establish-bug-reporting-and-feedback-process-linked-from-the-admin-ui-e-g-footer-link-to-i.md)

## 16. Observability & Maintenance
- [ ] [Instrument server metrics (page render time, htmx fragment duration, error rates) and expose to Cloud Monitoring.](doc/admin/tasks/081-instrument-server-metrics-page-render-time-htmx-fragment-duration-error-rates-and-expose-t.md)
- [ ] [Configure structured logging with request IDs correlating to backend API calls.](doc/admin/tasks/082-configure-structured-logging-with-request-ids-correlating-to-backend-api-calls.md)
- [ ] [Set up uptime checks and alerts for critical admin endpoints (login, orders list, notifications).](doc/admin/tasks/083-set-up-uptime-checks-and-alerts-for-critical-admin-endpoints-login-orders-list-notificatio.md)
- [ ] [Plan ongoing data retention/cleanup jobs for historical audit logs and UI caches.](doc/admin/tasks/084-plan-ongoing-data-retention-cleanup-jobs-for-historical-audit-logs-and-ui-caches.md)
