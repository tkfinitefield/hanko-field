# Implement splash screen logic (`/splash`) checking auth state, app version, feature flags.

**Parent Section:** 3. Onboarding & Auth Flow
**Task ID:** 018

## Goal
Implement splash logic deciding next route.

## Implementation Steps
1. Display splash animation while initializing Firebase, remote config, local caches.
2. Evaluate forced update, onboarding completion, auth state to determine next screen.
3. Handle cold start vs warm start conditions (resume).
4. Emit analytics event for app open.

## Material Design 3 Components
- **Background:** Full-bleed `Surface` tinted with `surfaceContainerHighest` color to anchor the splash artwork.
- **Status indicator:** Center `Circular progress indicator (indeterminate)` while startup checks execute.
- **Brand lockup:** Prominent `Icon` rendered with `DisplayLarge` typography tokens above the loader.
