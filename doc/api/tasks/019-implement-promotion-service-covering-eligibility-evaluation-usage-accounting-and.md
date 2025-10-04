# Implement promotion service covering eligibility evaluation, usage accounting, and validations.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 019

## Goal
Encapsulate promotion eligibility, usage accounting, and validation rules for both user checkout flow and admin tools.

## Responsibilities
- Evaluate promotion conditions (date ranges, customer segments, product filters).
- Maintain usage counters per promotion and per user with atomic increments.
- Interface with admin endpoints for creation/update/validation.

## Data Model
- Collection `promotions` with fields: `code`, `name`, `type`, `discount`, `constraints`, `activeFrom`, `activeTo`, `usageLimit`, `limitPerUser`, `status`.
- Sub-collection `usages` keyed by user ID tracking counts and lastAppliedAt.

## Steps
1. Implement rules engine to evaluate promotion constraints (cart total, SKU inclusion/exclusion, customer tier).
2. Provide methods `ApplyToCart(cart, context)` returning discount breakdown and validation errors.
3. Implement atomic counter updates using Firestore transactions in `promotion_usages`.
4. Support admin validation endpoint by returning diagnostics (which rule failed).
5. Write unit tests for typical promotions (percentage, fixed, free shipping) and edge cases.
