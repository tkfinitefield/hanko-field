# Implement notifications list (`/notifications`) with pagination, read/unread state, and push navigation handling.

**Parent Section:** 4. Home & Discovery
**Task ID:** 025

## Goal
Build notifications list with read/unread support.

## Implementation Steps
1. Retrieve notifications via repository with pagination.
2. Display grouped by date with category icons and CTA buttons.
3. Provide mark-as-read actions (single, bulk) updating backend and local state.
4. Handle navigation from push notifications deep linking to content.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with bulk-action `Overflow menu`.
- **Filter strip:** `Segmented buttons` to toggle All vs Unread states.
- **Content:** `Two-line list items` with leading status `Icon` and trailing `Assist chip` for type.
- **Feedback:** Swipe actions trigger `Snackbar` with undo for accidental dismiss.
