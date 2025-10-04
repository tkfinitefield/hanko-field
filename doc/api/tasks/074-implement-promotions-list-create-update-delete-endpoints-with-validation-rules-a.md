# Implement promotions list/create/update/delete endpoints with validation rules and schedule handling.

**Parent Section:** 6. Admin / Staff Endpoints > 6.2 Promotions
**Task ID:** 074

## Purpose
Provide staff interface to manage promotions lifecycle including creation, activation, and archival.

## Endpoints
- `GET /promotions`
- `POST /promotions`
- `PUT /promotions/{{promoId}}`
- `DELETE /promotions/{{promoId}}` (deactivate)

## Implementation Steps
1. Implement filtering by status, active window, type; support pagination.
2. Validate payload structure (discount type, conditions, usage limits) using shared promotion service.
3. Enforce immutability for certain fields once promotion started (e.g., discount value) unless admin override flag set.
4. Log changes in audit log with diff.
5. Tests verifying validation rules, status transitions, and RBAC.
