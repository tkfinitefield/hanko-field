# Build onboarding/tutorial screens (`/onboarding`) with skipping, progress indicator, and analytics events.

**Parent Section:** 3. Onboarding & Auth Flow
**Task ID:** 019

## Goal
Deliver onboarding tutorial with progress tracking.

## Implementation Steps
1. Build multi-page carousel with localized illustrations and copy.
2. Persist completion flag locally (Hive/shared prefs) and to backend when logged in.
3. Provide skip, back, next controls and progress indicator.
4. Track analytics for completion/skip rates.

## Material Design 3 Components
- **Structure:** `Medium top app bar` with `Text button` for Skip and `Icon button` for back navigation.
- **Slides:** Horizontal onboarding `Carousel` built from `Elevated cards` pairing illustration and copy.
- **Progress:** `Linear progress indicator` pinned below the cards to show step position.
- **Actions:** Primary `Filled tonal button` for Next and secondary `Outlined button` for Back/Skip.
