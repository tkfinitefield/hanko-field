# Build dashboard page (`/admin`) with KPI fragment endpoints (`/admin/fragments/kpi`, `/admin/fragments/alerts`).

**Parent Section:** 4. Shared Utilities & System Pages
**Task ID:** 021

## Goal
Implement admin dashboard with KPI and alerts fragments.

## Implementation Steps
1. Create `/admin` handler rendering `dashboard.html` with placeholders for KPI and alerts partials.
2. Implement fragment endpoints `/admin/fragments/kpi` and `/admin/fragments/alerts` returning partial templates (cards/list).
3. Use htmx to poll or refresh fragments on interval, optionally SSE for live updates.
4. Source data from backend API (orders summary, open tickets) via internal client.
5. Provide loading states and error fallbacks.
