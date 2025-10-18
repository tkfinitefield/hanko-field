# Build security/linked accounts page (`/account/security`) covering auth providers and 2FA prompts.

**Parent Section:** 6. Account & Library
**Task ID:** 040

## Goal
Build security page for linked accounts/2FA.

## Implementation Steps
1. Display linked providers with status; provide link/unlink modals.
2. Surface 2FA status and setup instructions.
3. Integrate with backend for token revocation.

## UI Components
- **Layout:** `AccountLayout` with security `SectionHeader`.
- **Provider list:** `ProviderCard` stack showing providers with connect `Button` and status `Badge`.
- **2FA setup:** `TwoFactorCard` containing QR code modal trigger and backup code list.
- **Password card:** `PasswordCard` with last updated info and change `Button`.
- **Sessions:** `SessionTable` for device logins.
- **Alert strip:** `AlertBanner` for suspicious activity warnings.
