# Implement payment integration abstraction (Stripe, PayPal) for checkout session management and reconciliation.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 021

## Goal
Provide unified interface over PSPs (Stripe, PayPal) for checkout session management, captures, refunds, and webhook reconciliation.

## Design
- Interface `PaymentProvider` with methods `CreateCheckoutSession`, `Confirm`, `Capture`, `Refund`, `LookupPayment`.
- Implement providers: `StripeProvider`, `PayPalProvider` using respective SDKs.
- Central service selects provider based on currency, customer preference, or cart metadata.
- Store PSP references in `payments` sub-collection under orders with fields `provider`, `intentId`, `status`, `amount`, `currency`, `capturedAt`.

## Steps
1. Implement provider adapters with retry/backoff and idempotency key usage per PSP.
2. Map PSP statuses to internal enums (`pending`, `succeeded`, `failed`, `refunded`).
3. Handle webhook reconciliation by updating payment records and triggering order state transitions.
4. Add abstraction tests with fake provider to ensure contract stability.
5. Ensure sensitive keys loaded from Secret Manager and masked in logs.
