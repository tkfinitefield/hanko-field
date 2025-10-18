# Implement account profile page (`/account`) with profile form fragment and update flow.

**Parent Section:** 6. Account & Library
**Task ID:** 035

## Goal
Implement account profile page with editable fields.

## Implementation Steps
1. Render profile form fragment with display name, language, country.
2. Submit updates via htmx to `/account/profile/form` and display validation errors.
3. Update session context on success.

## UI Components
- **Layout:** `AccountLayout` with sidebar `AccountNav` and breadcrumb header.
- **Profile form:** `ProfileForm` (name, email, phone) using `Input`, `PhoneInput`, `AvatarUploader`.
- **Security banner:** `InlineAlert` reminding to enable 2FA.
- **Preference toggles:** `SettingsList` with `Switch` components for communication preferences.
- **Session table:** `SessionTable` listing active sessions with revoke `LinkButton`.
- **Action bar:** Sticky `ActionBar` for save/discard with success `Toast`.
