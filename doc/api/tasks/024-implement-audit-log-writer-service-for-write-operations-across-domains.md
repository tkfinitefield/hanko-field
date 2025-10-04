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
1. Implement `AuditLogService` with `Record(ctx, entry)` and `List(ctx, filter)`.
2. Ensure PII stored securely; include diff/old values only when compliant, otherwise store hashed references.
3. Add retention/archive strategy (e.g., automatic export to BigQuery) documented for ops.
4. Provide tests verifying entry schema and error-handling (e.g., when logging fails, underlying mutation should still succeed but emit warning).
