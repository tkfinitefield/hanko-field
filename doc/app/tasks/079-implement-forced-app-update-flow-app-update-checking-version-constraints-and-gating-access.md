# Implement forced app update flow (`/app-update`) checking version constraints and gating access.

**Parent Section:** 13. System Utilities
**Task ID:** 079

## Goal
Implement forced update flow gating app usage when version outdated.

## Implementation Steps
1. Check Remote Config or backend version policy on startup.
2. Display blocking dialog with store links when update required.
3. Provide optional non-blocking reminder for soft updates.
