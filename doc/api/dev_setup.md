# API Developer Setup

These steps prepare the Go Cloud Run service for local development.

## Prerequisites

- Go 1.21 or later (install via `goenv`, `asdf`, or the official installer).
- Taskfile (`brew install go-task/tap/go-task`) or GNU Make.
- Firestore/Firebase emulators (optional for local integration tests).

## Bootstrap Commands

```bash
cd api
make deps    # install gofumpt, golangci-lint, staticcheck, govulncheck
make fmt     # format the codebase
make lint    # run golangci-lint + staticcheck
make test    # execute unit tests
make run     # start the HTTP server on http://localhost:8080
```

### Alternative (Taskfile)

```bash
cd api
task deps
task run
```

## Local Server

`make run` honours the `PORT` environment variable. By default the server listens on `:8080` and exposes:

- `GET /healthz` → basic health status payload.

Use `CTRL+C` to stop; the server performs a graceful shutdown with a 10s timeout.

## Tooling Notes

- Formatting: enforced by `gofumpt` (extra rules enabled). Editors should respect `.editorconfig`.
- Linting: `golangci-lint` plus `staticcheck` and `govulncheck` for security scanning.
- Dependencies: `make tidy` wraps `go mod tidy` to keep `go.mod` / `go.sum` synchronized.

## CI Integration

CI can reuse `make deps`, `make lint`, and `make test` steps. `make build` produces a binary at `api/bin/hanko-api` suitable for container packaging.
