# Implement rate limiting/throttling strategy for sensitive endpoints.

**Parent Section:** 11. Security & Compliance
**Task ID:** 119

## Goal
Implement rate limiting/throttling for sensitive endpoints (login, design AI requests, promotions) to mitigate abuse.

## Plan
- Evaluate Cloud Armor, API Gateway, or in-app token bucket limiter (Redis/Memory) depending on deployment.
- Configure limits per route and per identity (IP/user ID).
- Provide override controls for staff/backoffice operations.
- Instrument metrics for throttled requests.
