# Build style selection (`/design/style`) with preview carousel, filtering by script/shape, and template fetching.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 029

## Goal
Enable user to choose script and template style.

## Implementation Steps
1. Fetch template metadata, filter by persona and available fonts.
2. Provide preview carousel with script toggles and shape filters.
3. Update selection in view model and prefetch assets for editor.

## Material Design 3 Components
- **App bar:** `Medium top app bar` with help `Icon button` for typography guidance.
- **Script filters:** `Segmented buttons` for script family (kanji/kana/roman).
- **Preview rail:** Horizontal `Elevated cards` displaying style thumbnails and metadata.
- **Secondary controls:** `Filter chips` for shape/material and `Assist chips` for favorites.
