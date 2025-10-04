# Define service interfaces and layering (handler → service → repository) for maintainability and testing.

**Parent Section:** 0. Planning & Alignment
**Task ID:** 003

## Goal
Establish clear contracts for handlers, services, repositories, and background jobs so teams can implement features independently while respecting layering rules.

## Deliverables
- Package layout proposal (`/cmd/api`, `/internal/handlers`, `/internal/services`, `/internal/repositories`, `/internal/jobs`, `/internal/platform`).
- Go interface definitions for domain services (`DesignService`, `CartService`, `PromotionService`, etc.) with method signatures.
- Dependency injection plan (wire/fx/custom) covering lifecycle and test doubles.
- Testing matrix describing unit vs integration vs emulator coverage per layer.

## Steps
1. Draft handler/service/repository boundaries for each domain using flows in `doc/api/api_design.md`.
2. Identify shared middleware/utilities and define their packages within `/internal/platform`.
3. Document interface signatures including context usage, DTO structs, and error contracts.
4. Decide mocking libraries/emulator strategies to support isolated service tests.
5. Publish ADR describing layering rules, exceptions, and dependency direction.

## Acceptance Criteria
- Interfaces committed with documentation comments and sample implementation skeletons.
- Unit tests can target services with mocks or fakes through defined interfaces.
- Team agreement recorded (meeting notes/ADR) on layering approach.

---

## Layering Sign-off (2025-04-01)

### Deliverables
- ✅ Package structure documented in `doc/api/package-layout.md` outlining directories for handlers, services, repositories, domain types, platform adapters, jobs, DI, and tests.
- ✅ Service contracts published in `api/internal/services/interfaces.go` with domain DTO aliases and command structs covering designs, carts, checkout, orders, promotions, users, inventory, content, catalog, assets, and system utilities.
- ✅ Repository interfaces + registry scaffold captured in `api/internal/repositories/interfaces.go`, including filters, transaction abstraction, and lifecycle hooks for Firestore/Storage adapters.
- ✅ Shared domain models consolidated under `api/internal/domain/types.go` to keep data structures consistent between services and repositories.
- ✅ DI container stub (`api/internal/di/container.go`) defines service bundle, registry dependency, option overrides, and close semantics ready for wire-based providers.
- ✅ Layering ADR agreed and archived at `doc/api/adr/0001-layering-and-dependency-injection.md` with decisions on dependency direction, error translation, and provider strategy.
- ✅ Testing coverage expectations enumerated in `doc/api/testing-matrix.md` for each layer with tooling/emulator requirements.

### Key Decisions
- Adopt Handlers → Services → Repositories → Platform layering with domain models in a shared package to prevent circular dependencies.
- Use `google/wire` for compile-time dependency injection; container accepts repository registry and supports overrides for fakes during tests.
- Standardise repository error surfaces (`RepositoryError`) and translation via `services.ErrorTranslator` for consistent HTTP mappings.
- Provide repository registry accessor pattern (`repositories.Registry`) so handlers and jobs receive cohesive service bundles while enabling resource cleanup.

### Next Actions
- Generate `internal/di/providers.go` with wire provider sets mapping platform clients, repositories, and services (Target: 2025-04-10, Owner: Backend TL).
- Scaffold in-memory repository fakes under `internal/repositories/memory` to unblock service unit tests (Target: 2025-04-17, Owner: Backend squad).
- Prepare emulator docker-compose and CI wiring for repository integration tests per `doc/api/testing-matrix.md` (Target: 2025-05-01, Owner: DevOps).
