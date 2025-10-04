# Implement checkout session creation endpoint integrating PSP session API and reserving stock when required.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 055

## Purpose
Initiate PSP checkout session (Stripe/PayPal) and optionally reserve stock prior to redirecting customer.

## Endpoint
- `POST /checkout/session`

## Implementation Steps
1. Validate cart readiness (non-empty, shipping info present, promotions valid).
2. Invoke inventory service to hold stock if required (`internal/checkout/reserve-stock`).
3. Call payment integration abstraction to create checkout session; persist session ID + provider in cart or temporary record.
4. Return PSP session info (redirect URL, client secret) to client.
5. Tests verifying stock reservation, PSP call, and error rollback (release reservation on failure).
