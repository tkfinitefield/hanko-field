# Implement cart retrieval endpoint keyed by user/session with lazy creation.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 050

## Purpose
Fetch or lazily create the active cart for the authenticated user/session.

## Endpoint
- `GET /cart`

## Implementation Steps
1. Use user UID (or session cookie for guests if supported) to load cart document from `carts` collection.
2. When absent, create new cart with default currency and empty items; return consistent cart ID.
3. Include computed totals (run pricing engine) and metadata (lastUpdatedAt, estimatedTotals).
4. Ensure caching disabled to avoid stale data; include ETag for concurrency if needed.
5. Tests verifying lazy creation and separation between users.
