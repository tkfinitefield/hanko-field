# Implement promotion usage retrieval endpoint aggregating counts per user.

**Parent Section:** 6. Admin / Staff Endpoints > 6.2 Promotions
**Task ID:** 075

## Purpose
Allow staff to monitor promotion usage per user for support and analytics.

## Endpoint
- `GET /promotions/{{promoId}}/usages`

## Implementation Steps
1. Query usage sub-collection aggregated by user; support pagination and filters (e.g., `minUsage`).
2. Include user details (UID, email) by joining with user service (with rate limiting).
3. Provide CSV export option via background job if large dataset.
4. Tests verifying pagination, sorting, and access control.
