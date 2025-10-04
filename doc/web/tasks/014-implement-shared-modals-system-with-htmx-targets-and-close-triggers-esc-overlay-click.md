# Implement shared modals system with htmx targets and close triggers (ESC, overlay click).

**Parent Section:** 2. Shared Layout & Components
**Task ID:** 014

## Goal
Provide standardized modal system for htmx injections.

## Implementation Steps
1. Add `<div id="modal" hx-target="this">` in layout with base markup.
2. Create JS helper to show/hide modal, trap focus, close on ESC/overlay.
3. Document fragment structure for modals (header/body/footer) and close buttons.
