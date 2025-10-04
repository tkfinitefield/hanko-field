# Implement reorder endpoint cloning design snapshot and cart lines to new order draft.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 063

## Purpose
Allow users to quickly create new order draft based on prior order's design snapshot and cart lines.

## Endpoint
- `POST /orders/{{orderId}}:reorder`

## Implementation Steps
1. Validate original order belongs to user and status completed/delivered.
2. Copy line items and design snapshots into new cart or draft order document.
3. Ensure inventory availability before confirming reorder; notify user if items discontinued.
4. Return new cart/order reference for checkout.
5. Tests verifying snapshot integrity and unavailability scenarios.
