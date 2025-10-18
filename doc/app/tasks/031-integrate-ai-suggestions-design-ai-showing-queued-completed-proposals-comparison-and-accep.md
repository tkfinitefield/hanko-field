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

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with status `Badge` indicating queue length.
- **Status tabs:** `Segmented buttons` for Queued, Ready, and Applied suggestions.
- **Proposal cards:** `Elevated cards` pairing preview thumbnails with diff summaries and `Assist chips` for tags.
- **Action row:** `Filled button` to accept and `Outlined button` to reject with `Snackbar` feedback.
