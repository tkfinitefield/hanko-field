# Implement audit log writer service for write operations across domains.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 024

## Goal
Provide centralised audit log recording for sensitive mutations across the system, ensuring compliance and traceability.

## Responsibilities
- Write immutable entries to `auditLogs` collection with fields: `id`, `timestamp`, `actor`, `actorType`, `action`, `targetRef`, `metadata`, `ip`, `userAgent`.
- Offer helper to emit logs from services and middleware (e.g., login, profile updates, order state changes).
- Support querying by target or actor for admin endpoints and exports.

## Steps
- [x] Implement `AuditLogService` with `Record(ctx, entry)` and `List(ctx, filter)`.
- [x] Ensure PII stored securely; include diff/old values only when compliant, otherwise store hashed references.
- [x] Add retention/archive strategy (e.g., automatic export to BigQuery) documented for ops.
- [x] Provide tests verifying entry schema and error-handling (e.g., when logging fails, underlying mutation should still succeed but emit warning).

## Completion Notes
- Added `api/internal/services/audit_log_service.go` implementing the audit writer with sanitisation, hashing of IP/marked fields, safe error handling, and a corresponding test suite.
- Extended domain/repository/service contracts to expose new audit log fields and filtering, wired the service through DI, and refactored `userService` to emit structured audit records via the helper.
- Documented nightly BigQuery export plus monthly archival workflow in `doc/api/infrastructure.md` to guide ops on retention/monitoring expectations.
