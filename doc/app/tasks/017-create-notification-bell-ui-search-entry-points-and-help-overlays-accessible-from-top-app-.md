# Create notification bell UI, search entry points, and help overlays accessible from top app bar.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 017

## Goal
Implement shared app bar actions (notification bell, search, help overlay).

## Implementation Steps
1. Create reusable app bar widget with configurable title/actions.
2. Notification bell: subscribe to unread count provider, navigate to `/notifications` on tap.
3. Search icon opens `/search`; help icon displays contextual tips or FAQ shortcuts.
4. Ensure actions accessible via semantics and keyboard shortcuts.
