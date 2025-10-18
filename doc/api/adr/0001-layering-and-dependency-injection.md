# ADR 0001: API Service Layering & Dependency Injection

- **Status:** Accepted (2025-04-01)
- **Deciders:** R. Suzuki (Product), K. Nakamura (Tech Lead), K. Watanabe (DevOps), H. Kimura (QA)
- **Consulted:** Backend Squad, Web Squad, Data Engineering

## Context

The API v1 program spans public surfaces, authenticated user flows, admin tooling, webhooks, and internal jobs. We need a maintainable Go codebase that allows independent workstreams, predictable testing, and production-grade observability. Prior work in `doc/api/api_design.md` defines endpoints and data models, but we have not yet committed to how code will be layered, how dependencies are wired together, or how teams can test in isolation.

## Decision

1. **Layering Contract**
   - Split runtime into four layers: `handlers` (transport adapters), `services` (business orchestration), `repositories` (Firestore/Storage persistence), and `platform` (cross-cutting concerns: logging, tracing, IAM, PSP, AI gateways).
   - Handlers depend only on service interfaces; services depend on repositories + platform abstractions; repositories wrap Cloud SDK clients; platform packages do not depend on domain-level packages.
   - Domain models live in `internal/domain` and are imported by both services and repositories.

2. **Package Layout**
   ```text
   api/
     cmd/api/              # entrypoint wiring HTTP server + DI container
     internal/
       handlers/           # HTTP + background handlers (htmx, webhook)
       services/           # interface contracts + command DTOs
       repositories/       # persistence interfaces + filter DTOs
       domain/             # shared value objects/aggregates
       platform/           # firebase, firestore, storage, logging, auth adapters
       jobs/               # background processors (AI worker, cleanup)
       di/                 # dependency injection container + provider sets
   ```

3. **Dependency Injection**
   - Adopt [`google/wire`](https://github.com/google/wire) for compile-time DI. Provider sets live in `internal/di/providers.go` (to be implemented) and export constructors for repositories, platform clients, and services.
   - `internal/di/container.go` exposes `NewContainer(ctx, repositories.Registry, ...Option)` to bootstrap runtime. Production path uses real registry; tests pass fake registries or override options for targeted services.
   - Repository registry implements lifecycle management (Close) so Cloud clients are reused and shut down gracefully.

4. **Error Handling**
   - Introduce `services.ErrorTranslator` to normalize repository/platform errors into domain-specific `DomainError`s with stable codes for handlers.
   - Repository implementations return `RepositoryError` (with `IsNotFound`, `IsConflict`, `IsUnavailable`) to standardize translation.

5. **Testing Strategy**
   - Services are unit-tested using fake repositories/platform clients injected through the DI container options. We standardize on [`testify/mock`](https://github.com/stretchr/testify) for mocks and hand-written fakes for hot paths (cart, checkout).
   - Repository tests run against Firestore emulator + fake GCS via test harness under `api/test/integration`. Integration suites cover transaction semantics, TTL/index usage, and data shaping.
   - Handler tests use `net/http/httptest` with DI container assembled via in-memory fakes.
   - Background jobs instrumented with contract tests to assert idempotency and error retries using the same fake registry.

## Consequences

- Clear separation makes parallel development feasible; teams can implement handlers/services without waiting for Firestore migrations.
- Compile-time DI reduces runtime reflection overhead and gives early feedback on missing providers.
- Additional upfront work is required to maintain provider sets and fake implementations, but QA gains reliable surface-level tests.
- The repository registry abstraction must be implemented before feature work; DevOps will contribute scaffolding for Firestore client pooling and emulator wiring.
- Documentation (package layout, testing matrix) must be kept current as new domains emerge; ADR will be revisited post-beta.

## Follow-up Actions

1. Implement provider set skeletons in `internal/di/providers.go` (Tech Lead, 2025-04-10).
2. Create fake repositories under `internal/repositories/memory` for unit tests (Backend Squad, 2025-04-17).
3. Add CI job invoking Firestore emulator integration tests before merge (DevOps, 2025-05-01).
