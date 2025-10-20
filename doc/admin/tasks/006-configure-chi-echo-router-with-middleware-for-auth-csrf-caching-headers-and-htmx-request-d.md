# Configure chi/echo router with middleware for auth, CSRF, caching headers, and htmx request detection. ✅

**Parent Section:** 1. Project & Infrastructure Setup
**Task ID:** 006

## Goal
Configure HTTP router with middleware stack supporting SSR pages and htmx fragments.

## Key Elements
- Authentication middleware verifying Firebase tokens and injecting user context.
- CSRF protection (per design, meta tag and header) with bypass for safe GET fragments.
- Request logging, panic recovery, caching headers (disallow caching on auth pages).
- htmx detection via `HX-Request` header to adjust response (partial templates, status codes).

## Steps
1. Instantiate router with route groups for pages vs fragments vs modals.
2. Apply middleware order: logging → recover → auth → rbac → csrf (for non-GET).
3. Provide helper to register fragment endpoints with shared wrappers.
4. Add tests ensuring middleware invoked and unauthorized requests redirect to login.
