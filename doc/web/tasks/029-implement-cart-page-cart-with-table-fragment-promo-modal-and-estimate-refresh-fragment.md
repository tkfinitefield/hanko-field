# Implement cart page (`/cart`) with table fragment, promo modal, and estimate refresh fragment.

**Parent Section:** 5. Cart & Checkout
**Task ID:** 029

## Goal
Implement cart page with dynamic fragments.

## Implementation Steps
1. Render summary page and include `/cart/table` fragment with line items and totals.
2. Implement promo modal posting to `/cart:apply-promo` and refreshing totals.
3. Provide estimate fragment `/cart/estimate` updating totals on changes.

## UI Components
- **Layout:** `CheckoutLayout` with progress `Stepper` across cart > shipping > payment > review.
- **Cart table:** `CartTable` fragment listing items, options, price, quantity `Stepper`, remove `IconButton`.
- **Summary card:** `SummarySidebar` with totals, tax estimate, shipping estimator `Form`.
- **Promo modal:** `PromoModal` hosting code `Input` and status message.
- **Recommended rail:** `CrossSellRail` showing upsell `ProductCard`s.
- **Empty state:** `EmptyCart` component with continue shopping CTA.
