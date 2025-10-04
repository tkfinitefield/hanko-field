# Implement admin CRUD for materials capturing stock and supplier info.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 070

## Purpose
Manage materials inventory metadata (supplier info, safety stock) for storefront listing and production.

## Endpoints
- `POST /materials`
- `PUT /materials/{{id}}`
- `DELETE /materials/{{id}}`

## Implementation Steps
1. Store supplier references, lead times, and pricing details used by operations.
2. Enforce relationship with inventory service (update safety thresholds when material updated).
3. On delete/unpublish, ensure products referencing material updated or flagged.
4. Emit audit events for traceability.
5. Tests verifying cross-validation with inventory.
