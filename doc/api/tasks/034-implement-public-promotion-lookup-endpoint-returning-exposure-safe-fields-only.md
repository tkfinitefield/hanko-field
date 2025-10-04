# Implement public promotion lookup endpoint returning exposure-safe fields only.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 034

## Purpose
Allow clients to check promotion availability without revealing sensitive discount details.

## Endpoint
- `GET /promotions/{{code}}/public`

## Behaviour
- Validate promotion existence, date window, status, and exposure flag.
- Return limited payload: `code`, `isAvailable`, `startsAt`, `endsAt`, `descriptionPublic`, `eligibleAudiences` (non-sensitive).
- Do not return discount values or usage counts.

## Implementation Steps
1. Reuse promotion service repository to fetch promotion with `code` index.
2. Filter out promotions flagged `internalOnly` or `requiresAuth`.
3. Implement rate limiting to avoid brute force on codes.
4. Add tests verifying hidden fields omitted and expired promotions correctly flagged.
