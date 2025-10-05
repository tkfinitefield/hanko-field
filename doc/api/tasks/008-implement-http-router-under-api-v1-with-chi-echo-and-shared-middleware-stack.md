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

---

## Router Implementation Summary (2025-10-05)

- `handlers.NewRouter` now builds on `chi` with request ID, real IP, logger, recoverer, and timeout middleware plus `/healthz` and `/api/v1` grouping.
- Route option helpers (`WithPublicRoutes`, `WithAdminRoutes`, etc.) allow mounting domain registrars while default groups return structured 501 JSON until implemented.
- JSON error writers provide consistent `404`/`405` responses; wildcard groups share the same helper to aid future middleware layering.
- Added unit tests validating health, default fallbacks, injected registrars, and JSON error bodies.

## Follow-ups
- [ ] Implement real route registrars as domain handlers come online (public, me, designs, cart, orders, admin, webhooks, internal).
- [ ] Replace placeholder middleware with concrete auth, RBAC, rate limit, tracing, and idempotency implementations once available.
- [ ] Populate integration tests asserting full request flow once handlers and middleware are built.
