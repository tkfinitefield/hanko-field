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
