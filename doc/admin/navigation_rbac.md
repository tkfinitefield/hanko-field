# Admin Navigation Taxonomy & RBAC Mapping

## Roles

| Role Key | Description | Typical Persona |
|----------|-------------|------------------|
| `admin` | Full platform administrator with access to all modules, including system settings and staff management. | Platform Owner / CTO |
| `ops` | Operations lead handling orders, production, shipping, and system jobs. | Operations Lead |
| `support` | Customer support agent handling customer inquiries, refunds, and review moderation. | CS Agent |
| `marketing` | Marketing manager managing catalog, content, and promotions. | Marketing Manager |

Notes:
- Multiple roles can be assigned per user through Firebase custom claims (e.g., `["ops", "support"]`).
- `admin` is superset; include in every permissions set for clarity.

## Navigation Taxonomy

Sidebar groups mirror information architecture in `doc/admin/admin_design.md`. Each group owns specific routes and breadcrumbs.

| Group | Sidebar Label | Primary Routes | Breadcrumb Pattern |
|-------|----------------|----------------|--------------------|
| `dashboard` | ダッシュボード | `/admin`, `/admin/fragments/kpi`, `/admin/fragments/alerts` | `Dashboard` |
| `orders` | 受注管理 | `/admin/orders`, `/admin/orders/{id}`, `/admin/shipments/*`, `/admin/production/*` | `Orders › {OrderID}` |
| `catalog` | カタログ | `/admin/catalog/templates|fonts|materials|products` | `Catalog › {Kind}` |
| `content` | コンテンツ | `/admin/content/guides`, `/admin/content/pages` | `Content › {Page}` |
| `marketing` | マーケ | `/admin/promotions`, `/admin/reviews` | `Marketing › {Feature}` |
| `customers` | 顧客 | `/admin/customers`, `/admin/customers/{uid}` | `Customers › {UID}` |
| `system` | システム | `/admin/audit-logs`, `/admin/system/*`, `/admin/org/*` | `System › {Feature}` |

Additional utility routes (search, profile, notifications) sit outside the main taxonomy but inherit parent visibility (`/admin/search`, `/admin/notifications`, `/admin/profile`).

## Role Visibility Matrix

| Sidebar Group | admin | ops | support | marketing | Rationale |
|---------------|:-----:|:---:|:-------:|:---------:|-----------|
| Dashboard | ✅ | ✅ | ✅ | ✅ | Shared overview widgets. |
| Orders | ✅ | ✅ | ✅ | ◻️ | Operational & CS workflows require order visibility. |
| Catalog | ✅ | ✅ | ◻️ | ✅ | Ops manage materials/SKU, marketing curates templates/promos. |
| Content | ✅ | ◻️ | ◻️ | ✅ | Marketing-led content edits. |
| Marketing | ✅ | ◻️ | ✅* | ✅ | Review moderation shared with support; promotions for marketing. |
| Customers | ✅ | ✅ | ✅ | ◻️ | Ops & support need customer data; marketing excluded for privacy. |
| System | ✅ | ✅† | ◻️ | ◻️ | Ops need limited system dashboards (tasks/jobs); audits & staff management remain admin-only. |

Legend: ✅ visible, ✅* conditional sub-items, ✅† partial visibility via child route restrictions. Use child-level overrides for finer control (see next section).

## Route-Level Permissions

Detailed mapping supports conditional rendering of nested links or actions.

| Route | Description | Required Roles |
|-------|-------------|----------------|
| `/admin/orders` | Order list and filters | `admin`, `ops`, `support` |
| `/admin/orders/{id}` | Order detail tabs | `admin`, `ops`, `support` |
| `/admin/orders/{id}/modal/refund` | Refund modal | `admin`, `support` |
| `/admin/shipments/tracking` | Shipment monitor | `admin`, `ops` |
| `/admin/production/queues` | Production Kanban | `admin`, `ops` |
| `/admin/catalog/templates` | Template management | `admin`, `ops`, `marketing` |
| `/admin/catalog/fonts` | Font management | `admin`, `marketing` |
| `/admin/content/guides` | Guide CMS | `admin`, `marketing` |
| `/admin/promotions` | Promotions CRUD | `admin`, `marketing` |
| `/admin/promotions/{id}/usages` | Promo usage analytics | `admin`, `marketing` |
| `/admin/reviews` | Review moderation | `admin`, `support`, `marketing` |
| `/admin/customers` | Customer list/detail | `admin`, `ops`, `support` |
| `/admin/notifications` | Alerts feed | `admin`, `ops`, `support` |
| `/admin/audit-logs` | Audit log viewer | `admin` |
| `/admin/system/tasks` | Background job monitor | `admin`, `ops` |
| `/admin/system/counters` | Counter management | `admin` |
| `/admin/org/staff` | Staff management | `admin` |
| `/admin/profile` | Personal settings/2FA | `admin`, `ops`, `support`, `marketing` |
| `/admin/search` | Cross-entity search | `admin`, `ops`, `support` |

## Configuration Shape (Go)

```go
var SidebarConfig = []nav.Section{
    {
        ID:       "orders",
        Label:    "受注管理",
        Roles:    nav.Roles{"admin", "ops", "support"},
        Children: []nav.Link{
            {ID: "orders.index", Path: "/admin/orders", Roles: nav.Roles{"admin", "ops", "support"}},
            {ID: "shipments.tracking", Path: "/admin/shipments/tracking", Roles: nav.Roles{"admin", "ops"}},
            {ID: "production.queues", Path: "/admin/production/queues", Roles: nav.Roles{"admin", "ops"}},
        },
    },
}
```

- `nav.Roles` is a helper type implementing `Has(role string) bool`.
- Template renderer filters sections/links by intersecting user role claims with `Roles`.
- Routes absent from user scope should not render; server handlers additionally enforce RBAC middleware checks to prevent direct access.

## Testing Plan

- Unit tests exercising `FilterNavigation(userRoles)` to ensure sections hide when no matching role.
- Integration test for key combinations (e.g., `support` should see Orders, Marketing, Customers but not System).
- Golden tests verifying breadcrumbs per route follow taxonomy.

## Implementation Notes

- RBAC claims sourced from Firebase custom claims (`roles` array) cached per session.
- `admin` users implicitly gain all sections even if claims omit certain roles.
- Document updates feed `doc/admin/architecture.md` references and inform forthcoming task `012` for utility functions.

