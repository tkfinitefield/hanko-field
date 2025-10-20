package dashboard

import (
	"time"

	"github.com/a-h/templ"

	"finitefield.org/hanko-admin/internal/admin/templates/helpers"
	"finitefield.org/hanko-admin/internal/admin/templates/partials"
)

type orderRow struct {
	ID       string
	Customer string
	Total    int64
	Placed   time.Time
}

func navigationItems() []partials.NavItem {
	return []partials.NavItem{
		{Label: helpers.I18N("admin.nav.dashboard"), Href: "/admin", Active: true},
		{Label: helpers.I18N("admin.nav.orders"), Href: "/admin/orders"},
		{Label: helpers.I18N("admin.nav.catalog"), Href: "/admin/catalog"},
	}
}

func breadcrumbItems() []partials.Breadcrumb {
	return []partials.Breadcrumb{{Label: helpers.I18N("admin.dashboard.breadcrumb")}}
}

func orderHeaders() []string {
	return []string{"Order", "Customer", "Total", "Placed"}
}

func sampleOrders() []orderRow {
	now := time.Now()
	return []orderRow{
		{ID: "#1001", Customer: "山田 太郎", Total: 158000, Placed: now.Add(-2 * time.Hour)},
		{ID: "#1000", Customer: "鈴木 花子", Total: 9800, Placed: now.Add(-26 * time.Hour)},
	}
}

func orderRowsComponents(orders []orderRow) [][]templ.Component {
	rows := make([][]templ.Component, 0, len(orders))
	for _, o := range orders {
		rows = append(rows, []templ.Component{
			helpers.TextComponent(o.ID),
			helpers.TextComponent(o.Customer),
			helpers.TextComponent(helpers.Currency(o.Total, "JPY")),
			helpers.TextComponent(helpers.Relative(o.Placed)),
		})
	}
	return rows
}
