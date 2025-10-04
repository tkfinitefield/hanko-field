# Implement review moderation service (status transitions, replies, visibility).

**Parent Section:** 3. Shared Domain Services
**Task ID:** 025

## Goal
Manage review submission, moderation status transitions, and staff replies in support of both user and admin endpoints.

## Responsibilities
- Maintain `reviews` collection with fields: `id`, `orderRef`, `userRef`, `rating`, `comment`, `status` (`pending`, `approved`, `rejected`), `moderatedBy`, `moderatedAt`, `reply`.
- Validate that review creation is tied to completed orders and enforce one review per order.
- Support admin moderation actions and notifications.

## Steps
1. Implement `ReviewService` with methods `Create`, `GetByOrder`, `ListByUser`, `Moderate`, `StoreReply`.
2. Enforce text sanitization and profanity filters at creation time.
3. Emit events or notifications when reviews approved/rejected to update storefront.
4. Add tests covering moderation rules and duplicate prevention.
