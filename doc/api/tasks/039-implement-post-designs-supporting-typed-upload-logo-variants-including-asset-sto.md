# Implement `POST /designs` supporting typed/upload/logo variants, including asset storage and validation.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 039

## Purpose
Create user designs from typed input, uploaded artwork, or logo, storing assets and metadata for iteration.

## Flow
1. Validate request payload (type, text content, font/material selections, uploaded asset references).
2. Generate initial vector/bitmap asset (invoke design rendering service) and save to Storage path `designs/{{designId}}/v1/...`.
3. Persist design document with status `draft`, owner UID, and configuration snapshot.
4. Emit audit event and return created design with version info.

## Data Model
- Collection `designs`: `id`, `ownerUid`, `label`, `type`, `textLines[]`, `fontId`, `materialId`, `status`, `thumbnailURL`, `currentVersionId`, `createdAt`, `updatedAt`.
- Sub-collection `versions` for history snapshots.

## Implementation Steps
1. Define request/response DTOs and validation (character limits, banned words, asset size).
2. Call renderer (internal service) or queue job to produce preview; handle asynchronous case by returning `processing` status.
3. Store design and initial version within Firestore transaction.
4. Handle idempotency to avoid duplicate designs on retries (e.g., idempotency key referencing payload hash).
5. Add tests covering typed vs upload flows and validation errors.
