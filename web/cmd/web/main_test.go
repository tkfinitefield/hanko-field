package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"finitefield.org/hanko-web/internal/cms"
	"finitefield.org/hanko-web/internal/i18n"
	mw "finitefield.org/hanko-web/internal/middleware"
	"finitefield.org/hanko-web/internal/status"
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
	r.Get("/design/new", DesignNewHandler)
	r.Get("/design/preview", DesignPreviewHandler)
	r.Get("/design/preview/image", DesignPreviewImageFrag)

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

func TestDesignNewPageRenders(t *testing.T) {
	srv := newTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/design/new", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "data-design-option") {
		t.Fatalf("expected design option markers in body; status=%d body=%s", rec.Code, body)
	}
	if !strings.Contains(body, "design-primary-cta") {
		t.Fatalf("expected primary CTA button id in body; status=%d body=%s", rec.Code, body)
	}
}

func TestDesignPreviewPageRenders(t *testing.T) {
	srv := newTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/design/preview", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "design-preview-stage") {
		t.Fatalf("expected preview stage marker in body; status=%d body=%s", rec.Code, body)
	}
	if !strings.Contains(body, "Background material") {
		t.Fatalf("expected background control copy in body; status=%d body=%s", rec.Code, body)
	}
}

func TestDesignPreviewFragmentPushesQuery(t *testing.T) {
	srv := newTestRouter(t, nil)
	req := httptest.NewRequest(http.MethodGet, "/design/preview/image?bg=transparent&dpi=1200&frame=desk&grid=1", nil)
	req.Header.Set("HX-Request", "true")
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	want := "/design/preview?bg=transparent&dpi=1200&frame=desk&grid=1"
	if got := rec.Header().Get("HX-Push-Url"); got != want {
		t.Fatalf("expected HX-Push-Url %q, got %q", want, got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Transparent") && !strings.Contains(body, `value="transparent"`) {
		t.Fatalf("expected transparent label in fragment response")
	}
	if !strings.Contains(body, "Measurement grid") && !strings.Contains(body, "製図ガイド") {
		t.Fatalf("expected grid copy in fragment response")
	}
	if !strings.Contains(body, `/design/preview?bg=transparent&amp;dpi=1200&amp;frame=desk&amp;grid=0`) {
		t.Fatalf("expected disable grid action to preserve query parameters; body=%s", body)
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
	if rec1.Code != http.StatusOK {
		t.Fatalf("GET / expected 200, got %d; body=%s", rec1.Code, rec1.Body.String())
	}
	csrfCookie := ""
	sessCookie := ""
	t.Logf("Set-Cookie headers: %v", rec1.Result().Header["Set-Cookie"])
	for _, c := range rec1.Result().Cookies() {
		if c.Name == "csrf_token" {
			csrfCookie = c.Value
		}
		if c.Name == "HANKO_WEB_SESSION" {
			sessCookie = c.Value
		}
	}
	if csrfCookie == "" {
		t.Fatalf("missing csrf_token cookie from GET /")
	}
	if sessCookie == "" {
		t.Fatalf("missing session cookie from GET /")
	}

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
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var seen bool
	for _, c := range rec.Result().Cookies() {
		if c.Name == "HANKO_WEB_SESSION" {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatalf("expected HANKO_WEB_SESSION cookie to be set, got %v", rec.Result().Header["Set-Cookie"])
	}
}

func TestDesignAISuggestionsPageRenders(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/design/ai", DesignAISuggestionsHandler)
		r.Get("/design/ai/table", DesignAISuggestionTableFrag)
		r.Get("/design/ai/preview", DesignAISuggestionPreviewFrag)
	})
	req := httptest.NewRequest(http.MethodGet, "/design/ai", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "design-ai-table") {
		t.Fatalf("expected design ai table markup in body; body=%s", body)
	}
	if !strings.Contains(body, "design-ai-preview") {
		t.Fatalf("expected preview container in body; body=%s", body)
	}
}

func TestDesignAISuggestionsTableFragment(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/design/ai/table", DesignAISuggestionTableFrag)
	})
	req := httptest.NewRequest(http.MethodGet, "/design/ai/table?status=ready", nil)
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("HX-Push-Url"); got == "" || !strings.HasPrefix(got, "/design/ai") {
		t.Fatalf("expected HX-Push-Url header, got %q", got)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "data-suggestion-card") {
		t.Fatalf("expected suggestion cards in fragment; body=%s", body)
	}
}

func TestDesignAISuggestionsEmptyFilterKeepsPreviewNeutral(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/design/ai", DesignAISuggestionsHandler)
		r.Get("/design/ai/table", DesignAISuggestionTableFrag)
		r.Get("/design/ai/preview", DesignAISuggestionPreviewFrag)
	})

	req := httptest.NewRequest(http.MethodGet, "/design/ai?status=ready&persona=government", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Select a suggestion to open the preview drawer") {
		t.Fatalf("expected neutral preview message when empty; body=%s", body)
	}
	if !strings.Contains(body, "No suggestions match the current filters yet.") {
		t.Fatalf("expected empty state copy when no results; body=%s", body)
	}

	fragReq := httptest.NewRequest(http.MethodGet, "/design/ai/table?status=ready&persona=government", nil)
	fragReq.Header.Set("Accept-Language", "en")
	fragReq.Header.Set("HX-Request", "true")
	fragRec := httptest.NewRecorder()
	srv.ServeHTTP(fragRec, fragReq)
	if fragRec.Code != http.StatusOK {
		t.Fatalf("expected 200 for fragment, got %d; body=%s", fragRec.Code, fragRec.Body.String())
	}
	if got := fragRec.Header().Get("HX-Push-Url"); strings.Contains(got, "focus=") {
		t.Fatalf("expected HX-Push-Url without focus param, got %q", got)
	}
}

func TestDesignAISuggestionAccept(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/design/ai", DesignAISuggestionsHandler)
		r.Get("/design/ai/table", DesignAISuggestionTableFrag)
		r.Get("/design/ai/preview", DesignAISuggestionPreviewFrag)
		r.MethodFunc(http.MethodPost, "/design/ai/suggestions/{suggestionID}/accept", DesignAISuggestionAcceptHandler)
	})

	// prime session + CSRF
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/design/ai", nil)
	req.Header.Set("Accept-Language", "en")
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /design/ai expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	var csrfCookie, sessionCookie string
	for _, c := range rec.Result().Cookies() {
		if c.Name == "csrf_token" {
			csrfCookie = c.Value
		}
		if c.Name == "HANKO_WEB_SESSION" {
			sessionCookie = c.Value
		}
	}
	if csrfCookie == "" || sessionCookie == "" {
		t.Fatalf("expected csrf and session cookies, got csrf=%q session=%q", csrfCookie, sessionCookie)
	}

	postRec := httptest.NewRecorder()
	postReq := httptest.NewRequest(http.MethodPost, "/design/ai/suggestions/sg-401/accept", nil)
	postReq.Header.Set("HX-Request", "true")
	postReq.Header.Set("X-CSRF-Token", csrfCookie)
	postReq.Header.Set("Cookie", "csrf_token="+csrfCookie+"; HANKO_WEB_SESSION="+sessionCookie)
	srv.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("expected 200 accept response, got %d; body=%s", postRec.Code, postRec.Body.String())
	}
	trigger := postRec.Header().Get("HX-Trigger")
	if trigger == "" || !strings.Contains(trigger, "design-ai:suggestion-accepted") {
		t.Fatalf("expected HX-Trigger for acceptance, got %q", trigger)
	}
	body := postRec.Body.String()
	if !strings.Contains(body, "Applied to design editor preview") {
		t.Fatalf("expected acceptance note in response body; body=%s", body)
	}
	if !strings.Contains(body, "data-action=\"accept\"") {
		t.Fatalf("expected accept button markup present; body=%s", body)
	}
}

func setupStaticTestRouter(t *testing.T) http.Handler {
	t.Helper()
	cmsClient = cms.NewClient("")
	cmsClient.SetContentDir("../../content")
	cms.SetContentCacheDuration(500 * time.Millisecond)
	contentRenderCache.mu.Lock()
	contentRenderCache.items = map[string]renderedContentEntry{}
	contentRenderCache.mu.Unlock()
	statusClient = status.NewClient("")
	t.Cleanup(func() {
		cmsClient = nil
		statusClient = nil
		contentRenderCache.mu.Lock()
		contentRenderCache.items = map[string]renderedContentEntry{}
		contentRenderCache.mu.Unlock()
	})
	return newTestRouter(t, func(r chi.Router) {
		r.Get("/content/{slug}", ContentPageHandler)
		r.Get("/legal/{slug}", LegalPageHandler)
		r.Get("/status", StatusHandler)
	})
}

func TestContentPageMarkdownRendering(t *testing.T) {
	srv := setupStaticTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/content/about-hanko-field", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "About Hanko Field") {
		t.Fatalf("expected page title in body, got %s", body)
	}
	if !strings.Contains(body, "content-prose") {
		t.Fatalf("expected prose wrapper in body, got %s", body)
	}
	if !strings.Contains(body, "Now supporting bilingual orders") {
		t.Fatalf("expected banner copy in body, got %s", body)
	}
	if !strings.Contains(body, `aria-label="On this page"`) {
		t.Fatalf("expected table of contents to render, got %s", body)
	}
	cache := rec.Header().Get("Cache-Control")
	if cache != "public, max-age=600" {
		t.Fatalf("expected Cache-Control=public, max-age=600, got %q", cache)
	}
	lastMod := rec.Header().Get("Last-Modified")
	if lastMod == "" {
		t.Fatalf("expected Last-Modified header")
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/content/about-hanko-field", nil)
	req2.Header.Set("If-None-Match", etag)
	req2.Header.Set("Accept-Language", "en")
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 for matching ETag, got %d", rec2.Code)
	}
}

func TestLegalPageVersionFooter(t *testing.T) {
	srv := setupStaticTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/legal/privacy-policy", nil)
	req.Header.Set("Accept-Language", "ja")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "プライバシーポリシー") {
		t.Fatalf("expected Japanese policy title, got %s", body)
	}
	if !strings.Contains(body, "ドキュメント版 2025.1.2") {
		t.Fatalf("expected version footer, got %s", body)
	}
	if !strings.Contains(body, "privacy@hanko-field.jp") {
		t.Fatalf("expected contact email in body, got %s", body)
	}
	if !strings.Contains(body, "全文をダウンロード") {
		t.Fatalf("expected download CTA in body, got %s", body)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}
	req2 := httptest.NewRequest(http.MethodGet, "/legal/privacy-policy", nil)
	req2.Header.Set("If-None-Match", etag)
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 for matching ETag, got %d", rec2.Code)
	}
}

func TestStatusHandlerFallback(t *testing.T) {
	srv := setupStaticTestRouter(t)
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "System Status") {
		t.Fatalf("expected status title, got %s", body)
	}
	if !strings.Contains(body, "All systems operational") {
		t.Fatalf("expected fallback state label, got %s", body)
	}
	if !strings.Contains(body, "Scheduled maintenance") {
		t.Fatalf("expected incident timeline in body, got %s", body)
	}
	cache := rec.Header().Get("Cache-Control")
	if cache != "public, max-age=60" {
		t.Fatalf("expected Cache-Control=public, max-age=60, got %q", cache)
	}
	etag := rec.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/status", nil)
	req2.Header.Set("If-None-Match", etag)
	req2.Header.Set("Accept-Language", "en")
	rec2 := httptest.NewRecorder()
	srv.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusNotModified {
		t.Fatalf("expected 304 for matching ETag, got %d", rec2.Code)
	}
}

func TestModalPickFontSelectedState(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/modal/pick/font", ModalPickFont)
	})

	req := httptest.NewRequest(http.MethodGet, "/modal/pick/font?font=jp-gothic", nil)
	req.Header.Set("Accept-Language", "en")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "font-picker-modal_title") {
		t.Fatalf("expected modal title wrapper in body, got %s", body)
	}
	if !strings.Contains(body, "Gothic Modern") {
		t.Fatalf("expected selected font name in body, got %s", body)
	}
	if strings.Count(body, "Currently applied") != 1 {
		t.Fatalf("expected exactly one selected marker, got %s", body)
	}
}

func TestModalPickTemplateLocalizedTitle(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.Get("/modal/pick/template", ModalPickTemplate)
	})

	req := httptest.NewRequest(http.MethodGet, "/modal/pick/template", nil)
	req.Header.Set("Accept-Language", "ja")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "template-picker-modal_title") {
		t.Fatalf("expected modal title id in body, got %s", body)
	}
	if !strings.Contains(body, "テンプレートを選択") {
		t.Fatalf("expected localized title in body, got %s", body)
	}
	if !strings.Contains(body, "現在の選択") {
		t.Fatalf("expected selected badge copy, got %s", body)
	}
}

func TestModalKanjiMapPOSTRendersCandidates(t *testing.T) {
	srv := newTestRouter(t, func(r chi.Router) {
		r.MethodFunc(http.MethodGet, "/modal/kanji-map", ModalKanjiMap)
		r.MethodFunc(http.MethodPost, "/modal/kanji-map", ModalKanjiMap)
	})

	bootReq := httptest.NewRequest(http.MethodGet, "/", nil)
	bootReq.Header.Set("Accept-Language", "ja")
	bootRec := httptest.NewRecorder()
	srv.ServeHTTP(bootRec, bootReq)
	if bootRec.Code != http.StatusOK {
		t.Fatalf("expected bootstrapping GET to succeed, got %d; body=%s", bootRec.Code, bootRec.Body.String())
	}
	var csrfToken, sessionCookie string
	for _, c := range bootRec.Result().Cookies() {
		switch c.Name {
		case "csrf_token":
			csrfToken = c.Value
		case "HANKO_WEB_SESSION":
			sessionCookie = c.Value
		}
	}
	if csrfToken == "" || sessionCookie == "" {
		t.Fatalf("expected csrf and session cookies, got csrf=%q session=%q", csrfToken, sessionCookie)
	}

	form := strings.NewReader("name=Saito")
	req := httptest.NewRequest(http.MethodPost, "/modal/kanji-map", form)
	req.Header.Set("Accept-Language", "ja")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", csrfToken)
	req.Header.Set("Cookie", "csrf_token="+csrfToken+"; HANKO_WEB_SESSION="+sessionCookie)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "kanji-mapper-modal_title") {
		t.Fatalf("expected kanji mapper modal wrapper, got %s", body)
	}
	if !strings.Contains(body, "斎藤") {
		t.Fatalf("expected mapped kanji candidate in body, got %s", body)
	}
	if !strings.Contains(body, "信頼度") {
		t.Fatalf("expected confidence label in body, got %s", body)
	}
}
