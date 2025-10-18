# Implement payment methods management (`/profile/payments`) referencing PSP tokens, default selection, and removal.

**Parent Section:** 11. Profile & Settings
**Task ID:** 065

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Small top app bar` with add payment `Icon button`.
- **Method list:** `Two-line list items` with brand icon leading and trailing `Radio button` for default.
- **Security card:** `Outlined card` summarizing security tips with `Assist chips` linking to FAQ.
- **Dialogs:** `Full-screen dialog` for card entry using `Outlined text fields` and `Segmented buttons` for billing address choice.
