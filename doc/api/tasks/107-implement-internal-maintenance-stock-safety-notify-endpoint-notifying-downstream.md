# Implement internal maintenance stock-safety-notify endpoint notifying downstream systems.

**Parent Section:** 8. Internal Endpoints
**Task ID:** 107

## Purpose
Notify downstream systems or Slack when stock falls below safety threshold.

## Endpoint
- `POST /internal/maintenance/stock-safety-notify`

## Implementation Steps
1. Query inventory service for low stock results; format notification payloads.
2. Send notifications (email/Slack) and record last-notified timestamp to prevent spam.
3. Tests verifying de-duplication logic.
