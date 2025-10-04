# Build session middleware storing user info, CSRF tokens, and feature flags; add remember-me support if required.

**Parent Section:** 2. Authentication, Authorization, and Session Management
**Task ID:** 011

## Goal
Manage server-side session data for admin UI (user profile, CSRF, feature flags).

## Implementation Steps
1. Choose session storage (secure cookie vs Firestore/Redis). For sensitive data prefer encrypted cookies.
2. On login, store user info, CSRF token, last active timestamp. Provide `SessionManager` abstraction.
3. Implement remember-me by extending session expiry and storing refresh token if policy allows.
4. Ensure logout clears session server-side and cookie invalidated.
5. Add idle timeout handling (auto logout after inactivity).
