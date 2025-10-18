# Implement templates listing/detail (`/templates`, `/templates/{templateId}`) with filter fragment and preview.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 019

## Goal
Build templates list and detail pages.

## Implementation Steps
1. Implement filters (script, shape, registrability) hooking to `/templates/table` fragment.
2. Render detail page showing recommended sizes, constraints, preview.
3. Provide CTA to start design with selected template.

## UI Components
- **Layout:** `SiteLayout` with design `SectionHeader` and create CTA.
- **Filter controls:** `FilterToolbar` using search `Input`, category `ChipGroup`, style `Select`, locale `Toggle` (htmx).
- **Template grid:** `TemplateMasonry` rendering `TemplateCard` tiles with preview overlays.
- **Detail drawer:** `TemplateDrawer` slide-over showing metadata, usage stats, actions.
- **Pagination:** Infinite scroll `WaypointLoader` with skeleton cards.
- **Empty state:** `EmptyState` encouraging upload or request template.
