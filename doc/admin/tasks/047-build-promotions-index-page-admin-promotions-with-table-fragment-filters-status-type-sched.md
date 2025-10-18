# Build promotions index page (`/admin/promotions`) with table fragment, filters (status, type, schedule), and mass actions.

**Parent Section:** 8. Promotions & Marketing
**Task ID:** 047

## Goal
Provide promotions list with filters and mass actions.

## Implementation Steps
1. Table displays code, name, status, start/end dates, usage counts.
2. Filters for status, type, schedule, createdBy; implement with htmx fragment.
3. Bulk actions for activate/deactivate.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` (active promo count, revenue uplift metric chips).
- **Filter toolbar:** Search `Input`, promotion type `MultiSelect`, status `ChipGroup`, schedule `DateRangePicker`.
- **Promotion table:** `DataTable` with name, channel, status `Badge`, schedule, usage counts, inline action `MenuButton`.
- **Batch controls:** `BulkActionBar` enabling activate, pause, clone, delete flows.
- **Segment preview:** `DetailDrawer` summarizing targeting rules, last modified, audit log snippet.
- **Empty/error states:** `IllustratedEmpty` and `InlineAlert` components for dataset hints.
