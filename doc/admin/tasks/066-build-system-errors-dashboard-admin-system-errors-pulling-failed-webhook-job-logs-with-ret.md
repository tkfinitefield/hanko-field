# Build system errors dashboard (`/admin/system/errors`) pulling failed webhook/job logs with retry actions when permitted.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 066

## Goal
Show failed jobs/webhooks with retry options.

## Implementation Steps
1. Table with columns: job name, error, lastAttempt, retries, actions (retry, acknowledge).
2. Integrate with monitoring API or Firestore collection.
3. Provide filters for job type/severity.
4. Add quick links to related order/promotion.

## UI Components
- **Page shell:** `AdminLayout` + `PageHeader` projecting current error rate KPI chips.
- **Filter tray:** `FilterToolbar` for source (webhook/job/api), severity, service, time range.
- **Error table:** `DataTable` with error message excerpt, target, count, last seen, retry `Button`.
- **Metrics cards:** `SummaryCard` row for total failures, retry success, queue backlog.
- **Detail inspector:** `DetailDrawer` showing payload, stack trace, retry controls.
- **Runbooks:** `InlineAlert` linking to docs & `Accordion` for mitigation steps.
