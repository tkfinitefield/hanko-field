# Implement design preview page (`/design/preview`) with background options fragment (`/design/preview/image`).

**Parent Section:** 4. Design Creation Flow
**Task ID:** 026

## Goal
Render final design preview with background options.

## Implementation Steps
1. Provide controls for background (washi, wood, transparent) and DPI; update preview fragment accordingly.
2. Offer download buttons generating signed URLs for PNG/SVG.
3. Display measurement overlay and share actions.

## UI Components
- **Layout:** `EditorLayout` with simplified header containing back `IconButton` and export actions.
- **Preview area:** `MockupViewport` enabling zoom, pan, and responsive frame toggles.
- **Background controls:** `ControlPanel` housing background-color `SwatchPicker`, material `ChipGroup`, grid toggle `Switch`.
- **Metadata bar:** `InfoBar` showing version, last saved, owner.
- **Action footer:** `ActionBar` with download/share buttons and `SnackbarHost`.
- **Inline tips:** `Tooltip` or `Coachmark` to explain gestures.
