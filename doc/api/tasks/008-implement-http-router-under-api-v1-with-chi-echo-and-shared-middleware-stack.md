# Implement HTTP router under `/api/v1` with chi/echo and shared middleware stack.

**Parent Section:** 2. Core Platform Services
**Task ID:** 008

## Goal
Offer a central router under `/api/v1` that wires shared middleware and mounts domain routers for public, authenticated, admin, webhook, and internal endpoints.

## Design
- Adopt `chi` for lightweight routing with context-aware middleware.
- Route tree: `/api/v1/public`, `/api/v1/me`, `/api/v1/designs`, `/api/v1/cart`, `/api/v1/orders`, `/api/v1/admin/*`, `/api/v1/webhooks`, `/api/v1/internal`.
- Register middleware: request ID, structured logging, tracing, recovery, idempotency, auth, RBAC, rate limiting.
- Expose router builder `internal/handlers/router.go` returning configured `chi.Router`.

## Steps
1. Implement router factory injecting config and dependencies (services, middlewares).
2. Mount domain-specific route registrars from packages (e.g., `handlers/designs.Register(r, svc)`).
3. Provide 404/405 handlers returning JSON error payloads.
4. Unit test registration to ensure expected routes exist.
