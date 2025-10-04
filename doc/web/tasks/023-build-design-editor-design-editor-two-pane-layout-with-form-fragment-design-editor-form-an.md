# Build design editor (`/design/editor`) two-pane layout with form fragment (`/design/editor/form`) and live preview fragment (`/design/editor/preview`).

**Parent Section:** 4. Design Creation Flow
**Task ID:** 023

## Goal
Build two-pane design editor with form and live preview.

## Implementation Steps
1. Implement form fragment `/design/editor/form` handling name input, kanji mapping link, font/template selection, style controls.
2. Implement preview fragment `/design/editor/preview` triggered by form changes (debounced) to render updated SVG/PNG.
3. Integrate modals for font/template pickers and kanji mapping; ensure accessible navigation.
4. Handle save/draft actions, AI invocation, and validation feedback.
