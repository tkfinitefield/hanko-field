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
1. Define order schema with sub-collections: `payments`, `shipments`, `productionEvents`, `auditTrail`.
2. Implement service methods `CreateFromCart`, `Cancel`, `AppendProductionEvent`, `RequestInvoice`, `CloneForReorder`.
3. Ensure transactional integrity (order creation + reservation commit) using Firestore transactions or sagas.
4. Publish domain events (Cloud Pub/Sub) for analytics or notifications when status changes occur.
5. Unit/integration tests covering transition validation, concurrency, and reorder logic.
