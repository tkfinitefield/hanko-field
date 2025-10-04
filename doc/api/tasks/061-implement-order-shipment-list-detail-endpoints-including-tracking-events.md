# Implement order shipment list/detail endpoints including tracking events.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 061

## Purpose
Expose shipment status and tracking events to the user.

## Endpoints
- `GET /orders/{{orderId}}/shipments`
- `GET /orders/{{orderId}}/shipments/{{shipmentId}}`

## Implementation Steps
1. Store shipments under `orders/{{id}}/shipments` with fields: `carrier`, `trackingNumber`, `status`, `labelURL`, `items`, `events[]`.
2. Return aggregated view with latest event summarised for list endpoint.
3. Ensure events sanitized (no internal notes) before exposing.
4. Tests verifying unauthorized access and event ordering.
