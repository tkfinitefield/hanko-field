# Implement internal checkout commit endpoint finalizing reservations, marking orders paid, and emitting events.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 102

## Purpose
Finalize checkout after successful payment, committing reservation and updating order status.

## Endpoint
- `POST /internal/checkout/commit`

## Implementation Steps
1. Validate reservation exists and status `reserved`.
2. Mark reservation `committed`, finalize order document (`status=paid`), unlock production queue creation.
3. Update promotion usage and emit events for downstream systems.
4. Tests covering concurrency and idempotency (multiple webhook calls).
