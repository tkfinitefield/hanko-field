# Write integration tests exercising representative endpoint flows with emulators.

**Parent Section:** 10. Testing Strategy
**Task ID:** 113

## Scope
Exercise key endpoint flows (signup, cart, checkout, admin operations) against emulators.

## Plan
- Use `httptest` or end-to-end harness spinning server with emulator config.
- Seed fixtures for Firestore, Storage, Pub/Sub with deterministic data.
- Verify request/response payloads, Firestore side effects, and event emissions.
- Run as part of CI (longer stage) and as gating for releases.
