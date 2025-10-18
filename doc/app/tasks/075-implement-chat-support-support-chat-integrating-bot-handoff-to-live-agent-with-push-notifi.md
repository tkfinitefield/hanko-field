# Implement chat support (`/support/chat`) integrating bot handoff to live agent with push notifications.

**Parent Section:** 12. Support & Status
**Task ID:** 075

## Goal
Integrate chat support with bot-to-human escalation.

## Implementation Steps
1. Implement chat UI with message bubbles, typing indicators, and history persistence.
2. Integrate with chat backend (bot -> live agent) via WebSocket/Firebase.
3. Handle push notifications for new messages.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` showing agent presence via `Assist chip`.
- **Conversation list:** `List items` styled as `Surface` bubbles with opposing alignment.
- **Composer:** `Bottom app bar` with `Outlined text field`, attachment `Icon button`, and `Filled tonal button` for send.
- **System alerts:** `Banner` for connection loss and `Snackbar` for retries.
