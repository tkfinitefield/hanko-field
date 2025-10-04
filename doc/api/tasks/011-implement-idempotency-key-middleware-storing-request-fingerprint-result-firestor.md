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
