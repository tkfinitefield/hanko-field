# Implement order status transition endpoint enforcing workflow (`paid → in_production → shipped → delivered`) with audit log.

**Parent Section:** 6. Admin / Staff Endpoints > 6.3 Orders / Payments / Inventory Operations
**Task ID:** 078

## Purpose
Allow staff to progress order status while enforcing workflow and automation triggers.

## Endpoint
- `PUT /orders/{{orderId}}:status`

## Implementation Steps
1. Enforce allowed transitions (`paid -> in_production -> shipped -> delivered -> completed`).
2. Update audit trail with actor and reason; trigger downstream actions (production queue assignment, notification, invoice).
3. Guard against regression (can't move backwards except via dedicated operations).
4. Tests verifying transition matrix and concurrency conflict.
