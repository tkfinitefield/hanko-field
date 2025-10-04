# Implement admin user search/list/detail endpoints with flexible query support.

**Parent Section:** 6. Admin / Staff Endpoints > 6.5 Users / Reviews / Audit
**Task ID:** 088

## Purpose
Provide staff search and detail view of users for support and moderation.

## Endpoints
- `GET /users?query=` (supports email, name, UID, phone)
- `GET /users/{{uid}}`

## Implementation Steps
1. Implement search index (Firestore composite indexes or Algolia) for fuzzy matching.
2. Return user profile, orders summary, last login, flags.
3. Ensure PII masked according to role; log access in audit log.
4. Tests verifying search results and RBAC restrictions.
