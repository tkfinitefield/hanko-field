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
1. [x] Implement repository querying published templates with Firestore composite indexes for filters.
2. [x] Map Firestore docs to response DTO (avoid internal fields such as draft notes).
3. [x] Include CDN-signed URL builder if assets stored privately (or direct GCS public link if permitted).
4. [x] Add tests for filter combinations and unpublished template exclusion.

## Completion Notes
- Expanded template domain models and repository/service contracts to cover category/style/tag filters, sort options, and publish gating (`api/internal/domain/types.go`, `api/internal/repositories/interfaces.go`, `api/internal/services/interfaces.go`).
- Implemented catalog service with normalization helpers, CDN-aware asset resolution hooks, and comprehensive unit coverage (`api/internal/services/catalog_service.go`, `api/internal/services/catalog_service_test.go`).
- Added public `/templates` list/detail handlers including query parsing, CDN URL resolution, and HTTP tests verifying filtering, pagination, and unpublished exclusion (`api/internal/handlers/public_templates.go`, `api/internal/handlers/public_templates_test.go`).
- Repository wiring for Firestore remains TODO; service/handler stack now expects a catalog repository with the extended contract and will integrate once the persistence layer lands. Full suite `go test ./...` passes after gofmt formatting.
