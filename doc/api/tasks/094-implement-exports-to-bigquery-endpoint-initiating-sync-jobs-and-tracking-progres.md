# Implement exports to BigQuery endpoint initiating sync jobs and tracking progress.

**Parent Section:** 6. Admin / Staff Endpoints > 6.6 Operations Utilities
**Task ID:** 094

## Purpose
Trigger data export jobs syncing operational data to BigQuery for analytics.

## Endpoint
- `POST /exports:bigquery-sync`

## Implementation Steps
1. Accept payload specifying entities (orders, users, promotions) and time window.
2. Publish background job to export service; ensure idempotency to avoid duplicate loads.
3. Track job status (pending, running, completed, failed) and surface via `/system/tasks`.
4. Tests verifying job enqueuing and validation.
