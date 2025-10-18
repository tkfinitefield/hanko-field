# Build AI suggestions page (`/design/ai`) with table fragment, accept/reject actions, and polling.

**Parent Section:** 4. Design Creation Flow
**Task ID:** 025

## Goal
Implement AI suggestions gallery.

## Implementation Steps
1. Display grid of AI suggestion cards with score tags and diff highlights.
2. Use fragment `/design/ai/table` to poll for new suggestions.
3. Provide accept/reject buttons posting to API; refresh preview on accept.
4. Handle error states and show queue status.

## UI Components
- **Layout:** `SiteLayout` with `SectionHeader` summarizing AI queue and poll toggle.
- **Status filters:** `FilterToolbar` hosting status `SegmentedControl`, persona `Select`, sort `Dropdown`.
- **Suggestion table:** `SuggestionTable` (htmx fragment) listing prompt summary, preview thumbnail, score, actions.
- **Preview drawer:** `SuggestionPreview` sliding panel with diff view and accept/reject `ButtonGroup`.
- **Polling indicator:** `InlineNotice` showing next refresh countdown.
- **Analytics strip:** `StatsBar` summarizing adoption and success metrics.
