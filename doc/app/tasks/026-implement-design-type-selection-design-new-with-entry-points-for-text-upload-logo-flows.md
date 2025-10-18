# Implement design type selection (`/design/new`) with entry points for text/upload/logo flows.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 026

## Goal
Provide entry screen for selecting design creation mode.

## Implementation Steps
1. Display options (text input, image upload, logo engraving) with contextual details.
2. Validate prerequisites (storage permissions) before navigating.
3. Track analytics event for chosen mode.
4. Persist selection in creation view model.

## Material Design 3 Components
- **Header:** `Medium top app bar` reinforcing the flow title with contextual help `Icon button`.
- **Options grid:** `Elevated cards` representing text/upload/logo flows with illustration and body copy.
- **Quick filters:** `Filter chips` for popular use cases pinned above the grid.
- **Primary action:** `Extended FAB` to advance into the selected creation pathway.
