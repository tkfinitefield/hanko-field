package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hanko-field/api/internal/platform/httpx"
)

// RouteRegistrar registers a set of routes against the provided router.
type RouteRegistrar func(r chi.Router)

type routerConfig struct {
	basePath    string
	middlewares []func(http.Handler) http.Handler
	health      *HealthHandlers

	public   RouteRegistrar
	me       RouteRegistrar
	designs  RouteRegistrar
	nameMaps RouteRegistrar
	cart     RouteRegistrar
	orders   RouteRegistrar
	admin    RouteRegistrar
	webhooks RouteRegistrar
	internal RouteRegistrar

	webhookMiddlewares  []func(http.Handler) http.Handler
	internalMiddlewares []func(http.Handler) http.Handler
}

// Option customises the router configuration before construction.
type Option func(*routerConfig)

const (
	defaultAPIPrefix  = "/api/v1"
	defaultTimeout    = 60 * time.Second
	errorNotFoundCode = "route_not_found"
)

// NewRouter constructs the chi router with shared middleware and expected route groups.
func NewRouter(opts ...Option) chi.Router {
	cfg := routerConfig{
		basePath: defaultAPIPrefix,
		middlewares: []func(http.Handler) http.Handler{
			middleware.RequestID,
			middleware.RealIP,
			middleware.Timeout(defaultTimeout),
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()

	if cfg.health == nil {
		cfg.health = NewHealthHandlers()
	}

	for _, mw := range cfg.middlewares {
		if mw != nil {
			r.Use(mw)
		}
	}

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		httpx.WriteError(req.Context(), w, httpx.NewError(errorNotFoundCode, fmt.Sprintf("no route for %s", req.URL.Path), http.StatusNotFound))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		httpx.WriteError(req.Context(), w, httpx.NewError("method_not_allowed", fmt.Sprintf("method %s not allowed on %s", req.Method, req.URL.Path), http.StatusMethodNotAllowed))
	})

	r.Get("/healthz", cfg.health.Healthz)
	r.Get("/readyz", cfg.health.Readyz)

	r.Route(cfg.basePath, func(api chi.Router) {
		mount := func(path string, registrar RouteRegistrar, name string, groupMW []func(http.Handler) http.Handler) {
			api.Route(path, func(group chi.Router) {
				for _, mw := range groupMW {
					if mw != nil {
						group.Use(mw)
					}
				}
				if registrar != nil {
					registrar(group)
					return
				}
				registerNotImplemented(group, name)
			})
		}

		mount("/public", cfg.public, "public", nil)
		mount("/me", cfg.me, "me", nil)
		mount("/designs", cfg.designs, "designs", nil)
		if cfg.nameMaps != nil {
			cfg.nameMaps(api)
		} else {
			registerNotImplementedRoute(api, "/name-mappings:convert", "nameMappings")
			registerNotImplementedRoute(api, "/name-mappings/{mappingId}:select", "nameMappings")
		}
		mount("/cart", cfg.cart, "cart", nil)
		mount("/orders", cfg.orders, "orders", nil)
		mount("/admin", cfg.admin, "admin", nil)
		mount("/webhooks", cfg.webhooks, "webhooks", cfg.webhookMiddlewares)
		mount("/internal", cfg.internal, "internal", cfg.internalMiddlewares)
	})

	return r
}

// WithMiddlewares appends additional global middleware to the router.
func WithMiddlewares(mw ...func(http.Handler) http.Handler) Option {
	return func(cfg *routerConfig) {
		cfg.middlewares = append(cfg.middlewares, mw...)
	}
}

// WithHealthHandlers overrides the handlers used for /healthz and /readyz endpoints.
func WithHealthHandlers(h *HealthHandlers) Option {
	return func(cfg *routerConfig) {
		cfg.health = h
	}
}

// WithPublicRoutes configures the registrar responsible for public endpoints.
func WithPublicRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.public = reg
	}
}

// WithMeRoutes configures the registrar responsible for user scoped endpoints.
func WithMeRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.me = reg
	}
}

// WithDesignRoutes configures the registrar responsible for design endpoints.
func WithDesignRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.designs = reg
	}
}

// WithNameMappingRoutes configures the registrar responsible for name mapping endpoints.
func WithNameMappingRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.nameMaps = reg
	}
}

// WithCartRoutes configures the registrar responsible for cart endpoints.
func WithCartRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.cart = reg
	}
}

// WithOrderRoutes configures the registrar responsible for order endpoints.
func WithOrderRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.orders = reg
	}
}

// WithAdminRoutes configures the registrar responsible for admin endpoints.
func WithAdminRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.admin = reg
	}
}

// WithWebhookRoutes configures the registrar responsible for webhook endpoints.
func WithWebhookRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.webhooks = reg
	}
}

// WithWebhookMiddlewares configures middlewares applied to the /webhooks group.
func WithWebhookMiddlewares(mw ...func(http.Handler) http.Handler) Option {
	return func(cfg *routerConfig) {
		cfg.webhookMiddlewares = append(cfg.webhookMiddlewares, mw...)
	}
}

// WithInternalRoutes configures the registrar responsible for internal endpoints.
func WithInternalRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.internal = reg
	}
}

// WithInternalMiddlewares configures middlewares applied to the /internal group.
func WithInternalMiddlewares(mw ...func(http.Handler) http.Handler) Option {
	return func(cfg *routerConfig) {
		cfg.internalMiddlewares = append(cfg.internalMiddlewares, mw...)
	}
}

func registerNotImplemented(r chi.Router, name string) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		httpx.WriteError(req.Context(), w, httpx.NewError("not_implemented", fmt.Sprintf("%s routes not implemented", name), http.StatusNotImplemented))
	}
	r.HandleFunc("/*", handler)
	r.HandleFunc("/", handler)
	r.NotFound(handler)
	r.MethodNotAllowed(handler)
}

func registerNotImplementedRoute(r chi.Router, path string, name string) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		httpx.WriteError(req.Context(), w, httpx.NewError("not_implemented", fmt.Sprintf("%s routes not implemented", name), http.StatusNotImplemented))
	}
	r.HandleFunc(path, handler)
}
