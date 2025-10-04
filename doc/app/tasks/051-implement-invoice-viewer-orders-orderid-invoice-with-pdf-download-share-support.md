# Implement invoice viewer (`/orders/:orderId/invoice`) with PDF download/share support.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 051

## Goal
Provide invoice viewer with download support.

## Implementation Steps
1. Fetch invoice metadata and secure download link.
2. Render summary details; use native PDF viewer where available.
3. Handle pending invoice state with refresh option.
