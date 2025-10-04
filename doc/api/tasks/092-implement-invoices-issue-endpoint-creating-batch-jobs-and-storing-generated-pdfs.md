# Implement invoices issue endpoint creating batch jobs and storing generated PDFs.

**Parent Section:** 6. Admin / Staff Endpoints > 6.6 Operations Utilities
**Task ID:** 092

## Purpose
Let operations trigger batch invoice generation for orders.

## Endpoint
- `POST /invoices:issue`

## Implementation Steps
1. Accept payload with `orderIds[]` or query filters (date range, status).
2. Invoke invoice service to assign sequential invoice numbers (via counter service) and generate PDFs stored in Storage.
3. Update order records with invoice references and statuses.
4. Return job tracking ID for progress monitoring.
5. Tests verifying batching, counter usage, and Storage paths.
