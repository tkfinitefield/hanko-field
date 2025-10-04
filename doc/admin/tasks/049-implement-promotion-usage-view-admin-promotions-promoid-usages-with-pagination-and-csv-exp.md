# Implement promotion usage view (`/admin/promotions/{promoId}/usages`) with pagination and CSV export capability.

**Parent Section:** 8. Promotions & Marketing
**Task ID:** 049

## Goal
Display promotion usage per user with export option.

## Implementation Steps
1. Table with user email, number of uses, last used, total discount given.
2. Support pagination and export to CSV via background job.
3. Provide filters (>=N uses, timeframe).
