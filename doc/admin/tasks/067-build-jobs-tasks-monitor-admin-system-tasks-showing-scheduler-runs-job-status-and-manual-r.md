# Build jobs/tasks monitor (`/admin/system/tasks`) showing scheduler runs, job status, and manual retry triggers.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 067

## Goal
Monitor scheduled jobs and manual tasks.

## Implementation Steps
1. Display list of recent job runs with status, duration, triggered by, logs link.
2. Allow manual trigger for specific tasks (cleanup reservations) with confirmation modal.
3. Support SSE/polling to update job status.
