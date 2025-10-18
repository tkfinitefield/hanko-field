# Build order detail page (`/admin/orders/{orderId}`) with tabbed content endpoints (`/admin/orders/{id}/tab/{summary|lines|payments|production|shipments|invoice|audit}`).

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 026

## Goal
Provide order detail with tabbed sections pulling relevant sub-resources.

## Tabs
- Summary (order header, customer info, pricing).
- Lines (items, design thumbnails).
- Payments (history, actions).
- Production (events timeline).
- Shipments (tracking, label status).
- Invoice (PDF link, issue history).
- Audit (audit log entries).

## Implementation Steps
1. Render base page with tab navigation using anchor/btns triggering htmx requests to `/admin/orders/{id}/tab/{name}`.
2. Each tab handler fetches necessary API data (e.g., `GET /orders/{id}/payments`).
3. Provide consistent error handling: show toast and fallback message if API fails.
4. Implement sticky header showing key actions (status change, refund, invoice) accessible from all tabs.

## UI Components
- **Page shell:** `AdminLayout` with breadcrumb `PageHeader` (order id, customer chip, status `Badge`).
- **Summary band:** `SummaryCard` row for financial totals, SLA clocks, outstanding tasks.
- **Primary tabs:** `UnderlineTabs` for Summary, Lines, Payments, Production, Shipments, Invoice, Audit.
- **Content regions:** Each tab renders `DetailCard` grids with tables (`LineItemTable`, `TimelineList`) and action `ButtonGroup`.
- **Context rail:** Right-side `InfoRail` containing customer profile snippet, notes accordion, and timeline feed.
- **Action footer:** Sticky `ActionBar` for refund/status/modal triggers with `SnackbarHost` feedback.
