# Implement table fragments (`/admin/catalog/{kind}/table`) with filter/sort controls and pagination.

**Parent Section:** 6. Catalog Management
**Task ID:** 038

## Goal
Implement table fragments for each catalog kind with filter/sort.

## Implementation Steps
1. Build handler `/admin/catalog/{kind}/table` parsing filters (status, category, updatedAt).
2. Map API response from `GET /admin/catalog/{kind}` to table rows with action buttons (Edit, Delete).
3. Provide pagination controls and `hx-push-url` to update query string.
4. Include badges for publish status, version tags.
