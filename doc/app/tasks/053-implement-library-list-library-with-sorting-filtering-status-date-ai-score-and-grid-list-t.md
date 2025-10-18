# Implement library list (`/library`) with sorting, filtering (status, date, AI score), and grid/list toggle.

**Parent Section:** 9. My Hanko Library
**Task ID:** 053

## Goal
Implement library list with sorting and filtering.

## Implementation Steps
1. Fetch designs from backend with pagination and filter parameters.
2. Provide sorting options (recent, AI score, name) and filter chips (status, persona).
3. Support grid/list toggle and quick actions (preview, share).
4. Implement offline caching for quick access.

## Material Design 3 Components
- **App bar:** `Small top app bar` with inline `Search bar`.
- **Filters:** `Filter chips` and `Segmented buttons` to switch grid/list layouts.
- **Grid layout:** `Elevated cards` with hero preview, metadata, and status `Assist chips`.
- **Empty state:** Centered `Outlined card` with illustration and `Filled tonal button` to create new design.
