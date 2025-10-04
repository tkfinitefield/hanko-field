# Implement shop listing (`/shop`) with filters form, results fragment (`/shop/table`), and pagination.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 017

## Goal
Implement shop listing with filters and fragment-driven results.

## Implementation Steps
1. Build filter form (material, shape, size, price range, sale flag) hooking into htmx request to `/shop/table`.
2. Implement `/shop/table` fragment returning table/list of products with pagination controls.
3. Integrate query parameters for deep linking and shareable filters.
4. Handle empty states and loading indicators.
