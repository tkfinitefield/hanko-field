# Web Middleware Stack

This service applies a shared middleware stack across SSR pages and htmx fragments.

## Components
- Session (`HANKO_WEB_SESSION` cookie):
  - Signed (HMAC-SHA256) JSON payload storing `id`, optional `uid`, `locale`, `cart`.
  - Signing key from `HANKO_WEB_SESSION_SIGNING_KEY` (Secret Manager in prod).
  - `HttpOnly`, `SameSite=Lax`, `Secure` in prod.

- CSRF (`csrf_token` cookie):
  - Non-HttpOnly cookie so client can read; htmx appends `X-CSRF-Token` via a small script.
  - All non-GET/HEAD/OPTIONS requests from browsers must provide a matching header.
  - Programmatic `Authorization: Bearer ...` requests are exempt (API-to-API).

- HTMX detection:
  - Marks requests with `HX-Request: true` in context for targeted responses.

- Structured Logging:
  - JSON per request with `method`, `path`, `status`, `duration_ms`, `request_id`, `user_id`, `htmx`.

- Static Assets Caching:
  - `/assets/*` served with `Cache-Control: public, max-age=604800, stale-while-revalidate=86400`.
  - Weak `ETag` computed at startup for cache validation; `Vary: Accept-Encoding` set.

- Locale Vary:
  - Adds `Vary: Accept-Language` to dynamic responses for downstream caches.

## Configuration
- `HANKO_WEB_SESSION_SIGNING_KEY`: HMAC key (required in prod).
- `HANKO_WEB_ENV=prod`: Enables secure cookies.

## Notes
- The auth user (`UserID`) is currently sourced from the session. Integrate Firebase Auth verification to populate it at login flow as needed.

