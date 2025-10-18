# Implement tax settings page (`/admin/finance/taxes`) if in scope, with country/region rules management.

**Parent Section:** 11. Finance & Accounting
**Task ID:** 063

## Goal
Manage tax rates per region.

## Implementation Steps
1. Table of jurisdictions with rates, effective dates.
2. Modals to add/edit rates, with validation for overlapping ranges.
3. Integrate with backend config service.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` (jurisdiction summary, policy links).
- **Region nav:** Left `NavigationList` grouping regions/countries with search `Input`.
- **Rule editor:** `FormCard` for thresholds, rates, registration IDs using `NumberField`, `Toggle`, `DatePicker`.
- **Rate table:** `DataTable` of existing rules with scope, rate, effective dates, actions.
- **Audit panel:** `SidePanel` listing change history `TimelineList`.
- **Validation alerts:** `InlineAlert` for incomplete configs, `Snackbar` on save.
