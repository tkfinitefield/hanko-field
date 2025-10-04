# Implement profile/account page (`/admin/profile`) with 2FA setup, password/API key management, and session history.

**Parent Section:** 4. Shared Utilities & System Pages
**Task ID:** 024

## Goal
Provide admin profile page for personal settings.

## Implementation Steps
1. Display user information (name, email, role) from session context.
2. Provide forms for changing password (if applicable), enabling MFA, generating API keys.
3. Show active sessions/devices with revoke button using htmx.
4. Integrate with backend endpoints for each action and display success/failure toasts.
5. Document security guidance on page.
