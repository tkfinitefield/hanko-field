# Implement addresses CRUD endpoints with validation, default management, and dedupe.

**Parent Section:** 5. Authenticated User Endpoints > 5.1 Profile & Account
**Task ID:** 036

## Purpose
Manage user shipping/billing addresses including validation, deduplication, and default selection.

## Endpoints
- `GET /me/addresses`
- `POST /me/addresses`
- `PUT /me/addresses/{{addressId}}`
- `DELETE /me/addresses/{{addressId}}`

## Implementation Steps
1. Define Firestore sub-collection `users/{{uid}}/addresses` with fields: `id`, `fullName`, `postalCode`, `prefecture`, `addressLine1/2`, `phone`, `isDefaultShipping`, `isDefaultBilling`, `createdAt`, `updatedAt`.
2. Validate addresses via postal regex and optional third-party address normalization service.
3. Ensure only one default shipping/billing; use transaction to unset others when setting new default.
4. Prevent deletion when address linked to open orders unless replacement provided.
5. Tests covering create/update/delete flows, default toggling, validation errors.
