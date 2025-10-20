# Admin Design Tokens

Tailwind configuration for the admin console defines a focused set of design tokens that align with the visual language captured in `doc/admin/admin_design.md`. These tokens live in `admin/tailwind.config.js` and are consumed via semantic utility classes and base components authored in `admin/web/styles/tailwind.css`.

## Palette

| Token | Description | Tailwind reference |
| --- | --- | --- |
| `brand.25` – `brand.950` | Primary "Hanko red" ramp used for CTA emphasis, active states, and focus outlines. | `bg-brand-600`, `text-brand-500`, `focus:ring-brand-500` |
| `surface.{DEFAULT,muted,subtle}` | Layered neutrals for backgrounds (app shell, cards, tables). | `bg-surface-muted`, `bg-surface-default` |
| `surface.overlay` | Semi-opaque overlay for modals and drawers. | `bg-[var(--surface-overlay)]` (applied via `.modal-overlay`) |
| `border.{subtle,bold}` | Borders and dividers (cards, tables, form controls). | `border-[var(--border-subtle)]`, `.table-wrapper` |
| `success`, `danger`, `warning`, `info` | Status colors mapped to semantic badge/toast styles. | `.badge-success`, `.toast-danger`, etc. |

Dark mode is handled via the `class` strategy (`darkMode: "class"`). Switching the `dark` class on `<html>` or `<body>` flips CSS variables declared in `@layer base` to their dark counterparts without changing component markup.

## Typography

- **Font families**: `Inter` is the default sans-serif (`font-sans`), while `Lexend` powers headline components (`font-display`).
- **Heading scale**: `h1`–`h4` utilities are pre-wired in base styles for consistent sizing and weight.
- **Body copy**: `text-sm` is the default for UI copy, with `text-xs` reserved for helper text and labels.

## Spacing, Radius, and Elevation

- **Spacing extensions**: `spacing.13`, `spacing.15`, and `spacing.18` fill gaps between Tailwind defaults for layout rhythm around cards and overlays.
- **Radius**: `rounded-xl`/`rounded-2xl` tokens deliver pill-style surfaces for cards and modals.
- **Shadows**: Custom shadows (`focus`, `surface`, `modal`, `toast`) establish hierarchy—e.g. `.card` uses `shadow-surface`, modals use `shadow-modal`.

## Base Components

Reusable component classes are defined in `admin/web/styles/tailwind.css` under `@layer components`. The key primitives are:

- **Buttons** (`.btn`, `.btn-{primary|secondary|outline|ghost|danger}`, size modifiers `.btn-sm/.btn-md/.btn-lg`, loading state `.btn-loading`).
- **Tables** (`.table-wrapper`, `.table`) with baked-in hover states, empty-state messaging, and spacing that matches dashboard mocks.
- **Forms** (`.form-field`, `.form-label`, `.form-control`, `.form-error`) to align inputs, selects, and textareas across fragments.
- **Modals** (`.modal-overlay`, `.modal-panel`, `.modal-header/body/footer`) with animation tokens (`animate-dialog-in`).
- **Toasts** (`.toast{,-success,-danger,-info,-warning}`, `.toast-actions`) ready for HTMX-triggered alerts.
- **Badges** (`.badge`, `.badge-{success|warning|danger|info}`) to surface statuses in tables and lists.

Each templ component in `internal/admin/templates/components` wraps these classes so htmx fragments compose in a consistent way:

- `components.Button` + `ButtonOptions`
- `components.TextInput`, `Select`, `TextArea`
- `components.Modal`
- `components.Toast`
- `components.Table`, `Card`, `Badge`

`helpers.ButtonClass`, `helpers.ToastClass`, and related utilities centralise class composition so variants stay in sync with Tailwind tokens.

## Build Pipeline

`make css` now executes Tailwind with `NODE_ENV=production`, enabling purge/minify passes and Autoprefixer within the Tailwind standalone binary. In watch mode the same tokens remain available without the production purge. Any custom post-processing should hook into the generated asset `admin/public/static/app.css`.

> When adding new UI primitives, extend the existing tokens before inventing new ad-hoc colors, spacings, or shadows. This keeps the admin console coherent across dashboards, tables, and modal workflows documented in `doc/admin/admin_design.md`.
