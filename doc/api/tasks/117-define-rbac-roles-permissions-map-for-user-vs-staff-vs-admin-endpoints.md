# Define RBAC roles/permissions map for user vs staff vs admin endpoints.

**Parent Section:** 11. Security & Compliance
**Task ID:** 117

## Goal
Define role-based access control matrix for user, staff, admin endpoints and document enforcement strategy.

## Plan
- Enumerate roles (`user`, `staff`, `admin`, `system`) and permissions per endpoint group.
- Document in `doc/api/security/rbac.md` and expose as configuration in code.
- Implement automated tests verifying middleware denies unauthorized access.
- Align with Firebase custom claims and admin UI roles.
