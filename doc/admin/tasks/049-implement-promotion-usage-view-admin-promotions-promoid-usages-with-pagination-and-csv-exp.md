# Implement promotion usage view (`/admin/promotions/{promoId}/usages`) with pagination and CSV export capability.

**Parent Section:** 8. Promotions & Marketing
**Task ID:** 049

## Goal
Display promotion usage per user with export option.

## Implementation Steps
1. Table with user email, number of uses, last used, total discount given.
2. Support pagination and export to CSV via background job.
3. Provide filters (>=N uses, timeframe).

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` referencing parent promotion and quick back `Breadcrumb`.
- **Metrics band:** `SummaryCard` row for redemption total, conversion %, avg discount.
- **Filters:** `FilterToolbar` for order source, channel, timeframe, customer segment.
- **Usage table:** `DataTable` listing order id, customer, amount, status `Badge`, and link to order detail.
- **Export controls:** `Toolbar` with CSV export `Button`, scheduled report `MenuButton`, auto-refresh toggle.
- **Anomaly alert:** `InlineAlert` with link to analytics when redemption spikes.
