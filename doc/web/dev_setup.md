# Web Dev Setup

This guide sets up the Go web module with chi router, html/template rendering, Tailwind (standalone CLI), local htmx asset, and dev tooling.

## Prerequisites
- Go 1.23+
- Optional: Air for hot reload (`go install github.com/cosmtrek/air@latest`)

No Node.js/npm is required. TailwindCSS is built using its standalone CLI binary, and htmx is fetched once into `public/assets/js`.

## Running the dev server
Two terminals recommended: one for CSS watch, one for the Go server.

Terminal A (Tailwind watch/build):
```bash
cd web
# Download tailwindcss standalone (once), put it at web/tools/tailwindcss and make it executable.
# See: https://github.com/tailwindlabs/tailwindcss/releases
# macOS (Apple Silicon example):
#   curl -L -o tools/tailwindcss https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-macos-arm64
#   chmod +x tools/tailwindcss

make css-watch
```

Terminal B (Go server):
```bash
cd web
make dev        # uses air if available; else falls back to go run
# or
make run
```

Then open http://localhost:8080

Environment variables:
- `HANKO_WEB_PORT`: listen port (fallback to Cloud Run `PORT`), default 8080
- `HANKO_WEB_DEV=1`: enable template re-parse on each request
- `HANKO_WEB_ENV`: environment name (e.g., `dev`, `staging`, `prod`)

## Useful commands
```bash
cd web
make htmx       # download htmx.min.js into public/assets/js
make css        # one-shot Tailwind build to public/assets/app.css
make css-watch  # watch mode; rebuild on template/CSS changes
make build      # build Go binary to web/bin
make test       # run Go tests
make tidy       # go mod tidy
```

## Structure
- `web/cmd/web`: main entry
- `web/templates`: layouts, pages, partials for html/template
- `web/public/assets`: output CSS/JS (served at `/assets/...`)
- `web/assets/css/input.css`: Tailwind source
- TailwindCSS is compiled from `assets/css/input.css` to `public/assets/app.css`.
- The standalone binary path is `web/tools/tailwindcss` (ignored by Git). Place the downloaded binary there and `chmod +x` it.
- Run `make htmx` once to fetch `public/assets/js/htmx.min.js` locally; the base layout references `/assets/js/htmx.min.js`.

## Notes
- This scaffold uses `html/template`. If/when migrating to `templ`, maintain the same directory structure and route organization.
