# Implement design duplication endpoint producing new design with copied assets/metadata.

**Parent Section:** 5. Authenticated User Endpoints > 5.2 Designs & AI Workflow
**Task ID:** 043

## Purpose
Allow users to create a new design from an existing one quickly while copying assets and metadata.

## Endpoint
- `POST /designs/{{designId}}/duplicate`

## Implementation Steps
1. Validate source design ownership and status.
2. Create new design ID, copy latest configuration and assets to new storage path (`designs/{{newId}}/v1/...`).
3. Persist new design + initial version within transaction.
4. Handle asynchronous asset copy using Cloud Storage rewrite if large.
5. Tests ensuring duplicate inherits metadata but new IDs and audit entries created.
