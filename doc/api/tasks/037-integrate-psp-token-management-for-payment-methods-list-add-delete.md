# Integrate PSP token management for payment methods list/add/delete.

**Parent Section:** 5. Authenticated User Endpoints > 5.1 Profile & Account
**Task ID:** 037

## Purpose
Store references to payment methods from PSP without holding sensitive card data.

## Endpoints
- `GET /me/payment-methods`
- `POST /me/payment-methods`: accepts tokenized reference (e.g., Stripe payment method ID) and optional metadata.
- `DELETE /me/payment-methods/{{paymentMethodId}}`

## Implementation Steps
1. Define Firestore sub-collection `users/{{uid}}/paymentMethods` with fields: `id`, `provider`, `token`, `brand`, `last4`, `expMonth`, `expYear`, `isDefault`, `createdAt`.
2. Validate token with PSP API (e.g., retrieve payment method from Stripe) before storing reference.
3. Encrypt or hash tokens if storage security requires; ensure they are references not PAN.
4. Provide `PaymentMethodService` methods ensuring default selection and verifying no outstanding invoices before deletion.
5. Write tests using PSP sandbox mocks verifying validation and default toggling.
