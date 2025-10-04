# Implement manual payment capture and refund endpoints integrating PSP APIs.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 079

## Purpose
Enable staff to capture outstanding payments or issue refunds via PSP connectors.

## Endpoints
- `POST /orders/{{orderId}}/payments:manual-capture`
- `POST /orders/{{orderId}}/payments:refund`

## Implementation Steps
1. Validate order payment status and ensure actor has permission.
2. Call payment provider through abstraction with idempotency guard.
3. Update payments sub-collection and order balance fields; log event.
4. Handle partial refunds by capturing amount in payload.
5. Tests mocking PSP responses for success/failure, ensuring audit log entry produced.
