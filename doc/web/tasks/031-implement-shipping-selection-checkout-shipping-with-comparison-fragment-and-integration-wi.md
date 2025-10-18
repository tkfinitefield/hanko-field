# Implement shipping selection (`/checkout/shipping`) with comparison fragment and integration with estimate API.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 031

## Goal
Implement shipping method selection.

## Implementation Steps
1. Call backend to fetch available methods; render comparison table fragment.
2. Update totals/ETA upon selection; handle international restrictions.

## UI Components
- **Layout:** `CheckoutLayout` with progress `Stepper`.
- **Rate options:** `ShippingOptions` fragment rendering `OptionCard` rows with carrier badge, cost, ETA, select `Radio`.
- **Comparison chart:** `RateComparisonChart` summarizing speed vs cost for top carriers.
- **Sidebar:** `SummarySidebar` with updated totals and delivery address snapshot.
- **Alerts:** `InlineAlert` for restrictions/outages.
- **Action footer:** `ActionBar` continue/back buttons with `Spinner` during recalculation.
