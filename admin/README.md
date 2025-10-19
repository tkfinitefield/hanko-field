# Hanko Admin (Go + templ Scaffold)

## Prerequisites

- Go 1.23+
- `curl` available for downloading the Tailwind standalone binary
- Go tooling:
  - `go install github.com/air-verse/air@latest`
  - `go install github.com/a-h/templ/cmd/templ@latest`

`air` and `templ` should be on your `PATH` (typically `$GOPATH/bin`).

## Setup

```bash
cd admin
make ensure-tailwind   # downloads tailwindcss standalone binary
```

## Common Commands

- `make dev` – run templ generation, tidy modules, start Tailwind watcher, then launch `air`.
- `make templ` – regenerate `templ` components.
- `make css` – single build of Tailwind output (minified) to `public/static/app.css`.
- `make css-watch` – Tailwind watch mode without starting the Go server.
- `make lint` – `gofmt` and `go vet`.

`air` watches `*.go` and `*.templ` files (configured via `.air.toml`). Tailwind scans the paths listed in `tailwind.config.js`.

## Configuration

Environment variables:

- `ADMIN_HTTP_ADDR` – bind address (default `:8080`)
- `ADMIN_BASE_PATH` – mount point for the admin UI (default `/admin`)

Run `make ensure-tailwind` after changing `TAILWIND_VERSION` in the `Makefile`; the rule verifies the installed binary matches the requested version and re-downloads if needed.

## Layout

- `cmd/admin` – entrypoint.
- `internal/admin/httpserver` – chi router + handlers.
- `internal/admin/templates` – templ components organised by feature.
- `public/static` – compiled CSS/JS assets served by Go via `embed`.
- `web/styles` – Tailwind source files.

Generated `*_templ.go` files are committed to keep `go build ./...` working without extra steps. Regenerate after editing `.templ` files.
