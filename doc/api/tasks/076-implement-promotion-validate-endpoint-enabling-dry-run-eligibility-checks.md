# Implement promotion validate endpoint enabling dry-run eligibility checks.

**Parent Section:** 6. Admin / Staff Endpoints > 6.2 Promotions
**Task ID:** 076

## Purpose
Provide dry-run validation for promotion rules without requiring cart context.

## Endpoint
- `POST /promotions:validate`

## Implementation Steps
1. Accept payload describing promotion definition; reuse promotion service validation logic.
2. Return detailed result listing passes/fails for each constraint (date range, audience, stacking rules).
3. Ensure endpoint restricted to staff and not persisted unless requested.
4. Tests verifying complex rule validation and error messages.
