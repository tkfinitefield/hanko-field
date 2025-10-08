# API Testing Matrix

| Layer | Scope | Primary Tooling | Emulator/Fixture Needs | Ownership | Notes |
| --- | --- | --- | --- | --- | --- |
| Handlers (`internal/handlers`) | HTTP routing, middleware, request/response mapping | `net/http/httptest`, `chi` test helpers | Fake Firebase auth tokens, stub services via DI overrides | Backend squad | Focus on serialization, auth scopes, idempotency headers. |
| Services (`internal/services`) | Business rules, orchestration, error translation | `testing`, `testify/mock`, in-memory fakes | Fake repositories (`internal/repositories/memory`), fake platform clients | Backend squad | Each method must have unit tests covering happy path, domain errors, and repository failures. |
| Repositories (`internal/repositories`) | Firestore/Storage persistence, transactions, indexes | Firestore emulator, fake GCS (`fsouza/fake-gcs-server`) | Emulator compose file (`tools/emulators/docker-compose.yml`), seed fixtures | Backend + Data Eng | Integration tests validate indexes, TTL behaviour, and transaction retries. |
| Platform (`internal/platform`) | Firebase auth, PSP, AI gateway adapters | `httptest`, contract suites against sandbox APIs | External sandbox creds (Stripe), AI mock server | Platform sub-team | Use golden files for signature verification and ensure retry policies instrumented. |
| Jobs (`internal/jobs`) | Background workers, scheduler targets | `testing`, job harness, fake queues | Pub/Sub emulator, Cloud Tasks emulator | Backend squad | Validate idempotency + error handling using controlled fake registries. |
| End-to-End Smoke | Minimal flows across HTTP + PSP + Firestore | Postman/Newman or k6, GitHub Actions nightly | Firestore emulator (local), Stripe test mode | QA | Runs after deployments; uses DI to switch to staging endpoints. |

## Coverage Targets

- **Unit tests**: ≥80% coverage for services, ≥70% for handlers.
- **Integration tests**: Execute on every PR (emulator-based) with parallel execution capped at 5 suites to control runtime.
- **End-to-end smoke**: Nightly on staging; manual trigger for release candidate sign-off.

## Test Data Strategy

- Seed minimal fixtures via `tools/scripts/seed.go`, relying on JSON documents stored in `test/fixtures/`.
- Use ULIDs from `doc/api/models/external-ids.yaml` to keep deterministic ordering in tests.
- For PSP integrations, rely on Stripe sandbox webhooks captured through signed payload fixtures.
