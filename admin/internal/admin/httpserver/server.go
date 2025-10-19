package httpserver

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"finitefield.org/hanko-admin/internal/admin/httpserver/ui"
	"finitefield.org/hanko-admin/public"
)

type Config struct {
	Address  string
	BasePath string
}

func New(cfg Config) *http.Server {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(60 * time.Second))

	staticContent, err := public.StaticFS()
	if err != nil {
		log.Fatalf("embed static: %v", err)
	}
	router.Handle("/public/static/*", http.StripPrefix("/public/static/", http.FileServer(http.FS(staticContent))))

	basePath := normalizeBasePath(cfg.BasePath)
	mountAdminRoutes(router, basePath)

	return &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
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

func mountAdminRoutes(router chi.Router, base string) {
	if base == "/" {
		router.Get("/", ui.DashboardHandler)
		return
	}

	router.Get(base, ui.DashboardHandler)

	router.Route(base, func(r chi.Router) {
		r.Get("/", ui.DashboardHandler)
		// Future admin routes will be registered here.
	})
}
