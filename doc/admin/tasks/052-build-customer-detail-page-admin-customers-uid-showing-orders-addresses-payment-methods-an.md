# Build customer detail page (`/admin/customers/{uid}`) showing orders, addresses, payment methods, and support notes.

**Parent Section:** 9. Customers, Reviews, and KYC
**Task ID:** 052

## Goal
Provide comprehensive customer profile view.

## Implementation Steps
1. Tabs for overview, orders, addresses, payment methods, notes.
2. Display summary metrics and quick actions (send email, create order).
3. Use htmx fragments for each tab to load data lazily.
