# Build system errors dashboard (`/admin/system/errors`) pulling failed webhook/job logs with retry actions when permitted.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 066

## Goal
Show failed jobs/webhooks with retry options.

## Implementation Steps
1. Table with columns: job name, error, lastAttempt, retries, actions (retry, acknowledge).
2. Integrate with monitoring API or Firestore collection.
3. Provide filters for job type/severity.
4. Add quick links to related order/promotion.
