# Admin Template Conventions

## Directory Layout

- `internal/admin/templates/layouts` – page shells (`base`, `modal`) and shared chrome components.
- `internal/admin/templates/partials` – reusable view fragments such as the sidebar and breadcrumbs.
- `internal/admin/templates/components` – atomic UI components (buttons, cards, tables, badges) usable across features.
- `internal/admin/templates/dashboard` – feature-specific templates (example implementation).
- `internal/admin/templates/helpers` – Go helper functions for formatting (currency, relative time, navigation classes) and component utilities.

Generated `*_templ.go` files live alongside their `.templ` sources and must remain checked in to keep builds reproducible.

## Helper Functions

Located in `internal/admin/templates/helpers/format.go`:

- `Currency(amount int64, currency string)` – renders monetised values using ISO codes (minor units input).
- `Date(ts time.Time, layout string)` – formats timestamps (defaults to `2006-01-02 15:04 MST`).
- `Relative(ts time.Time)` – returns coarse "time ago" strings.
- `I18N(key string, args ...any)` – placeholder translation helper for future localization.
- `NavClass(active bool)` and `BadgeClass(tone string)` – utility class helpers for navigation/badges.
- `TextComponent(value string)` & `TableRows(rows [][]string)` – convert raw strings to templ components for composition.

## Usage Patterns

- Layouts accept navigation + breadcrumb slices and body components: `layouts.Base(title, navItems, breadcrumbs, content)`.
- Partials expose typed structs (`partials.NavItem`, `partials.Breadcrumb`) to keep data consistent across pages.
- Components expose templ functions (`components.Card`, `components.Table`, `components.Button`, `components.Text`) to render UI atoms.
- Feature templates import the layout/components/helpers packages and delegate data shaping to local Go helpers (`dashboard/data.go`).
- Fragment routes should return only the relevant component (`components.Table(...)`, etc.) while pages compose via `layouts.Base`.

## Regeneration

Run `make templ` (or `go generate ./internal/admin/templates/...`) whenever `.templ` files change. `go test ./...` ensures helper packages compile and that generated files remain in-sync.
