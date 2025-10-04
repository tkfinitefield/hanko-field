# Implement admin CRUD for content guides including localization and category tagging.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 072

## Purpose
Manage localized guide content accessible via public endpoints.

## Endpoints
- `POST /content/guides`
- `PUT /content/guides/{{id}}`
- `DELETE /content/guides/{{id}}`

## Implementation Steps
1. Support draft/publish states; allow scheduling via `publishAt`.
2. Store content in `contentGuides` with localization arrays or per-language docs.
3. Sanitize HTML/markdown input; optionally convert to sanitized blocks.
4. Track revision history with author info for compliance.
5. Tests verifying scheduling, localization, and sanitization.
