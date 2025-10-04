# Implement cart pricing engine (tax, shipping, discounts) with pluggable rules.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 020

## Goal
Calculate cart totals (subtotal, discounts, tax, shipping, total) using pluggable rules and service integrations (tax service, shipping service).

## Responsibilities
- Accept cart state (items, quantities, addresses, promotion codes) and produce pricing breakdown.
- Interface with promotion service, tax calculators, and shipping rate providers.
- Provide deterministic calculations for estimations and order finalisation.

## Design
- Modules: item pricing, discount application, tax calculator, shipping estimator.
- Configurable via strategy objects so tax/shipping providers can be swapped per market.
- Support multi-currency with rounding rules and currency precision.

## Steps
1. Define data structs `Cart`, `CartItem`, `PricingBreakdown` in shared package.
2. Implement pipeline applying in order: base price → item-level discount → cart-level promotions → tax → shipping.
3. Integrate with promotion service for discounts and inventory service for stock validation.
4. Provide caching for shipping rates (per address, weight) to reduce API calls.
5. Add regression tests comparing expected totals versus scenario fixtures.
