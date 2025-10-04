# Implement deactivate-and-mask endpoint anonymizing user data and revoking access.

**Parent Section:** 6. Admin / Staff Endpoints > 6.5 Users / Reviews / Audit
**Task ID:** 089

## Purpose
Facilitate compliance request to deactivate user and anonymize PII.

## Endpoint
- `POST /users/{{uid}}:deactivate-and-mask`

## Implementation Steps
1. Set user `isActive=false`, `piiMasked=true`, remove roles; call Firebase Admin to disable account.
2. Scrub PII fields (name/email/phone) replacing with tokens; detach payment methods.
3. Emit audit log and notify downstream systems (CRM, marketing suppression list).
4. Tests verifying data masking, idempotency, and side effects.
