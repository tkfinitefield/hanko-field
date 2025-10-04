# Implement review moderation endpoints (list pending, approve/reject, store reply) updating moderation status.

**Parent Section:** 6. Admin / Staff Endpoints > 6.5 Users / Reviews / Audit
**Task ID:** 090

## Purpose
Allow moderators to list pending reviews, approve/reject, and add staff replies.

## Endpoints
- `GET /reviews?moderation=pending`
- `PUT /reviews/{{id}}:moderate`
- `POST /reviews/{{id}}:store-reply`

## Implementation Steps
1. Provide filter for status and search by order/user.
2. Moderate endpoint accepts action (`approve`, `reject`) with reason; updates review status and triggers notifications.
3. Replies stored with staff identity and timestamps; visible to user once approved.
4. Tests verifying moderation rules, audit logging, and reply visibility.
