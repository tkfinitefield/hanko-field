# Implement audit log retrieval endpoint with filtering by target reference.

**Parent Section:** 6. Admin / Staff Endpoints > 6.5 Users / Reviews / Audit
**Task ID:** 091

## Purpose
Enable staff to query audit logs filtered by target reference for investigations.

## Endpoint
- `GET /audit-logs?targetRef=/orders/{{id}}`

## Implementation Steps
1. Query audit log collection by `targetRef` with pagination and timeframe filters.
2. Provide export capability (CSV) when requested.
3. Enforce RBAC and redaction of sensitive metadata based on operator role.
4. Tests verifying filtering and redaction.
