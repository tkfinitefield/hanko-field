# Implement modal container (`#modal`) with htmx target wiring, animations, and escape-key close behaviour.

**Parent Section:** 3. Layout, Navigation, and Shared UX
**Task ID:** 018

## Goal
Provide reusable modal container for htmx to inject forms and handle lifecycle.

## Implementation Steps
1. Add `<div id="modal" class="hidden" hx-target="this">` to layout.
2. Create small Alpine/vanilla script to toggle visibility, lock scroll, and handle ESC key.
3. Standardize htmx responses to include `HX-Trigger` header for closing modal/refreshing parts of DOM.
4. Provide CSS for overlay and animation tokens.
5. Document fragment guidelines (wrap modal content in `_modal.html`).
