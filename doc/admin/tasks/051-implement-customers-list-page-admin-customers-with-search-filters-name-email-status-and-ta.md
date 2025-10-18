# Implement customers list page (`/admin/customers`) with search filters (name, email, status) and table fragment.

**Parent Section:** 9. Customers, Reviews, and KYC
**Task ID:** 051

## Goal
Implement customers list with search filters.

## Implementation Steps
1. Filters: name/email, status (active, deactivated), tier.
2. Table columns: customer, total orders, lifetime value, last order, flags.
3. Integrate with search API or `GET /users` admin endpoint.
4. Provide row action to open detail.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` showing total customers and segmentation chips.
- **Filter + search:** `FilterToolbar` containing global search `Input`, status `ChipGroup`, persona `MultiSelect`, tag `Combobox`, spending `RangeSlider`.
- **Customers table:** `DataTable` with avatar, name/email, lifetime value, last order date, risk `Badge`, actions menu.
- **Bulk operations:** `BulkActionBar` for segment tagging, export, messaging triggers.
- **Preview drawer:** `DetailDrawer` summarizing profile, notes, last interactions, quick links.
- **Empty state:** `IllustratedEmpty` with CTA to import CSV.
