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
