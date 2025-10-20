package helpers

import (
	"context"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/rbac"
)

// HasCapability reports whether the authenticated user possesses the capability.
// Empty capability strings default to true to avoid guarding unconstrained actions.
func HasCapability(ctx context.Context, capability string) bool {
	if capability == "" {
		return true
	}
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return false
	}
	return rbac.HasCapability(user.Roles, rbac.Capability(capability))
}

// HasAnyRole reports whether the authenticated user has any of the provided roles.
// Accepts raw role strings to simplify usage from templates.
func HasAnyRole(ctx context.Context, roles ...string) bool {
	if len(roles) == 0 {
		return true
	}
	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return false
	}
	normalised := rbac.NormaliseRoles(roles)
	return rbac.HasAnyRole(user.Roles, normalised)
}
