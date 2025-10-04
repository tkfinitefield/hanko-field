# Write end-to-end tests covering critical workflows (order status change, shipment label generation, promotion creation, review moderation).

**Parent Section:** 15. Quality Assurance & Documentation
**Task ID:** 077

## Goal
Build E2E regression suite for critical admin flows.

## Implementation Steps
1. Choose framework (Playwright, Cypress, or Go chromedp) to automate browser flows.
2. Cover scenarios: login, order status change, shipment label generation, promotion creation, review moderation.
3. Seed fixtures via backend setup API/emulator before tests.
4. Integrate with CI nightly and gating pipeline.
