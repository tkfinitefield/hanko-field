# Build account delete flow (`/profile/delete`) with confirmation steps and backend call.

**Parent Section:** 11. Profile & Settings
**Task ID:** 072

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Medium top app bar` with prominent danger color tokens.
- **Warning card:** `Outlined card` with iconography and `BodyLarge` copy.
- **Acknowledgement list:** `List items` containing `Checkbox` for policy confirmations.
- **CTA:** `Filled button` styled with `errorContainer` colors and secondary `Text button` to cancel.
