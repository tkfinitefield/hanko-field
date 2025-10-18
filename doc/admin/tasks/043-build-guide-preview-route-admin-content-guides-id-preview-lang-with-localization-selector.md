# Build guide preview route (`/admin/content/guides/{id}/preview?lang=`) with localization selector.

**Parent Section:** 7. CMS (Guides & Pages)
**Task ID:** 043

## Goal
Provide localized preview for guides/pages.

## Implementation Steps
1. Implement preview route rendering article using same components as web view.
2. Provide language selector and shareable preview token if needed.

## UI Components
- **Layout:** `PreviewLayout` (full-width) with sticky `PreviewHeader` containing locale `SegmentedControl`, status `Badge`, and open-in-new `IconButton`.
- **Viewport:** `DeviceFrame` component showcasing responsive preview (desktop/tablet/mobile toggles).
- **Sidebar:** Collapsible `MetaPanel` listing metadata, publish windows, author notes.
- **Feedback bar:** Bottom `ActionBar` with Approve/Request changes buttons and comment `Textarea`.
- **Skeleton state:** `SkeletonBlocks` covering hero/body during render fetch.
- **Error banner:** `InlineAlert` for preview generation failures.
