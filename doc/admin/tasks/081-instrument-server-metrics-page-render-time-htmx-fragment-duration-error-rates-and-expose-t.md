# Instrument server metrics (page render time, htmx fragment duration, error rates) and expose to Cloud Monitoring.

**Parent Section:** 16. Observability & Maintenance
**Task ID:** 081

## Goal
Instrument server to emit metrics for Cloud Monitoring.

## Implementation Steps
1. Use OpenTelemetry or expvar to publish latency, error rate, fragment render duration.
2. Add counters for key events (order status change submissions, promotions created).
3. Export metrics to Cloud Monitoring with labels (endpoint, environment).
