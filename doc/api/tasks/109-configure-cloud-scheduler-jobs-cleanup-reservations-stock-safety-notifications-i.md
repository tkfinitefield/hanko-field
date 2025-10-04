# Configure Cloud Scheduler jobs (cleanup reservations, stock safety notifications) invoking internal endpoints with auth.

**Parent Section:** 9. Background Jobs & Scheduling
**Task ID:** 109

## Purpose
Configure Cloud Scheduler to invoke internal maintenance endpoints at required cadence.

## Jobs
- Cleanup reservations (every 5 minutes).
- Stock safety notifications (daily or hourly).
- Optional: retry failed webhooks, invoice generation polling.

## Implementation Steps
1. Define scheduler jobs via IaC referencing service account with correct auth tokens.
2. Configure HTTP target with OIDC token (service account) and rate-limiting.
3. Document schedule and expected runtime.
4. Tests/integration: deploy to dev and confirm invocation with logs.
