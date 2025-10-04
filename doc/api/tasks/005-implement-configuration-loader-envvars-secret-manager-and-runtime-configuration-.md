# Implement configuration loader (envvars + Secret Manager) and runtime configuration schema.

**Parent Section:** 1. Project & Environment Setup
**Task ID:** 005

## Goal
Centralise configuration management with support for environment variables, local overrides, and Secret Manager values, exposing a typed struct to application layers.

## Design
- Package: `internal/platform/config` with exported `Load(ctx context.Context) (Config, error)`.
- Support cascading sources: defaults -> `.env` (local) -> environment variables -> Secret Manager references.
- `Config` struct sections: Server (port, timeouts), Firebase (project IDs, credentials path), Firestore (project, emulator), Storage buckets, PSP credentials, AI worker endpoints, webhook secrets, rate limits, feature flags.
- Use validation to fail fast on missing/invalid values.

## Steps
1. Define `Config` struct and default values constants.
2. Implement loader using `envconfig`-style tags or manual mapping to parse env vars.
3. Integrate Secret Manager fetcher with caching and clear error messages on missing secrets.
4. Add helper to inject config into dependency container for handlers/services.
5. Document configuration keys and default values in `doc/api/configuration.md`.

## Acceptance Criteria
- Unit tests cover parsing, default overrides, Secret Manager fallback.
- Local development can use `.env` without touching production secrets.
- Config loader returns typed errors and logs redacted failure details.
