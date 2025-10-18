# Implement permissions onboarding (`/permissions`) prompting for photo/storage/notification access with rationale.

**Parent Section:** 13. System Utilities
**Task ID:** 077

## Goal
Guide users through permissions onboarding (camera/storage/notifications).

## Implementation Steps
1. Explain rationale with persona-specific messaging and imagery.
2. Trigger native permission dialogs at appropriate flow steps.
3. Provide fallback instructions when permission denied.

## Material Design 3 Components
- **Layout:** Full-bleed `Surface` with brand illustration housed in an `Elevated card`.
- **Permission list:** `List items` each containing iconography and `Assist chips` for rationale.
- **Actions:** `Filled button` to grant all and `Outlined button` for not now.
- **Footer:** `Text button` linking to policy docs.
