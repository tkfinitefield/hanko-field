# Build guides list/detail (`/guides`, `/guides/{slug}`) with CMS integration and SEO metadata.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 020

## Goal
Implement guides list/detail with CMS integration.

## Implementation Steps
1. Fetch guides list via API; render cards with category/language filters tied to fragment.
2. Render article detail with markdown/HTML sanitization, table of contents, related articles.
3. Configure SEO metadata and structured data.
4. Provide share buttons and localized alternatives.

## UI Components
- **Layout:** `SiteLayout` with `SectionHeader` including filters summary and subscribe CTA.
- **Guide filters:** `FilterToolbar` with persona `ChipGroup`, topic `Select`, search `Input`.
- **Guide list:** `GuideList` of `GuideCard` entries containing cover image, reading time, and tags.
- **Detail view:** `GuideArticle` template using `Prose` typography styles for CMS HTML.
- **Aside:** `StickySidebar` for table of contents, share `IconButtons`, and download PDF link.
- **Recommendation band:** `RelatedGuidesRail` at article end.
