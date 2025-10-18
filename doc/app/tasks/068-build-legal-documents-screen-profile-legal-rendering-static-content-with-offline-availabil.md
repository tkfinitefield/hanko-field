# Build legal documents screen (`/profile/legal`) rendering static content with offline availability.

**Parent Section:** 11. Profile & Settings
**Task ID:** 068

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with download `Icon button`.
- **Document list:** `List items` leading icons for document type and trailing `Assist chip` for version.
- **Content viewer:** `Outlined card` surface rendering markdown/HTML with scroll.
- **Footer:** `Text button` for open in browser.
