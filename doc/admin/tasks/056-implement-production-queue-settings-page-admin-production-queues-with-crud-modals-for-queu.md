# Implement production queue settings page (`/admin/production-queues`) with CRUD modals for queue definitions.

**Parent Section:** 10. Production Queues & Org Management
**Task ID:** 056

## Goal
Manage production queue definitions (capacity, roles, stages).

## Implementation Steps
1. Table listing queues with priority, capacity, SLA.
2. CRUD modals to edit queue details and assign work centers.
3. Validate unique names and non-zero capacity.
4. Integrate with backend endpoints.

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` (manage queues, add queue `PrimaryButton`).
- **Queue table:** `DataTable` listing name, capacity, SLA, active toggles with inline edit `IconButton`.
- **Filters:** Simple `FilterToolbar` for workshop, status, product line.
- **Detail panel:** `SidePanel` with queue description, stage definitions, linked staff list.
- **Analytics strip:** `SummaryCard` duo for throughput, WIP limit utilisation.
- **Dialogs:** `Modal` forms for create/edit/delete with `FormField` components.
