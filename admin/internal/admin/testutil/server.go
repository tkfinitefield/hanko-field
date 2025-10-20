package testutil

import (
	"net/http/httptest"
	"testing"

	"finitefield.org/hanko-admin/internal/admin/httpserver"
	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
)

// ServerOption customises the HTTP server configuration for tests.
type ServerOption func(*httpserver.Config)

// WithAuthenticator overrides the authenticator used by the admin server.
func WithAuthenticator(auth middleware.Authenticator) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.Authenticator = auth
	}
}

// WithBasePath sets a custom base path for the admin routes.
func WithBasePath(path string) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.BasePath = path
	}
}

// NewServer constructs an httptest server running the admin HTTP stack with sensible defaults.
func NewServer(t testing.TB, opts ...ServerOption) *httptest.Server {
	t.Helper()

	cfg := httpserver.Config{
		Address:        ":0",
		BasePath:       "/admin",
		LoginPath:      "",
		CSRFCookieName: "csrf_token",
		CSRFHeaderName: "X-CSRF-Token",
		Authenticator:  middleware.DefaultAuthenticator(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	srv := httpserver.New(cfg)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}
