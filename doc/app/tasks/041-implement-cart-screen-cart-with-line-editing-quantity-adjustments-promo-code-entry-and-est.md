# Implement cart screen (`/cart`) with line editing, quantity adjustments, promo code entry, and estimate summary.

**Parent Section:** 7. Cart & Checkout
**Task ID:** 041

## Goal
Implement cart screen with editable lines and promo entry.

## Implementation Steps
1. Display items with thumbnails, quantity controls, option indicators.
2. Allow removal, undo, and editing of options/add-ons.
3. Integrate promo code application with error handling.
4. Show pricing breakdown and estimated delivery summary.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with edit `Icon button` for bulk actions.
- **Line items:** `Card`-wrapped `List items` featuring thumbnail, modifiers, and quantity `Stepper` built from `Icon buttons`.
- **Promo entry:** `Outlined text field` for promo codes with trailing `Assist chip` showing applied state.
- **Summary sheet:** Bottom `Elevated card` for totals with primary `Filled button` for checkout.
