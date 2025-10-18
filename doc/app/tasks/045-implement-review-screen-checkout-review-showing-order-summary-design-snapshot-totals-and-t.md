# Implement review screen (`/checkout/review`) showing order summary, design snapshot, totals, and terms acknowledgement.

**Parent Section:** 7. Cart & Checkout
**Task ID:** 045

## Goal
Render final review before placing order.

## Implementation Steps
1. Present order summary with design snapshot, totals, shipping/payment info.
2. Include terms acceptance and special instructions input.
3. Submit order to backend; handle success/failure states with toasts.

## Material Design 3 Components
- **App bar:** `Medium top app bar` with edit shortcuts via `Assist chips`.
- **Summary stack:** `Outlined cards` for shipment, payment, and contact info sections.
- **Design preview:** `Elevated card` showing thumbnail with `Supporting text`.
- **CTA:** Full-width `Filled button` for place order, plus `Checkbox` acknowledging terms.
