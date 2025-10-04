# Implement PayPal webhook handler validating signature and handling payment/refund lifecycle.

**Parent Section:** 7. Webhooks (Inbound)
**Task ID:** 097

## Purpose
Handle PayPal payment/refund events similar to Stripe, ensuring orders stay in sync.

## Endpoint
- `POST /webhooks/payments/paypal`

## Implementation Steps
1. Validate PayPal signature via `Transmission-Id`, `Certificate-Url`, `Transmission-Sig`, `Transmission-Time` headers.
2. Process events (`CHECKOUT.ORDER.APPROVED`, `PAYMENT.CAPTURE.COMPLETED`, `PAYMENT.CAPTURE.DENIED`, `PAYMENT.CAPTURE.REFUNDED`).
3. Update payment records and order statuses accordingly.
4. Store event payload for auditing with retention policy.
5. Tests with sandbox payloads verifying signature validation and transitions.
