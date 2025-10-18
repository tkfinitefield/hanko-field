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

## UI Components
- **Shell:** `EditorLayout` using full-height `TwoPane` structure with collapsible nav.
- **Left pane:** `FormPane` rendering `/design/editor/form` fragment with field `FieldGroup`, `Select`, `Slider` controls.
- **Canvas pane:** `PreviewPane` embedding `/design/editor/preview` iframe/canvas with toolbar `IconButton` cluster.
- **Toolbar:** `EditorTopBar` holding breadcrumb, autosave `StatusBadge`, undo/redo buttons.
- **Action rail:** Right `ActionRail` for share/export, AI suggestions, help.
- **Toast area:** `ToastStack` for save/validation feedback.
