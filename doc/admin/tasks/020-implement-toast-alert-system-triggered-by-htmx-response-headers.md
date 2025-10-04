# Implement toast/alert system triggered by htmx response headers.

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 020

## Goal
Provide toast/alert system that htmx handlers can trigger.

## Implementation Steps
1. Add toast container `<div id="toast-stack">` in layout.
2. Define JS utility to listen for `HX-Trigger` events with payload {type,message} and render toast.
3. Style success, error, warning variants using Tailwind.
4. Ensure toasts auto-dismiss after timeout and accessible (ARIA live region).
