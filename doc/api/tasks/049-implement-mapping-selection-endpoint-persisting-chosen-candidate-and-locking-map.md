# Implement mapping selection endpoint persisting chosen candidate and locking mapping.

**Parent Section:** 5. Authenticated User Endpoints > 5.3 Name Mapping
**Task ID:** 049

## Purpose
Allow user to finalize kanji mapping for future design usage.

## Endpoint
- `POST /name-mappings/{{mappingId}}:select`

## Implementation Steps
1. Validate mapping belongs to user and candidate index provided exists.
2. Persist selection by updating mapping document (`status=selected`, `selectedCandidate`, `selectedAt`).
3. Prevent re-selection unless explicitly allowed; optionally allow override with audit log.
4. Notify design creation flow of final mapping (store reference on user profile).
5. Tests verifying selection, unauthorized access, and idempotent repeated selection.
