# Implement generic error screen (`/error`) with diagnostics and support links.

**Parent Section:** 13. System Utilities
**Task ID:** 081

## Goal
Display generic error screen with diagnostics.

## Implementation Steps
1. Show friendly error message, error code, and support CTA.
2. Provide actions to retry, go home, or report problem.
3. Log error context for analytics.

## Material Design 3 Components
- **App bar:** `Center-aligned top app bar` providing contextual title.
- **Error summary:** `Outlined card` with icon, headline, and detailed supporting text.
- **Remediation chips:** `Assist chips` linking to retry/report/log export actions.
- **CTA:** `Filled button` for retry and `Text button` for contact support.
