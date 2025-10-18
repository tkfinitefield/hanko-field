# Build linked accounts screen (`/profile/linked-accounts`) showing social auth connections and unlink flow.

**Parent Section:** 11. Profile & Settings
**Task ID:** 070

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Small top app bar` with add account `Icon button`.
- **Account cards:** `Outlined cards` listing provider, status, and `Switch` for auto sign-in.
- **Security banner:** `Banner` reminding about password hygiene.
- **Action buttons:** `Text button` for unlink and `Filled tonal button` for save.
