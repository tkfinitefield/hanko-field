# Implement version history (`/design/versions`) with diff view and rollback actions.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 035

## Goal
Display design version history with diff and rollback.

## Implementation Steps
1. Fetch version list from backend; show timeline with thumbnails and metadata.
2. Provide diff viewer (side-by-side) and rollback confirmation dialog.
3. Log analytics and audit event on rollback.

## Material Design 3 Components
- **Top bar:** `Small top app bar` with compare `Icon button`.
- **Timeline:** Vertical `List` with `Divider` separators and `Assist chips` for status (current, archived).
- **Preview pairs:** `Outlined cards` showing before/after thumbnails.
- **Actions:** `Filled tonal button` to restore and `Outlined button` to duplicate version.
