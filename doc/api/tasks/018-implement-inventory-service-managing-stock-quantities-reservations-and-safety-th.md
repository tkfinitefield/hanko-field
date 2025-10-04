# Implement inventory service managing stock quantities, reservations, and safety thresholds.

**Parent Section:** 3. Shared Domain Services
**Task ID:** 018

## Goal
Provide stock management abstractions for carts, orders, and admin operations, including reservations and safety threshold monitoring.

## Responsibilities
- Manage collections `products`, `inventory`, `stockReservations`.
- Support reservation workflow (reserve → commit → release) used by checkout/internal endpoints.
- Calculate low stock alerts and integrate with Cloud Scheduler cleanup tasks.

## Data Model
- `stock` documents keyed by SKU with fields: `sku`, `productRef`, `onHand`, `reserved`, `safetyStock`, `updatedAt`.
- `stockReservations` documents: `reservationId`, `orderId`, `lines[]`, `status`, `expiresAt`, `createdAt`.

## Steps
1. Implement `InventoryRepository` providing transactional updates to adjust `onHand`/`reserved` counts.
2. Implement `InventoryService` with methods `ReserveStocks`, `CommitReservation`, `ReleaseReservation`, `ListLowStock`.
3. Ensure concurrency via Firestore transactions to prevent overselling.
4. Emit events for stock changes for audit/analytics.
5. Tests using Firestore emulator simulating concurrent reservations and expiry handling.
