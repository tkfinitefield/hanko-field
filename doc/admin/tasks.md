# Admin Console Implementation Task List

## 0. Planning & Architecture
- [ ] Confirm admin console scope, personas (ops, CS, marketing), and rollout priorities based on `doc/admin/admin_design.md`.
- [ ] Define UI architecture (Go server-side rendering + htmx partials) and routing conventions for full pages vs fragment endpoints.
- [ ] Model navigation taxonomy and RBAC visibility rules; document mapping between sidebar groups and user roles.
- [ ] Produce data contract checklist mapping every admin action to the corresponding API endpoint and required request/response fields.

## 1. Project & Infrastructure Setup
- [ ] Scaffold Go web module (`/admin` or `/web`) with templating pipeline, asset bundler (Tailwind), and dev tooling (hot reload, lint).
- [ ] Configure chi/echo router with middleware for auth, CSRF, caching headers, and htmx request detection.
- [ ] Establish HTML template structure (`layouts`, `partials`, `components`) and helper functions (i18n, currency, date formatting).
- [ ] Implement Tailwind configuration, design tokens, and base components library (buttons, tables, forms, modals, toasts).
- [ ] Set up integration tests harness (httptest + DOM assertions) and smoke test environment for admin flows.

## 2. Authentication, Authorization, and Session Management
- [ ] Implement Firebase ID token verification for staff/admin, including refresh and error handling UX.
- [ ] Build session middleware storing user info, CSRF tokens, and feature flags; add remember-me support if required.
- [ ] Implement RBAC guard utilities for template rendering (sidebar filtering, action visibility).
- [ ] Create login/logout flows (`/admin/login`, `/admin/logout`) with error states and redirect logic.
- [ ] Add MFA/API-key management UI under `/admin/profile`, integrating with backend secrets service.

## 3. Layout, Navigation, and Shared UX
- [ ] Implement global layout template (`/admin/_layout.html`) with sidebar, top bar, breadcrumbs, and responsive behaviour.
- [ ] Build dynamic sidebar rendering with route highlighting and permission-based sections.
- [ ] Create top bar components (environment badge, search shortcut, notification icon, user menu).
- [ ] Implement modal container (`#modal`) with htmx target wiring, animations, and escape-key close behaviour.
- [ ] Provide reusable pagination controls, table headers with sort indicators, and bulk action toolbar components.
- [ ] Implement toast/alert system triggered by htmx response headers.

## 4. Shared Utilities & System Pages
- [ ] Build dashboard page (`/admin`) with KPI fragment endpoints (`/admin/fragments/kpi`, `/admin/fragments/alerts`).
- [ ] Implement global search page (`/admin/search`) and result fragment (`/admin/search/table`) with filtering and keyboard shortcuts.
- [ ] Create notifications page (`/admin/notifications`) and table fragment for failed jobs, stock alerts, and shipping exceptions.
- [ ] Implement profile/account page (`/admin/profile`) with 2FA setup, password/API key management, and session history.

## 5. Orders & Operations
### 5.1 Orders List & Detail
- [ ] Implement orders index page (`/admin/orders`) with filter form, table fragment (`/admin/orders/table`), pagination, and bulk actions (status updates, label generation, CSV export).
- [ ] Build order detail page (`/admin/orders/{orderId}`) with tabbed content endpoints (`/admin/orders/{id}/tab/{summary|lines|payments|production|shipments|invoice|audit}`).
- [ ] Implement status update modal (`/admin/orders/{id}/modal/status`) posting to `PUT /admin/orders/{id}:status` and updating the UI inline.
- [ ] Implement refund modal (`/admin/orders/{id}/modal/refund`) integrating with `POST /orders/{id}/payments:refund` and showing validation errors.
- [ ] Implement invoice request modal (`/admin/orders/{id}/modal/invoice`) calling `POST /admin/invoices:issue` and reflecting result in tabs.
- [ ] Add bulk export and print actions (CSV, PDF) with progress feedback.

### 5.2 Shipments & Tracking
- [ ] Create shipment batch page (`/admin/shipments/batches`) for label generation workflows and integration with shipment POST endpoints.
- [ ] Implement shipment tracking monitor (`/admin/shipments/tracking`) with table fragment (`/admin/shipments/tracking/table`) and filtering by carrier/status.
- [ ] Hook carrier webhook data or Firestore views to populate tracking dashboard, including exception badges and SLA indicators.

### 5.3 Production & Workshop
- [ ] Implement production kanban page (`/admin/production/queues`) with board fragment (`/admin/production/queues/board`) and drag-and-drop updates posting production events.
- [ ] Create work order view (`/admin/production/workorders/{orderId}`) summarizing design assets, instructions, and tasks.
- [ ] Build QC page (`/admin/production/qc`) to record pass/fail events and trigger rework flows.

## 6. Catalog Management
- [ ] Develop catalog overview page with tabs for templates, fonts, materials, and products (`/admin/catalog/{kind}`).
- [ ] Implement table fragments (`/admin/catalog/{kind}/table`) with filter/sort controls and pagination.
- [ ] Build CRUD modals (`/admin/catalog/{kind}/modal/{new|edit|delete}`) connecting to `POST/PUT/DELETE /admin/catalog/{kind}` endpoints.
- [ ] Support asset uploads (preview images, SVGs) via integration with assets signed URL workflow inside modals.
- [ ] Add versioning/publish status indicators and scheduled publish support where applicable.

## 7. CMS (Guides & Pages)
- [ ] Implement guides page (`/admin/content/guides`) with table fragment, draft/publish toggles, and scheduled publish UI.
- [ ] Build guide preview route (`/admin/content/guides/{id}/preview?lang=`) with localization selector.
- [ ] Implement edit modal or two-pane editor with live preview using htmx for partial refresh.
- [ ] Implement pages management (`/admin/content/pages`) with block-based editor, preview, and publish scheduling.
- [ ] Ensure content editing includes markdown/HTML sanitization and version history display.

## 8. Promotions & Marketing
- [ ] Build promotions index page (`/admin/promotions`) with table fragment, filters (status, type, schedule), and mass actions.
- [ ] Implement promotion modals for create/edit (`/admin/promotions/modal/{new|edit}`) with validation and conditional fields.
- [ ] Implement promotion usage view (`/admin/promotions/{promoId}/usages`) with pagination and CSV export capability.
- [ ] Integrate promotion dry-run validation UI linking to `POST /admin/promotions:validate` with rule breakdown display.

## 9. Customers, Reviews, and KYC
- [ ] Implement customers list page (`/admin/customers`) with search filters (name, email, status) and table fragment.
- [ ] Build customer detail page (`/admin/customers/{uid}`) showing orders, addresses, payment methods, and support notes.
- [ ] Implement deactivate-and-mask modal tied to `POST /users/{uid}:deactivate-and-mask` with confirmation and audit log output.
- [ ] Implement review moderation dashboard (`/admin/reviews?moderation=pending`) with table fragment showing review details and filters.
- [ ] Build moderation modal(s) for approve/reject (`PUT /admin/reviews/{id}:moderate`) and store reply (`POST /admin/reviews/{id}:store-reply`).

## 10. Production Queues & Org Management
- [ ] Implement production queue settings page (`/admin/production-queues`) with CRUD modals for queue definitions.
- [ ] Provide queue WIP summary view and metrics (capacity, SLA) for operations oversight.
- [ ] Build staff/role management pages (`/admin/org/staff`, `/admin/org/roles`) or placeholder hooking into Firebase Console if deferred; document admin-only access.
- [ ] Implement role assignment UI once supporting APIs available, including invitations and access revocation.

## 11. Finance & Accounting
- [ ] Implement payments transactions page (`/admin/payments/transactions`) with filters by provider, status, date, and amount.
- [ ] Provide manual capture modal linking to `POST /orders/{id}/payments:manual-capture` with PSP response handling.
- [ ] Provide refund modal integration (reuse from orders or dedicated UI) ensuring accounting notes.
- [ ] Implement tax settings page (`/admin/finance/taxes`) if in scope, with country/region rules management.
- [ ] Surface reconciliation reports or export links as required by accounting stakeholders.

## 12. Logs, Counters, and System Operations
- [ ] Implement audit log viewer (`/admin/audit-logs`) with table fragment, diff collapsible rows, and filters by target/user/date.
- [ ] Build system errors dashboard (`/admin/system/errors`) pulling failed webhook/job logs with retry actions when permitted.
- [ ] Build jobs/tasks monitor (`/admin/system/tasks`) showing scheduler runs, job status, and manual retry triggers.
- [ ] Implement counters management UI (`/admin/system/counters`) allowing admins to view and test `POST /admin/counters/{name}:next`.
- [ ] Implement settings page for environment configuration toggles or at minimum a read-only configuration summary.

## 13. Notifications & Real-Time Feedback
- [ ] Integrate WebSocket or polling mechanism for real-time updates (new orders, failed jobs, queue changes) within notifications center.
- [ ] Provide badge counts on sidebar/top bar for pending reviews, alerts, and tasks.
- [ ] Implement toast/alert patterns for success/failure responses from htmx requests using response headers or JSON payload.

## 14. Accessibility, Localization, and UX Enhancements
- [ ] Audit templates for accessibility (semantic HTML, ARIA attributes, focus management for modals and drag/drop).
- [ ] Ensure internationalization support (i18n helper usage, date/number formatting, locale switch readiness).
- [ ] Implement keyboard shortcuts (`/` search, `f` filter, `j/k` navigation, `o` open detail, `g` tab switch) with hint overlay.
- [ ] Ensure responsive behaviour for tablet view with collapsible sidebar and touch-friendly controls.

## 15. Quality Assurance & Documentation
- [ ] Write end-to-end tests covering critical workflows (order status change, shipment label generation, promotion creation, review moderation).
- [ ] Document admin console deployment checklist, including environment variables, Firebase auth setup, and CDN configuration.
- [ ] Create user guide for staff with screenshots/workflow instructions hosted in `doc/admin/guide.md`.
- [ ] Establish bug reporting and feedback process linked from the admin UI (e.g., footer link to issue tracker).

## 16. Observability & Maintenance
- [ ] Instrument server metrics (page render time, htmx fragment duration, error rates) and expose to Cloud Monitoring.
- [ ] Configure structured logging with request IDs correlating to backend API calls.
- [ ] Set up uptime checks and alerts for critical admin endpoints (login, orders list, notifications).
- [ ] Plan ongoing data retention/cleanup jobs for historical audit logs and UI caches.
