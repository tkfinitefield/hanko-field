# Build checkout address page (`/checkout/address`) with address selection fragment and forms.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 030

## Goal
Build checkout address selection.

## Implementation Steps
1. Display saved addresses via fragment; allow selecting shipping/billing.
2. Provide form for new address using modal; validate fields.
3. Persist selection in session and route to shipping step.

## UI Components
- **Layout:** `CheckoutLayout` with breadcrumb progress indicator.
- **Saved addresses:** `AddressList` fragment with radio selection and edit `LinkButton`.
- **Address form:** `AddressForm` using `Input`, `Select`, `Textarea` with inline validation.
- **Sidebar:** `SummarySidebar` showing order totals and shipping ETA.
- **Action footer:** `ActionBar` with Continue `PrimaryButton` and Back `SecondaryButton`.
- **Inline alerts:** `InlineAlert` for validation or geocoding errors.
