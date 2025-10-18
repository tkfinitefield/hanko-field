# Implement payment integration abstraction (Stripe) for checkout session management and reconciliation.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 021

## Goal
Provide unified interface over PSPs (starting with Stripe) for checkout session management, captures, refunds, and webhook reconciliation.

## Design
- Interface `PaymentProvider` with methods `CreateCheckoutSession`, `Confirm`, `Capture`, `Refund`, `LookupPayment`.
- Implement providers: `StripeProvider` using the official SDK.
- Central service selects provider based on currency, customer preference, or cart metadata.
- Store PSP references in `payments` sub-collection under orders with fields `provider`, `intentId`, `status`, `amount`, `currency`, `capturedAt`.

## Steps
- [x] Implement Stripe provider adapter with retry/backoff and idempotency key usage.
- [x] Map PSP statuses to internal enums (`pending`, `succeeded`, `failed`, `refunded`).
- [x] Handle webhook reconciliation by updating payment records and triggering order state transitions.
- [x] Add abstraction tests with fake provider to ensure contract stability.
- [x] Ensure sensitive keys loaded from Secret Manager and masked in logs.

## Completion Notes (2025-10-07)
- Added `internal/payments` package with provider interface, routing manager, and domain-level payment normalization for PSP integrations.
- Implemented Stripe provider adapter using official SDK with idempotency headers, account routing, and safe logging defaults.
- Introduced common `PaymentDetails` structure for storing PSP metadata including intent IDs, capture and refund timestamps, and mapped statuses.
- Expanded domain `Payment` model to persist PSP intent IDs and adjusted Go module dependencies to include Stripe SDK only.
- Added manager-focused unit tests with fake providers to verify routing by preference, currency, and default fallback; formatted all new code and ensured `go test ./...` passes.
