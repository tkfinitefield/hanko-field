# Implement production events POST endpoint allowing operations to append workflow events.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 082

## Purpose
Enable production staff to append workflow events (engraving complete, QC failed) to orders.

## Endpoint
- `POST /orders/{{orderId}}/production-events`

## Implementation Steps
1. Accept payload with `stage`, `status`, `notes`, `operator`, optional attachments.
2. Append to `productionEvents` sub-collection; update summary fields on order.
3. Integrate with production queue service to adjust WIP metrics.
4. Tests verifying stage validation and notifications.
