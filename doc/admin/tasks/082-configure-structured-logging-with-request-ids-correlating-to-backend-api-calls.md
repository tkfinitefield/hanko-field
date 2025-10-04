# Configure structured logging with request IDs correlating to backend API calls.

**Parent Section:** 16. Observability & Maintenance
**Task ID:** 082

## Goal
Configure structured logging with trace correlation.

## Implementation Steps
1. Use structured logger (zap/zerolog) with fields: requestID, userID, route, status.
2. Propagate trace ID from frontend to backend API calls.
3. Provide log context helpers for fragments.
