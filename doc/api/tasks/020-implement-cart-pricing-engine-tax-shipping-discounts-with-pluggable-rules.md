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
- [x] Define data structs `Cart`, `CartItem`, `PricingBreakdown` in shared package.
- [x] Implement pipeline applying in order: base price → item-level discount → cart-level promotions → tax → shipping.
- [x] Integrate with promotion service for discounts and inventory service for stock validation.
- [x] Provide caching for shipping rates (per address, weight) to reduce API calls.
- [x] Add regression tests comparing expected totals versus scenario fixtures.

## Completion Notes
- Added shared pricing models and enriched cart/item structs for weight, shipping, tax metadata (`api/internal/domain/pricing.go`, `api/internal/domain/types.go`).
- Implemented `CartPricingEngine` with pluggable item discount rules, promotion validation, tax/shipping collaborators, and per-address shipping cache (`api/internal/services/pricing_engine.go`).
- Covered core scenarios with unit tests, including caching behaviour and error paths (`api/internal/services/pricing_engine_test.go`), and verified via `go test ./...` from `api` module.
