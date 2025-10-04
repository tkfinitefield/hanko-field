# Implement Stripe webhook handler validating signature and processing payment intent succeeded/failed and refund events.

**Parent Section:** 7. Webhooks (Inbound)
**Task ID:** 096

## Purpose
Process Stripe payment lifecycle events (intent succeeded/failed, charge refunded) to reconcile orders.

## Endpoint
- `POST /webhooks/payments/stripe`

## Implementation Steps
1. Validate `Stripe-Signature` header using signing secret from Secret Manager; enforce tolerance window.
2. Parse event payload, handle relevant event types (`payment_intent.succeeded`, `payment_intent.payment_failed`, `charge.refunded`).
3. Map Stripe IDs to internal payment records, update order status, reservation commitments, promotion usage.
4. Ensure idempotency by tracking processed event IDs.
5. Respond quickly (within 2s); offload heavy work to background job if needed.
6. Tests using Stripe webhook fixtures verifying signature validation and state transitions.
