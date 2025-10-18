# Implement review page (`/checkout/review`) summarizing order and linking to confirmation action.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 033

## Goal
Render order review summary.

## Implementation Steps
1. Display design snapshot, items, totals, shipping/payment summary.
2. Provide final confirmation button linking to PSP or order commit.
3. Display terms acceptance check.

## UI Components
- **Layout:** `CheckoutLayout` with progress `Stepper` final stage highlight.
- **Summary grid:** `ReviewGrid` containing shipment, payment, items `DetailCard`s.
- **Terms acknowledgment:** `TermsCheckbox` with link to policies.
- **Edit links:** `InlineLink` for each section to back-navigate.
- **Sidebar:** `SummarySidebar` locking totals and loyalty info.
- **Action footer:** `PrimaryButton` place order, `SecondaryButton` return to payment with `Spinner` during submit.
