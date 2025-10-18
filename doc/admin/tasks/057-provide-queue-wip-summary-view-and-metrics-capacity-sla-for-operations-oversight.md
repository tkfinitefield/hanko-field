# Provide queue WIP summary view and metrics (capacity, SLA) for operations oversight.

**Parent Section:** 10. Production Queues & Org Management
**Task ID:** 057

## Goal
Provide overview of queue workload.

## Implementation Steps
1. Fetch metrics from backend (orders per stage, avg age).
2. Display charts/cards summarizing backlog.
3. Support filters by queue or date.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` emphasising WIP totals and SLA status.
- **Metrics band:** `SummaryCard` grid showing per-queue WIP, capacity, SLA breach count.
- **Visualization:** `BarChartCard`/`LineChartCard` for trend view.
- **Detail table:** `DataTable` enumerating queues with drilldown `LinkButton`.
- **Filters:** `FilterToolbar` for facility, shift, queue type.
- **Alerts:** `InlineAlert` for queues over threshold with quick assign `Button`.
