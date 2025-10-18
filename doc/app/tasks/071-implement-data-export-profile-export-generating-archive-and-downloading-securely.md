# Implement data export (`/profile/export`) generating archive and downloading securely.

**Parent Section:** 11. Profile & Settings
**Task ID:** 071

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with status `Badge` for last export.
- **Summary card:** `Elevated card` describing export contents with supporting text.
- **Preference list:** `List items` with `Switches` for including assets, orders, history.
- **CTA:** `Filled button` to start export and `Text button` to view previous archives.
