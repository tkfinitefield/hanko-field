# Implement Firestore client factory, transaction helpers, and strongly typed repository abstractions.

**Parent Section:** 2. Core Platform Services
**Task ID:** 014

## Goal
Abstract Firestore client creation, transactions, and typed access patterns to minimise boilerplate and centralise emulator configuration.

## Design
- Package `internal/platform/firestore` providing `Provider` (lazy singleton), `RunTransaction` helper, typed mapper utilities.
- Support emulator detection via env vars.
- Provide context deadlines and retry/backoff configuration.

## Steps
- [x] Implement provider with thread-safe initialisation and graceful shutdown hook.
- [x] Add repository base struct exposing helper methods (Get, Set, Update, Query) with error wrapping.
- [x] Provide transaction helper that surfaces context cancellation and convert Firestore errors to domain errors.
- [x] Create integration tests using Firestore emulator docker container.

## Work Summary
- Added `api/internal/platform/firestore` package with provider, error classification, transaction helper, and generic repository base/codec utilities.
- Added integration test (`api/internal/platform/firestore/firestore_integration_test.go`) that launches the Firestore emulator via Docker and validates typed repository operations and transaction behaviour.
