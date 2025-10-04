# Implement AI suggestion request endpoint queuing jobs and returning suggestion IDs.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 044

## Purpose
Queue AI-powered design enhancements (balance, candidate generation) for asynchronous processing.

## Endpoint
- `POST /designs/{{designId}}/ai-suggestions`

## Implementation Steps
1. Validate design ownership and ensure design state eligible (not archived).
2. Construct AI job payload (method, model, parameters, design snapshot) and call dispatcher to enqueue.
3. Return response containing `suggestionId`, `status=queued`, and polling URL.
4. Record job reference in `designSuggestions` with initial status.
5. Tests verifying queue invocation and duplicate prevention via idempotency key.
