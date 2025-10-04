# Implement internal maintenance cleanup-reservations endpoint releasing expired reservations.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 106

## Purpose
Release expired reservations via scheduled invocation.

## Endpoint
- `POST /internal/maintenance/cleanup-reservations`

## Implementation Steps
1. Scan reservations with `expiresAt < now` and status `reserved` (batch to avoid timeouts).
2. Release via inventory service; accumulate metrics for counts released.
3. Log summary and errors for monitoring.
4. Tests verifying batching and partial failure handling.
