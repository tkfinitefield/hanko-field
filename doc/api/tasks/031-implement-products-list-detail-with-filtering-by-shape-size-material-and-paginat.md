# Implement products list/detail with filtering by shape/size/material and pagination.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 031

## Purpose
Provide public product catalog (shape, size, material) to feed e-commerce browsing and cart building.

## Endpoints
- `GET /products`: filters `shape=`, `size=`, `material=`, `isCustomizable=`; pagination.
- `GET /products/{{productId}}`: includes price tiers, compatible templates.

## Data Model
- Collection `products`: `id`, `sku`, `name`, `shape`, `sizes[]`, `defaultMaterial`, `price`, `currency`, `images[]`, `isPublished`, `inventoryStatus`, `compatibleTemplateIds[]`.

## Implementation Steps
1. Implement query builder supporting multi-filter combinations with Firestore indexes.
2. Join with materials/templates if necessary to include human-readable names (cache to avoid N+1).
3. Include price display rules (tax inclusive/exclusive) based on config.
4. Provide tests covering filter combinations, pagination tokens, and hidden products.
