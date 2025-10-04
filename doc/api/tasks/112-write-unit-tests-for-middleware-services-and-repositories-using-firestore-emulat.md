# Write unit tests for middleware, services, and repositories (using Firestore emulator/mocks).

**Parent Section:** 10. Testing Strategy
**Task ID:** 112

## Scope
Cover middleware, services, repositories with fast, isolated tests using mocks/emulators.

## Plan
- Establish testing framework (`testing` package, `stretchr/testify`).
- Provide helper for Firestore emulator setup and teardown.
- Ensure each service has behavioural tests covering success/error paths.
- Integrate into CI (`go test ./...` with race detector optionally).
- Track coverage targets per package (e.g., >=75%).
