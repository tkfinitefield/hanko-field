# Build moderation modal(s) for approve/reject (`PUT /admin/reviews/{id}:moderate`) and store reply (`POST /admin/reviews/{id}:store-reply`).

**Parent Section:** 9. Customers, Reviews, and KYC
**Task ID:** 055

## Goal
Build moderation modals for approve/reject and reply.

## Implementation Steps
1. Approve/reject modal includes moderation notes and optional email to customer.
2. Reply modal stores staff response via backend API and updates UI.
3. Ensure actions update review status column via htmx triggers.
