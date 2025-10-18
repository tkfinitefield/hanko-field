# Implement production timeline (`/orders/:orderId/production`) visualizing stages and timestamps.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 049

## Goal
Visualize production progress for an order.

## Implementation Steps
1. Fetch production events from backend; sort chronologically.
2. Display timeline with stage icons, timestamps, notes.
3. Provide estimated completion and alerts for delays.
4. Support real-time updates via polling or push notifications.

## Material Design 3 Components
- **App bar:** `Small top app bar` with refresh `Icon button`.
- **Timeline:** Vertical `List` using `Step list items` separated by `Dividers` and decorated `Icons`.
- **Status chips:** `Assist chips` representing stage SLA health (On track, Attention, Delayed).
- **Context:** `Outlined card` summarizing order metadata at top.
