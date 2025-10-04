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
1. Implement repository with transaction-based increment supporting custom step/formatters.
2. Provide service-level helpers for specific counters (invoices, job numbers) applying prefix logic.
3. Document error handling for exhaustion (max value) and concurrency.
4. Tests verifying increments under contention using emulator.
