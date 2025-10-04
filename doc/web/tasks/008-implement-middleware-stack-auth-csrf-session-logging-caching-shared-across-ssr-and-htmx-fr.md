# Implement middleware stack (auth, CSRF, session, logging, caching) shared across SSR and htmx fragments.

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 008

## Goal
Implement middleware for auth, CSRF, sessions, logging, caching.

## Implementation Steps
1. Authentication: verify Firebase tokens or session cookies, inject user context.
2. CSRF: set meta tag + header usage for htmx POST/PUT/DELETE requests.
3. Sessions: store user preferences, cart IDs using secure cookies.
4. Logging: structured logs, request IDs, error handling returning JSON for fragments.
5. Caching: ETag/Cache-Control for static content; vary by locale.
