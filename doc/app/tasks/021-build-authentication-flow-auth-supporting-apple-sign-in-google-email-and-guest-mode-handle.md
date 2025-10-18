# Build authentication flow (`/auth`) supporting Apple Sign-In, Google, Email, and guest mode; handle link with Firebase Auth.

**Parent Section:** 3. Onboarding & Auth Flow
**Task ID:** 021

## Goal
Implement authentication screen supporting Apple, Google, Email, guest mode.

## Implementation Steps
1. Provide branded login UI with provider buttons and guest option.
2. Integrate Firebase Auth for each provider; handle linking and error messaging.
3. On success, fetch user profile and update session provider.
4. Support guest mode with limited capabilities and prompt to upgrade later.

## Material Design 3 Components
- **Top area:** `Large top app bar` with branded logo and help `Icon button`.
- **Credential fields:** `Outlined text fields` for email/password with `Supporting text` for validation.
- **Federated options:** `Filled tonal buttons` with provider icons laid out as a vertical stack.
- **Alt paths:** `Text button` for guest mode and `Snackbar` for auth errors and retry.
