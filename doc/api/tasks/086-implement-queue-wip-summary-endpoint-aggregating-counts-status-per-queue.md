# Implement queue WIP summary endpoint aggregating counts/status per queue.

**Parent Section:** 6. Admin / Staff Endpoints > 6.4 Production Queues
**Task ID:** 086

## Purpose
Provide aggregated counts/status for each production queue.

## Endpoint
- `GET /production-queues/{{queueId}}/wip`

## Implementation Steps
1. Aggregate orders assigned to queue by status (waiting, in_progress, blocked).
2. Include metrics such as average age, SLA breach counts.
3. Optimize with Firestore aggregation queries or cached metrics refreshed by background job.
4. Tests verifying aggregation accuracy.
