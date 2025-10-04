# Develop catalog overview page with tabs for templates, fonts, materials, and products (`/admin/catalog/{kind}`).

**Parent Section:** 6. Catalog Management
**Task ID:** 037

## Goal
Provide unified catalog page with tabs for templates, fonts, materials, products.

## Implementation Steps
1. Render top-level page with tab navigation; default to templates.
2. Each tab loads table fragment via htmx specifying `kind` parameter.
3. Persist active tab in query string and highlight accordingly.
