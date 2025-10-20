package middleware

import (
	"net/http"

	"finitefield.org/hanko-admin/internal/admin/rbac"
)

// RequireRole aborts the request with 403 Forbidden when the authenticated user
// lacks any of the provided roles.
func RequireRole(required ...rbac.Role) func(http.Handler) http.Handler {
	roles := rbac.Roles(required)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				forbidden(w, r)
				return
			}
			if len(roles) > 0 && !rbac.HasAnyRole(user.Roles, roles) {
				forbidden(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireCapability aborts the request when the authenticated user lacks the required capability.
func RequireCapability(capability rbac.Capability) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := UserFromContext(r.Context())
			if !ok || user == nil {
				forbidden(w, r)
				return
			}
			if !rbac.HasCapability(user.Roles, capability) {
				forbidden(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	if IsHTMXRequest(r.Context()) {
		w.Header().Set("HX-Refresh", "true")
	}
	http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
}
