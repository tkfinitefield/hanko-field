# Integrate promotion dry-run validation UI linking to `POST /admin/promotions:validate` with rule breakdown display.

**Parent Section:** 8. Promotions & Marketing
**Task ID:** 050

## Goal
Implement UI for promotion dry-run validation.

## Implementation Steps
1. Form capturing prospective cart data (subtotal, items, customer segment).
2. Call `POST /admin/promotions:validate` and render rule evaluation table (pass/fail per rule).
3. Highlight blockers and provide copyable JSON result for debugging.
