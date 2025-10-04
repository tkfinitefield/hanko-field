# Implement order invoice request endpoint producing task to generate PDF.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 059

## Purpose
Capture user request for invoice/receipt to trigger asynchronous PDF generation.

## Endpoint
- `POST /orders/{{orderId}}:request-invoice`

## Implementation Steps
1. Validate order belongs to user and is paid.
2. Record request in `orders/{{id}}/invoiceRequests` or flag on order document with timestamp.
3. Publish job to invoice service (Cloud Run job or Pub/Sub) to generate PDF via internal endpoint.
4. Return acknowledgement including expected delivery channel (email, dashboard download).
5. Tests verifying duplicate suppression and authorization.
