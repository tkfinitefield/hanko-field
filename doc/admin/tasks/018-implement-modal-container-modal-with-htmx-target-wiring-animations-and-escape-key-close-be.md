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

## Fragment Guidelines
- Return modal fragments wrapped with `layouts.Modal` (or `components.Modal`) so the panel includes the correct ARIA attributes, focus trap hooks, and close affordances.
- When a modal action succeeds, respond with `HX-Trigger: {"modal:close": true, "refresh:fragments": {"targets": ["#fragment-id"]}}` to close the dialog and re-fetch any dependent fragments via their `hx-trigger="refresh"` listeners.
- Fragments that should respond to modal workflows must declare `hx-trigger="refresh from:body"` alongside their `hx-get`/`hx-target` attributes so they react to the global refresh events.
- Always render an empty `<div id="modal" class="modal hidden" aria-hidden="true" data-modal-state="closed" data-modal-open="false" hx-swap-oob="true"></div>` in htmx responses when a modal should be cleared to avoid leaving stale markup in the container.
