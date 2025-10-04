# Implement review creation endpoint validating order ownership and completion.

**Parent Section:** 5. Authenticated User Endpoints > 5.6 Reviews
**Task ID:** 064

## Purpose
Allow users to submit reviews for completed orders while enforcing eligibility and moderation pipeline.

## Endpoint
- `POST /reviews`

## Implementation Steps
1. Validate order belongs to user, status `delivered` or `completed`, review not already submitted.
2. Accept payload (`rating`, `title`, `body`, `photos[]`) with validation (rating bounds, profanity filter).
3. Persist review with status `pending` and queue for moderation notifications.
4. Emit event for analytics and email confirmation.
5. Tests verifying eligibility checks, duplicate prevention, and content sanitization.
