# Implement refund modal (`/admin/orders/{id}/modal/refund`) integrating with `POST /orders/{id}/payments:refund` and showing validation errors.

**Parent Section:** 5. Orders & Operations > 5.1 Orders List & Detail
**Task ID:** 028

## Goal
Provide refund initiation UI referencing payments endpoint.

## Implementation Steps
1. Modal form includes payment selection, amount, reason, notify customer checkbox.
2. Submit to proxy handler calling `POST /orders/{id}/payments:refund`.
3. Display PSP response (success/failure) and update payments tab via htmx trigger.
4. Ensure partial refunds allowed; validate amount <= available.
