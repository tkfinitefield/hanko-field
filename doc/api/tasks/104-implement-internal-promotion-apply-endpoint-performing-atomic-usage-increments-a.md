# Implement internal promotion apply endpoint performing atomic usage increments and validation.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 104

## Purpose
Provide atomic promotion usage increment for server processes (checkout finalization).

## Endpoint
- `POST /internal/promotions/apply`

## Implementation Steps
1. Accept `code`, `userId`, `cartTotals`; validate eligibility via promotion service.
2. Increment usage counters within transaction to prevent double counting.
3. Return usage snapshot for logging.
4. Tests verifying concurrency and limit enforcement.
