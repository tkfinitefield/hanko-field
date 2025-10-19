# Scaffold Go web module (`/admin`) with templating pipeline, asset bundler (Tailwind), and dev tooling (hot reload, lint). ✅

**Parent Section:** 1. Project & Infrastructure Setup
**Task ID:** 005

## Scope
Create Go module for admin web app including build scripts, dependency management, and dev tooling.

## Implementation Steps
1. Initialise module (e.g., `github.com/hanko-field/admin`) within repository.
2. Add dependencies: router (`chi`), template helpers, csrf middleware, Firebase auth client, htmx-friendly utilities.
3. Configure make/task commands (`dev`, `build`, `lint`, `test`, `tailwind` watcher).
4. Integrate Tailwind CLI (standalone binary) with make targets for asset compilation; include watch mode for local dev.
5. Provide local hot reload script using `air` or `fresh` to reload Go server on file change.
6. Document setup in `doc/admin/dev_setup.md`.

## Acceptance Criteria
- `make dev` starts server with Tailwind watcher and reload.
- Lint/test commands pass in clean checkout.
