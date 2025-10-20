# Create reusable components (hero, cards, tables, forms, button sets, skeleton loaders) with Tailwind variants.

**Parent Section:** 2. Shared Layout & Components
**Task ID:** 013

## Goal
Create shared component library (hero, cards, tables, forms, skeletons).

## Implementation Steps
1. Build partial templates or macros for each component with Tailwind classes.
2. Document usage patterns and context-specific modifiers.
3. Ensure components accessible (ARIA roles, keyboard navigation) and responsive.

## Components Added (Go templates + Tailwind)
- Buttons: `c_button`, `c_button_set` — variants: `primary`, `secondary`, `outline`, `ghost`, `danger`; sizes: `sm`, `md`, `lg`.
- Hero: `c_hero` — props: `Align` (`center`/`left`), `Bg` (`muted`/`brand`/plain), `Title`, `Subtitle`, `Eyebrow`, `Actions` (list of button props).
- Card: `c_card` — variants: `outlined`, `elevated`, `plain`; supports `Title`, `Description`, `MediaURL`, and content/footer slot templates via `ContentTmpl`/`ContentData`, `FooterTmpl`/`FooterData`.
- Table: `c_table` — `Columns` (list of `{Key, Label, Align}`), `Rows` (list of maps keyed by `Key`), `EmptyMessage`.
- Forms: `c_input`, `c_textarea`, `c_select` — support `Label`, `Help`, `Error`, `Required`, ARIA bindings.
- Skeletons: `c_skeleton_text`, `c_skeleton_card`, `c_skeleton_table`.

All component templates live under `web/templates/partials/components/*.tmpl` and are included automatically in page renders.

### Helper Template Functions
Added to template FuncMap (`web/cmd/web/main.go`):
- `dict(k1, v1, k2, v2, ...)` to build map props.
- `list(v1, v2, ...)` to build slices.
- `seq(n)` to generate `[0..n-1]` for skeleton loops.

### Usage Examples
Buttons
```
{{ template "c_button_set" (dict "Align" "left" "Buttons" (list 
  (dict "Label" "Primary" "Variant" "primary")
  (dict "Label" "Outline" "Variant" "outline")
)) }}
```

Hero
```
{{ template "c_hero" (dict 
  "Align" "center" "Bg" "muted"
  "Title" "Composable components"
  "Subtitle" "Build pages quickly."
  "Actions" (list (dict "Label" "Start" "Href" "/" "Variant" "primary"))
) }}
```

Card with custom body/footer slots
```
{{ define "product_card_body" }}<p class="text-sm text-gray-600">{{ .Desc }}</p>{{ end }}
{{ define "product_card_footer" }}{{ template "c_button" (dict "Label" "View" "Href" .URL "Variant" "outline" "Size" "sm") }}{{ end }}
{{ template "c_card" (dict "Variant" "outlined" "Title" .Name "ContentTmpl" "product_card_body" "ContentData" (dict "Desc" .Desc) "FooterTmpl" "product_card_footer" "FooterData" (dict "URL" .URL)) }}
```

Table
```
{{ $cols := (list (dict "Key" "name" "Label" "Name") (dict "Key" "price" "Label" "Price" "Align" "right")) }}
{{ $rows := (list (dict "name" "Classic" "price" "$12") (dict "name" "Deluxe" "price" "$24")) }}
{{ template "c_table" (dict "Columns" $cols "Rows" $rows) }}
```

Forms
```
{{ template "c_input" (dict "Name" "email" "Type" "email" "Label" "Email" "Placeholder" "you@example.com" "Required" true) }}
```

Skeletons
```
{{ template "c_skeleton_table" (dict "Cols" 4 "Rows" 6) }}
```

### Accessibility
- Buttons include focus rings and `aria-label` support; disabled state uses `aria-disabled`.
- Form controls bind `aria-describedby`/`aria-errormessage` to help/error text.
- Tables use proper `<th scope="col">` headers and readable contrast.

### Demo Page
The `/templates` page showcases all components with sample data to aid development.
