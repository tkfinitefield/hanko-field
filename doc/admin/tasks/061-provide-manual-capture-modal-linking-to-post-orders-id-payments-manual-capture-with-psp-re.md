# Provide manual capture modal linking to `POST /orders/{id}/payments:manual-capture` with PSP response handling.

**Parent Section:** 11. Finance & Accounting
**Task ID:** 061

## Goal
Provide manual capture modal.

## Implementation Steps
1. Form selects payment intent, optionally partial amount, capture reason.
2. Call backend capture endpoint and show PSP response.
3. Update payments tab on success.
