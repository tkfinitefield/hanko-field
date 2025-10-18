# Build guide detail (`/guides/:slug`) rendering CMS content with markdown/HTML and offline caching.

**Parent Section:** 10. Guides & Cultural Content
**Task ID:** 060

## Goal
Render guide article detail.

## Implementation Steps
1. Display markdown/HTML content using rich text renderer with theme styling.
2. Support offline caching and sharing.
3. Provide related content recommendations.

## Material Design 3 Components
- **App bar:** `Medium top app bar` with bookmark `Icon button`.
- **Hero:** `Elevated card` with cover image and metadata `Assist chips` (persona, duration).
- **Content body:** `Rich text` styled with Material 3 typography tokens inside `Surface`.
- **Utilities:** `Filled tonal button` for share and `Text button` for open in browser.
