# Build material detail screen (`/materials/:materialId`) showing specs, gallery, availability.

**Parent Section:** 6. Shop & Product Browsing
**Task ID:** 038

## Goal
Show material detail page with specs and media.

## Implementation Steps
1. Fetch material info (hardness, texture, photos) and availability.
2. Present gallery with pinch-to-zoom, video support if available.
3. Provide actions to add to cart or view compatible products.

## Material Design 3 Components
- **App bar:** `Medium top app bar` with favorite `Icon button`.
- **Gallery:** `Carousel` of imagery inside `Elevated cards` with rounded corners.
- **Specs:** `List items` with leading icons and trailing `Assist chips` for availability.
- **Action rail:** Bottom `Filled button` to start order and `Outlined button` for share.
