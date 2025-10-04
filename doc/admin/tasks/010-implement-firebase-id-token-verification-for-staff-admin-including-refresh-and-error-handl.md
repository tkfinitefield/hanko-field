# Implement Firebase ID token verification for staff/admin, including refresh and error handling UX.

**Parent Section:** 2. Authentication, Authorization, and Session Management
**Task ID:** 010

## Goal
Authenticate staff/admin users via Firebase ID tokens for each request.

## Implementation Steps
1. Integrate Firebase Admin SDK to verify ID tokens (cached JWKS) and extract custom claims (`role`).
2. Implement middleware to read `Authorization` cookie/header, validate, and inject `UserContext`.
3. Handle token expiration by redirecting to login or initiating silent refresh via JS if available.
4. Log auth failures with reason codes for audit.
5. Add tests using stubbed Firebase verifier.
