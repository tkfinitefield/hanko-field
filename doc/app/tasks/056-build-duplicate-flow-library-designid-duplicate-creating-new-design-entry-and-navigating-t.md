# Build duplicate flow (`/library/:designId/duplicate`) creating new design entry and navigating to editor.

**Parent Section:** 9. My Hanko Library
**Task ID:** 056

## Goal
Create duplicate design and navigate to editor.

## Implementation Steps
1. Call backend duplication endpoint; create new design entry.
2. Navigate to `/design/editor` with new design ID, preloading assets.
3. Notify user of success/failure.
