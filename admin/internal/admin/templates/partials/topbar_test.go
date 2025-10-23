package partials

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/rbac"
)

func TestTopbarActionsRenderForAdmin(t *testing.T) {
	t.Parallel()

	ctx := buildTopbarContext(t, "/admin/orders", "Staging")
	ctx = middleware.ContextWithUser(ctx, &middleware.User{
		UID:   "ops-1",
		Email: "ops@example.com",
		Roles: []string{string(rbac.RoleAdmin)},
		Token: "token",
	})

	doc := renderTopbar(t, ctx)

	badge := doc.Find("[data-environment-badge] span[aria-hidden='true']")
	require.Equal(t, 1, badge.Length(), "environment badge should render")
	require.Equal(t, "STG", strings.TrimSpace(badge.Text()), "staging environment should render STG badge")

	search := doc.Find("[data-topbar-search-trigger]")
	require.Equal(t, 1, search.Length(), "search shortcut should be visible for admin")
	require.Equal(t, "/admin/search?overlay=1", search.AttrOr("hx-get", ""))
	require.Equal(t, "/admin/search", search.AttrOr("data-search-href", ""))

	notifications := doc.Find("[data-notifications-root]")
	require.Equal(t, 1, notifications.Length(), "notifications badge should render for admin")
	require.Equal(t, "/admin/notifications/badge", notifications.AttrOr("hx-get", ""))

	userMenu := doc.Find("[data-user-menu]")
	require.Equal(t, 1, userMenu.Length(), "user menu should render")
	require.Equal(t, "/admin/logout", doc.Find("[data-user-menu-logout]").AttrOr("action", ""), "logout form should post to logout route")
	require.Equal(t, 1, doc.Find("[data-user-menu-logout] input[name=\"_csrf\"]").Length(), "logout form should include CSRF field")
}

func TestTopbarHidesRestrictedActions(t *testing.T) {
	t.Parallel()

	ctx := buildTopbarContext(t, "/admin/catalog/products", "Development")
	ctx = middleware.ContextWithUser(ctx, &middleware.User{
		UID:   "marketer",
		Email: "marketing@example.com",
		Roles: []string{string(rbac.RoleMarketing)},
	})

	doc := renderTopbar(t, ctx)

	require.Equal(t, 0, doc.Find("[data-topbar-search-trigger]").Length(), "search shortcut must be hidden without capability")
	require.Equal(t, 0, doc.Find("[data-notifications-root]").Length(), "notifications badge must be hidden without capability")

	userSummary := doc.Find("[data-user-menu] .truncate.text-sm")
	require.Equal(t, 1, userSummary.Length(), "user menu should still render for marketing role")
	require.Contains(t, strings.TrimSpace(userSummary.Text()), "marketer")
}

func buildTopbarContext(t *testing.T, requestPath string, environment string) context.Context {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, requestPath, nil)
	rec := httptest.NewRecorder()

	var ctx context.Context
	handler := middleware.RequestInfoMiddleware("/admin")(middleware.Environment(environment)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		ctx = r.Context()
	})))
	handler.ServeHTTP(rec, req)

	require.NotNil(t, ctx, "middleware stack must provide context")
	return ctx
}

func renderTopbar(t *testing.T, ctx context.Context) *goquery.Document {
	t.Helper()

	var buf bytes.Buffer
	err := TopbarActions().Render(ctx, &buf)
	require.NoError(t, err, "topbar must render without error")

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err, "html must parse")
	return doc
}
