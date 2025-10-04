# Implement shipping carrier webhook handler accepting updates per carrier and mapping payloads to shipment events.

**Parent Section:** 7. Webhooks (Inbound)
**Task ID:** 098

## Purpose
Accept tracking updates from carriers (DHL, JP Post, Yamato, UPS, FedEx) to update shipment status.

## Endpoint
- `POST /webhooks/shipping/{carrier}`

## Implementation Steps
1. Differentiate carriers via path parameter; each carrier has unique payload schema/signature method.
2. Validate authenticity (IP whitelist, HMAC, OAuth) per carrier requirements.
3. Map tracking numbers to order shipments, append events to `orders/{{orderId}}/shipments/{{shipmentId}}/events`.
4. Update shipment and order statuses when delivered or exception occurs; notify customers.
5. Tests using fixture payloads for each carrier and validation failure scenarios.
