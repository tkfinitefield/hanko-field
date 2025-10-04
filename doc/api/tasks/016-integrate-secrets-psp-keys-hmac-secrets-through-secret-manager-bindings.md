# Integrate secrets (PSP keys, HMAC secrets) through Secret Manager bindings.

**Parent Section:** 2. Core Platform Services
**Task ID:** 016

## Goal
Integrate Google Secret Manager for sensitive values and expose ergonomic API for runtime consumption.

## Design
- `internal/platform/secrets.Fetcher` reads `secret://` URIs, caches values, and supports reload notifications.
- Provide local development fallback to read from `.secrets.local` file when Secret Manager unavailable.
- Expose metrics for fetch latency and cache hits.

## Steps
1. Implement fetcher with environment-specific project IDs and version pins (latest by default).
2. Integrate with config loader to resolve secret references automatically.
3. Add panic-on-start option for required secrets missing; log redacted names.
4. Provide rotation playbook and optional Pub/Sub push trigger for hot reload.
