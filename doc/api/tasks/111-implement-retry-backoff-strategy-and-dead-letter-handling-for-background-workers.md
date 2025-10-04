# Implement retry/backoff strategy and dead-letter handling for background workers.

**Parent Section:** 9. Background Jobs & Scheduling
**Task ID:** 111

## Purpose
Define retry/backoff policies and dead-letter queues for background jobs to ensure resilience.

## Implementation Steps
1. Standardize retry configuration (exponential backoff, max attempts) per job type.
2. Configure Pub/Sub subscription dead-letter topics; implement handler to surface alerts when messages land there.
3. For Cloud Run jobs, implement manual retry/cron schedule and stateful tracking.
4. Document policies and integrate with logging/metrics (retry count, DLQ size).
5. Tests verifying retry wrappers with mocked failures.
