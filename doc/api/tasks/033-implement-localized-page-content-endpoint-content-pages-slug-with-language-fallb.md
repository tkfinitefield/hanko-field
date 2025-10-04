# Implement localized page content endpoint (`/content/pages/{slug}`) with language fallback.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 033

## Purpose
Deliver localized static pages (FAQ, company info) for marketing web application.

## Endpoint
- `GET /content/pages/{{slug}}?lang=ja`

## Data Model
- Collection `contentPages`: `slug`, `lang`, `title`, `bodyHtml`, `seo`, `isPublished`, `updatedAt`.

## Implementation Steps
1. Fetch page by slug and language, fallback to default language if translation missing.
2. Render sanitized HTML or structured blocks as required by frontend contract.
3. Provide 404 when page not found or unpublished.
4. Add tests for fallback, caching headers, and sanitization.
