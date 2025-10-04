# Establish integration/unit testing harness (httptest, HTML assertions, htmx interactions).

**Parent Section:** 1. Project Setup & Tooling
**Task ID:** 010

## Goal
Establish testing harness for SSR pages and htmx fragments.

## Implementation Steps
1. Use `httptest` to render pages; assert HTML using goquery.
2. Write tests for fragment endpoints verifying partial structure and HTTP headers.
3. Integrate end-to-end tests with Playwright/Cypress for critical flows.
4. Include tests in CI pipeline.
