# Build CRUD modals (`/admin/catalog/{kind}/modal/{new|edit|delete}`) connecting to `POST/PUT/DELETE /admin/catalog/{kind}` endpoints.

**Parent Section:** 6. Catalog Management
**Task ID:** 039

## Goal
Provide create/edit/delete modal workflows tying into admin catalog APIs.

## Implementation Steps
1. Modal form fields tailored per kind (templates require preview asset, fonts license info, etc.).
2. On submit call `POST/PUT/DELETE /admin/catalog/{kind}`; handle validation errors by re-rendering partial with messages.
3. After success, trigger table refresh via `HX-Trigger`.
4. Confirm delete with irreversible warning and dependency checks.
