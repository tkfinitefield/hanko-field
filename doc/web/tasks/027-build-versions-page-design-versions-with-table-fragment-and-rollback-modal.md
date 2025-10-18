# Build versions page (`/design/versions`) with table fragment and rollback modal.

**Parent Section:** 4. Design Creation Flow
**Task ID:** 027

## Goal
Show version history with rollback functionality.

## Implementation Steps
1. Implement fragment returning table/list of versions with metadata.
2. Provide rollback modal to confirm revert and call API.
3. Refresh editor preview after rollback.

## UI Components
- **Layout:** `SiteLayout` with breadcrumb `SectionHeader`.
- **Versions table:** `HistoryTable` listing version number, author, timestamp, notes, with compare `IconButton`.
- **Diff preview:** `SplitPreview` component showing Before/After canvases (lazy load).
- **Action bar:** `Toolbar` with restore `PrimaryButton`, duplicate `SecondaryButton`, delete `DangerButton`.
- **Filters:** `FilterChips` for author/date tags.
- **Audit drawer:** `TimelineDrawer` showing change log entries.
