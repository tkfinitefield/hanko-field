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
1. Implement provider with thread-safe initialisation and graceful shutdown hook.
2. Add repository base struct exposing helper methods (Get, Set, Update, Query) with error wrapping.
3. Provide transaction helper that surfaces context cancellation and convert Firestore errors to domain errors.
4. Create integration tests using Firestore emulator docker container.
