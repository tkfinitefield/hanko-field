# Implement materials listing/detail endpoints, including availability flags.

**Parent Section:** 4. Public Endpoints (Unauthenticated)
**Task ID:** 030

## Purpose
Return available material types (wood, metal, stamp pad) with attributes for selection experience.

## Endpoints
- `GET /materials`
- `GET /materials/{{materialId}}`

## Data Model
- Collection `materials`: fields `id`, `name`, `description`, `category`, `grain`, `color`, `isAvailable`, `leadTimeDays`, `previewImageURL`.

## Implementation Steps
1. Query Firestore for `isAvailable == true`; include inventory `leadTimeDays` computed from stock service when necessary.
2. Provide translation support for localized fields (use fallback labels from CMS when translation missing).
3. Cache responses using CDN or memory caching due to low churn.
4. Tests verifying available filter and localized response behaviour.
