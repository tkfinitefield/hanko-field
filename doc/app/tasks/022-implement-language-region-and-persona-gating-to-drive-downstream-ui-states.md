# Implement language/region and persona gating to drive downstream UI states.

**Parent Section:** 3. Onboarding & Auth Flow
**Task ID:** 022

## Goal
Implement authentication screen supporting Apple, Google, Email, guest mode.

## Implementation Steps
1. Provide branded login UI with provider buttons and guest option.
2. Integrate Firebase Auth for each provider; handle linking and error messaging.
3. On success, fetch user profile and update session provider.
4. Support guest mode with limited capabilities and prompt to upgrade later.
