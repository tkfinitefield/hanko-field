# Establish HTML template structure (`layouts`, `partials`, `components`) and helper functions (i18n, currency, date formatting). ✅

**Parent Section:** 1. Project & Infrastructure Setup
**Task ID:** 007

## Goal
Establish template directory hierarchy, layout inheritance, and helper functions.

## Implementation Plan
- Layouts: `_layout.html` (base), `_modal.html`, `_table.html` partials.
- Partials: table rows, filter forms, pagination, alerts, KPI cards.
- Components: macros or Go functions for buttons, inputs, icons.
- Helpers: `func Map`, `func FormatMoney`, `func Timeago`, i18n translation function.
- Template caching strategy for production vs dev (auto reload in dev).

## Acceptance Criteria
- Developers can render view with `Render(w, "orders/index", data)` and include partials easily.
- Template helper library documented and unit tested (where possible).
