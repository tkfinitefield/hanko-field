# Implement admin order listing endpoint with status/date filters for operations dashboards.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 077

## Purpose
Provide operations dashboard to filter orders by status, production stage, date, or channel.

## Endpoint
- `GET /orders?status=in_production&since=...`

## Implementation Steps
1. Support extensive filters: status, payment status, production queue, customer email, promotion code.
2. Return operational fields (assigned queue, last event, outstanding tasks).
3. Use pagination and sorting (default newest first).
4. Tests verifying filter combinations and RBAC.
