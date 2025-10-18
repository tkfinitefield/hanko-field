# Implement kanji mapping flow (`/design/input/kanji-map`) displaying candidate list, meanings, and selection persistence.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 028

## Goal
Implement kanji mapping flow for foreign users.

## Implementation Steps
1. Fetch candidate kanji via backend service; display meaning, pronunciation, popularity.
2. Allow multi-select/compare, bookmarking, and final selection updating design state.
3. Provide fallback manual entry and offline caching of suggestions.
4. Log analytics for selected kanji.

## Material Design 3 Components
- **Header:** `Center-aligned top app bar` with search affordance.
- **Lookup:** `Search bar` pinned below the bar feeding results.
- **Results list:** `Supporting list items` with leading glyph preview and trailing `Radio button`.
- **Context chips:** `Filter chips` to narrow by stroke count or radical category.
