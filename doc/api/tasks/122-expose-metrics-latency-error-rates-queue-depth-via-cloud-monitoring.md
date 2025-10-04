# Expose metrics (latency, error rates, queue depth) via Cloud Monitoring.

**Parent Section:** 12. Observability & Operations
**Task ID:** 122

## Goal
Publish application metrics (latency, error rate, queue depth) to Cloud Monitoring.

## Plan
- Integrate OpenTelemetry metrics exporter for HTTP server and custom domain metrics.
- Instrument key operations (checkout, AI queue, background jobs) with counters/histograms.
- Define metric naming conventions and labels (environment, endpoint, status).
- Provide `/metrics` endpoint if using Prometheus-to-Cloud Monitoring bridge.
