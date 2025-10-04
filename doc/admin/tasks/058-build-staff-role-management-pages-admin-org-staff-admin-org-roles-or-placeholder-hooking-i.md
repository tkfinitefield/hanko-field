# Build staff/role management pages (`/admin/org/staff`, `/admin/org/roles`) or placeholder hooking into Firebase Console if deferred; document admin-only access.

**Parent Section:** 10. Production Queues & Org Management
**Task ID:** 058

## Goal
Provide UI for managing staff accounts and roles (or document deferment).

## Implementation Steps
1. If backend ready, implement list with invite/remove actions.
2. Show role assignments, last login, 2FA status.
3. Provide modal to change roles using RBAC mapping.
4. If backend not ready, add placeholder page noting management in Firebase console and track TODO.
