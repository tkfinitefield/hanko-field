# Implement deactivate-and-mask modal tied to `POST /users/{uid}:deactivate-and-mask` with confirmation and audit log output.

**Parent Section:** 9. Customers, Reviews, and KYC
**Task ID:** 053

## Goal
Provide modal to process account deletion/PII masking.

## Implementation Steps
1. Show summary of effects (orders remain, PII removed).
2. Require confirmation phrase before submission.
3. Trigger backend endpoint and refresh customer detail with status updates.
4. Log action to audit log.
