# Implement internal checkout release endpoint restoring stock on failure/timeout.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 103

## Purpose
Release reservation when payment fails or times out.

## Endpoint
- `POST /internal/checkout/release`

## Implementation Steps
1. Accept reservation ID or order reference.
2. Update reservation to `released`, increment stock counts, log reason.
3. Notify promotion service to roll back usage increments if applicable.
4. Tests verifying idempotent release and concurrency.
