# Produce data contract checklist mapping every admin action to the corresponding API endpoint and required request/response fields. ✅

**Parent Section:** 0. Planning & Architecture
**Task ID:** 004

## Goal
Map every admin UI action to backend API payloads to guarantee consistent integration.

## Tasks
- Build spreadsheet or YAML manifest with columns: UI route, fragment endpoint, HTTP method, API endpoint, request fields, response fields.
- Highlight dependencies where backend work pending or additional parameters required.
- Circulate manifest with API/FE leads to confirm contract and versioning expectations.

## Acceptance Criteria
- Manifest checked into `doc/admin/contracts/` and referenced across stories.
- No handler is implemented without corresponding API details confirmed.
- API team acknowledges any gaps and plans enhancements.
