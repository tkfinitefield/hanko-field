# Implement versions view (`/library/:designId/versions`) reusing diff/rollback components.

**Parent Section:** 9. My Hanko Library
**Task ID:** 055

## Goal
Display version history for saved designs.

## Implementation Steps
1. Reuse versions UI from creation flow with timeline/diff features.
2. Allow rollback and show annotations for reasons.

## Material Design 3 Components
- **Top bar:** `Small top app bar` with compare toggle `Icon button`.
- **Version list:** `List items` with trailing `Radio button` for selection and `Assist chips` for status.
- **Diff display:** `Outlined cards` side-by-side with preview thumbnails.
- **Actions:** `Filled tonal button` to restore and `Outlined button` to export snapshot.
