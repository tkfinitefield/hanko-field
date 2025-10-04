# Build offline screen (`/offline`) with retry and cached content access.

**Parent Section:** 13. System Utilities
**Task ID:** 080

## Goal
Provide offline screen with retry options.

## Implementation Steps
1. Detect offline state via connectivity provider; show offline UI.
2. Offer cached content access where available and retry button.
3. Resume navigation automatically once connection restored.
