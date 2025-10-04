# Implement shipment update endpoint for correcting tracking statuses/events.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 081

## Purpose
Allow manual correction of shipment status or tracking details when carriers send updates.

## Implementation Steps
1. Validate shipment exists; update fields (status, expectedDelivery, notes) with audit logging.
2. Append manual event to shipments events array.
3. Notify customers if status change to delivered/cancelled.
4. Tests verifying concurrency and event append.
