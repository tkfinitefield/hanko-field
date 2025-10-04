# Implement structured logging, trace propagation (Cloud Trace), and panic/error handling middleware producing JSON error responses.

**Parent Section:** 2. Core Platform Services
**Task ID:** 012

## Goal
Provide structured JSON logging, panic recovery, and tracing propagation compatible with Cloud Logging/Trace.

## Design
- Use `zap` logger with fields: timestamp, severity, trace ID, request ID, user ID, route, latency.
- Recovery middleware converts panics to 500 responses, logs stack trace.
- Integrate OpenTelemetry to propagate `X-Cloud-Trace-Context`.

## Steps
1. Initialize logger in `main` and inject into context.
2. Implement middleware wrapping handler to log request start/finish with latency.
3. Sanitize log fields to avoid PII leakage; centralize redaction helpers.
4. Provide structured error response builder ensuring consistent JSON format.
