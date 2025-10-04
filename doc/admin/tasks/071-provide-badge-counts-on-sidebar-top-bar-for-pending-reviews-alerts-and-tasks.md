# Provide badge counts on sidebar/top bar for pending reviews, alerts, and tasks.

**Parent Section:** 13. Notifications & Real-Time Feedback
**Task ID:** 071

## Goal
Show live badge counts for pending items.

## Implementation Steps
1. Notifications service returns counts for pending reviews, alerts, tasks.
2. Update counts when SSE/poll event received or when relevant fragment reloads.
3. Ensure counts accessible for screen readers.
