# Define UI architecture (Go server-side rendering + htmx partials) and routing conventions for full pages vs fragment endpoints. ✅

**Parent Section:** 0. Planning & Architecture
**Task ID:** 002

## Goal
Define the technical approach for Go + htmx SSR admin console so implementation teams share conventions.

## Decisions to Make
- Router framework (chi vs echo) and route naming for full pages vs fragment endpoints.
- Template structure and partials naming (`/admin/layouts`, `/admin/partials`).
- htmx usage patterns (form submissions, swaps, indicators, error handling).
- Asset pipeline (Tailwind build, Alpine.js optional enhancements).

## Deliverables
- ADR covering SSR + htmx architecture, template organisation, fragment naming conventions.
- Diagram showing request flow (auth middleware → handler → template render → fragment swap).
- Coding standards doc (naming, template helpers, CSS utilities).

## Acceptance Criteria
- Engineers can create new page or fragment following documented conventions without ambiguity.
- Architecture doc stored under `doc/admin/architecture.md` and reviewed with team.
