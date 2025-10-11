# Implement fonts listing/detail endpoints with metadata needed for rendering.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 029

## Purpose
Expose list of available fonts for design composition with metadata needed by clients (name, script support, license info).

## Endpoints
- `GET /fonts`: optional `?script=` filter (kanji, kana, latin) and `?isPremium=`.
- `GET /fonts/{{fontId}}`: includes license URL, preview assets.

## Data Model
- Collection `fonts`: fields `id`, `displayName`, `family`, `scripts[]`, `previewImageURL`, `letterSpacing`, `isPremium`, `supportedWeights[]`, `license`, `isPublished`.

## Implementation Steps
1. [x] Build query functions with script and premium filters.
2. [x] Provide DTO with relevant fields, hide internal licensing notes.
3. [x] Preload CDN URLs for previews and ensure caching headers set appropriately.
4. [x] Add tests verifying filtering and 404 for unpublished or missing fonts.

## Completion Notes
- Extended font domain/repository/service contracts to include license metadata, premium flag, publication filtering, and detail retrieval (`api/internal/domain/types.go`, `api/internal/repositories/interfaces.go`, `api/internal/services/interfaces.go`, `api/internal/services/catalog_service.go`).
- Implemented public `/fonts` list/detail handlers with query parsing, CDN URL resolution, and cache-control headers (`api/internal/handlers/public_templates.go`).
- Added HTTP handler tests covering filter normalization, caching headers, and not-found behaviour (`api/internal/handlers/public_templates_test.go`).
- Updated catalog service tests/stubs to satisfy the expanded repository interface (`api/internal/services/catalog_service_test.go`).
