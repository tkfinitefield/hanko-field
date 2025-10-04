# Build dynamic sidebar rendering with route highlighting and permission-based sections.

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 016

## Goal
Render sidebar navigation dynamically based on role and current route.

## Implementation Steps
1. Build menu configuration map (group name, items, icon, required capability, route pattern).
2. Template iterates over groups, filtering items with `HasCapability` helper.
3. Apply active class when request path matches item route prefix.
4. Support collapsible behaviour on small screens (toggle button controlling CSS state).
5. Provide unit tests ensuring hidden items for limited roles.
