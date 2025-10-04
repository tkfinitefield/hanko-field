# Implement signed download endpoint verifying ownership/permission before issuing link.

**Parent Section:** 5. Authenticated User Endpoints > 5.7 Assets
**Task ID:** 067

## Purpose
Provide time-limited download links for assets ensuring caller has permission.

## Endpoint
- `POST /assets/{{assetId}}:signed-download`

## Implementation Steps
1. Validate requesting user/staff has access (owner or admin with scope).
2. Ensure asset status `ready` and not soft-deleted; log access for audit.
3. Generate signed GET URL via storage helper with short expiry and appropriate content disposition.
4. Return URL to client and optionally include checksum for verification.
5. Tests verifying access control and audit log emission.
