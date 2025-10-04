# Finalize domain data models (Firestore collections/structured documents, storage layout, external IDs) covering users, designs, orders, payments, shipments, promotions, inventory, content, and audit logs.

**Parent Section:** 0. Planning & Alignment
**Task ID:** 002

## Goal
Produce canonical data models so every team relies on a single source of truth for Firestore collections, Storage buckets, and external identifiers referenced by the API design.

## Deliverables
- ERD covering users, designs, design versions, AI suggestions, carts, orders, payments, shipments, promotions, inventory, content, reviews, assets, audit logs, invoices, production queues, counters.
- JSON/YAML schemas for each primary Firestore document and sub-collection in `doc/api/models/`.
- Storage bucket layout (e.g., `assets/designs/{designId}/{filename}`) with retention and IAM policies.

## Steps
1. Inventory all entities from `api_design.md`, map ownership and relationships (references, history, soft deletes).
2. Define Firestore collection names, document IDs, required composite indexes, and TTL settings.
3. Specify field types, enum ranges, timestamp fields, audit metadata, and masking requirements.
4. Standardise external ID formats (`d_`, `o_`, `promo_`, etc.) and document generators.
5. Review schemas with backend, frontend, and data teams to ensure API payloads and reporting needs align.

## Acceptance Criteria
- Models accommodate all endpoints without further structural changes.
- Required indexes and TTL policies documented for infrastructure provisioning.
- Sensitive fields classified for compliance and logging redaction.

---

## Data Model Sign-off (2025-04-01)

### Outcomes
- ✅ Created canonical Firestore inventory in `doc/api/models/firestore.collections.yaml`, covering 18 top-level collections plus sub-collections with schema pointers, TTLs, and composite index requirements aligned to API queries.
- ✅ Added missing JSON Schema definitions for orders, cart items, AI jobs, name mappings, and invoices under `doc/db/schema/` with YAML mirrors in `doc/api/models/` to keep all primary documents typed.
- ✅ Documented identifier conventions (`doc/api/models/external-ids.yaml`) establishing ULID/base32 prefixes for public surfaces and deterministic IDs for transactional subcollections.
- ✅ Published storage layout (`doc/api/models/storage-layout.md`) detailing bucket hierarchy, lifecycle policies, and CMEK usage alongside access patterns for signed URLs.
- ✅ Captured data classification and redaction rules (`doc/api/models/data-protection.md`) to satisfy acceptance criteria on sensitive field handling.
- ✅ Updated ERD (`doc/api/models/README.md`) to reflect relationships among users, designs, carts, orders, promotions, and operational logs.

### Key Design Decisions
- Firestore remains single-project with region `asia-northeast1`; composite indexes prioritise user/order queries, production triage, and promotions usage caps.
- Orders adopt ULID-based primary keys plus human-friendly counters (`HF-YYYY-######`) to support chronological sorting and offline reconciliation.
- Ephemeral collections (`stockReservations`, `aiJobs`, `designs.aiSuggestions`) use Firestore TTL on `expiresAt` to limit storage costs and simplify cleanup jobs.
- Promotion usages store user-referenced DocumentRef strings to enable transactional checks and fraud analysis without duplicating PII.
- Storage segregation uses a single CMEK-protected assets bucket with object metadata tags (`pii`, `origin`) to drive lifecycle automation and compliance reporting.

### Implementation Notes
- Provision composite indexes listed in `firestore.collections.yaml` before seeding data; most require collection group indexes spanning nested fields (e.g. `production.queueRef`).
- Terraform modules should map new schemas to validation tests (e.g. jsonschema CLI) as part of CI to prevent regressions.
- Ensure backend services import schema definitions for runtime validation where appropriate (e.g. order creation, AI job enqueue).

### Next Actions
- Coordinate with DevOps to add TTL and composite index definitions to Firestore deployment scripts by 2025-04-05.
- Update API backlog items to reference new schema file names and ID prefixes (tags: `schemas`, `infra`).
- Schedule schema walkthrough with frontend + data teams to confirm payload mapping and analytics needs before Beta freeze (target 2025-04-08).
