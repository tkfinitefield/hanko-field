# Implement shared modals system with htmx targets and close triggers (ESC, overlay click).

**Parent Section:** 2. Shared Layout & Components
**Task ID:** 014

## Goal
Provide standardized modal system for htmx injections.

## Implementation Steps
1. Use shared mount: `<div id="modal-root"></div>` (already in base layout).
2. Add reusable modal component partial and JS for lifecycle (open/close, ESC/overlay click, focus trap).
3. Document fragment structure and HTMX usage.

## What Was Added
- Component template: `web/templates/partials/components/modal.tmpl` defines `c_modal` with props:
  - `ID` (string, optional; default `modal`)
  - `Title` (string)
  - `Size` (`sm|md|lg|xl`)
  - `BodyTmpl` + `BodyData` (slot-style) or `Body` (string)
  - `FooterTmpl` + `FooterData` (slot-style). If omitted, a default Close button is rendered.
  - Close controls emit `data-modal-close`; overlay has `data-modal-overlay`.

- Base layout JS augments modal behavior:
  - Opens on `htmx:afterSwap` into `#modal-root`; traps focus and prevents body scroll.
  - Closes on ESC key, overlay click, or any `[data-modal-close]` click.
  - Restores focus to the opener; exposes `window.hankoModalClose()`.

- Helper template funcs used: `dict`, `list`, `slot` (executes named template with data), `safe`.

## Usage (HTMX)
Open a modal by swapping the fragment into `#modal-root`:
```
<a hx-get="/modals/demo" hx-target="#modal-root" hx-swap="innerHTML">Open</a>
```
Server handler should render `c_modal` with desired props. Example demo route is implemented:
- GET `/modals/demo` → returns a simple modal fragment.

### Custom Body/Footer via Slots
Define templates (in any parsed file) and reference by name:
```
{{ define "product_body" }}<p class="text-sm">{{ .Desc }}</p>{{ end }}
{{ template "c_modal" (dict "Title" "Details" "BodyTmpl" "product_body" "BodyData" (dict "Desc" .Desc)) }}
```

### Close Behavior
- Overlay click or `[data-modal-close]` elements close the modal.
- Pressing ESC closes the modal.
- You can also call `hankoModalClose()` from custom scripts.

## Demo
- A button on `/templates` opens the demo modal via HTMX for quick manual verification.
