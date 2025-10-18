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

---

## Configuration Loader Summary (2025-04-01)

### Deliverables
- ✅ Expanded `internal/platform/config` with typed configuration schema, cascading source resolution, secret reference support, and validation (`api/internal/platform/config/config.go`).
- ✅ Added comprehensive unit coverage for defaults, overrides, dotenv fallback, and secret resolution behaviours (`api/internal/platform/config/config_test.go`).
- ✅ Updated DI container to carry the loaded configuration for downstream wiring (`api/internal/di/container.go`).
- ✅ Documented environment keys, defaults, and secret usage patterns in `doc/api/configuration.md` for team reference.
- ✅ Main entrypoint now consumes the typed config, applying runtime-specific timeouts (`api/cmd/api/main.go`).

### Key Decisions
- `sm://` URI scheme indicates Secret Manager references; applications must provide a `SecretResolver` at load time or receive a typed `SecretError`.
- Required fields limited to `Firebase.ProjectID`, `Firestore.ProjectID` (defaults to Firebase), and `Storage.AssetsBucket`; other values may remain empty for local development.
- Feature flags default to conservative values (`AISuggestions=false`, `Promotions=true`) to avoid surprising changes in production.

### Next Actions
- Implement a `SecretResolver` backed by Google Secret Manager and wire it through DI (Owner: DevOps, Target: 2025-04-12).
- Extend configuration tests once PSP and AI service integrations add new required settings.
