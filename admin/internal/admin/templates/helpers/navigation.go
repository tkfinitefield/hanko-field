package helpers

import (
	"context"
	"strings"

	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
)

// RequestPath returns the current request URL path for template helpers.
func RequestPath(ctx context.Context) string {
	return normalizeRoute(middleware.RequestPathFromContext(ctx))
}

// BasePath returns the configured admin base path.
func BasePath(ctx context.Context) string {
	return normalizeRoute(middleware.BasePathFromContext(ctx))
}

// NavActive reports whether the current request should highlight the menu item.
func NavActive(ctx context.Context, pattern string, prefix bool) bool {
	current := RequestPath(ctx)
	target := normalizeRoute(pattern)

	if target == "" {
		return false
	}

	if prefix {
		if target == "/" {
			return current == "/"
		}
		if current == target {
			return true
		}
		return strings.HasPrefix(current, target+"/")
	}

	return current == target
}

func normalizeRoute(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
		if path == "" {
			return "/"
		}
	}
	return path
}
