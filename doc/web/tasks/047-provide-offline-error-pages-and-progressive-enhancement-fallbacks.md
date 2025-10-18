# Provide offline/error pages and progressive enhancement fallbacks.

**Parent Section:** 8. Notifications, Search, and Utilities
**Task ID:** 047

## Goal
Provide offline/error pages and progressive enhancement fallback.

## Implementation Steps
1. Create static offline page with retry option.
2. Implement error pages for 404/500 with support links.
3. Ensure core functionality accessible without JS (progressive enhancement).

## UI Components
- **Offline page:** `OfflineTemplate` with illustration `EmptyState`, retry `PrimaryButton`, cached content list `CardGrid`.
- **Error page:** `ErrorTemplate` for 500/404 using `HeroCard`, diagnostic `DetailsAccordion`, support `LinkButton`.
- **Maintenance:** `MaintenanceTemplate` with countdown timer and status subscription `Form`.
- **Progressive fallback:** `ProgressiveShell` delivering static `Skeleton` or `PlaceholderCard` when JS disabled.
- **Toast hooks:** `ToastHost` triggered for offline detection and reconnection events.
- **Analytics:** `TelemetryBeacon` component logging offline/error occurrences.
