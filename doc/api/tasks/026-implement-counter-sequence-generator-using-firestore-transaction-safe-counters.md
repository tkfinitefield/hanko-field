# Implement counter/sequence generator using Firestore transaction-safe counters.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 026

## Goal
Provide atomic sequence generation for invoices, order numbers, and other counters using Firestore-safe operations.

## Design
- Collection `counters` keyed by `{scope}:{name}` storing `currentValue`, `step`, `updatedAt`.
- `CounterService.Next(scope, name)` runs Firestore transaction to increment and return formatted value (with prefixes, zero padding).
- Optional caching for high-throughput sequences using sharded counters.

## Steps
- [x] Implement repository with transaction-based increment supporting custom step/formatters.
- [x] Provide service-level helpers for specific counters (invoices, job numbers) applying prefix logic.
- [x] Document error handling for exhaustion (max value) and concurrency.
- [x] Tests verifying increments under contention using emulator.

## Completion Notes
- Added Firestore-backed counter repository with transactional increments, optional configuration, and exhaustion signalling (`api/internal/repositories/firestore/counter_repository.go`, `api/internal/repositories/counter_errors.go`).
- Exposed formatted sequence generation via new `CounterService` with helpers for orders/invoices plus unit coverage (`api/internal/services/counter_service.go`, `api/internal/services/counter_service_test.go`).
- Wired counter service into DI and updated repository interfaces and stubs to support configuration (`api/internal/di/container.go`, `api/internal/repositories/interfaces.go`, `api/internal/services/order_service_test.go`).
- Added Firestore emulator integration test validating concurrent increments and max-value handling (`api/internal/repositories/firestore/counter_repository_integration_test.go`).
