# Implement version history (`/design/versions`) with diff view and rollback actions.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 035

## Goal
Display design version history with diff and rollback.

## Implementation Steps
1. Fetch version list from backend; show timeline with thumbnails and metadata.
2. Provide diff viewer (side-by-side) and rollback confirmation dialog.
3. Log analytics and audit event on rollback.
