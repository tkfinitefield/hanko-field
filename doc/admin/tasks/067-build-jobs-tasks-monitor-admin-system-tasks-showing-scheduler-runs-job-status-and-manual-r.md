# Build jobs/tasks monitor (`/admin/system/tasks`) showing scheduler runs, job status, and manual retry triggers.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 067

## Goal
Monitor scheduled jobs and manual tasks.

## Implementation Steps
1. Display list of recent job runs with status, duration, triggered by, logs link.
2. Allow manual trigger for specific tasks (cleanup reservations) with confirmation modal.
3. Support SSE/polling to update job status.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` (scheduler health `Badge`).
- **Filter controls:** `FilterToolbar` (job type, state, host, next run window).
- **Jobs table:** `DataTable` with job id, description, schedule, last/next run, status, controls.
- **Timeline pane:** `RunHistoryChart` showing runtime durations, success/failure trend.
- **Action drawer:** `DetailDrawer` for selected job exposing parameters, logs, manual trigger `Button`.
- **Notification panel:** `InlineAlert` for stalled jobs with quick escalate `Button`.
