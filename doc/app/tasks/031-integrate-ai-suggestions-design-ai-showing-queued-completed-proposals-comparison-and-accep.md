# Integrate AI suggestions (`/design/ai`) showing queued/completed proposals, comparison, and accept/reject actions.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 031

## Goal
Integrate AI suggestion interface for design improvements.

## Implementation Steps
1. Trigger backend job via repository; show pending state with spinner.
2. Poll for suggestions; display cards with preview comparison slider.
3. Provide accept/reject actions updating design data and notifying backend.
4. Handle rate limiting and error states gracefully.
