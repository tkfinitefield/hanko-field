# Implement contract tests for webhooks to ensure payload parsing and idempotency.

**Parent Section:** 10. Testing Strategy
**Task ID:** 114

## Scope
Validate webhook handlers against provider payload samples ensuring schema compatibility and idempotency.

## Plan
- Maintain fixture library for Stripe, PayPal, carriers, AI worker payloads.
- Simulate signature headers and verify validation logic.
- Assert resulting state changes (orders, payments, shipments).
- Run contract suite in CI to catch spec changes from providers.
