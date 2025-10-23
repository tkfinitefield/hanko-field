package partials

import (
	"context"
	"testing"

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
