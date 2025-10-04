# Implement design editor (`/design/editor`) with canvas controls (layout, stroke, margins, rotation, grid) using custom painter widgets.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 030

## Goal
Build interactive design editor with fine-grained controls.

## Implementation Steps
1. Implement canvas using `CustomPainter`, parameterized by view model state.
2. Provide controls for layout (alignment), stroke width, margins, rotation, grid overlay.
3. Support undo/redo, reset to template, and live preview updates.
4. Auto-save edits and handle device orientation/resizing.
