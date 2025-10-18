# Define Go + htmx architecture (router layout, template structure, partials naming) and directory conventions.

**Parent Section:** 0. Planning & Architecture
**Task ID:** 002

## Goal
Define Go + htmx architecture, routing conventions, and template strategy.

## Decisions
- Router choice (chi) and route grouping for full pages vs fragments vs modals.
- Template directory structure (`layouts`, `partials`, `pages`) and naming conventions.
- htmx usage patterns (trigger rules, swap strategies, error handling) and progressive enhancement.
- Asset pipeline (Tailwind build, lightweight vanilla JS/htmx helpers) and caching strategy.
