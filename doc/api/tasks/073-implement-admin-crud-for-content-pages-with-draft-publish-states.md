# Implement admin CRUD for content pages with draft/publish states.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 073

## Purpose
Manage marketing/static pages.

## Endpoints
- `POST /content/pages`
- `PUT /content/pages/{{id}}`
- `DELETE /content/pages/{{id}}`

## Implementation Steps
1. Model pages with fields: `slug`, `language`, `title`, `body`, `seoMeta`, `status` (`draft`, `published`, `archived`).
2. Provide preview tokens for reviewing unpublished pages.
3. Enforce unique slug per language.
4. Trigger CDN purge after publish/unpublish.
5. Tests covering slug uniqueness, preview access, and audit logs.
