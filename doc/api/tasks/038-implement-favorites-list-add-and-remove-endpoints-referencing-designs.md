# Implement favorites list, add, and remove endpoints referencing designs.

**Parent Section:** 5. Authenticated User Endpoints > 5.1 Profile & Account
**Task ID:** 038

## Purpose
Enable users to bookmark designs for quick access across devices.

## Endpoints
- `GET /me/favorites`
- `PUT /me/favorites/{{designId}}` (add)
- `DELETE /me/favorites/{{designId}}` (remove)

## Implementation Steps
1. Store favorites in `users/{{uid}}/favorites` with fields: `designRef`, `addedAt`.
2. Ensure referenced design belongs to user or is shareable when adding.
3. Enforce size limit (e.g., max 200 favorites) with proper error response when exceeded.
4. Return favorites list joined with design metadata (name, preview) via batched fetch.
5. Tests verifying add idempotency, removal, and limit enforcement.
