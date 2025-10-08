# Build payment page (`/checkout/payment`) with PSP session initiation and confirmation handling.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 032

## Goal
Integrate PSP session initiation and confirmation.

## Implementation Steps
1. Provide button for Stripe; call `POST /checkout/session` for server intent.
2. Handle redirection or embedded payments; update UI on success/failure.
