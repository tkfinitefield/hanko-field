# Implement shipment creation endpoint generating labels via carrier integrations and storing tracking info.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 080

## Purpose
Allow operations to create shipments, optionally integrating with carrier APIs to generate labels.

## Implementation Steps
1. Accept payload with carrier, service level, tracking preference, package dimensions.
2. Call shipping integration to generate label/tracking number; store in shipments sub-collection.
3. Optionally emit event to notify customer.
4. Update order status to `shipped` if all items dispatched.
5. Tests verifying integration mocks, error handling, and partial shipment scenarios.
