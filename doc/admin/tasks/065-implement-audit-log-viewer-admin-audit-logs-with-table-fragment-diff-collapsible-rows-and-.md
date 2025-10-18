# Implement audit log viewer (`/admin/audit-logs`) with table fragment, diff collapsible rows, and filters by target/user/date.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 065

## Goal
Implement audit log viewer with filtering and diff presentation.

## Implementation Steps
1. Handler queries audit log API with filters (targetRef, actor, date range).
2. Table displays action, actor, timestamp, summary; row expands to show diff JSON.
3. Support export to CSV and pagination.
4. Provide search input with debounced htmx requests.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` summarising timeframe and actor filters.
- **Filter controls:** `FilterToolbar` (actor `Combobox`, resource `MultiSelect`, action type chips, date `RangePicker`).
- **Audit table:** `ExpandableDataTable` showing timestamp, actor, action, target; expand to reveal diff `CodePanel`.
- **Diff viewer:** `DiffCard` with before/after JSON using syntax-highlight `CodeBlock`.
- **Export/alerts:** `Toolbar` for export CSV, subscribe to events via `MenuButton`.
- **Alert rail:** `InlineAlert` when retention threshold near or search too broad.
