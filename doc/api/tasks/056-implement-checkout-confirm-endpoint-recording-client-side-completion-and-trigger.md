# Implement checkout confirm endpoint recording client-side completion and triggering post-checkout workflow.

**Parent Section:** 5. Authenticated User Endpoints > 5.4 Cart & Checkout
**Task ID:** 056

## Purpose
Receive client-side confirmation after PSP checkout completes to kick off order finalisation.

## Endpoint
- `POST /checkout/confirm`

## Implementation Steps
1. Accept payload containing session ID/payment intent reference.
2. Verify status with payment provider; mark cart `status=pending_capture` and schedule internal commit job.
3. Trigger background workflow or Pub/Sub message for order creation while awaiting webhook authoritative confirmation.
4. Respond with provisional order ID or status message to client.
5. Tests verifying double submission handling, mismatch scenarios (failed payment), and idempotency.
