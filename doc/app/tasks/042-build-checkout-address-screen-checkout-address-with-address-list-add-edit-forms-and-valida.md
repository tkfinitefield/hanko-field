# Build checkout address screen (`/checkout/address`) with address list, add/edit forms, and validation (JP/international formats).

**Parent Section:** 7. Cart & Checkout
**Task ID:** 042

## Goal
Manage shipping address selection/creation.

## Implementation Steps
1. List saved addresses; highlight default and allow editing.
2. Provide add form with validation, postal code lookup, persona-specific instructions.
3. Persist selection to checkout state.

## Material Design 3 Components
- **App bar:** `Small top app bar` with add address `Icon button`.
- **Saved addresses:** `List items` with leading location icon, trailing `Radio button`, and `Assist chips` for default/billing.
- **Form modal:** `Full-screen dialog` containing `Outlined text fields` and `Segmented buttons` for domestic/international layouts.
- **Footer:** Primary `Filled button` to confirm shipping address.
