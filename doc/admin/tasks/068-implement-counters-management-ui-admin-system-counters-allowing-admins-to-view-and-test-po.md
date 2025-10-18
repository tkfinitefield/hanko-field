# Implement counters management UI (`/admin/system/counters`) allowing admins to view and test `POST /admin/counters/{name}:next`.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 068

## Goal
Provide counters management UI.

## Implementation Steps
1. Table showing counter name, scope, current value, last updated.
2. Form to call `POST /admin/counters/{name}:next` for testing (optionally specify scope).
3. Display next value and log action to audit.

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` with namespace selector `Combobox`.
- **Counters table:** `DataTable` listing counter name, current value, increment size, last updated.
- **Test controls:** `InlineForm` housing increment/decrement `NumberField`, action `Button`, result `Badge`.
- **History drawer:** `DetailDrawer` showing recent operations timeline and linked jobs.
- **Alert banner:** `InlineAlert` for counters nearing threshold.
- **Export:** `Toolbar` send-to CSV / copy name `IconButton`.
