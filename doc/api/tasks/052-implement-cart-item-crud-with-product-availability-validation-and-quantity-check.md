# Implement cart item CRUD with product availability validation and quantity checks.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 052

## Purpose
Manage cart line items with validation against product availability and quantity rules.

## Endpoints
- `GET /cart/items`
- `POST /cart/items`
- `PUT /cart/items/{{itemId}}`
- `DELETE /cart/items/{{itemId}}`

## Implementation Steps
1. Represent items as sub-collection `carts/{{cartId}}/items` or embedded array with fields: `id`, `productId`, `designId`, `quantity`, `unitPrice`, `options`, `createdAt`, `updatedAt`.
2. Validate quantity >0, product availability (call inventory service), and design ownership.
3. Support merge behaviour for identical items (increase quantity) with idempotency key for POST.
4. Recompute pricing engine after modifications.
5. Tests verifying validation, concurrency, and merge logic.
