# Create login/logout flows (`/admin/login`, `/admin/logout`) with error states and redirect logic.

**Parent Section:** 2. Authentication, Authorization, and Session Management
**Task ID:** 013

## Goal
Implement admin login page and logout workflow with error handling.

## Implementation Steps
1. Build `/admin/login` page (Go template) with email/password or federated login instructions.
2. Integrate Firebase client or custom auth API to exchange credentials for ID token.
3. Set session cookie, redirect to last visited page or dashboard.
4. Handle auth errors (invalid credentials, disabled account) with inline messaging.
5. Provide `/admin/logout` route clearing session and redirecting to login with confirmation.
