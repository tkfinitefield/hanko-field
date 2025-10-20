# Web Testing Harness

This project uses Go's `httptest` for integration and unit tests.

What’s covered
- Router assembly: builds a `chi` router with the production middleware stack (session, CSRF, i18n, logging).
- SSR HTML assertions: requests `/` and checks for localized content.
- htmx interaction checks: simulates `HX-Request` and validates CSRF error JSON, and success when header + cookie are present.

Key tests
- `web/cmd/web/main_test.go`
  - `TestHealthzOK` – probes `/healthz`.
  - `TestHomeLocalizedNav_EN` – sets `Accept-Language: en` and asserts translated nav.
  - `TestHTMXPostRequiresCSRF` – end-to-end CSRF double-submit flow (cookie + header tied to session), with both failure and success paths.
  - `TestSessionMiddlewareSetsCookie` – verifies session cookie is set on first response.
- `web/internal/i18n/i18n_test.go` – ensures `Accept-Language` q-values are respected (e.g., `ja;q=0.8, en;q=0.9` selects `en`).

Running tests
```bash
cd web
go test ./...
```

Notes
- Tests run with dev-mode template parsing and use on-disk templates and locales (`../../templates`, `../../locales`).
- Session middleware sets cookies just before the first write to ensure compatibility with `httptest` and real servers.
