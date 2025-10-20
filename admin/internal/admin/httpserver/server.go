package httpserver

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/httpserver/ui"
	"finitefield.org/hanko-admin/public"
)

// Config holds runtime options for the admin HTTP server.
type Config struct {
	Address          string
	BasePath         string
	LoginPath        string
	Authenticator    custommw.Authenticator
	CSRFCookieName   string
	CSRFCookiePath   string
	CSRFCookieSecure bool
	CSRFHeaderName   string
}

// New constructs the HTTP server with middleware stack and embedded assets.
func New(cfg Config) *http.Server {
	router := chi.NewRouter()
	router.Use(chimw.RequestID)
	router.Use(chimw.RealIP)
	router.Use(chimw.Logger)
	router.Use(chimw.Recoverer)
	router.Use(chimw.Timeout(60 * time.Second))

	staticContent, err := public.StaticFS()
	if err != nil {
		log.Fatalf("embed static: %v", err)
	}
	router.Handle("/public/static/*", http.StripPrefix("/public/static/", http.FileServer(http.FS(staticContent))))

	basePath := normalizeBasePath(cfg.BasePath)
	loginPath := resolveLoginPath(basePath, cfg.LoginPath)

	authenticator := cfg.Authenticator
	if authenticator == nil {
		authenticator = custommw.DefaultAuthenticator()
	}

	csrfCfg := custommw.CSRFConfig{
		CookieName: cfg.CSRFCookieName,
		CookiePath: firstNonEmpty(cfg.CSRFCookiePath, basePath),
		HeaderName: cfg.CSRFHeaderName,
		Secure:     cfg.CSRFCookieSecure,
	}

	mountAdminRoutes(router, basePath, routeOptions{
		Authenticator: authenticator,
		LoginPath:     loginPath,
		CSRF:          csrfCfg,
	})

	return &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

type routeOptions struct {
	Authenticator custommw.Authenticator
	LoginPath     string
	CSRF          custommw.CSRFConfig
}

func mountAdminRoutes(router chi.Router, base string, opts routeOptions) {
	if base != "/" {
		router.Get(base, ui.DashboardHandler)
	}

	router.Route(base, func(r chi.Router) {
		r.Use(custommw.HTMX())
		r.Use(custommw.NoStore())
		r.Use(custommw.Auth(opts.Authenticator, opts.LoginPath))
		r.Use(custommw.CSRF(opts.CSRF))

		r.Get("/", ui.DashboardHandler)
		// Future admin routes will be registered here.
	})
}

func normalizeBasePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return "/admin"
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimRight(p, "/")
	}
	if p == "" {
		return "/"
	}
	return p
}

func resolveLoginPath(base string, override string) string {
	if strings.TrimSpace(override) != "" {
		return override
	}
	if base == "/" {
		return "/login"
	}
	return base + "/login"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// RegisterFragment registers a GET handler intended for htmx fragment rendering.
func RegisterFragment(r chi.Router, pattern string, handler http.HandlerFunc) {
	r.With(custommw.RequireHTMX()).Get(pattern, handler)
}
