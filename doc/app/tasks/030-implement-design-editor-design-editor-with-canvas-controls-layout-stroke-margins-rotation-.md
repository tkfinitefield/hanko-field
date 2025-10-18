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

## Material Design 3 Components
- **Top chrome:** `Medium top app bar` with undo/redo `Icon buttons` and overflow menu.
- **Tool rail:** Left-side `Navigation rail` hosting tool icons (select, text, layout, export).
- **Canvas:** Central `Surface` framed by `Outlined card` to present live preview with grid overlay.
- **Property sheet:** Right-side `Modal side sheet` exposing sliders, `Segmented buttons`, and `Switches` for settings.
- **Primary action:** Bottom `Extended FAB` for preview/export entry.
