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
- [x] Define DTO for AI requests (method, model, prompt, design snapshot) and ensure payload stored securely.
- [x] Implement dispatcher `QueueSuggestion(designID, parameters)` pushing Pub/Sub message with idempotency key.
- [x] Provide methods for polling job status and retrieving completed suggestions.
- [x] Integrate with webhook `/webhooks/ai/worker` to update job status and persist outputs.
- [x] Add tests using Pub/Sub emulator to assert enqueue behaviour and state updates.

## Completion Notes
- Introduced AI job domain types and repository contracts plus the concrete dispatcher in `api/internal/services/background_job_dispatcher.go`, covering queueing, status transitions, idempotency, and suggestion persistence.
- Added Pub/Sub adapter `api/internal/platform/jobs/pubsub_publisher.go` (and tests) publishing JSON job envelopes without exposing prompts, leveraging `pstest` to assert message payloads.
- Wrote dispatcher unit tests in `api/internal/services/background_job_dispatcher_test.go` validating enqueue behaviour, idempotent replay, success/failure completions, and ensured `go test ./...` passes after updating module deps for Pub/Sub support.
