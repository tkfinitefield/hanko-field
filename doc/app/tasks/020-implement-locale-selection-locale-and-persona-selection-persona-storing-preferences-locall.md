# Implement locale selection (`/locale`) and persona selection (`/persona`) storing preferences locally and server-side.

**Parent Section:** 3. Onboarding & Auth Flow
**Task ID:** 020

## Goal
Provide locale and region selection.

## Implementation Steps
1. Display supported locales with description and sample content.
2. Persist selection locally and propagate to app-wide locale provider.
3. Sync with backend profile once authenticated.
4. Handle fallback to device locale when user skips.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` with up navigation and Save action.
- **Locale grid:** `List items` with `Radio buttons` and supporting text for language/region metadata.
- **Persona chips:** `Assist chips` grouped by persona archetype to toggle contextual defaults.
- **Confirmation:** Footer `Filled button` to persist selections across providers.
