# Implement shipping selection (`/checkout/shipping`) supporting domestic/international options and delivery estimates.

**Parent Section:** 7. Cart & Checkout
**Task ID:** 043

## Goal
Let users choose shipping method.

## Implementation Steps
1. Display available shipping options with cost and ETA, segmented by domestic/international.
2. Update totals when option selected; handle restrictions (e.g., promotions requiring express).
3. Persist selection to checkout view model.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with info `Icon button` linking to shipping FAQs.
- **Options list:** `Elevated cards` per carrier with delivery estimate displayed via supporting text.
- **Filters:** `Assist chips` for speed vs cost emphasis.
- **Alerts:** `Banner` for service interruptions above the list, with `Snackbar` on selection failure.
