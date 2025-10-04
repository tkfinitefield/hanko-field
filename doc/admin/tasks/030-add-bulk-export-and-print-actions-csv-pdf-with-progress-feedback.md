# Add bulk export and print actions (CSV, PDF) with progress feedback.

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 030

## Goal
Support mass CSV/PDF export of orders.

## Implementation Steps
1. Bulk action triggers background job request with filters and selected IDs.
2. Show progress indicator (modal or toast) and provide link once job completes (polling or SSE).
3. Ensure exports respect RBAC (only allowed fields) and handle large data via streaming.
