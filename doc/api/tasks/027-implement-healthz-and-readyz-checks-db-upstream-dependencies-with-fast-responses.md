# Implement `/healthz` and `/readyz` checks (DB, upstream dependencies) with fast responses.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 027

## Purpose
Expose lightweight endpoints for load balancers and Cloud Run to verify service health without requiring authentication.

## Behaviour
- `GET /healthz`: returns 200 `{"status":"ok"}` immediately if process alive.
- `GET /readyz`: verifies dependencies (Firestore connectivity, Storage signed URL signer, Pub/Sub, Secret Manager) before responding.
- Responses must include build metadata (`version`, `commitSha`, `environment`).

## Implementation Steps
1. [x] Implement handler `HealthzHandler` returning static response.
2. [x] Implement handler `ReadyzHandler` executing dependency checks with timeouts and aggregated error reporting (`details[]`).
3. [x] Ensure endpoints bypass auth and idempotency middleware but include logging/metrics.
4. [x] Add unit tests mocking dependency clients to simulate pass/fail scenarios.

## Completion Notes
- Added dependency-backed health repository and system service to surface build metadata and readiness data (`api/internal/repositories/health_repository.go`, `api/internal/services/system_service.go`).
- Reworked health handlers and router wiring to expose `/healthz` and `/readyz` with structured responses plus detailed check output (`api/internal/handlers/health.go`, `api/internal/handlers/router.go`).
- Bootstrapped build metadata and readiness checks in main, wiring Firestore and Secret Manager probes along with handler configuration (`api/cmd/api/main.go`).
- Extended DI scaffolding and comprehensive unit coverage for repository, service, handlers, and routing (`api/internal/di/container.go`, `api/internal/handlers/health_test.go`, `api/internal/repositories/health_repository_test.go`, `api/internal/services/system_service_test.go`, `api/internal/handlers/router_test.go`).
