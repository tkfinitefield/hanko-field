# Implement settings page for environment configuration toggles or at minimum a read-only configuration summary.

**Parent Section:** 12. Logs, Counters, and System Operations
**Task ID:** 069

## Goal
Provide settings page summarizing environment config.

## Implementation Steps
1. Display read-only config values (feature flags, integration status) with caution about sensitive data.
2. Optionally allow toggling feature flags with confirmation.
3. Document manual procedures linked from page.

## UI Components
- **Page shell:** `AdminLayout` with `PageHeader` (environment switcher `SegmentedControl`).
- **Category nav:** Left `NavigationList` for features, integrations, risk, experiments.
- **Setting groups:** `Accordion` sections containing `Switch`, `Select`, `NumberField`, `MultiSelect` controls with helper text.
- **Audit summary:** Inline `ChangeLog` component listing last edits with actor/time.
- **Action footer:** Sticky `ActionBar` for Save/Discard + `SnackbarHost` on completion.
- **Guardrails:** `InlineAlert` for read-only environments and `Tooltip` for locked fields.
