# Implement production queue CRUD endpoints storing capacity, priorities, and metadata.

**Parent Section:** 6. Admin / Staff Endpoints > 6.4 Production Queues
**Task ID:** 085

## Purpose
Manage production queues (engraving, finishing) with metadata controlling order assignments.

## Endpoints
- `GET /production-queues`
- `POST /production-queues`
- `PUT /production-queues/{{queueId}}`
- `DELETE /production-queues/{{queueId}}`

## Implementation Steps
1. Model queue documents: `id`, `name`, `capacity`, `workCenters`, `priority`, `status`, `notes`.
2. Prevent deletion when queue has active assignments; require reassign first.
3. Provide audit logs for configuration changes.
4. Tests verifying RBAC and constraint enforcement.
