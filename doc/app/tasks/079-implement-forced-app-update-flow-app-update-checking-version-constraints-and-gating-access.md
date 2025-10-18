# Implement forced app update flow (`/app-update`) checking version constraints and gating access.

**Parent Section:** 13. System Utilities
**Task ID:** 079

## Goal
Implement forced update flow gating app usage when version outdated.

## Implementation Steps
1. Check Remote Config or backend version policy on startup.
2. Display blocking dialog with store links when update required.
3. Provide optional non-blocking reminder for soft updates.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with support `Text button`.
- **Alert card:** `Outlined card` using `errorContainer` tokens to communicate urgency.
- **Action set:** Primary `Filled button` for update now, secondary `Text button` opening store listing.
- **Blocking notice:** `Banner` explaining restrictions until update completes.
