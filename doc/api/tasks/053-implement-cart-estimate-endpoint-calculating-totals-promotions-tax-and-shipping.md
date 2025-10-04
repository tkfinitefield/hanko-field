# Implement cart estimate endpoint calculating totals, promotions, tax, and shipping.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 053

## Purpose
Provide client with estimate of totals (tax, shipping, discounts) without committing order.

## Endpoint
- `POST /cart:estimate`

## Implementation Steps
1. Load cart items and metadata; optionally accept override payload for estimation scenario.
2. Invoke pricing engine; include breakdown fields (`subtotal`, `discount`, `tax`, `shipping`, `total`, `promotion` details).
3. Return structured response matching spec; include warnings for missing shipping address or invalid promotions.
4. Optionally cache recent estimates per cart to reduce recomputation.
5. Tests verifying calculations and error handling when cart empty.
