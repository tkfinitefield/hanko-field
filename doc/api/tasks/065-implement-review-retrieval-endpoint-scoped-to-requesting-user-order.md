# Implement review retrieval endpoint scoped to requesting user/order.

**Parent Section:** 5. Authenticated User Endpoints > 5.6 Reviews
**Task ID:** 065

## Purpose
Allow user to fetch own reviews optionally filtered by order.

## Endpoint
- `GET /reviews?orderId=` (optional)

## Implementation Steps
1. Query `reviews` by `userUid`; support filter by `orderId`.
2. Return only published fields (status, rating, body, reply) including moderation outcome.
3. Hide moderated-out content if rejected with reason.
4. Tests verifying filter behavior and access control.
