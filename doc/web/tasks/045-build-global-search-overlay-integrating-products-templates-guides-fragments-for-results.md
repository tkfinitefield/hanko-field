# Build global search overlay integrating products, templates, guides (fragments for results).

**Parent Section:** 8. Notifications, Search, and Utilities
**Task ID:** 045

## Goal
Build global search overlay combining products/templates/guides.

## Implementation Steps
1. Implement overlay triggered by search icon or keyboard shortcut.
2. Call search endpoints for each category; render results via fragments.
3. Provide keyboard navigation and highlight matches.

## UI Components
- **Container:** `SearchOverlay` full-screen dialog triggered from header.
- **Search bar:** `OverlaySearchBar` with inline icon, voice button, close control.
- **Result tabs:** `SegmentedTabs` for Products, Templates, Guides, Accounts, each loading fragment lists.
- **Result list:** `SearchResultList` using `ResultItem` entries with thumbnail, metadata, action.
- **Recent/pinned:** `SearchHistoryPanel` for recent queries and pinned items.
- **Footer shortcuts:** `ShortcutRow` documenting keyboard commands.
