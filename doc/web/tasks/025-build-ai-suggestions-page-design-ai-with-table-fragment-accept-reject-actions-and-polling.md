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
