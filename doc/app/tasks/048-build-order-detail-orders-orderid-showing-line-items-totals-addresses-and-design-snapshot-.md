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

## Material Design 3 Components
- **Top bar:** `Medium top app bar` with actions for reorder and share.
- **Section tabs:** `Primary tabs` for Summary, Timeline, and Files.
- **Content sections:** `Outlined cards` for addresses/payment and `Elevated cards` for design previews.
- **Support banner:** `Banner` for escalation or delays, with `Assist chips` for quick actions.
