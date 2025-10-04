# Implement static content pages (`/content/{slug}`, `/legal/{slug}`, `/status`) with caching and markdown rendering.

**Parent Section:** 3. Landing & Exploration
**Task ID:** 021

## Goal
Implement static content pages (content/legal/status) with caching.

## Implementation Steps
1. Fetch content from CMS or markdown files; render with HTML sanitization.
2. Cache responses, support localization, and handle not-found gracefully.
3. Implement status page showing incidents from external status API.
