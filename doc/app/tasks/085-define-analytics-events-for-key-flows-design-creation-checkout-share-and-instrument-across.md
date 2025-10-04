# Define analytics events for key flows (design creation, checkout, share) and instrument across view models.

**Parent Section:** 15. Analytics, Telemetry, and Monitoring
**Task ID:** 085

## Goal
Define and instrument analytics events across the app.

## Implementation Steps
1. Create analytics spec listing events, parameters, and triggers per flow.
2. Implement analytics wrapper to log events with strongly typed enums.
3. Ensure PII-safe parameters and respect user consent preferences.
