# Implement local persistence (Hive/Isar/shared_preferences) for caching, offline screen data, and onboarding state.

**Parent Section:** 2. Core Infrastructure & Shared Components
**Task ID:** 013

## Goal
Support offline caching and storing user preferences.

## Implementation Steps
1. Select persistence engine (Hive/Isar) and set up adapters/migrations.
2. Design cache schemas for designs, cart, guides, notifications, onboarding flags.
3. Implement repository caching policies (stale-while-revalidate, TTL).
4. Provide encryption for sensitive data where required.
