# Provide request context helpers for pagination (pageSize/pageToken), sorting, and filter parsing.

**Parent Section:** 2. Core Platform Services
**Task ID:** 013

## Goal
Offer reusable helpers for pagination (`pageSize`, `pageToken`), sorting (`orderBy`), and filtering (`filter=field op value`) matching the API spec.

## Design
- `internal/platform/pagination` with `Params` struct storing size, token, order, filters.
- Support Firestore cursor tokens encoded as base64 JSON.
- Validate order/filter fields against allowlist passed by handlers.

## Steps
1. Implement parser that handles defaults and max limits (e.g., max pageSize=100).
2. Provide token encode/decode using `startAfter` or `startAt` values for Firestore.
3. Support filter operators (`==`, `>=`, `<=`, `array-contains`) with sanitized input.
4. Add unit tests for parsing errors and valid combinations.
