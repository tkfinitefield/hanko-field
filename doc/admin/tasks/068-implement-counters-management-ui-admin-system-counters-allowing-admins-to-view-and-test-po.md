# Implement counters management UI (`/admin/system/counters`) allowing admins to view and test `POST /admin/counters/{name}:next`.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 068

## Goal
Provide counters management UI.

## Implementation Steps
1. Table showing counter name, scope, current value, last updated.
2. Form to call `POST /admin/counters/{name}:next` for testing (optionally specify scope).
3. Display next value and log action to audit.
