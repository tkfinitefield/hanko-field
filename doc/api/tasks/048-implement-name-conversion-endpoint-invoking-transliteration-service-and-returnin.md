# Implement name conversion endpoint invoking transliteration service and returning ranked candidates.

**Parent Section:** 5. Authenticated User Endpoints > 5.3 Name Mapping
**Task ID:** 048

## Purpose
Convert Latin names into candidate kanji representations using transliteration/AI services.

## Endpoint
- `POST /name-mappings:convert`

## Implementation Steps
1. Validate payload (`latin`, `locale`, optional `gender`/context).
2. Invoke external transliteration service or in-house model returning ranked kanji candidates.
3. Store conversion job in `nameMappings` collection with fields: `id`, `input`, `candidates[]`, `status`, `createdAt`.
4. Return response with candidate list, each containing `kanji`, `kana`, `score`, `notes`.
5. Tests verifying service fallback, caching repeated requests, and handling unsupported locale errors.
