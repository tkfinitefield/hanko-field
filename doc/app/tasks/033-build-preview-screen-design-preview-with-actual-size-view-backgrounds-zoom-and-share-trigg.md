# Build preview screen (`/design/preview`) with actual size view, backgrounds, zoom, and share triggers.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 033

## Goal
Offer final preview with actual size and background options.

## Implementation Steps
1. Render preview at true-to-size with measurement overlay and pinch-to-zoom.
2. Provide background toggles (paper, wood, transparent) and lighting effects.
3. Expose share/export buttons and confirm design ready for ordering.

## Material Design 3 Components
- **Top bar:** `Medium top app bar` with share and edit `Icon buttons`.
- **Preview area:** Center `Surface` framed by `Outlined card` supporting pinch gestures.
- **Background selector:** `Segmented buttons` for background colors and `Assist chips` for material textures.
- **Actions:** Bottom `Filled tonal button` for export and `Outlined button` to reopen editor.
