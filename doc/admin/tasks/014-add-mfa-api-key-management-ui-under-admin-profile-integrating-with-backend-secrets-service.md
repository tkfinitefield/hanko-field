# Add MFA/API-key management UI under `/admin/profile`, integrating with backend secrets service.

**Parent Section:** 2. Authentication, Authorization, and Session Management
**Task ID:** 014

## Goal
Provide profile page for managing MFA and API keys per staff user.

## Implementation Steps
1. Retrieve current user info via API (phone, MFA status, API keys list).
2. Render forms for enabling MFA, rotating keys, revoking sessions; use modals with confirmation.
3. Integrate with backend endpoints for key creation/deletion and MFA enrollment (TOTP/Email).
4. Display session history with ability to revoke.
5. Include security warnings and documentation links.
