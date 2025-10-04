# Implement global layout template (`/admin/_layout.html`) with sidebar, top bar, breadcrumbs, and responsive behaviour.

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 015

## Goal
Build master layout for admin pages with responsive grid and shared chrome.

## Structure
- `<html>` root with language attribute.
- `<body>` containing sidebar (nav), topbar, `<main id="content">` area.
- Slot for toast container and modal container appended at end of body.
- Inject CSRF meta tag, hx-headers script, environment badge in topbar.

## Implementation Steps
1. Create `_layout.html` template using Go `template` with `{{block "content" .}}` placeholder.
2. Include dynamic breadcrumb component.
3. Provide CSS classes ensuring fixed navigation, scrollable content, dark mode support.
