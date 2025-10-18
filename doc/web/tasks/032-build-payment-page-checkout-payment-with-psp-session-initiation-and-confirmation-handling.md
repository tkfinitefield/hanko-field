# Build payment page (`/checkout/payment`) with PSP session initiation and confirmation handling.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 032

## Goal
Integrate PSP session initiation and confirmation.

## Implementation Steps
1. Provide button for Stripe; call `POST /checkout/session` for server intent.
2. Handle redirection or embedded payments; update UI on success/failure.

## UI Components
- **Layout:** `CheckoutLayout` reuse with progress `Stepper`.
- **Payment form:** `PaymentGatewayForm` embedding PSP elements (card fields, wallet buttons).
- **Saved methods:** `SavedMethodList` with radio selection, edit `LinkButton`.
- **Security callout:** `TrustBadge` row listing encryption icons.
- **Sidebar:** `SummarySidebar` with totals, due today, invoice note.
- **Feedback:** `InlineAlert` for decline errors and `SuccessToast` after confirmation.
