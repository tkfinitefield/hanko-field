# Implement audit log viewer (`/admin/audit-logs`) with table fragment, diff collapsible rows, and filters by target/user/date.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 065

## Goal
Implement audit log viewer with filtering and diff presentation.

## Implementation Steps
1. Handler queries audit log API with filters (targetRef, actor, date range).
2. Table displays action, actor, timestamp, summary; row expands to show diff JSON.
3. Support export to CSV and pagination.
4. Provide search input with debounced htmx requests.
