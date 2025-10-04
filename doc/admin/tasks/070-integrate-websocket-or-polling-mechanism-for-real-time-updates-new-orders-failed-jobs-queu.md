# Integrate WebSocket or polling mechanism for real-time updates (new orders, failed jobs, queue changes) within notifications center.

**Parent Section:** 13. Notifications & Real-Time Feedback
**Task ID:** 070

## Goal
Deliver real-time updates to admin UI.

## Implementation Steps
1. Evaluate SSE vs WebSocket for notifications; implement server endpoint streaming events.
2. Client subscribes and updates notifications center / badges.
3. Fallback to polling if browser not supported.
4. Handle reconnection/backoff logic.
