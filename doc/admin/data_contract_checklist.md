# Admin UI Data Contract Checklist

Each admin interaction references a backend endpoint. This checklist enumerates the expected contracts so API and frontend teams stay aligned. Unless noted otherwise, responses follow the common envelope `{ "data": ... }` pattern with standard error schema.

## Dashboard & Global

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Load KPI cards | `/admin/fragments/kpi` | GET | `since` (optional ISO datetime) | `kpis[]` (id, label, value, delta, trend) | Returns HTML fragment via `templ`; backing API aggregates Firestore metrics. |
| Load alert feed | `/admin/fragments/alerts` | GET | — | `alerts[]` (id, severity, message, createdAt) | Provides HTML fragment. Source data pulled from Firestore/Cloud Tasks logs. |
| Global search | `/admin/search` | GET | `q`, `type[]`, `status`, pagination cursors | `results[]` (type, id, headline, snippet, url, badges) | Requires search index (Algolia/Firestore). Fragment endpoint `/admin/search/table`. |
| Notifications list | `/admin/notifications` | GET | `cursor`, `filter` | `notifications[]` (id, category, message, entityRef, createdAt, read) | Shows failure jobs/inventory alerts; ensure streaming updates. |

## Orders & Fulfilment

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Fetch orders table | `/admin/orders` | GET | `status`, `since`, `currency`, `amountMin`, `amountMax`, `hasRefund`, `page`, `pageSize`, `sort` | `orders[]` (id, orderNumber, status, paymentStatus, customerName, totalAmount, currency, createdAt, updatedAt), `pagination` (page, totalPages, nextCursor) | Full page for SSR. Fragment reload via `/admin/orders/table`. |
| Fetch order detail | `/orders/{id}` | GET | `expand` (tabs) | `order` (id, lineItems[], statusHistory[], paymentInfo, shippingInfo, productionEvents[]) | Shared API with public admin scope; ensure RBAC. |
| Update order status | `/admin/orders/{id}:status` | PUT | `status`, `note`, `notifyCustomer` | `orderID`, `status`, `previousStatus`, `updatedAt`, `actor` | Trigger timeline entry + email when `notifyCustomer` true. |
| Issue refund | `/orders/{id}/payments:refund` | POST | `paymentID`, `amount`, `reason`, `memo` | `refundID`, `status`, `amount`, `processedAt` | PSP integration; restrict to `admin`,`support`. |
| Issue invoice | `/admin/invoices:issue` | POST | `orderID`, `deliveryEmail`, `note`, `language` | `invoiceID`, `status`, `pdfURL`, `issuedAt` | Saves audit log entry. |
| Bulk status transition | `/admin/orders:bulk-status` | POST | `orderIDs[]`, `status`, `note` | `results[]` (orderID, status, error?) | Endpoint name TBD in API design; highlight dependency. |
| Generate shipment labels | `/admin/orders/{id}/shipments` | POST | `carrier`, `service`, `packages[]` (weight, dimensions), `shipDate` | `shipmentID`, `trackingNumber`, `labelURL`, `cost`, `estimatedDelivery` | For batch mode use `/admin/shipments/batches` (TBD). |
| Track shipments | `/admin/shipments/tracking` | GET | `status`, `carrier`, `cursor` | `shipments[]` (orderID, shipmentID, trackingNumber, status, eta, lastEventAt) | Fragment `/admin/shipments/tracking/table`. |
| Add production event | `/admin/orders/{id}/production-events` | POST | `type` (queued/engraving/polishing/qc/packed), `note`, `occurredAt` | `eventID`, `type`, `status`, `createdAt` | Used by Kanban D&D updates. |
| Fetch production queues | `/admin/production/queues` | GET | `workspace`, `lane` | `lanes[]` (name, orders[] with summary) | Fragment `/admin/production/queues/board`. |
| QC update | `/admin/production/qc` | POST | `orderID`, `result` (`pass`/`fail`), `notes` | `qcID`, `result`, `inspectedAt` | Failing triggers notification. |
| Generate CSV export | `/admin/orders:export` | POST | `filters` (same as list), `fields[]`, `format` | `jobID`, `status`, `downloadURL` (when ready) | Async job; use Notifications when ready. |

## Catalog Management

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Create catalog item | `/admin/catalog/{kind}` | POST | Common: `name`, `description`, `status`; Templates: `templateID`, `svgPath`; Fonts: `fontID`, `family`, `weights[]`; Materials: `materialID`, `sku`, `color`, `inventory`; Products: `productID`, `price`, `currency`, `leadTime`, `photoURLs[]` | `id`, `kind`, `createdAt`, echo of fields | `kind ∈ {templates, fonts, materials, products}`. |
| Update catalog item | `/admin/catalog/{kind}/{id}` | PUT | Same as create + `version` for concurrency | `id`, `updatedAt`, `version` | Use optimistic locking. |
| Delete catalog item | `/admin/catalog/{kind}/{id}` | DELETE | `version` | `{ "deleted": true }` | Soft delete if needed. |
| Fetch catalog table | `/admin/catalog/{kind}` | GET | `status`, `q`, pagination | `items[]` (id, name, status, tags, updatedAt) | Fragment `/admin/catalog/{kind}/table`. |

## Content (CMS)

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Create guide | `/admin/content/guides` | POST | `title`, `slug`, `lang`, `body`, `tags[]`, `publishAt`, `authorID`, `status` (`draft`/`published`) | `guideID`, `status`, `publishAt`, `version` | Provides preview link. |
| Update guide | `/admin/content/guides/{id}` | PUT | Fields as above + `version` | `guideID`, `status`, `updatedAt`, `version` | |
| Delete guide | `/admin/content/guides/{id}` | DELETE | `version`, `deleteReason` | `{ "deleted": true }` | |
| Preview guide | `/admin/content/guides/{id}/preview` | GET | `lang` | `html` preview payload | Rendered HTML fragment. |
| Manage fixed page | `/admin/content/pages/{id}` | PUT | `blocks[]` (type, content, order), `seo`, `version` | `pageID`, `updatedAt`, `version` | Block editor sync. |

## Promotions & Marketing

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Create promotion | `/admin/promotions` | POST | `code`, `type` (`percent`/`fixed`), `value`, `currency`, `startAt`, `endAt`, `usageLimit`, `eligibleProducts[]`, `notes` | `promotionID`, `status`, `createdAt` | |
| Update promotion | `/admin/promotions/{id}` | PUT | Same as create + `version`, `status` (`active`/`paused`) | `promotionID`, `status`, `updatedAt`, `version` | |
| Delete promotion | `/admin/promotions/{id}` | DELETE | `version` | `{ "deleted": true }` | Soft delete recommended. |
| Promotion usage report | `/admin/promotions/{id}/usages` | GET | `cursor`, `orderStatus` | `usages[]` (orderID, userID, amount, redeemedAt) | Table fragment. |
| Review moderation | `/admin/reviews/{id}:moderate` | PUT | `decision` (`approve`/`reject`), `reason`, `publicReply?` | `reviewID`, `status`, `moderatedBy`, `moderatedAt` | |
| Store reply | `/admin/reviews/{id}:store-reply` | POST | `body`, `visible` (bool) | `replyID`, `status`, `createdAt` | |

## Customers & Support

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| Fetch customers | `/admin/customers` | GET | `q`, `status`, `since`, pagination | `customers[]` (userID, name, email, lastOrderAt, lifetimeValue, status) | |
| View customer detail | `/admin/customers/{uid}` | GET | `expand` | `customer` (profile, addresses[], paymentMethods[], ordersSummary[]) | |
| Deactivate & mask user | `/users/{uid}:deactivate-and-mask` | POST | `reason`, `initiator` | `userID`, `status`, `maskedAt`, `auditID` | Irreversible; requires admin. |
| Create support note | `/admin/customers/{uid}/notes` | POST | `body`, `category`, `visibility` | `noteID`, `createdAt`, `author` | Endpoint naming TBD. |
| Fetch notifications | `/admin/notifications` | GET | `since`, `category` | `notifications[]` | Shared with dashboard table. |

## System & Operations

| Action | Endpoint | Method | Request Fields | Response Fields | Notes |
|--------|----------|--------|----------------|-----------------|-------|
| View audit logs | `/admin/audit-logs` | GET | `targetRef`, `actor`, `since`, `eventType`, pagination | `entries[]` (id, actor, action, targetRef, diff, createdAt) | Fragment `/admin/audit-logs/table`. |
| Fetch system errors | `/admin/system/errors` | GET | `service`, `severity`, pagination | `errors[]` (id, service, message, occurredAt, status) | Display stack snippet. |
| Retry job | `/admin/system/tasks/{id}:retry` | POST | `reason` | `taskID`, `status`, `queuedAt`, `nextRunAt` | Endpoint inferred; confirm API support. |
| View counters | `/admin/system/counters` | GET | `search` | `counters[]` (name, currentValue, updatedAt) | |
| Increment counter | `/admin/counters/{name}:next` | POST | `amount` (default 1) | `name`, `value`, `incrementedAt` | Returns new value. |
| Manage production queue settings | `/admin/production-queues` | POST/PUT | `queueID`, `name`, `steps[]`, `enabled` | `queueID`, `updatedAt` | Required for ops configuration. |
| Staff management list | `/admin/org/staff` | GET | `role`, `search` | `staff[]` (userID, email, roles[], lastActiveAt) | Pending API completion. |
| Update staff roles | `/admin/org/staff/{uid}` | PUT | `roles[]`, `notes`, `initiator` | `userID`, `roles[]`, `updatedAt` | Works with Firebase custom claims. |

## Checklist Gaps & Dependencies

- **Bulk order actions** (`:bulk-status`, batch shipment label, CSV export) require API definition and Firestore job orchestration.
- **Support notes** and **job retry** endpoints are inferred; confirm alignment with backend roadmap.
- **Search** relies on indexing strategy (Firestore export vs Algolia); finalize response schema before implementation.
- **Staff management** depends on RBAC service to mutate Firebase claims atomically.

Teams should reference this document during API reviews. Update entries as endpoints stabilize; any schema change must include versioning notes and template adjustments.
