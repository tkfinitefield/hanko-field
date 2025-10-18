# Implement home screen (`/home`) showing featured items, recent designs, and recommended templates using async providers.

**Parent Section:** 4. Home & Discovery
**Task ID:** 023

## Goal
Build home screen surfaces curated content and personalized recommendations.

## Implementation Steps
1. Fetch featured content, recent designs, recommended templates using `AsyncNotifier` providers.
2. Compose sections as independent widgets (carousal, grid) with skeleton loaders.
3. Support pull-to-refresh and analytics events for section interactions.
4. Personalize ordering based on persona, locale, and usage history.

## Material Design 3 Components
- **Navigation:** `Center-aligned top app bar` with search `Icon button` and notification `Badge` actions.
- **Hero:** `Elevated card` with image + headline typography for featured campaign.
- **Sections:** Horizontal `List` of `Filled cards` for recommendations and `Outlined cards` for recents.
- **Global nav:** Persistent `Navigation bar` anchored to the shell for tab switching.
