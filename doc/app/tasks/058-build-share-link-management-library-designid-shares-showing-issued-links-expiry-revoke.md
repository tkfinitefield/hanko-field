# Build share link management (`/library/:designId/shares`) showing issued links, expiry, revoke.

**Parent Section:** 9. My Hanko Library
**Task ID:** 058

## Goal
Manage share links with expiry and revoke options.

## Implementation Steps
1. List existing share links with expiry, usage stats.
2. Provide actions to extend, revoke, or create new link.
3. Display copy/share button with analytics tracking.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with create link `Icon button`.
- **Active links:** `List items` with trailing `Assist chips` for expiry and `Icon button` for revoke.
- **History:** `Outlined card` summarizing expired links with `Text button` to view more.
- **Footer:** `Filled button` for new link generation.
