# Implement accept/reject endpoints mutating suggestion status and updating design state.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 046

## Purpose
Provide actions to accept or reject AI suggestions, updating design state when accepted.

## Endpoints
- `POST /designs/{{designId}}/ai-suggestions/{{suggestionId}}:accept`
- `POST /designs/{{designId}}/ai-suggestions/{{suggestionId}}:reject`

## Implementation Steps
1. Validate suggestion belongs to design+user and in `completed` state.
2. On accept: create new design version from suggestion output, update design preview, mark suggestion status `accepted`.
3. On reject: mark suggestion `rejected` with optional reason.
4. Prevent repeated accept/reject using Firestore transactions.
5. Tests verifying state transitions and resulting design updates.
