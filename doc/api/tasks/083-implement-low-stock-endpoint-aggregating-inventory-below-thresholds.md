# Implement low stock endpoint aggregating inventory below thresholds.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 083

## Purpose
Provide dashboard of SKUs below safety stock thresholds.

## Endpoint
- `GET /stock/low`

## Implementation Steps
1. Query inventory service for items where `onHand - reserved < safetyStock`.
2. Return data with supplier info, projected depletion date, recent sales velocity.
3. Optionally include export action.
4. Tests verifying query results and RBAC.
