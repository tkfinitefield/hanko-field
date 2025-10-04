# Implement networking layer with HTTP client, interceptors (auth, logging), retries, and response parsing.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 011

## Goal
Provide reusable HTTP client with auth, logging, and resilience.

## Implementation Steps
1. Choose HTTP client (`dio` recommended) and configure base options (timeouts, user-agent, locale headers).
2. Implement interceptors for auth token injection, request logging (sensitive data redaction), and error mapping.
3. Add retry/backoff strategy and offline detection fallback.
4. Expose client via provider for dependency injection and testing with mocks.
