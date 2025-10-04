# Implement AI worker webhook handler updating AI job status and persisting generated suggestions.

**Parent Section:** 7. Webhooks (Inbound)
**Task ID:** 099

## Purpose
Receive callbacks from AI worker after processing design suggestion jobs.

## Endpoint
- `POST /webhooks/ai/worker`

## Implementation Steps
1. Authenticate using HMAC or OIDC depending on worker deployment.
2. Payload includes `jobId`, `suggestionId`, `status`, `outputs`, `error`.
3. Update `aiJobs` and `designSuggestions` documents with results, store generated asset references.
4. Trigger notifications (email/push) to user when suggestion ready.
5. Tests covering success, failure, and duplicate webhook deliveries.
