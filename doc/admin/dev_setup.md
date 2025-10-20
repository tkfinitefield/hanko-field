# Admin Dev Setup

## Prerequisites

- Go 1.23+
- `curl` (used to fetch Tailwind standalone binary)
- `air` live reload (`go install github.com/air-verse/air@latest`)
- `templ` generator (`go install github.com/a-h/templ/cmd/templ@latest`)

Ensure `$GOPATH/bin` (where `air` and `templ` are installed) is on your `PATH`.

## Initial Setup

```bash
cd admin
make ensure-tailwind
```

This downloads the Tailwind standalone executable into `admin/bin/`.

`make ensure-tailwind` is idempotent and re-downloads the binary when `TAILWIND_VERSION` changes.

## Common Tasks

- `make dev` – runs `templ` generation, `go mod tidy`, starts Tailwind watcher, then launches `air`.
- `make templ` – regenerate templ components after editing `.templ`.
- `make css` – build a minified Tailwind bundle at `public/static/app.css`.
- `make css-watch` – run Tailwind in watch mode only.
- `make lint` – `gofmt` and `go vet`.
- `make test-ui` – execute httptest-based integration smoke tests (see `internal/admin/httpserver/server_integration_test.go`).

Go build cache is redirected to `.gocache` to remain within the repo sandbox. Static assets are embedded from `public/static`.

## Configuration

- `ADMIN_HTTP_ADDR` (default `:8080`) controls the listen address.
- `ADMIN_BASE_PATH` (default `/admin`) sets the mount point for all admin routes.
- Attach an `Authorization: Bearer <token>` header (any non-empty token accepted by the default authenticator) when exploring authenticated routes locally. Browsers without a token will be redirected to `/admin/login`.
- Set `FIREBASE_PROJECT_ID` together with `GOOGLE_APPLICATION_CREDENTIALS` to enable Firebase ID token verification. When using the Firebase Auth emulator, also provide `FIREBASE_AUTH_EMULATOR_HOST`.

## Notes

- Generated `*_templ.go` files are committed. Run `make templ` whenever templates change.
- `.air.toml` controls hot reload for `.go` and `.templ`.
- Tailwind scans `.templ` and generated component files (`tailwind.config.js` content globs).
