# Set up logging/trace pipeline for client errors and attach device context.

**Parent Section:** 15. Analytics, Telemetry, and Monitoring
**Task ID:** 088

## Goal
Set up client logging/trace pipeline.

## Implementation Steps
1. Implement structured logging with context (user id hash, device info).
2. Send logs to remote sink (Sentry, custom backend) respecting privacy.
3. Capture non-fatal errors for debugging.
