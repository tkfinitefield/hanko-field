# Implement admin CRUD for fonts including licensing data.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 069

## Purpose
Allow staff to manage fonts, including licensing and subscription flags.

## Endpoints
- `POST /fonts`
- `PUT /fonts/{{id}}`
- `DELETE /fonts/{{id}}` (disable)

## Implementation Steps
1. Validate payload includes licensing URLs, allowed usages, and preview assets.
2. Enforce uniqueness of `family` + `weight` combination; maintain slug for referencing.
3. Provide ability to upload new preview assets (call assets service) and update metadata.
4. Soft delete by setting `isPublished=false`; ensure dependent designs fallback gracefully.
5. Tests covering validation and dependent data checks (cannot remove if used in active design without fallback).
