# Build offline screen (`/offline`) with retry and cached content access.

**Parent Section:** 13. System Utilities
**Task ID:** 080

## Goal
Provide offline screen with retry options.

## Implementation Steps
1. Detect offline state via connectivity provider; show offline UI.
2. Offer cached content access where available and retry button.
3. Resume navigation automatically once connection restored.

## Material Design 3 Components
- **Background:** `Surface` with illustration inside `Elevated card`.
- **Status message:** `BodyLarge` typography with `Assist chip` for last sync timestamp.
- **Actions:** `Filled tonal button` to retry and `Text button` to open cached library.
- **Footer:** `Navigation bar` disabled state to indicate offline restrictions.
