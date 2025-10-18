# Implement Idempotency-Key middleware storing request fingerprint + result (Firestore or Redis) for POST/PUT/PATCH/DELETE.

**Parent Section:** 2. Core Platform Services
**Task ID:** 011

## Goal
Ensure safe retries for mutating requests by deduplicating `Idempotency-Key` operations per user and endpoint.

## Design
- Middleware intercepts POST/PUT/PATCH/DELETE and requires header.
- Persist entry in `idempotency_keys` collection or Redis with key hash, status, response payload, expiry.
- On duplicate, short-circuit and replay stored response.

## Steps
1. Implement `IdempotencyStore` interface with Firestore implementation using transactions for concurrency.
2. Wrap handlers in response recorder capturing status, headers, body.
3. Enforce TTL config (e.g., 24h) and cleanup job for expired keys.
4. Unit test duplicate request behaviour and missing header path.

## Completion Notes
- Implemented Firestore-backed idempotency store with scoped keys, TTL extension, and cleanup helpers.
- Added HTTP middleware capturing responses, hashing request fingerprints, and replaying stored responses for duplicates.
- Wired middleware and cleanup ticker into API server startup with configurable headers, TTL, and cleanup cadence.
- Added configuration schema for idempotency settings and unit tests covering success, replay, conflict, and pending scenarios.
