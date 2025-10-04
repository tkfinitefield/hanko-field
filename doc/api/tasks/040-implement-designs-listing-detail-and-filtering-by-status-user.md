# Implement designs listing/detail and filtering by status/user.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 040

## Purpose
Allow users to browse their designs with filtering and pagination.

## Endpoints
- `GET /designs`: filters `status=`, `type=`, `updatedAfter=`; pagination token support.
- `GET /designs/{{designId}}`: returns full metadata, current version, AI suggestions summary.

## Implementation Steps
1. Authorize by owner UID; support staff override for support cases.
2. Query Firestore with composite indexes on owner + status + updatedAt.
3. Embed latest version metadata (denormalized fields) for quick display; optionally fetch from `versions` sub-collection when `includeHistory=true`.
4. Return sanitized asset URLs (signed URLs for private assets).
5. Tests for pagination tokens and unauthorized access.
