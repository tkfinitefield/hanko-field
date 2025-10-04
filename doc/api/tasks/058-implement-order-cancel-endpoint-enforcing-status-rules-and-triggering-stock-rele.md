# Implement order cancel endpoint enforcing status rules and triggering stock release.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 058

## Purpose
Allow user to cancel order prior to production/shipping, releasing stock and reversing payments as needed.

## Endpoint
- `POST /orders/{{orderId}}:cancel`

## Implementation Steps
1. Validate order belongs to user and status in cancellable states (`pending_payment`, `paid`).
2. Trigger inventory reservation release and inform payment provider (void/cancel intent if not captured).
3. Update order status to `cancelled`, record reason, emit audit log.
4. Notify downstream services (email, admin dashboards).
5. Tests covering status validation, concurrency, and failure rollbacks.
