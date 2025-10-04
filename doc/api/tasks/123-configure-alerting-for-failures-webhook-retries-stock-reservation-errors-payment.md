# Configure alerting for failures (webhook retries, stock reservation errors, payment mismatches).

**Parent Section:** 12. Observability & Operations
**Task ID:** 123

## Goal
Configure alerting policies for critical failures (webhook retries, stock reservation errors, payment mismatches).

## Plan
- Identify SLOs and thresholds (e.g., webhook failure rate >5% for 5m).
- Create Cloud Monitoring alerting policies tied to notification channels (Slack, PagerDuty).
- Document runbook links for each alert.
- Test alert triggers using synthetic incidents.
