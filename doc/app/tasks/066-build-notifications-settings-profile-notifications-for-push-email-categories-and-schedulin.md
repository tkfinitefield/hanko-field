# Build notifications settings (`/profile/notifications`) for push/email categories and scheduling.

**Parent Section:** 11. Profile & Settings
**Task ID:** 066

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with reset `Text button`.
- **Category list:** `List items` each containing `Switch` for channel enablement and supporting text.
- **Digest controls:** `Segmented buttons` for frequency selection (daily, weekly, monthly).
- **Footer:** `Filled tonal button` to save preferences with `Snackbar` on success.
