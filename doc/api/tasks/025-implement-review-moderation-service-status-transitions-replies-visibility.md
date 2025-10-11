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
- [x] Implement `ReviewService` with methods `Create`, `GetByOrder`, `ListByUser`, `Moderate`, `StoreReply`.
- [x] Enforce text sanitization and profanity filters at creation time.
- [x] Emit events or notifications when reviews approved/rejected to update storefront.
- [x] Add tests covering moderation rules and duplicate prevention.

## Completion Notes
- Added review domain models and repository contracts, including moderation metadata and reply support (`api/internal/domain/types.go`, `api/internal/repositories/interfaces.go`).
- Implemented `ReviewService` with validation, duplicate prevention, moderation transitions, replies, and event emission (`api/internal/services/review_service.go`).
- Wired service into DI and exposed via the services facade (`api/internal/services/interfaces.go`, `api/internal/di/container.go`).
- Created unit tests covering creation sanitization, duplicate guarding, moderation transitions, and reply management (`api/internal/services/review_service_test.go`).
