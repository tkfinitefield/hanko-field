# Implement templates listing/detail endpoints with optional filters and CDN URLs.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 028

## Purpose
Provide public catalog of hanko templates with optional filters and CDN asset references.

## Endpoints
- `GET /templates`: supports `?category=`, `?style=`, pagination, sort by `popularity`/`createdAt`.
- `GET /templates/{{templateId}}`: returns full metadata including preview URLs.

## Data Model
- Collection `templates` fields: `id`, `name`, `description`, `category`, `style`, `tags[]`, `previewImageURL`, `svgPath`, `isPublished`, `createdAt`, `updatedAt`.
- Optional caching via Cloud CDN; ensure data flagged `isPublished` only.

## Implementation Steps
1. Implement repository querying published templates with Firestore composite indexes for filters.
2. Map Firestore docs to response DTO (avoid internal fields such as draft notes).
3. Include CDN-signed URL builder if assets stored privately (or direct GCS public link if permitted).
4. Add tests for filter combinations and unpublished template exclusion.
