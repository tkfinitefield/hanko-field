# Implement production kanban page (`/admin/production/queues`) with board fragment (`/admin/production/queues/board`) and drag-and-drop updates posting production events.

**Parent Section:** 5. Orders & Operations > 5.3 Production & Workshop
**Task ID:** 034

## Goal
Provide production kanban board with drag-and-drop updates.

## Implementation Steps
1. Render column per stage (queued, engraving, polishing, qc, packed) using board fragment.
2. Cards show order info, due date, blocking flags.
3. DnD triggers htmx POST to `/admin/orders/{id}/production-events` with new stage.
4. Handle optimistic UI update and revert on failure.
5. Provide filtering by queue or priority.
