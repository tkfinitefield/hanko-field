# API Data Models

This package is the canonical reference for Firestore collections, document schemas, storage structure, and ID conventions backing the API v1 surface described in `doc/api/api_design.md`.

- `firestore.collections.yaml` — collection inventory with schema pointers, composite indexes, TTLs.
- `external-ids.yaml` — prefixes and generators for public- and internal-facing identifiers.
- `storage-layout.md` — GCS bucket layout, retention, IAM, and encryption notes.
- `data-protection.md` — data classification, masking, and logging redaction policy.
- `*.schema.yaml` — JSON Schema documents (YAML bindings) sourced from `doc/db/schema`.

```mermaid
erDiagram
    USERS ||--o{ USER_ADDRESSES : has
    USERS ||--o{ USER_PAYMENT_METHODS : stores
    USERS ||--o{ USER_FAVORITES : bookmarks
    USERS ||--o{ CARTS : owns
    USERS ||--o{ ORDERS : places
    USERS ||--o{ REVIEWS : writes
    USERS ||--o{ NAME_MAPPINGS : requests

    CARTS ||--o{ CART_ITEMS : contains
    CARTS }|..|{ STOCK_RESERVATIONS : secures

    DESIGNS ||--o{ DESIGN_VERSIONS : versioned
    DESIGNS ||--o{ AI_SUGGESTIONS : proposes
    DESIGNS }o--|| USERS : created_by
    DESIGNS }o--|| ASSETS : references

    ORDERS ||--o{ PAYMENTS : records
    ORDERS ||--o{ SHIPMENTS : ships
    ORDERS ||--o{ PRODUCTION_EVENTS : tracks
    ORDERS }o--|| CARTS : sourced_from
    ORDERS }o--|| STOCK_RESERVATIONS : commits
    ORDERS }o--|| ASSETS : captures

    PROMOTIONS ||--o{ PROMOTION_USAGES : logs
    PROMOTIONS }o--|| USERS : targeted

    CONTENT_GUIDES }o--|| ASSETS : renders
    CONTENT_PAGES }o--|| ASSETS : renders

    PRODUCTION_QUEUES ||--o{ ORDERS : stages

    AUDIT_LOGS }o--|| USERS : actor
    AUDIT_LOGS }o--|| ORDERS : subject
    AUDIT_LOGS }o--|| DESIGNS : subject

    COUNTERS }o--|| ORDERS : enumerates
    COUNTERS }o--|| INVOICES : enumerates

    TEMPLATES }o--|| PRODUCTS : configures
    PRODUCTS }o--|| MATERIALS : uses
    PRODUCTS }o--|| FONTS : default_font
```

> Diagram shows logical relationships; see `firestore.collections.yaml` for cardinality and field-level references.
