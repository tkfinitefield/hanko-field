# Ensure content editing includes markdown/HTML sanitization and version history display.

**Parent Section:** 7. CMS (Guides & Pages)
**Task ID:** 046

## Goal
Ensure editing includes sanitization and version history.

## Implementation Steps
1. Sanitize HTML server-side using allowlist (Bleach-like library) to prevent XSS.
2. Keep version history with diff display and ability to revert.
3. Log changes in audit log with actor info.
