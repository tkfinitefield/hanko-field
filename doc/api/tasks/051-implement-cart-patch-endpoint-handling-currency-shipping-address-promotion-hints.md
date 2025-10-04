# Implement cart patch endpoint handling currency, shipping address, promotion hints.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 051

## Purpose
Allow updates to cart-level metadata such as currency, shipping address, and promotion hints.

## Endpoint
- `PATCH /cart`

## Implementation Steps
1. Accept partial updates for `currency`, `shippingAddressId`, `billingAddressId`, `notes`, `promotionHint`.
2. Validate currency supported and addresses belong to user.
3. Update Firestore document with optimistic locking (precondition on updateTime).
4. Recalculate pricing when relevant fields change; return updated totals.
5. Tests covering currency swap, address update, and concurrency conflict.
