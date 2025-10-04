# Implement design version listing/detail endpoints maintaining history snapshots.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 042

## Purpose
Expose version history of a design for user review and reversion.

## Endpoints
- `GET /designs/{{designId}}/versions`
- `GET /designs/{{designId}}/versions/{{versionId}}`

## Implementation Steps
1. Store version documents with fields: `id`, `designId`, `sequence`, `config`, `assets`, `createdAt`, `createdBy`.
2. Provide list with pagination sorted by `sequence desc`.
3. Optionally allow query parameter `includeAssets` to fetch signed URLs for version assets.
4. Ensure permission check restricts to owner/staff.
5. Tests verifying order, access control, and 404 for missing versions.
