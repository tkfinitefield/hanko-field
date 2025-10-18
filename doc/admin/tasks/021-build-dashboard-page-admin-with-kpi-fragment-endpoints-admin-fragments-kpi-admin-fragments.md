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

## UI Components
- **Page shell:** `AdminLayout` with sidebar, top bar, and breadcrumb trail bound to section metadata.
- **Hero band:** `PageHeader` hosting title, environment badge, and refresh `SplitButton` for fragment polling.
- **KPI grid:** Responsive `SummaryCard` components (2xl:4-col, md:2-col) rendering `/fragments/kpi` metrics with sparkline mini charts.
- **Alerts panel:** `ListCard` exposing `/fragments/alerts` entries with severity `Badge` and CTA `IconButton` for ack.
- **Activity rail:** Right-hand `ActivityFeed` column (collapsible) showing latest orders/jobs with inline filters.
- **Empty/error states:** `InlineNotice` slots for fragment timeout or no data, surfaced per section.
