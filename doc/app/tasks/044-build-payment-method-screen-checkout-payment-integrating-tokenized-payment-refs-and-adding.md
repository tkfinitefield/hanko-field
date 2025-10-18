# Build payment method screen (`/checkout/payment`) integrating tokenized payment refs and adding new methods if allowed.

**Parent Section:** 7. Cart & Checkout
**Task ID:** 044

## Goal
Present payment methods and manage tokenized references.

## Implementation Steps
1. Fetch stored payment methods; show brand, last4, expiry.
2. Provide add-new flow via native PSP SDK or web view if required.
3. Manage default selection and ensure secure storage.

## Material Design 3 Components
- **App bar:** `Small top app bar` with add method `Icon button`.
- **Saved methods:** `List items` with `Radio buttons` and brand `Icon` as leading element.
- **Entry form:** `Outlined text fields`, `Dropdown menu`, and `Segmented buttons` for billing address reuse.
- **Confirmation:** `Filled button` to continue, with `Snackbar` for declined payments.
