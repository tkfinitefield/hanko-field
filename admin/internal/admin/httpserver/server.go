package httpserver

import (
	"crypto/rand"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"finitefield.org/hanko-admin/internal/admin/dashboard"
	custommw "finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
	"finitefield.org/hanko-admin/internal/admin/httpserver/ui"
	adminnotifications "finitefield.org/hanko-admin/internal/admin/notifications"
	adminorders "finitefield.org/hanko-admin/internal/admin/orders"
	"finitefield.org/hanko-admin/internal/admin/profile"
	"finitefield.org/hanko-admin/internal/admin/search"
	appsession "finitefield.org/hanko-admin/internal/admin/session"
	adminshipments "finitefield.org/hanko-admin/internal/admin/shipments"
	"finitefield.org/hanko-admin/public"
)

// Config holds runtime options for the admin HTTP server.
type Config struct {
	Address              string
	BasePath             string
	LoginPath            string
	Authenticator        custommw.Authenticator
	DashboardService     dashboard.Service
	ProfileService       profile.Service
	SearchService        search.Service
	NotificationsService adminnotifications.Service
	OrdersService        adminorders.Service
	ShipmentsService     adminshipments.Service
	Session              SessionConfig
	SessionStore         custommw.SessionStore
	CSRFCookieName       string
	CSRFCookiePath       string
	CSRFCookieSecure     bool
	CSRFHeaderName       string
	Environment          string
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
		log.Fatalf("admin: authenticator is required; refusing to start without configured authenticator")
	}

	sessionStore := cfg.SessionStore
	if sessionStore == nil {
		sessionStore = mustBuildSessionStore(cfg.Session, basePath)
	}

	environment := strings.TrimSpace(cfg.Environment)

	csrfCfg := custommw.CSRFConfig{
		CookieName: cfg.CSRFCookieName,
		CookiePath: firstNonEmpty(cfg.CSRFCookiePath, basePath),
		HeaderName: cfg.CSRFHeaderName,
		Secure:     cfg.CSRFCookieSecure,
	}

	uiHandlers := ui.NewHandlers(ui.Dependencies{
		DashboardService:     cfg.DashboardService,
		ProfileService:       cfg.ProfileService,
		SearchService:        cfg.SearchService,
		NotificationsService: cfg.NotificationsService,
		OrdersService:        cfg.OrdersService,
		ShipmentsService:     cfg.ShipmentsService,
	})

	mountAdminRoutes(router, basePath, routeOptions{
		SessionStore:  sessionStore,
		Authenticator: authenticator,
		LoginPath:     loginPath,
		CSRF:          csrfCfg,
		UI:            uiHandlers,
		Environment:   environment,
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
	UI            *ui.Handlers
	Environment   string
}

func mountAdminRoutes(router chi.Router, base string, opts routeOptions) {
	authHandlers := newAuthHandlers(opts.Authenticator, base, opts.LoginPath)

	var shared []func(http.Handler) http.Handler
	if opts.SessionStore != nil {
		shared = append(shared, custommw.Session(opts.SessionStore))
	}
	shared = append(shared,
		custommw.RequestInfoMiddleware(base),
		custommw.HTMX(),
		custommw.NoStore(),
		custommw.Environment(opts.Environment),
	)

	loginChain := router.With(shared...)
	loginChain = loginChain.With(custommw.CSRF(opts.CSRF))
	loginChain.Get(authHandlers.loginPath, authHandlers.LoginForm)
	loginChain.Post(authHandlers.loginPath, authHandlers.LoginSubmit)

	uiHandlers := opts.UI
	if uiHandlers == nil {
		uiHandlers = ui.NewHandlers(ui.Dependencies{})
	}

	router.Route(base, func(r chi.Router) {
		for _, mw := range shared {
			r.Use(mw)
		}

		r.Group(func(protected chi.Router) {
			protected.Use(custommw.Auth(opts.Authenticator, opts.LoginPath))
			protected.Use(custommw.CSRF(opts.CSRF))

			protected.Get("/", uiHandlers.Dashboard)
			protected.Get("/fragments/kpi", uiHandlers.DashboardKPIs)
			protected.Get("/fragments/alerts", uiHandlers.DashboardAlerts)
			protected.Route("/profile", func(pr chi.Router) {
				pr.Get("/", uiHandlers.ProfilePage)
				pr.Get("/mfa/totp", uiHandlers.MFATOTPStart)
				pr.Post("/mfa/totp", uiHandlers.MFATOTPConfirm)
				pr.Post("/mfa/email", uiHandlers.EmailMFAEnable)
				pr.Post("/mfa/disable", uiHandlers.DisableMFA)
				pr.Get("/api-keys/new", uiHandlers.NewAPIKeyForm)
				pr.Post("/api-keys", uiHandlers.CreateAPIKey)
				pr.Post("/api-keys/{keyID}/revoke", uiHandlers.RevokeAPIKey)
				pr.Post("/sessions/{sessionID}/revoke", uiHandlers.RevokeSession)
			})
			protected.Get("/logout", authHandlers.Logout)
			protected.Post("/logout", authHandlers.Logout)
			protected.Route("/search", func(sr chi.Router) {
				sr.Get("/", uiHandlers.SearchPage)
				sr.Get("/table", uiHandlers.SearchTable)
			})
			protected.Route("/notifications", func(nr chi.Router) {
				nr.Get("/", uiHandlers.NotificationsPage)
				nr.Get("/table", uiHandlers.NotificationsTable)
				nr.Get("/badge", uiHandlers.NotificationsBadge)
			})
			protected.Route("/orders", func(or chi.Router) {
				or.Get("/", uiHandlers.OrdersPage)
				or.Get("/table", uiHandlers.OrdersTable)
				or.Post("/bulk/status", uiHandlers.OrdersBulkStatus)
				or.Post("/bulk/labels", uiHandlers.OrdersBulkLabels)
				or.Post("/bulk/export", uiHandlers.OrdersBulkExport)
				or.Get("/bulk/export/jobs/{jobID}", uiHandlers.OrdersBulkExportJobStatus)
				or.Get("/{orderID}/modal/status", uiHandlers.OrdersStatusModal)
				or.Put("/{orderID}:status", uiHandlers.OrdersStatusUpdate)
				or.Get("/{orderID}/modal/refund", uiHandlers.OrdersRefundModal)
				or.Post("/{orderID}/payments:refund", uiHandlers.OrdersSubmitRefund)
				or.Get("/{orderID}/modal/invoice", uiHandlers.OrdersInvoiceModal)
				or.Post("/{orderID}/shipments", uiHandlers.ShipmentsCreateOrderShipment)
			})
			protected.Route("/shipments", func(sr chi.Router) {
				sr.Get("/tracking", uiHandlers.ShipmentsTrackingPage)
				sr.Get("/tracking/table", uiHandlers.ShipmentsTrackingTable)
				sr.Get("/batches", uiHandlers.ShipmentsBatchesPage)
				sr.Get("/batches/table", uiHandlers.ShipmentsBatchesTable)
				sr.Get("/batches/{batchID}/drawer", uiHandlers.ShipmentsBatchDrawer)
				sr.Post("/batches", uiHandlers.ShipmentsCreateBatch)
				sr.Post("/batches/regenerate", uiHandlers.ShipmentsRegenerateLabels)
			})
			protected.Post("/invoices:issue", uiHandlers.InvoicesIssue)
			protected.Get("/invoices/jobs/{jobID}", uiHandlers.InvoiceJobStatus)
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
