# Implement global search page (`/admin/search`) and result fragment (`/admin/search/table`) with filtering and keyboard shortcuts.

**Parent Section:** 4. Shared Utilities & System Pages
**Task ID:** 022

## Goal
Build global search spanning orders, users, reviews as described in design.

## Implementation Steps
1. Create `/admin/search` page with search form (input, filters) and results region.
2. Implement `/admin/search/table` fragment accepting query params and returning results table grouped by entity type.
3. Integrate with search API or call multiple endpoints; handle pagination (per type or aggregated).
4. Add keyboard shortcuts (focus on `/` key) and highlight search terms.
5. Provide empty-state and error messaging.
