# Build QC page (`/admin/production/qc`) to record pass/fail events and trigger rework flows.

**Parent Section:** 5. Orders & Operations > 5.3 Production & Workshop
**Task ID:** 036

## Goal
Implement QC page to record quality control outcomes.

## Implementation Steps
1. List orders awaiting QC with filters.
2. Allow pass/fail actions with reason codes, attachments (photos).
3. On fail, route order back to appropriate stage via production events.
4. Track metrics for QC failure rates.
