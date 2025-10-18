# Implement pages management (`/admin/content/pages`) with block-based editor, preview, and publish scheduling.

**Parent Section:** 7. CMS (Guides & Pages)
**Task ID:** 045

## Goal
Provide localized preview for guides/pages.

## Implementation Steps
1. Implement preview route rendering article using same components as web view.
2. Provide language selector and shareable preview token if needed.

## UI Components
- **Page shell:** `AdminLayout` breadcrumbs to Content > Pages plus `PrimaryButton` for new page.
- **Content tree:** Left `NavigationList` showing page hierarchy, search `Input`, and status `Badge` per node.
- **Editor workspace:** Split `TwoPane` layout with `BlockEditor` on left (drag blocks, reorder, inline toolbar) and `LivePreview` iframe on right.
- **Properties panel:** `SidePanel` for SEO fields, publish schedule, tags with `Accordion` sections.
- **Action footer:** Sticky `ActionBar` hosting Save draft, Preview, Publish `ButtonGroup`.
- **History modal:** Trigger `Modal` listing previous versions via `TimelineList`.
