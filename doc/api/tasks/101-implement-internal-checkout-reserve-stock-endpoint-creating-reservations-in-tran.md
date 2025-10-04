# Implement internal checkout reserve-stock endpoint creating reservations in transaction-safe manner and decrementing stock.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 101

## Purpose
Reserve inventory for checkout prior to payment completion.

## Endpoint
- `POST /internal/checkout/reserve-stock`

## Implementation Steps
1. Authenticate via OIDC/IAP token; available only to services.
2. Accept payload with `orderId`, `userRef`, `lines[]`, `ttlSec`.
3. Use inventory service transaction to decrement available stock and create reservation document.
4. Return reservation status and expiry.
5. Emit monitoring metrics (reservations created, failures).
6. Tests using Firestore emulator verifying concurrency and TTL behaviour.
