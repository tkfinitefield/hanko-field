# Implement RBAC guard utilities for template rendering (sidebar filtering, action visibility).

**Parent Section:** 2. Authentication, Authorization, and Session Management
**Task ID:** 012

## Goal
Provide utilities to enforce role-based access both in handlers and templates.

## Implementation Steps
1. Define role constants and capability map (e.g., `RoleOps => orders, shipments`, `RoleMarketing => promotions`).
2. Build middleware `RequireRole` to guard routes.
3. Expose template helper `HasCapability` to conditionally render actions/buttons.
4. Unit test capability matrix to prevent regressions.
