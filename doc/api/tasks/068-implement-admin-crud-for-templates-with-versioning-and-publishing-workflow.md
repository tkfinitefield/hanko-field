# Implement admin CRUD for templates with versioning and publishing workflow.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 068

## Purpose
Provide staff CRUD over templates with versioning and publish workflow impacting public catalog.

## Endpoints
- `POST /templates`
- `PUT /templates/{{id}}`
- `DELETE /templates/{{id}}` (soft delete / unpublish)

## Implementation Steps
1. Require admin role via middleware; log all actions in audit log.
2. Accept payload with `draft` fields (internal notes, preview assets) separate from public metadata.
3. Manage template versions via `templates/{{id}}/versions` storing change history; enforce publish toggle updates `isPublished` and `publishedAt`.
4. Trigger cache invalidation or CDN purge when publishing/unpublishing.
5. Tests verifying RBAC, version history retention, and soft delete behaviour.
