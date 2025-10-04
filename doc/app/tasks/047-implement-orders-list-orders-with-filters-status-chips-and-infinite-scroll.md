# Implement orders list (`/orders`) with filters, status chips, and infinite scroll.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 047

## Goal
Implement orders list with filters and infinite scroll.

## Implementation Steps
1. Fetch paginated orders using repository with caching.
2. Provide filter chips for status and date range; integrate with provider state.
3. Display cards containing snapshot, total, status timeline snippet.
4. Implement pull-to-refresh and skeleton states.
