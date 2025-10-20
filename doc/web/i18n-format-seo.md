# i18n, Formatting, and SEO Utilities

## i18n
- Dictionaries live in `web/locales/*.json` (e.g., `ja.json`, `en.json`).
- Load at startup; fallback is `ja`. Supported locales can be set via `HANKO_WEB_LOCALES` (comma-separated).
- Middleware sets the language in session (`Session.Locale`) via `hl` query param, `hl` cookie, or `Accept-Language`.
- Templates use `{{ tlang $.Lang "key" }}` to render translations.

## Formatting
- Template helpers:
  - `fmtDate time lang` → locale-friendly date (e.g., `2006-01-02` for `ja`).
  - `fmtMoney amountMinor currency lang` → currencies like `¥12,345` for JPY.

## SEO
- View models may include an `SEO` field with `Title`, `Description`, `Canonical`, and nested `OG`/`Twitter` fields.
- The partial `templates/partials/head.tmpl` consumes `.SEO` when present and falls back to defaults.

## Example
- Update a handler to set `Lang` and `SEO`:
  - See `web/cmd/web/main.go: HomeHandler` and `web/internal/handlers/home.go: BuildHomeData`.
