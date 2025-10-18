# Implement library (`/account/library`) with design list fragment, filters, and actions (duplicate/export/share).

**Parent Section:** 6. Account & Library
**Task ID:** 039

## Goal
Render design library with filters and actions.

## Implementation Steps
1. Build fragment listing designs with search, sort, filter options.
2. Provide actions (duplicate, export, share) using modals and API calls.
3. Render design previews and metadata.

## UI Components
- **Layout:** `AccountLayout` with library `SectionHeader`.
- **Filter bar:** `FilterToolbar` (status chips, tag combobox, sort select, search input).
- **Design grid:** `DesignCardGrid` showing preview, status `Badge`, action menu.
- **Batch actions:** `BulkActionBar` for export/share/delete.
- **Detail drawer:** `DesignDrawer` showing metadata, analytics, quick links.
- **Empty state:** `EmptyState` encouraging new design creation.
