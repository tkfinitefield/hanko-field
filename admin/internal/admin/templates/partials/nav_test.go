package partials

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/navigation"
	"finitefield.org/hanko-admin/internal/admin/rbac"
)

func TestHasVisibleItemsRespectRoles(t *testing.T) {
	t.Parallel()

	// Support staff should not see system-level tools.
	ctx := middleware.ContextWithUser(context.Background(), &middleware.User{
		Roles: []string{string(rbac.RoleSupport)},
	})
	menu := navigation.BuildMenu("/admin")

	var system navigation.MenuGroup
	var operations navigation.MenuGroup
	for _, group := range menu {
		switch group.Key {
		case "system":
			system = group
		case "operations":
			operations = group
		}
	}

	require.NotEmpty(t, system.Items, "system group must contain navigation items")
	require.False(t, hasVisibleItems(system, ctx), "support role must not see system group")

	require.True(t, hasVisibleItems(operations, ctx), "support role should access operations features")
}

func TestVisibleItemsFiltersByCapability(t *testing.T) {
	t.Parallel()

	group := navigation.MenuGroup{
		Key:        "system",
		Label:      "システム",
		Capability: rbac.CapSystemTasks,
		Items: []navigation.MenuItem{
			{
				Key:         "system-tasks",
				Label:       "タスク/ジョブ",
				Capability:  rbac.CapSystemTasks,
				Href:        "/admin/system/tasks",
				Pattern:     "/admin/system/tasks",
				MatchPrefix: true,
			},
			{
				Key:         "audit-logs",
				Label:       "監査ログ",
				Capability:  rbac.CapAuditLogView,
				Href:        "/admin/audit-logs",
				Pattern:     "/admin/audit-logs",
				MatchPrefix: true,
			},
		},
	}

	ctxSupport := middleware.ContextWithUser(context.Background(), &middleware.User{
		Roles: []string{string(rbac.RoleSupport)},
	})
	ctxOps := middleware.ContextWithUser(context.Background(), &middleware.User{
		Roles: []string{string(rbac.RoleOps)},
	})

	require.Empty(t, visibleItems(group, ctxSupport), "support role lacks group capability so items must be hidden")

	items := visibleItems(group, ctxOps)
	require.Len(t, items, 1, "ops role should only see allowed items")
	require.Equal(t, "system-tasks", items[0].Key)
}

func TestSidebarRenderingFiltersAndHighlights(t *testing.T) {
	t.Parallel()

	menu := navigation.BuildMenu("/admin")

	req := httptest.NewRequest(http.MethodGet, "/admin/orders/123", nil)
	var ctx context.Context
	handler := middleware.RequestInfoMiddleware("/admin")(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	}))
	handler.ServeHTTP(httptest.NewRecorder(), req)
	ctx = middleware.ContextWithUser(ctx, &middleware.User{
		Roles: []string{string(rbac.RoleSupport)},
	})

	var buf bytes.Buffer
	err := Sidebar(menu).Render(ctx, &buf)
	require.NoError(t, err)

	doc := parseHTML(t, buf.Bytes())

	// Support role should not see system tools.
	require.Equal(t, 0, doc.Find(`a[href="/admin/system/tasks"]`).Length(), "system tasks link must be hidden")

	ordersLink := doc.Find(`a[href="/admin/orders"]`)
	require.Equal(t, 1, ordersLink.Length(), "orders link should render")
	require.Equal(t, "page", ordersLink.AttrOr("aria-current", ""), "active route highlights current page")
	require.Contains(t, ordersLink.AttrOr("class", ""), "bg-slate-900", "active link should use highlighted class")
}

func parseHTML(t *testing.T, body []byte) *goquery.Document {
	t.Helper()

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	require.NoError(t, err)
	return doc
}
