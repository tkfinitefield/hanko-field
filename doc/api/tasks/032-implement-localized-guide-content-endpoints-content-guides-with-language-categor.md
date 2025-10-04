# Implement localized guide content endpoints (`/content/guides`) with language/category query support.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 032

## Purpose
Serve localized guides (culture, etiquette) for marketing pages with optional category filtering.

## Endpoints
- `GET /content/guides?lang=ja&category=culture`
- `GET /content/guides/{{slug}}?lang=ja`

## Data Model
- Collection `contentGuides`: `slug`, `title`, `lang`, `category`, `summary`, `bodyHtml`, `heroImage`, `isPublished`, `publishedAt`.

## Implementation Steps
1. Design repository to fetch by primary language and fallback to default when translation missing.
2. Sanitize/scope HTML output using allowlist to prevent injection.
3. Support caching/etag using `updatedAt` timestamp.
4. Tests verifying language fallback and category filtering.
