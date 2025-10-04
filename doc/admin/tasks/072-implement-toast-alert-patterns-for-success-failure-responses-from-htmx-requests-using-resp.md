# Implement toast/alert patterns for success/failure responses from htmx requests using response headers or JSON payload.

**Parent Section:** 13. Notifications & Real-Time Feedback
**Task ID:** 072

## Goal
Implement toast/alert patterns from htmx responses.

## Implementation Steps
1. Standardize backend responses to include `HX-Trigger: showToast` header with payload.
2. Client toast manager displays message with severity, optional action buttons.
3. Provide documentation for backend developers on how to trigger.
