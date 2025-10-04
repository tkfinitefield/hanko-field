# Implement `/me` GET/PUT respecting editable fields and audit logging changes.

**Parent Section:** 5. Authenticated User Endpoints > 5.1 Profile & Account
**Task ID:** 035

## Purpose
Allow authenticated users to retrieve and update profile attributes while enforcing field-level permissions.

## Endpoints
- `GET /me`: returns profile document merged with Firebase identity info.
- `PUT /me`: accepts editable fields (`displayName`, `preferredLanguage`, `notificationPrefs`, `avatarAssetId`).

## Implementation Steps
1. Fetch user profile via `UserService`; if absent, seed from Firebase user record.
2. Define update DTO validating allowed fields and rejecting attempts to change `role`, `isActive`, `piiMasked`.
3. Persist updates through `UserService.UpdateProfile` with audit logging.
4. Return response containing derived fields (e.g., `hasPassword`, `onboardedAt`).
5. Add tests ensuring unauthorized fields ignored and audit entry generated.
