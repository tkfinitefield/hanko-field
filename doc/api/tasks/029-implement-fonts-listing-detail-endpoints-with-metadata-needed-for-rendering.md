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
1. Build query functions with script and premium filters.
2. Provide DTO with relevant fields, hide internal licensing notes.
3. Preload CDN URLs for previews and ensure caching headers set appropriately.
4. Add tests verifying filtering and 404 for unpublished or missing fonts.
