# Implement production events retrieval endpoint exposing timeline.

**Parent Section:** 5. Authenticated User Endpoints > 5.5 Orders / Payments / Shipments
**Task ID:** 062

## Purpose
Provide timeline of production steps for transparency (engraving, QA, packaging).

## Endpoint
- `GET /orders/{{orderId}}/production-events`

## Implementation Steps
1. Retrieve events from `orders/{{id}}/productionEvents` sorted ascending; fields: `timestamp`, `stage`, `operator`, `notes` (sanitised), `attachments`.
2. Support filtering (e.g., `?includeNotes=false`) to hide internal notes if needed.
3. Tests verifying ordering and redaction.
