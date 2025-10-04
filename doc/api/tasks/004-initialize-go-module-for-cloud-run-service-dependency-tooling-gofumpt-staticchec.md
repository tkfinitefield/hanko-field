# Initialize Go module for Cloud Run service, dependency tooling (gofumpt, staticcheck, vulncheck), and Makefile/Taskfile helpers.

**Parent Section:** 1. Project & Environment Setup
**Task ID:** 004

## Scope
Bootstrap the API service for Cloud Run with consistent tooling so developers share identical linting, testing, and build workflows.

## Plan
- Create `go.mod` under `/api` with module path `github.com/hanko-field/api` (adjust if monorepo rules differ).
- Configure Go >= 1.21, enable module mode, and add basic dependencies (`chi`/`echo`, Firebase Admin SDK).
- Provide developer tooling via `Makefile`/`Taskfile` (targets: `deps`, `fmt`, `lint`, `test`, `run`, `build`, `generate`).
- Register formatting and lint tools (`gofumpt`, `golangci-lint`, `staticcheck`, `govulncheck`) using `tools.go` pattern.
- Set up `.golangci.yml` and editorconfig/VSCode settings for formatting consistency.

## Steps
1. Initialise module and vendor base dependencies.
2. Scaffold directory structure (`cmd/api`, `internal/{handlers,services,repositories,platform}`) with placeholder files.
3. Implement `main.go` wiring config loading and router startup (stub handlers for now).
4. Add convenience scripts (e.g., `make dev` to run with emulators) and documentation in `doc/api/dev_setup.md`.

## Acceptance Criteria
- `make lint` and `make test` succeed from clean checkout.
- Running `go run ./cmd/api` starts HTTP server locally using default config.
- CI can reuse the same commands without bespoke scripting.
