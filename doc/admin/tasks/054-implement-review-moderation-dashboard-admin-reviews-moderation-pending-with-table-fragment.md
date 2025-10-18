# Implement review moderation dashboard (`/admin/reviews?moderation=pending`) with table fragment showing review details and filters.

**Parent Section:** 9. Customers, Reviews, and KYC
**Task ID:** 054

## Goal
Implement moderation interface for pending reviews.

## Implementation Steps
1. Table listing pending reviews with order link, rating, submitted text, attachments.
2. Filters for rating, product, reported status.
3. Inline actions or modals for approve/reject with reason.
4. Provide preview of storefront display.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` and open reviews count chips.
- **Filter tray:** `FilterToolbar` (channel select, rating chip group, flag type, age bucket).
- **Moderation table:** `DataTable` listing review text preview, customer, product, age, flags, actions.
- **Queue controls:** `BulkActionBar` for approve/reject/batch escalate with confirmation modals.
- **Detail inspector:** `SplitPane` with review detail, product snippet, prior moderation history, `ButtonGroup`.
- **Productivity meters:** `SummaryCard` row showing today processed, backlog, SLA timer.
