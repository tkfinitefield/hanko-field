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

---

## Initialization Summary (2025-04-01)

### Deliverables
- ✅ Established Go module with Cloud Run-ready entrypoint; `api/go.mod` now uses `github.com/hanko-field/api` on Go 1.21 and `api/cmd/api/main.go` provides graceful HTTP server bootstrapping with `/healthz` route via internal handlers.
- ✅ Added lightweight config loader and router/health handlers (`api/internal/platform/config/config.go`, `api/internal/handlers/router.go`, `api/internal/handlers/health.go`) to keep transport and configuration concerns separated.
- ✅ Provisioned developer tooling (`api/Makefile`, `api/Taskfile.yml`, `api/.golangci.yml`, `api/tools/tools.go`, root `.editorconfig`) covering dependencies, formatting (gofumpt), linting (golangci-lint + staticcheck), testing, builds, and vulnerability checks.
- ✅ Documented workflow in `doc/api/dev_setup.md` so contributors can bootstrap via Make or Taskfile.
- ✅ Updated repository ignore rules (`.gitignore`) to exclude build artefacts and local caches from version control.

### Verification
- `go build ./...` and `go test ./...` executed with local cache path to confirm module compiles and tests pass within the sandbox.

### Next Actions
- Integrate Firebase Admin SDK and other platform adapters as dependencies once credentials and emulator strategy are defined.
- Add CI pipeline steps invoking `make deps`, `make lint`, and `make test` to keep parity with local tooling (owner: DevOps, target 2025-04-12).
