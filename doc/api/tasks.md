# API Implementation Task List

## 0. Planning & Alignment
- [ ] Confirm scope, success criteria, and sequencing for API v1 with stakeholders based on `doc/api/api_design.md`.
- [ ] Finalize domain data models (Firestore collections/structured documents, storage layout, external IDs) covering users, designs, orders, payments, shipments, promotions, inventory, content, and audit logs.
- [ ] Define service interfaces and layering (handler → service → repository) for maintainability and testing.

## 1. Project & Environment Setup
- [ ] Initialize Go module for Cloud Run service, dependency tooling (gofumpt, staticcheck, vulncheck), and Makefile/Taskfile helpers.
- [ ] Implement configuration loader (envvars + Secret Manager) and runtime configuration schema.
- [ ] Provision infrastructure prerequisites (Cloud Run service, Firestore DB, Pub/Sub topics, Cloud Storage buckets, Cloud Scheduler jobs) in IaC or documented scripts.
- [x] Configure CI/CD pipeline (lint, unit/integration tests, build, deploy) and set up environment promotion workflow.

## 2. Core Platform Services
- [x] Implement HTTP router under `/api/v1` with chi/echo and shared middleware stack.
- [ ] Implement Firebase ID token verification, role extraction, and authentication middleware for user/staff separation.
- [ ] Implement OIDC/IAP token checker and HMAC signature validator for internal/server-to-server and webhook endpoints.
- [x] Implement Idempotency-Key middleware storing request fingerprint + result (Firestore or Redis) for POST/PUT/PATCH/DELETE.
- [ ] Implement structured logging, trace propagation (Cloud Trace), and panic/error handling middleware producing JSON error responses.
- [x] Provide request context helpers for pagination (pageSize/pageToken), sorting, and filter parsing.
- [ ] Implement Firestore client factory, transaction helpers, and strongly typed repository abstractions.
- [ ] Provide Cloud Storage signed URL helper for asset upload/download.
- [ ] Integrate secrets (PSP keys, HMAC secrets) through Secret Manager bindings.

## 3. Shared Domain Services
- [ ] Implement user profile service mapping Firebase Auth users to Firestore documents.
- [x] Implement inventory service managing stock quantities, reservations, and safety thresholds.
- [ ] Implement promotion service covering eligibility evaluation, usage accounting, and validations.
- [ ] Implement cart pricing engine (tax, shipping, discounts) with pluggable rules.
- [ ] Implement payment integration abstraction (Stripe) for checkout session management and reconciliation.
- [x] Implement order lifecycle service (cart → order creation, status transitions, production events, shipment updates).
- [ ] Implement AI suggestion job dispatcher interface (enqueuing jobs, tracking status, storing results).
- [ ] Implement audit log writer service for write operations across domains.
- [ ] Implement review moderation service (status transitions, replies, visibility).
- [ ] Implement counter/sequence generator using Firestore transaction-safe counters.

## 4. Public Endpoints (Unauthenticated)
- [ ] Implement `/healthz` and `/readyz` checks (DB, upstream dependencies) with fast responses.
- [ ] Implement templates listing/detail endpoints with optional filters and CDN URLs.
- [x] Implement fonts listing/detail endpoints with metadata needed for rendering.
- [x] Implement materials listing/detail endpoints, including availability flags.
- [ ] Implement products list/detail with filtering by shape/size/material and pagination.
- [ ] Implement localized guide content endpoints (`/content/guides`) with language/category query support.
- [ ] Implement localized page content endpoint (`/content/pages/{slug}`) with language fallback.
- [ ] Implement public promotion lookup endpoint returning exposure-safe fields only.

## 5. Authenticated User Endpoints
### 5.1 Profile & Account
- [ ] Implement `/me` GET/PUT respecting editable fields and audit logging changes.
- [ ] Implement addresses CRUD endpoints with validation, default management, and dedupe.
- [ ] Integrate PSP token management for payment methods list/add/delete.
- [ ] Implement favorites list, add, and remove endpoints referencing designs.

### 5.2 Designs & AI Workflow
- [ ] Implement `POST /designs` supporting typed/upload/logo variants, including asset storage and validation.
- [ ] Implement designs listing/detail and filtering by status/user.
- [ ] Implement design update/delete with permission checks and soft delete handling.
- [ ] Implement design version listing/detail endpoints maintaining history snapshots.
- [ ] Implement design duplication endpoint producing new design with copied assets/metadata.
- [ ] Implement AI suggestion request endpoint queuing jobs and returning suggestion IDs.
- [ ] Implement AI suggestion listing/detail retrieval from job store.
- [ ] Implement accept/reject endpoints mutating suggestion status and updating design state.
- [ ] Implement registrability-check endpoint integrating external service and caching results.

### 5.3 Name Mapping
- [ ] Implement name conversion endpoint invoking transliteration service and returning ranked candidates.
- [ ] Implement mapping selection endpoint persisting chosen candidate and locking mapping.

### 5.4 Cart & Checkout
- [ ] Implement cart retrieval endpoint keyed by user/session with lazy creation.
- [ ] Implement cart patch endpoint handling currency, shipping address, promotion hints.
- [ ] Implement cart item CRUD with product availability validation and quantity checks.
- [ ] Implement cart estimate endpoint calculating totals, promotions, tax, and shipping.
- [ ] Implement apply/remove promo endpoints interacting with promotion service and idempotency.
- [ ] Implement checkout session creation endpoint integrating PSP session API and reserving stock when required.
- [ ] Implement checkout confirm endpoint recording client-side completion and triggering post-checkout workflow.

### 5.5 Orders / Payments / Shipments
- [ ] Implement order list/detail endpoints with pagination and user scoping.
- [ ] Implement order cancel endpoint enforcing status rules and triggering stock release.
- [ ] Implement order invoice request endpoint producing task to generate PDF.
- [ ] Implement order payment history retrieval.
- [ ] Implement order shipment list/detail endpoints including tracking events.
- [ ] Implement production events retrieval endpoint exposing timeline.
- [ ] Implement reorder endpoint cloning design snapshot and cart lines to new order draft.

### 5.6 Reviews
- [ ] Implement review creation endpoint validating order ownership and completion.
- [ ] Implement review retrieval endpoint scoped to requesting user/order.

### 5.7 Assets
- [ ] Implement signed upload endpoint validating asset kind/purpose and returning pre-signed URL + metadata record.
- [ ] Implement signed download endpoint verifying ownership/permission before issuing link.

## 6. Admin / Staff Endpoints
### 6.1 Catalog & CMS
- [ ] Implement admin CRUD for templates with versioning and publishing workflow.
- [ ] Implement admin CRUD for fonts including licensing data.
- [ ] Implement admin CRUD for materials capturing stock and supplier info.
- [ ] Implement admin CRUD for products including SKU configuration and price tiers.
- [ ] Implement admin CRUD for content guides including localization and category tagging.
- [ ] Implement admin CRUD for content pages with draft/publish states.

### 6.2 Promotions
- [ ] Implement promotions list/create/update/delete endpoints with validation rules and schedule handling.
- [ ] Implement promotion usage retrieval endpoint aggregating counts per user.
- [ ] Implement promotion validate endpoint enabling dry-run eligibility checks.

### 6.3 Orders / Payments / Inventory Operations
- [ ] Implement admin order listing endpoint with status/date filters for operations dashboards.
- [ ] Implement order status transition endpoint enforcing workflow (`paid → in_production → shipped → delivered`) with audit log.
- [ ] Implement manual payment capture and refund endpoints integrating PSP APIs.
- [ ] Implement shipment creation endpoint generating labels via carrier integrations and storing tracking info.
- [ ] Implement shipment update endpoint for correcting tracking statuses/events.
- [ ] Implement production events POST endpoint allowing operations to append workflow events.
- [ ] Implement low stock endpoint aggregating inventory below thresholds.
- [ ] Implement stock reservation release endpoint for manual override of expired reservations.

### 6.4 Production Queues
- [ ] Implement production queue CRUD endpoints storing capacity, priorities, and metadata.
- [ ] Implement queue WIP summary endpoint aggregating counts/status per queue.
- [ ] Implement queue assign-order endpoint ensuring concurrency control and queue policies.

### 6.5 Users / Reviews / Audit
- [ ] Implement admin user search/list/detail endpoints with flexible query support.
- [ ] Implement deactivate-and-mask endpoint anonymizing user data and revoking access.
- [ ] Implement review moderation endpoints (list pending, approve/reject, store reply) updating moderation status.
- [ ] Implement audit log retrieval endpoint with filtering by target reference.

### 6.6 Operations Utilities
- [ ] Implement invoices issue endpoint creating batch jobs and storing generated PDFs.
- [ ] Implement counters next endpoint managing named sequences with concurrency safety.
- [ ] Implement exports to BigQuery endpoint initiating sync jobs and tracking progress.
- [ ] Implement system errors/tasks endpoints reading from failure queues/log storage.

## 7. Webhooks (Inbound)
- [ ] Implement Stripe webhook handler validating signature and processing payment intent succeeded/failed and refund events.
- [ ] Implement shipping carrier webhook handler accepting updates per carrier and mapping payloads to shipment events.
- [ ] Implement AI worker webhook handler updating AI job status and persisting generated suggestions.
- [ ] Implement webhook security middleware (IP filtering, replay protection) and monitoring.

## 8. Internal Endpoints
- [ ] Implement internal checkout reserve-stock endpoint creating reservations in transaction-safe manner and decrementing stock.
- [ ] Implement internal checkout commit endpoint finalizing reservations, marking orders paid, and emitting events.
- [ ] Implement internal checkout release endpoint restoring stock on failure/timeout.
- [ ] Implement internal promotion apply endpoint performing atomic usage increments and validation.
- [ ] Implement internal invoice issue-one endpoint generating invoice number, PDF, and updating order.
- [ ] Implement internal maintenance cleanup-reservations endpoint releasing expired reservations.
- [ ] Implement internal maintenance stock-safety-notify endpoint notifying downstream systems.
- [ ] Implement internal audit-log endpoint for structured audit writes from other services.

## 9. Background Jobs & Scheduling
- [ ] Configure Cloud Scheduler jobs (cleanup reservations, stock safety notifications) invoking internal endpoints with auth.
- [ ] Implement job runners (Cloud Run jobs/PubSub subscribers) for asynchronous tasks (AI processing, invoice generation, exports).
- [ ] Implement retry/backoff strategy and dead-letter handling for background workers.

## 10. Testing Strategy
- [ ] Write unit tests for middleware, services, and repositories (using Firestore emulator/mocks).
- [ ] Write integration tests exercising representative endpoint flows with emulators.
- [ ] Implement contract tests for webhooks to ensure payload parsing and idempotency.
- [ ] Add load/performance test plan for critical paths (checkout, AI requests, stock reservations).
- [ ] Document manual QA scenarios for admin workflows and edge cases.

## 11. Security & Compliance
- [ ] Define RBAC roles/permissions map for user vs staff vs admin endpoints.
- [ ] Enforce validation and sanitization for all user inputs to prevent injection/abuse.
- [ ] Implement rate limiting/throttling strategy for sensitive endpoints.
- [ ] Ensure PII masking/anonymization processes meet compliance and logging policies.
- [ ] Perform security review (HMAC secret rotation, OAuth scopes, firewall rules) before launch.

## 12. Observability & Operations
- [ ] Expose metrics (latency, error rates, queue depth) via Cloud Monitoring.
- [ ] Configure alerting for failures (webhook retries, stock reservation errors, payment mismatches).
- [ ] Implement centralized structured logging with correlation IDs and request IDs.
- [ ] Document on-call runbooks for incident handling and operational tasks.

## 13. Documentation & Support
- [ ] Document endpoint reference with request/response examples and auth requirements.
- [ ] Provide onboarding guide for developers (local setup, emulators, testing commands).
- [ ] Document deployment checklist and rollback procedures.
- [ ] Provide post-launch task list for monitoring and iterative improvements.
