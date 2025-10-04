# Implement Firebase ID token verification, role extraction, and authentication middleware for user/staff separation.

**Parent Section:** 2. Core Platform Services
**Task ID:** 009

## Goal
Authenticate Firebase users/staff and inject identity + roles into context for downstream handlers.

## Design
- Initialise Firebase Admin SDK once; cache JWKS certificates.
- Middleware `RequireFirebaseAuth(allowedRoles ...)` extracts `Authorization: Bearer` header, validates token, and loads custom claims (`role`, `locale`).
- On success attach `Identity{UID, Email, Roles, Locale}` to context.
- On failure respond 401 with standardized error structure.

## Steps
1. Implement verifier using `auth.Client.VerifyIDToken` with caching and network timeout.
2. Map Firebase custom claims to internal roles (user, staff, admin) and add fallback if claim missing.
3. Write helper to fetch user profile lazily when required.
4. Add tests covering valid token, expired token, missing role.
