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

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` (queue filter `Select`, today stats chips).
- **Work list:** `DataTable` enumerating pending QC items with type, stage, assigned member, SLA `Badge`.
- **Action drawer:** `DetailDrawer` for step-by-step checks, pass/fail `ButtonGroup`, comment `Textarea`.
- **Rework modal:** Trigger `Modal` for reassigning tasks and capturing issue category via `Select`.
- **Performance widgets:** `SummaryCard` row showing pass rate, rework %, average handle time.
- **Notifications:** `InlineAlert` for upstream blockers and `ToastHost` for completion feedback.
