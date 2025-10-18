# Develop catalog overview page with tabs for templates, fonts, materials, and products (`/admin/catalog/{kind}`).

**Parent Section:** 6. Catalog Management
**Task ID:** 037

## Goal
Provide unified catalog page with tabs for templates, fonts, materials, products.

## Implementation Steps
1. Render top-level page with tab navigation; default to templates.
2. Each tab loads table fragment via htmx specifying `kind` parameter.
3. Persist active tab in query string and highlight accordingly.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` and create asset `PrimaryButton`.
- **Category tabs:** `UnderlineTabs` for Templates, Fonts, Materials, Products.
- **Filter drawer:** `FilterToolbar` (status chips, owner select, tag combobox, updated range).
- **Catalog table/grid:** Toggle between `DataTable` and `CardGrid` views with preview, usage counts, status `Badge`.
- **Bulk actions:** `BulkActionBar` for publish/unpublish/archive with confirmation modals.
- **Metadata rail:** `DetailDrawer` showing selected asset preview, dependencies, audit trail.
