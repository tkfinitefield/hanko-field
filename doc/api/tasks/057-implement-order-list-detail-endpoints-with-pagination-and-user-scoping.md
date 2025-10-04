# Implement order list/detail endpoints with pagination and user scoping.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 057

## Purpose
Allow users to view their orders with pagination and detail view, including payments and shipments.

## Endpoints
- `GET /orders`
- `GET /orders/{{orderId}}`

## Implementation Steps
1. Query `orders` filtered by `userUid` and optional `status`/date range; use pagination helper.
2. Return summary fields for list view (order number, status, total, createdAt) and embed shipments/payment summaries for detail.
3. Enforce ownership and soft-deleted order visibility for reorders.
4. Tests verifying filters, pagination, and unauthorized access.
