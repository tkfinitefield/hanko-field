# Build product detail (`/products/{productId}`) with gallery fragment, review snippets, and add-to-cart form.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 018

## Goal
Build product detail page with gallery, reviews snippet, add-to-cart.

## Implementation Steps
1. Fetch product data (specs, pricing, stock); render header with CTA.
2. Implement gallery fragment for image switching; allow zoom/lightbox.
3. Embed review snippets fragment; link to full reviews.
4. Build add-to-cart form (quantity, options) posting to `/cart/items` via htmx.

## UI Components
- **Layout:** `SiteLayout` with sticky `PrimaryNav` and breadcrumb `SectionHeader`.
- **Gallery:** `MediaGallery` supporting image zoom, video, and material swatches.
- **Purchase column:** `DetailPanel` containing price `PriceBadge`, availability `StatusPill`, quantity `Stepper`, CTA `PrimaryButton`.
- **Details tabs:** `ContentTabs` for Description, Specs, Reviews, FAQ with htmx fragments per tab.
- **Recommendations:** `ProductRail` showcasing related materials/products via `ProductCard`.
- **Review snippet:** `RatingSummary` card with aggregate score and latest quotes.
