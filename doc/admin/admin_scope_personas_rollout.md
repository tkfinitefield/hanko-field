# Admin Console Scope & Rollout Alignment

## Personas & Primary Jobs

| Persona | Core Responsibilities | Critical Admin Areas | Notes |
|---------|-----------------------|----------------------|-------|
| Operations Lead (Ops) | Manage order lifecycle, production throughput, shipping accuracy | `/admin/orders`, `/admin/orders/{id}`, `/admin/shipments/*`, `/admin/production/*`, `/admin/system/tasks` | Needs bulk actions, production Kanban, shipment tracking, and system task visibility for incident response. |
| Customer Support Agent (CS) | Resolve customer inquiries, handle refunds/returns, manage customer records | `/admin/orders/{id}`, `/admin/customers`, `/admin/reviews`, `/admin/notifications`, `/admin/audit-logs` | Requires quick search, audit history, refund flows, and notification triage to support SLAs. |
| Marketing Manager (Marketing) | Launch promotions, curate catalog and content, monitor campaign results | `/admin/catalog/*`, `/admin/promotions`, `/admin/content/*`, `/admin/notifications` | Needs safe drafting/publishing, promo usage analytics, and material/template governance. |

## Scope Confirmation

- **Core navigation** (per `doc/admin/admin_design.md`): dashboard, order management, catalog, content, marketing, customer, and system sections with htmx fragments for partial updates.
- **Operational depth**: order detail tabs (summary, payments, production, shipments, invoice, audit) plus batch shipping, production Kanban, and QC flows confirm coverage for factory + logistics workflows.
- **Growth levers**: catalog CRUD, promotions, and CMS modules satisfy marketing self-serve requirements.
- **Support tooling**: customer detail view, review moderation, notifications, and audit logs anchor CS use cases.

## Rollout Phasing

| Phase | Included Modules | Rationale / Dependencies |
|-------|------------------|---------------------------|
| MVP (Launch) | Dashboard fragments, `/admin/orders` list + detail tabs, refund/status modals, shipment tracking, production Kanban & QC, customer detail, notifications feed, basic catalog (`templates`, `materials`, `products`), promotions CRUD | Unblocks Ops/CS day-one workflows. Depends on API readiness: `GET /admin/orders`, `POST /admin/orders/{id}/production-events`, `POST /orders/{id}/payments:refund`, shipping label service. |
| Phase 1.5 | CMS guides/pages, fonts management, review moderation, audit log viewer, system counters/tasks | Enables marketing and compliance once base API endpoints (`/admin/content/*`, `/admin/reviews`, `/admin/audit-logs`) stabilize; non-critical for cutover but near-term value. |
| Phase 2+ | advanced search (`/admin/search`), finance/tax settings, staff/role management UI, analytics enhancements | Requires additional backend work (search aggregation, RBAC service, finance integration). Can trail initial launch without blocking core operations. |

## Risks & Follow-Ups

- **API dependencies**: confirm delivery timeline for admin-specific endpoints (refunds, production events, content APIs) and coordinate with backend owners. Missing endpoints will block MVP flows.
- **Permission model**: RBAC UI is deferred (Phase 2); define interim policy (Firebase Console managed roles) and document guardrails.
- **Notification reliability**: clarify event sources feeding `/admin/notifications` and ensure monitoring for job failure alerts.
- **Search scope**: `/admin/search` cross-entity search depends on Firestore/Algolia index design; need explicit requirements before Phase 2.
- **Data freshness**: audit log and shipment tracking rely on near-real-time ingestion; confirm streaming vs batch update strategy.

## Alignment Actions

1. Share persona-to-navigation matrix with Ops, CS, Marketing leads for validation (owner: Admin PM).
2. Track API readiness in decision log with responsible backend owners and due dates.
3. Tag backlog items by phase (`MVP`, `P1.5`, `P2+`) and assign owners in project tracker.
4. Schedule rollout review ahead of MVP code freeze to re-confirm dependencies and risk mitigations.

