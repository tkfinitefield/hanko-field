# Implement support screen (`/profile/support`) linking to FAQ, chat, contact forms.

**Parent Section:** 11. Profile & Settings
**Task ID:** 069

## Goal
Implement profile home summarizing account info and quick links.

## Implementation Steps
1. Display avatar, display name, persona toggle, membership status.
2. Provide shortcuts to addresses, payments, notifications, support.
3. Fetch data via profile provider with optimistic updates.

## Material Design 3 Components
- **App bar:** `Large top app bar` with search `Icon button` for support content.
- **Support grid:** `Elevated cards` for FAQ, Chat, Call, each with `Assist chip` for availability.
- **Recent tickets:** `List items` with status `Assist chips`.
- **CTA:** `Filled tonal button` for create ticket.
