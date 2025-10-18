# Implement static content pages (`/content/{slug}`, `/legal/{slug}`, `/status`) with caching and markdown rendering.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 021

## Goal
Implement static content pages (content/legal/status) with caching.

## Implementation Steps
1. Fetch content from CMS or markdown files; render with HTML sanitization.
2. Cache responses, support localization, and handle not-found gracefully.
3. Implement status page showing incidents from external status API.

## UI Components
- **Layout:** `StaticLayout` reusing header/footer but simplified nav.
- **Hero header:** `ContentHeader` with breadcrumb, page icon, effective date metadata.
- **Body:** `MarkdownRenderer` styled with `Prose` for typographic hierarchy.
- **TOC:** Optional `TableOfContents` generated for long documents with in-page anchors.
- **Status banner:** `AlertBanner` summarizing update (for status/privacy pages).
- **Version footer:** `VersionFooter` listing last updated time and download links.
