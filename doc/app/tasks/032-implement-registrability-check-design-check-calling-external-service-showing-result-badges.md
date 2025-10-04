# Implement registrability check (`/design/check`) calling external service, showing result badges.

**Parent Section:** 5. Design Creation Flow (作成タブ)
**Task ID:** 032

## Goal
Present registrability check results for official seals.

## Implementation Steps
1. Call backend service with design data; display results (OK/Warning/Fail) with reasoning.
2. Provide guidance for adjustments and ability to re-run after edits.
3. Cache latest result for offline viewing.
