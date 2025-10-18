# Implement payments transactions page (`/admin/payments/transactions`) with filters by provider, status, date, and amount.

**Parent Section:** 11. Finance & Accounting
**Task ID:** 060

## Goal
Display PSP transactions with filters and linking to orders.

## Implementation Steps
1. Table columns: transaction ID, order, provider, amount, status, capturedAt.
2. Filters by provider, status, date range, amount.
3. Provide quick links to order detail and PSP dashboard.

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` showing gross volume, failure rate `SummaryCard` chips.
- **Advanced filters:** `FilterToolbar` with provider `MultiSelect`, status `ChipGroup`, amount `RangeSlider`, settlement date `DatePicker`, risk flag toggle.
- **Transactions table:** `DataTable` featuring expandable rows, PSP reference, order link, payout batch, actions.
- **Reconciliation drawer:** `DetailDrawer` presenting event timeline, raw gateway payload, dispute controls.
- **Batch actions:** `BulkActionBar` for capture, refund, resend receipt.
- **Export controls:** `Toolbar` with download `Button` and saved view `Select`.
