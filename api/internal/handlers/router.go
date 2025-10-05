package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RouteRegistrar registers a set of routes against the provided router.
type RouteRegistrar func(r chi.Router)

type routerConfig struct {
	basePath    string
	middlewares []func(http.Handler) http.Handler

	public   RouteRegistrar
	me       RouteRegistrar
	designs  RouteRegistrar
	cart     RouteRegistrar
	orders   RouteRegistrar
	admin    RouteRegistrar
	webhooks RouteRegistrar
	internal RouteRegistrar
}

// Option customises the router configuration before construction.
type Option func(*routerConfig)

const (
	defaultAPIPrefix  = "/api/v1"
	defaultTimeout    = 60 * time.Second
	errorContentType  = "application/json"
	errorNotFoundCode = "route_not_found"
)

// NewRouter constructs the chi router with shared middleware and expected route groups.
func NewRouter(opts ...Option) chi.Router {
	cfg := routerConfig{
		basePath: defaultAPIPrefix,
		middlewares: []func(http.Handler) http.Handler{
			middleware.RequestID,
			middleware.RealIP,
			middleware.Logger,
			middleware.Recoverer,
			middleware.Timeout(defaultTimeout),
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	r := chi.NewRouter()

	for _, mw := range cfg.middlewares {
		if mw != nil {
			r.Use(mw)
		}
	}

	r.NotFound(func(w http.ResponseWriter, req *http.Request) {
		writeJSONError(w, http.StatusNotFound, errorNotFoundCode, fmt.Sprintf("no route for %s", req.URL.Path))
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, req *http.Request) {
		writeJSONError(w, http.StatusMethodNotAllowed, "method_not_allowed", fmt.Sprintf("method %s not allowed on %s", req.Method, req.URL.Path))
	})

	r.Get("/healthz", health)

	r.Route(cfg.basePath, func(api chi.Router) {
		mount := func(path string, registrar RouteRegistrar, name string) {
			api.Route(path, func(group chi.Router) {
				if registrar != nil {
					registrar(group)
					return
				}
				registerNotImplemented(group, name)
			})
		}

		mount("/public", cfg.public, "public")
		mount("/me", cfg.me, "me")
		mount("/designs", cfg.designs, "designs")
		mount("/cart", cfg.cart, "cart")
		mount("/orders", cfg.orders, "orders")
		mount("/admin", cfg.admin, "admin")
		mount("/webhooks", cfg.webhooks, "webhooks")
		mount("/internal", cfg.internal, "internal")
	})

	return r
}

// WithMiddlewares appends additional global middleware to the router.
func WithMiddlewares(mw ...func(http.Handler) http.Handler) Option {
	return func(cfg *routerConfig) {
		cfg.middlewares = append(cfg.middlewares, mw...)
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

// WithInternalRoutes configures the registrar responsible for internal endpoints.
func WithInternalRoutes(reg RouteRegistrar) Option {
	return func(cfg *routerConfig) {
		cfg.internal = reg
	}
}

func registerNotImplemented(r chi.Router, name string) {
	handler := func(w http.ResponseWriter, req *http.Request) {
		writeJSONError(w, http.StatusNotImplemented, "not_implemented", fmt.Sprintf("%s routes not implemented", name))
	}
	r.HandleFunc("/*", handler)
	r.HandleFunc("/", handler)
	r.NotFound(handler)
	r.MethodNotAllowed(handler)
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", errorContentType)
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(map[string]any{
		"error":   code,
		"message": message,
		"status":  status,
	})
}
