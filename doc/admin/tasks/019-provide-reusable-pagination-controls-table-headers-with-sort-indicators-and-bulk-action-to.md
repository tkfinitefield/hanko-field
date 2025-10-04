# Provide reusable pagination controls, table headers with sort indicators, and bulk action toolbar components.

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 019

## Goal
Build reusable components for pagination, sorting, and bulk actions across tables.

## Implementation Steps
1. Create partial for table header cells with sort icons; support query param linking via htmx.
2. Build pagination partial reading `PageInfo` struct (pageSize, current, next, prev).
3. Implement bulk action toolbar component showing selected count, exposing actions via forms or modals.
4. Ensure keyboard accessibility (space/enter triggers, focus states).
