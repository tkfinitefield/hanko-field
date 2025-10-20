package httpserver

import (
	"crypto/rand"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/httpserver/ui"
	appsession "finitefield.org/hanko-admin/internal/admin/session"
	"finitefield.org/hanko-admin/public"
)

// Config holds runtime options for the admin HTTP server.
type Config struct {
	Address          string
	BasePath         string
	LoginPath        string
	Authenticator    custommw.Authenticator
	Session          SessionConfig
	SessionStore     custommw.SessionStore
	CSRFCookieName   string
	CSRFCookiePath   string
	CSRFCookieSecure bool
	CSRFHeaderName   string
}

// SessionConfig represents optional overrides for the admin session manager.
type SessionConfig struct {
	CookieName       string
	CookiePath       string
	CookieDomain     string
	CookieSecure     bool
	CookieHTTPOnly   *bool
	CookieSameSite   http.SameSite
	IdleTimeout      time.Duration
	Lifetime         time.Duration
	RememberLifetime time.Duration
	HashKey          []byte
	BlockKey         []byte
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

	sessionStore := cfg.SessionStore
	if sessionStore == nil {
		sessionStore = mustBuildSessionStore(cfg.Session, basePath)
	}

	csrfCfg := custommw.CSRFConfig{
		CookieName: cfg.CSRFCookieName,
		CookiePath: firstNonEmpty(cfg.CSRFCookiePath, basePath),
		HeaderName: cfg.CSRFHeaderName,
		Secure:     cfg.CSRFCookieSecure,
	}

	mountAdminRoutes(router, basePath, routeOptions{
		SessionStore:  sessionStore,
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
	SessionStore  custommw.SessionStore
	Authenticator custommw.Authenticator
	LoginPath     string
	CSRF          custommw.CSRFConfig
}

func mountAdminRoutes(router chi.Router, base string, opts routeOptions) {
	authHandlers := newAuthHandlers(opts.Authenticator, base, opts.LoginPath)

	var shared []func(http.Handler) http.Handler
	if opts.SessionStore != nil {
		shared = append(shared, custommw.Session(opts.SessionStore))
	}
	shared = append(shared, custommw.HTMX(), custommw.NoStore())

	loginChain := router.With(shared...)
	loginChain = loginChain.With(custommw.CSRF(opts.CSRF))
	loginChain.Get(authHandlers.loginPath, authHandlers.LoginForm)
	loginChain.Post(authHandlers.loginPath, authHandlers.LoginSubmit)

	router.Route(base, func(r chi.Router) {
		for _, mw := range shared {
			r.Use(mw)
		}

		r.Group(func(protected chi.Router) {
			protected.Use(custommw.Auth(opts.Authenticator, opts.LoginPath))
			protected.Use(custommw.CSRF(opts.CSRF))

			protected.Get("/", ui.DashboardHandler)
			protected.Get("/logout", authHandlers.Logout)
			protected.Post("/logout", authHandlers.Logout)
			// Future admin routes will be registered here.
		})
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

func mustBuildSessionStore(cfg SessionConfig, basePath string) custommw.SessionStore {
	hashKey := cfg.HashKey
	if len(hashKey) == 0 {
		hashKey = randomBytes(32)
		log.Printf("session: generated ephemeral hash key; set ADMIN_SESSION_HASH_KEY to persist sessions across restarts")
	}

	blockKey := cfg.BlockKey
	if blockKey == nil || len(blockKey) == 0 {
		blockKey = randomBytes(32)
	}

	path := firstNonEmpty(cfg.CookiePath, basePath, "/")
	httpOnly := true
	if cfg.CookieHTTPOnly != nil {
		httpOnly = *cfg.CookieHTTPOnly
	}

	manager, err := appsession.NewManager(appsession.Config{
		CookieName:       cfg.CookieName,
		HashKey:          hashKey,
		BlockKey:         blockKey,
		CookiePath:       path,
		CookieDomain:     cfg.CookieDomain,
		CookieSecure:     cfg.CookieSecure,
		CookieHTTPOnly:   &httpOnly,
		CookieSameSite:   cfg.CookieSameSite,
		IdleTimeout:      cfg.IdleTimeout,
		Lifetime:         cfg.Lifetime,
		RememberLifetime: cfg.RememberLifetime,
	})
	if err != nil {
		log.Fatalf("session manager init failed: %v", err)
	}
	return manager
}

func randomBytes(length int) []byte {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		log.Fatalf("generate random bytes: %v", err)
	}
	return buf
}
