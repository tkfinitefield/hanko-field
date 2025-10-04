# Implement AI suggestion listing/detail retrieval from job store.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 045

## Purpose
Allow users to inspect AI suggestion progress and retrieve generated assets.

## Endpoints
- `GET /designs/{{designId}}/ai-suggestions`
- `GET /designs/{{designId}}/ai-suggestions/{{suggestionId}}`

## Implementation Steps
1. Query `designSuggestions` filtered by `designId` and owner.
2. Include status, summary metrics, and signed URLs for preview assets when available.
3. Support pagination and filtering by status (queued/completed/rejected).
4. Tests verifying ownership checks and asset URL generation.
