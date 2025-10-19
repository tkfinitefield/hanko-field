# Web Dev Setup

This guide sets up the Go web module with chi router, html/template rendering, Tailwind (CDN), and dev tooling.

## Prerequisites
- Go 1.23+
- Optional: Air for hot reload (`go install github.com/cosmtrek/air@latest`)

No Node.js/npm is required. TailwindCSS and htmx are loaded via CDN.

## Running the dev server
Two terminals recommended: one for CSS watch, one for the Go server.

Terminal (Go server):
```bash
cd web
make dev        # uses air if available; else falls back to go run
# or
make run
```

Then open http://localhost:8080

Environment variables:
- `PORT`: listen port (default 8080)
- `DEV=1`: enable template re-parse on each request

## Useful commands
```bash
cd web
make build      # build Go binary to web/bin
make test       # run Go tests
make tidy       # go mod tidy
```

## Structure
- `web/cmd/web`: main entry
- `web/templates`: layouts, pages, partials for html/template
- `web/public/assets`: output CSS/JS (served at `/assets/...`)
- `web/assets/css/input.css`: Tailwind source
- TailwindCSS is included via `<script src="https://cdn.tailwindcss.com"></script>` in the base layout.

## Notes
- htmx is provided via npm (`htmx.org`) and copied to `public/assets/js/htmx.min.js` by `npm run assets:copy`.
- This scaffold uses `html/template`. If/when migrating to `templ`, maintain the same directory structure and route organization.
