package middleware

import (
	"context"
	"net/http"
	"strings"
)

type environmentContextKey struct{}

// Environment attaches the deployment environment label to the request context
// so templates and handlers can render environment-specific chrome/state.
// Empty values default to "Development".
func Environment(value string) func(http.Handler) http.Handler {
	label := strings.TrimSpace(value)
	if label == "" {
		label = "Development"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), environmentContextKey{}, label)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// EnvironmentFromContext returns the environment label registered for the
// current request, defaulting to "Development" when unavailable.
func EnvironmentFromContext(ctx context.Context) string {
	if ctx == nil {
		return "Development"
	}
	if value, ok := ctx.Value(environmentContextKey{}).(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return "Development"
}
