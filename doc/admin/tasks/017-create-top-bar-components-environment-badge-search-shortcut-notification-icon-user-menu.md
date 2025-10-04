# Create top bar components (environment badge, search shortcut, notification icon, user menu).

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 017

## Goal
Implement top bar with environment badge, search shortcut, notifications, user menu.

## Implementation Steps
1. Render environment badge (dev/stg/prod) using config injected into template.
2. Add search shortcut button triggering `/admin/search` overlay with keyboard binding (`/`).
3. Notifications icon shows count from notifications service via htmx poll or SSE.
4. User menu contains profile link and logout button.
5. Ensure accessibility (ARIA roles, focus trap on menu).
