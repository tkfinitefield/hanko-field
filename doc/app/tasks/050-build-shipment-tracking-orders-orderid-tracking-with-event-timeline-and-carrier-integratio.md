# Build shipment tracking (`/orders/:orderId/tracking`) with event timeline and carrier integration.

**Parent Section:** 8. Orders & Tracking
**Task ID:** 050

## Goal
Display shipment tracking events.

## Implementation Steps
1. Retrieve tracking data (carrier, events) and map to timeline UI.
2. Show location, status, timestamp; highlight current status.
3. Provide fallback instructions if tracking unavailable.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with map `Icon button`.
- **Status header:** `Elevated card` showing current shipment state with `Assist chips` for carrier.
- **Event list:** `Supporting list items` timestamped with trailing `Icon` for milestone.
- **Support actions:** `Filled tonal button` for contact carrier and `Text button` for copy tracking ID.
