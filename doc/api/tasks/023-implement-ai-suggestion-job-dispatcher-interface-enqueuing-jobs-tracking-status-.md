# Implement AI suggestion job dispatcher interface (enqueuing jobs, tracking status, storing results).

**Parent Section:** 3. Shared Domain Services
**Task ID:** 023

## Goal
Coordinate asynchronous AI jobs for design suggestions, tracking job lifecycle and providing data to API endpoints.

## Responsibilities
- Maintain `aiJobs` collection with status (`queued`, `processing`, `completed`, `failed`), inputs, outputs, and error messages.
- Publish jobs to Pub/Sub queue consumed by AI worker Cloud Run jobs.
- Store generated suggestions in `designSuggestions` with metadata and preview assets.

## Steps
1. Define DTO for AI requests (method, model, prompt, design snapshot) and ensure payload stored securely.
2. Implement dispatcher `QueueSuggestion(designID, parameters)` pushing Pub/Sub message with idempotency key.
3. Provide methods for polling job status and retrieving completed suggestions.
4. Integrate with webhook `/webhooks/ai/worker` to update job status and persist outputs.
5. Add tests using Pub/Sub emulator to assert enqueue behaviour and state updates.
