package main

import (
    "io"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/go-chi/chi/v5"
    chimw "github.com/go-chi/chi/v5/middleware"

    "finitefield.org/hanko-web/internal/i18n"
    mw "finitefield.org/hanko-web/internal/middleware"
)

// newTestRouter builds a router similar to main(), optionally adding extra routes.
func newTestRouter(t *testing.T, add func(r chi.Router)) http.Handler {
    t.Helper()
    // ensure templates reparse each request and set correct paths
    devMode = true
    templatesDir = "../../templates"
    publicDir = "../../public"
    if _, err := parseTemplates(); err != nil {
        t.Fatalf("parseTemplates failed: %v", err)
    }
    // load i18n for tests
    var err error
    i18nBundle, err = i18n.Load("../../locales", "ja", []string{"ja", "en"})
    if err != nil {
        t.Fatalf("load i18n: %v", err)
    }
    r := chi.NewRouter()
    r.Use(chimw.RequestID)
    r.Use(chimw.RealIP)
    r.Use(mw.HTMX)
    r.Use(mw.Session)
    r.Use(mw.Locale(i18nBundle))
    r.Use(mw.Auth)
    r.Use(mw.CSRF)
    r.Use(mw.VaryLocale)
    r.Use(mw.Logger)
    r.Use(chimw.Recoverer)

    // base routes used in app
    r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
        _, _ = io.WriteString(w, "ok")
    })
    // assets and home
    assets := http.StripPrefix("/assets", mw.AssetsWithCache("public/assets"))
    r.Handle("/assets/*", assets)
    r.Get("/", HomeHandler)

    if add != nil {
        r.Group(func(r chi.Router) {
            add(r)
        })
    }
    return r
}

func TestHealthzOK(t *testing.T) {
    srv := newTestRouter(t, nil)
    req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)
    t.Logf("/ status=%d body=%s", rec.Code, rec.Body.String())
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
    }
    if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
        t.Fatalf("expected body 'ok', got %q", got)
    }
}

func TestHomeLocalizedNav_EN(t *testing.T) {
    srv := newTestRouter(t, nil)
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    req.Header.Set("Accept-Language", "en")
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)
    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }
    body := rec.Body.String()
    if !strings.Contains(body, ">Shop<") {
        t.Fatalf("expected localized nav label 'Shop' in body; status=%d body=%s", rec.Code, body)
    }
}

func TestHTMXPostRequiresCSRF(t *testing.T) {
    srv := newTestRouter(t, func(r chi.Router) {
        r.Post("/echo", func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusOK)
            _, _ = io.WriteString(w, "ok")
        })
        r.Get("/debug", func(w http.ResponseWriter, r *http.Request) {
            s := mw.GetSession(r)
            _, _ = io.WriteString(w, s.CSRFToken)
        })
    })

    // First, GET / to receive CSRF cookie
    rec1 := httptest.NewRecorder()
    req1 := httptest.NewRequest(http.MethodGet, "/", nil)
    srv.ServeHTTP(rec1, req1)
    if rec1.Code != http.StatusOK { t.Fatalf("GET / expected 200, got %d; body=%s", rec1.Code, rec1.Body.String()) }
    csrfCookie := ""
    sessCookie := ""
    t.Logf("Set-Cookie headers: %v", rec1.Result().Header["Set-Cookie"])
    for _, c := range rec1.Result().Cookies() {
        if c.Name == "csrf_token" { csrfCookie = c.Value }
        if c.Name == "HANKO_WEB_SESSION" { sessCookie = c.Value }
    }
    if csrfCookie == "" { t.Fatalf("missing csrf_token cookie from GET /") }
    if sessCookie == "" { t.Fatalf("missing session cookie from GET /") }

    // POST without CSRF should 403 when HX-Request=true
    rec2 := httptest.NewRecorder()
    req2 := httptest.NewRequest(http.MethodPost, "/echo", nil)
    req2.Header.Set("HX-Request", "true")
    srv.ServeHTTP(rec2, req2)
    if rec2.Code != http.StatusForbidden {
        t.Fatalf("expected 403 for missing CSRF, got %d; body=%s", rec2.Code, rec2.Body.String())
    }

    // Verify session has same token
    recDbg := httptest.NewRecorder()
    reqDbg := httptest.NewRequest(http.MethodGet, "/debug", nil)
    reqDbg.Header.Set("Cookie", "csrf_token="+csrfCookie+"; HANKO_WEB_SESSION="+sessCookie)
    srv.ServeHTTP(recDbg, reqDbg)
    if strings.TrimSpace(recDbg.Body.String()) != csrfCookie {
        t.Fatalf("session token mismatch: got %q want %q", recDbg.Body.String(), csrfCookie)
    }

    // POST with CSRF header and cookie should succeed
    rec3 := httptest.NewRecorder()
    req3 := httptest.NewRequest(http.MethodPost, "/echo", nil)
    req3.Header.Set("HX-Request", "true")
    req3.Header.Set("X-CSRF-Token", csrfCookie)
    req3.Header.Set("Cookie", "csrf_token="+csrfCookie+"; HANKO_WEB_SESSION="+sessCookie)
    srv.ServeHTTP(rec3, req3)
    if rec3.Code != http.StatusOK {
        t.Fatalf("expected 200 with valid CSRF, got %d; body=%s", rec3.Code, rec3.Body.String())
    }
    if strings.TrimSpace(rec3.Body.String()) != "ok" {
        t.Fatalf("expected body ok, got %q", rec3.Body.String())
    }
}

func TestSessionMiddlewareSetsCookie(t *testing.T) {
    h := mw.Session(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        _, _ = io.WriteString(w, "ok")
    }))
    rec := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodGet, "/x", nil)
    h.ServeHTTP(rec, req)
    if rec.Code != http.StatusOK { t.Fatalf("expected 200, got %d", rec.Code) }
    var seen bool
    for _, c := range rec.Result().Cookies() {
        if c.Name == "HANKO_WEB_SESSION" { seen = true; break }
    }
    if !seen {
        t.Fatalf("expected HANKO_WEB_SESSION cookie to be set, got %v", rec.Result().Header["Set-Cookie"])
    }
}
