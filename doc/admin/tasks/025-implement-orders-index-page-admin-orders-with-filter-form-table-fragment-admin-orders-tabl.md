# Implement orders index page (`/admin/orders`) with filter form, table fragment (`/admin/orders/table`), pagination, and bulk actions (status updates, label generation, CSV export).

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 025

## Goal
Deliver orders list with filters, pagination, and bulk actions per design.

## UI Elements
- Filter form with fields: status, since, currency, amount range, hasRefund.
- Table with columns (order #, customer, total, status, updatedAt, badges).
- Bulk actions toolbar for status update, label generation, CSV export.
- htmx fragment `/admin/orders/table` for tbody swaps and pagination links.

## Implementation Steps
1. Build handler for full page rendering base layout and injecting initial table data (first page).
2. Implement fragment handler parsing filters, calling `GET /admin/orders`, mapping API pagination to UI.
3. Add forms for bulk actions using checkboxes; on submit call respective backend or open modal.
4. Provide spinner indicator using `hx-indicator` and maintain query params via `hx-push-url`.
5. Implement export action using async job (download link once ready).
