# Implement counters next endpoint managing named sequences with concurrency safety.

**Parent Section:** 6. Admin / Staff Endpoints > 6.6 Operations Utilities
**Task ID:** 093

## Purpose
Provide staff endpoint to request next value from named counters (diagnostic/manual use).

## Endpoint
- `POST /counters/{{name}}:next`

## Implementation Steps
1. Validate caller has admin role; optionally restrict to safe counters.
2. Invoke counter service to increment and return formatted value.
3. Log manual increments for audit.
4. Tests verifying concurrency and error scenarios.
