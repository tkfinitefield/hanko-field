# Build order detail (`/orders/:orderId`) showing line items, totals, addresses, and design snapshot gallery.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 048

## Goal
Render comprehensive order detail view.

## Implementation Steps
1. Show order header (id, status, action buttons).
2. Present items, pricing breakdown, addresses, payment summary, design snapshots.
3. Provide quick actions (contact support, reorder, download invoice).
4. Handle loading/error states via `AsyncValue`.
