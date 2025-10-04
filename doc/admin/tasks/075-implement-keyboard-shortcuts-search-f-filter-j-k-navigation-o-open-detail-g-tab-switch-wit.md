# Implement keyboard shortcuts (`/` search, `f` filter, `j/k` navigation, `o` open detail, `g` tab switch) with hint overlay.

**Parent Section:** 14. Accessibility, Localization, and UX Enhancements
**Task ID:** 075

## Goal
Implement keyboard shortcuts for productivity.

## Implementation Steps
1. Register key bindings (`/`, `f`, `j/k`, `o`, `g`) using unobtrusive JS.
2. Display help modal listing shortcuts (press `?`).
3. Ensure shortcuts respect input focus and accessibility (disable when modals open if needed).
