# Implement apply/remove promo endpoints interacting with promotion service and idempotency.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 054

## Purpose
Manage application of promotion codes on cart with proper validation and usage tracking.

## Endpoints
- `POST /cart:apply-promo`
- `DELETE /cart:remove-promo`

## Implementation Steps
1. Validate promo code via promotion service; ensure usage limits and eligibility satisfied.
2. Update cart document with `appliedPromotions[]` entry storing code, discount type, estimated value.
3. On remove, delete entry and recalculate totals.
4. For apply, store idempotency key to prevent duplicate evaluation.
5. Tests verifying eligibility errors, limit per user, and recalculation of totals.
