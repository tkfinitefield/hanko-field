# Implement order payment history retrieval.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 060

## Purpose
Provide itemized payment events associated with an order for transparency.

## Endpoint
- `GET /orders/{{orderId}}/payments`

## Implementation Steps
1. Fetch payment records from `orders/{{id}}/payments` sorted by `createdAt`.
2. Include fields: `provider`, `transactionId`, `amount`, `currency`, `status`, `capturedAt`, `refundedAmount`.
3. Hide internal metadata (webhook payloads) from user-facing response.
4. Tests verifying access control and completeness.
