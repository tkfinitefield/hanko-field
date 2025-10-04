# Set up uptime checks and alerts for critical admin endpoints (login, orders list, notifications).

**Parent Section:** 16. Observability & Maintenance
**Task ID:** 083

## Goal
Set up uptime checks and alerts for critical admin endpoints.

## Implementation Steps
1. Configure Cloud Monitoring uptime checks for `/admin/login`, `/admin/orders` (non-auth alternative via synthetic?).
2. Use service account token or headless browser script for auth-protected checks.
3. Set alert notification channels.
