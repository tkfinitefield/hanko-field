# Implement job runners (Cloud Run jobs/PubSub subscribers) for asynchronous tasks (AI processing, invoice generation, exports).

**Parent Section:** 9. Background Jobs & Scheduling
**Task ID:** 110

## Purpose
Implement background workers (Cloud Run jobs or Pub/Sub subscribers) for asynchronous processing (AI, invoices, exports).

## Implementation Steps
1. Create `cmd/jobs` entrypoints for each worker with dependency injection for services.
2. Subscribe to Pub/Sub topics (ai-jobs, webhook-retry, export-jobs) using push/pull model as appropriate.
3. Implement graceful shutdown, ack/nack handling, and metrics for throughput.
4. Provide local runner harness for emulator testing.
5. Tests verifying message handling using Pub/Sub emulator/fakes.
