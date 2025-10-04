# Hook carrier webhook data or Firestore views to populate tracking dashboard, including exception badges and SLA indicators.

**Parent Section:** 5. Orders & Operations > 5.2 Shipments & Tracking
**Task ID:** 033

## Goal
Connect webhook or Firestore data to tracking dashboard.

## Steps
1. Implement repository function to aggregate shipments by status using backend data store.
2. Provide caching layer for dashboard to avoid heavy queries; invalidate on webhook arrival.
3. Include SLA/exceptions pipeline marking shipments needing attention.
