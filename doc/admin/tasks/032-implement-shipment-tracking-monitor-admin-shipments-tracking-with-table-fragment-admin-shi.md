# Implement shipment tracking monitor (`/admin/shipments/tracking`) with table fragment (`/admin/shipments/tracking/table`) and filtering by carrier/status.

**Parent Section:** 5. Orders & Operations > 5.2 Shipments & Tracking
**Task ID:** 032

## Goal
Visualize shipment tracking across carriers.

## Implementation Steps
1. Table shows shipments with status, carrier, last event timestamp, SLA breach indicator.
2. Fragment endpoint fetches aggregated data from backend (may query analytics view) and supports filters.
3. Provide drill-down to order detail shipments tab.
4. Integrate auto-refresh (polling) or manual refresh.
