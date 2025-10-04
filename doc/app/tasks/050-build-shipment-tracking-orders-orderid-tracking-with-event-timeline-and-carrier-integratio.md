# Build shipment tracking (`/orders/:orderId/tracking`) with event timeline and carrier integration.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 050

## Goal
Display shipment tracking events.

## Implementation Steps
1. Retrieve tracking data (carrier, events) and map to timeline UI.
2. Show location, status, timestamp; highlight current status.
3. Provide fallback instructions if tracking unavailable.
