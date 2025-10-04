# Implement stock reservation release endpoint for manual override of expired reservations.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 084

## Purpose
Allow operations to manually trigger release of expired stock reservations.

## Endpoint
- `POST /stock/reservations:release-expired`

## Implementation Steps
1. Invoke inventory service to scan reservations with `expiresAt < now` and status `reserved`.
2. Update reservations to `released`, adjust stock counts, and log actions.
3. Return summary (count released, SKUs affected).
4. Tests verifying release logic and idempotency.
