# Set up integration tests harness (httptest + DOM assertions) and smoke test environment for admin flows.

**Parent Section:** 1. Project & Infrastructure Setup
**Task ID:** 009

## Goal
Set up testing infrastructure for admin UI (integration & smoke tests).

## Implementation Steps
1. Use `httptest` to spin server with in-memory dependencies (mock API client, fake auth).
2. Write utility to parse rendered HTML (e.g., goquery) for assertions on DOM structure.
3. Configure smoke tests hitting key routes (login challenge, orders list) under `make test-ui`.
4. Document how to run tests locally and include them in CI pipeline.
