# Implement design update/delete with permission checks and soft delete handling.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 041

## Purpose
Support editing and soft-deleting of designs while preserving history and enforcing ownership.

## Endpoints
- `PUT /designs/{{designId}}`
- `DELETE /designs/{{designId}}`

## Implementation Steps
1. Validate ownership and ensure design not locked (e.g., used in paid order) before allowing update/delete.
2. On update, create new version snapshot in `versions` and update `currentVersionId` + thumbnail.
3. On delete, mark `status=archived` and scrub sensitive data if required rather than hard delete.
4. Emit audit log entries capturing previous values.
5. Tests covering optimistic concurrency (updateTime precondition) and delete restrictions.
