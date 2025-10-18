# Implement orders list (`/account/orders`) with filterable table fragment and pagination.

**Parent Section:** 6. Account & Library
**Task ID:** 037

## Goal
Implement account order history.

## Implementation Steps
1. Build filter form (status/date range) and fragment `/account/orders/table` with pagination.
2. Link to order detail pages.
3. Handle empty state and loading.

## UI Components
- **Layout:** `AccountLayout` with orders `SectionHeader`.
- **Filter controls:** `FilterToolbar` (status chips, timeframe select, search input).
- **Orders table:** `OrdersTable` fragment listing number, date, total, status `Badge`, action link.
- **Pagination:** `Pager` with page count and load-more button.
- **Empty state:** `EmptyState` with CTA to start new order.
- **Support banner:** `InlineHelp` linking to support resources.
