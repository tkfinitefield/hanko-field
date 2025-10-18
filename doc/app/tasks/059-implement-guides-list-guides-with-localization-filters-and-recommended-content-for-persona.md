# Implement guides list (`/guides`) with localization filters and recommended content for personas.

**Parent Section:** 10. Guides & Cultural Content
**Task ID:** 059

## Goal
Implement guides list with localization support.

## Implementation Steps
1. Fetch guides from CMS API with filters (language, persona).
2. Display cards with hero image, summary, and duration.
3. Support offline caching and search.

## Material Design 3 Components
- **App bar:** `Large top app bar` with `Search bar` collapse behavior.
- **Filters:** `Filter chips` for persona, locale, and topic.
- **Guide cards:** `Elevated cards` with hero image, `HeadlineSmall`, and supporting text.
- **Global nav:** `Navigation bar` to maintain app-level destinations.
