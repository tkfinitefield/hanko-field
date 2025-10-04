# Implement system errors/tasks endpoints reading from failure queues/log storage.

**Parent Section:** 6. Admin / Staff Endpoints > 6.6 Operations Utilities
**Task ID:** 095

## Purpose
Provide visibility into failed jobs and background task status for operations troubleshooting.

## Endpoints
- `GET /system/errors`
- `GET /system/tasks`

## Implementation Steps
1. Aggregate error logs from error collection or monitoring sink (Firestore or Cloud Logging integration).
2. Provide filters by job type, status, date.
3. Expose retry actions when permissible (link to internal endpoints).
4. Ensure PII sanitized before exposing errors to general staff.
5. Tests verifying pagination, filtering, and redaction.
