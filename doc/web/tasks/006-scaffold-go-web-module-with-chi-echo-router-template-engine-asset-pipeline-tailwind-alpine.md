# Scaffold Go web module with chi router, template engine, Tailwind asset pipeline, and dev tooling.

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 006

## Goal
Set up Go web module with router, template engine, asset pipeline, and developer tooling.

## Implementation Steps
1. Initialize Go module (e.g., `github.com/hanko-field/web`) and directory structure (`cmd/web`, `internal/handlers`, `web/templates`).
2. Add dependencies (chi, html/template, Tailwind build scripts, htmx JS bundling).
3. Configure Makefile/Taskfile for `dev`, `build`, `lint`, `test`, `tailwind` watch.
4. Provide local dev script (air/fresh) for hot reload.
5. Document setup in `doc/web/dev_setup.md`.
