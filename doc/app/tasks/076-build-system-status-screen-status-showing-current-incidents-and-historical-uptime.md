# Build system status screen (`/status`) showing current incidents and historical uptime.

**Parent Section:** 12. Support & Status
**Task ID:** 076

## Goal
Display system status screen.

## Implementation Steps
1. Fetch status data (current incidents, uptime history) from status API.
2. Show current status banner and incident details.
3. Provide subscription or refresh controls.

## Material Design 3 Components
- **App bar:** `Large top app bar` with refresh `Icon button`.
- **Current incident:** `Elevated card` with severity `Assist chips` and timestamp.
- **History:** `List items` grouped by week with `Dividers`.
- **Filters:** `Segmented buttons` for services (API, App, Admin).
