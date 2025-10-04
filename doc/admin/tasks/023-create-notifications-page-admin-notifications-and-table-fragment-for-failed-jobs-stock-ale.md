# Create notifications page (`/admin/notifications`) and table fragment for failed jobs, stock alerts, and shipping exceptions.

**Parent Section:** 4. Shared Utilities & System Pages
**Task ID:** 023

## Goal
Display system notifications (failed jobs, stock alerts, shipping exceptions).

## Implementation Steps
1. Build `/admin/notifications` page with filters by category/severity.
2. Implement table fragment `/admin/notifications/table` supporting htmx refresh.
3. Show notification details with modals linking to actions (retry job, view order).
4. Integrate with alert API/backing store.
5. Provide badge counts for top bar integration.
