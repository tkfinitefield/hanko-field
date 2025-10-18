# Implement FAQ screen (`/support/faq`) with categories, search, and offline caching.

**Parent Section:** 12. Support & Status
**Task ID:** 073

## Goal
Implement FAQ screen with categories and offline cache.

## Implementation Steps
1. Fetch FAQ categories and entries from CMS; cache locally.
2. Provide search with keyword highlighting and suggestion tags.
3. Allow feedback on article helpfulness.

## Material Design 3 Components
- **App bar:** `Large top app bar` with inline `Search bar` and filter `Icon button`.
- **Category chips:** `Filter chips` horizontally scrollable for topic selection.
- **FAQ list:** `List items` using expandable supporting text for answers.
- **Fallback:** `Outlined card` promoting contact options when no result found.
