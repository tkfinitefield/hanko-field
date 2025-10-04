# Implement queue assign-order endpoint ensuring concurrency control and queue policies.

**Parent Section:** 6. Admin / Staff Endpoints > 6.4 Production Queues
**Task ID:** 087

## Purpose
Assign orders to production queue following capacity rules and concurrency control.

## Endpoint
- `POST /production-queues/{{queueId}}:assign-order`

## Implementation Steps
1. Validate queue capacity and order status before assignment.
2. Use transaction to update order document with `productionQueueId`, `assignedAt`, `assignedBy`.
3. Append assignment event to production events.
4. Return updated order summary.
5. Tests verifying concurrency and capacity enforcement.
