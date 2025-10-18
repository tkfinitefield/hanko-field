# Implement guides page (`/admin/content/guides`) with table fragment, draft/publish toggles, and scheduled publish UI.

**Parent Section:** 7. CMS (Guides & Pages)
**Task ID:** 042

## Goal
Manage guide articles with publish scheduling.

## Implementation Steps
1. Build page with filters (status, category, language) and table fragment.
2. Provide toggle buttons for publish/unpublish using htmx.
3. Include schedule UI (datetime picker) hooking into API fields.

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` (guides count, locale `ChipGroup`).
- **Control bar:** `FilterToolbar` containing search `Input`, persona `Select`, publish state `SegmentedControl`, schedule `DatePicker`.
- **Guides table:** `DataTable` with title, locale, author, status `Badge`, scheduled date, and inline actions.
- **Bulk actions:** `BulkActionBar` for publish, unschedule, archive, with progress `Stepper`.
- **Preview drawer:** `DetailDrawer` rendering excerpt, hero image, upcoming changes timeline.
- **Empty state:** `IllustratedEmpty` encouraging new guide creation with CTA button.
