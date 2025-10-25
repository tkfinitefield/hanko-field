package testutil

import (
	"net/http/httptest"
	"testing"

	"finitefield.org/hanko-admin/internal/admin/dashboard"
	"finitefield.org/hanko-admin/internal/admin/httpserver"
	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	adminorders "finitefield.org/hanko-admin/internal/admin/orders"
	adminproduction "finitefield.org/hanko-admin/internal/admin/production"
	"finitefield.org/hanko-admin/internal/admin/profile"
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

// WithProfileService wires a custom profile service implementation.
func WithProfileService(service profile.Service) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.ProfileService = service
	}
}

// WithDashboardService wires a custom dashboard service implementation.
func WithDashboardService(service dashboard.Service) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.DashboardService = service
	}
}

// WithOrdersService wires a custom orders service implementation.
func WithOrdersService(service adminorders.Service) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.OrdersService = service
	}
}

// WithProductionService overrides the production service implementation.
func WithProductionService(service adminproduction.Service) ServerOption {
	return func(cfg *httpserver.Config) {
		cfg.ProductionService = service
	}
}

// NewServer constructs an httptest server running the admin HTTP stack with sensible defaults.
func NewServer(t testing.TB, opts ...ServerOption) *httptest.Server {
	t.Helper()

	cfg := httpserver.Config{
		Address:           ":0",
		BasePath:          "/admin",
		LoginPath:         "",
		CSRFCookieName:    "csrf_token",
		CSRFHeaderName:    "X-CSRF-Token",
		Authenticator:     middleware.DefaultAuthenticator(),
		DashboardService:  dashboard.NewStaticService(),
		ProfileService:    profile.NewStaticService(nil),
		OrdersService:     adminorders.NewStaticService(),
		ProductionService: adminproduction.NewStaticService(),
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	srv := httpserver.New(cfg)
	ts := httptest.NewServer(srv.Handler)
	t.Cleanup(ts.Close)
	return ts
}
