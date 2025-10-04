# Implement registrability-check endpoint integrating external service and caching results.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 047

## Purpose
Integrate with external service to verify Hankos meet legal requirements (e.g., actual name, bank viability).

## Endpoint
- `POST /designs/{{designId}}:registrability-check`

## Implementation Steps
1. Validate design state and ensure necessary metadata (name, type) present.
2. Call external registrability API with required payload; handle asynchronous/ synchronous responses.
3. Cache result in `designs/{{id}}/registrability` sub-document with status, score, reason, expiresAt.
4. Return response with status and messages; surface to UI for guidance.
5. Tests mocking external service verifying success/failure caching and throttling (rate limits per user).
