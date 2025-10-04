# Implement admin CRUD for products including SKU configuration and price tiers.

**Parent Section:** 6. Admin / Staff Endpoints > 6.1 Catalog & CMS
**Task ID:** 071

## Purpose
Allow staff to configure product catalog (SKUs, pricing, availability) powering storefront.

## Endpoints
- `POST /products`
- `PUT /products/{{id}}`
- `DELETE /products/{{id}}`

## Implementation Steps
1. Validate SKU uniqueness, price currency alignment, and compatibility lists (templates/materials).
2. Provide nested configuration for variants (size, color) and per-variant pricing.
3. Integrate with inventory service to set initial stock and safety thresholds.
4. Soft delete/unpublish to keep history.
5. Tests verifying validation and integration with inventory/promotion references.
