# Implement webhook security middleware (IP filtering, replay protection) and monitoring.

**Parent Section:** 7. Webhooks (Inbound)
**Task ID:** 100

## Purpose
Provide reusable security middleware for webhook endpoints (IP filtering, replay protection, metrics).

## Implementation Steps
1. Implement middleware that checks source IP against allowlist (configurable) and logs rejections.
2. Add replay protection storing `signature + timestamp` for limited duration.
3. Record metrics (webhook latency, failures) exposed to Cloud Monitoring.
4. Integrate middleware into router for all `/webhooks/*` routes.
5. Tests verifying blocked IPs, replay detection, and logging.
