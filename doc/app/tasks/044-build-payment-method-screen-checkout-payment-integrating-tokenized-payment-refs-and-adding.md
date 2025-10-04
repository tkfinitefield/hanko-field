# Build payment method screen (`/checkout/payment`) integrating tokenized payment refs and adding new methods if allowed.

**Parent Section:** 7. Cart & Checkout
**Task ID:** 044

## Goal
Present payment methods and manage tokenized references.

## Implementation Steps
1. Fetch stored payment methods; show brand, last4, expiry.
2. Provide add-new flow via native PSP SDK or web view if required.
3. Manage default selection and ensure secure storage.
