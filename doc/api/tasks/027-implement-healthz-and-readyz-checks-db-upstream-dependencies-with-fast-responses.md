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
1. Implement handler `HealthzHandler` returning static response.
2. Implement handler `ReadyzHandler` executing dependency checks with timeouts and aggregated error reporting (`details[]`).
3. Ensure endpoints bypass auth and idempotency middleware but include logging/metrics.
4. Add unit tests mocking dependency clients to simulate pass/fail scenarios.
