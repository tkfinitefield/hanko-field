# Implement order lifecycle service (cart → order creation, status transitions, production events, shipment updates).

**Parent Section:** 3. Shared Domain Services
**Task ID:** 022

## Goal
Manage transition from cart to order, handle status changes, production events, and shipment updates coherently.

## Responsibilities
- Create order documents (`orders` collection) with snapshots of design, cart items, pricing, and shipping info.
- Provide state machine enforcing transitions (`draft -> pending_payment -> paid -> in_production -> shipped -> delivered -> completed`).
- Expose methods for user endpoints (list, detail, cancel, reorder) and admin operations (status changes, production events).
- Integrate with payment service, inventory reservations, and production queue service.

## Steps
- [x] Define order schema with sub-collections: `payments`, `shipments`, `productionEvents`, `auditTrail`.
- [x] Implement service methods `CreateFromCart`, `Cancel`, `AppendProductionEvent`, `RequestInvoice`, `CloneForReorder`.
- [x] Ensure transactional integrity (order creation + reservation commit) using Firestore transactions or sagas.
- [x] Publish domain events (Cloud Pub/Sub) for analytics or notifications when status changes occur.
- [x] Unit/integration tests covering transition validation, concurrency, and reorder logic.

## Completion Notes
- Expanded shared domain models for orders, line items, production events, and supporting structs, adding status constants and timeline metadata (`api/internal/domain/types.go`).
- Added repository contracts for order production events and wired the order service into the DI container (`api/internal/repositories/interfaces.go`, `api/internal/di/container.go`).
- Implemented `orderService` handling cart conversion, state transitions, cancellation with inventory coordination, production event updates, invoice requests, and reorder cloning, with domain event publishing hooks (`api/internal/services/order_service.go`).
- Created comprehensive unit tests covering creation, transitions, cancellation, production events, and reorders (`api/internal/services/order_service_test.go`).
