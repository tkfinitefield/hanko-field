# Build addresses management (`/profile/addresses`) with CRUD, defaults, and shipping sync.

**Parent Section:** 11. Profile & Settings
**Task ID:** 064

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Small top app bar` with add address `Icon button`.
- **Address list:** `List items` with trailing `Radio button` for default and `Icon buttons` for edit/delete.
- **Sync banner:** `Banner` indicating shipping sync status.
- **Dialog:** `Standard dialog` housing `Outlined text fields` for quick edits.
