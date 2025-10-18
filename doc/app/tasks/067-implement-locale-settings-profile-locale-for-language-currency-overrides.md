# Implement locale settings (`/profile/locale`) for language/currency overrides.

**Parent Section:** 11. Profile & Settings
**Task ID:** 067

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Small top app bar` with help `Icon button`.
- **Language picker:** `List items` paired with `Radio buttons` for locale selection.
- **Currency toggle:** `Segmented buttons` or `Assist chips` for currency override.
- **Confirmation:** `Filled button` to apply changes and trigger `Snackbar`.
