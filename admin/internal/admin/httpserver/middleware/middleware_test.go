package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	appsession "finitefield.org/hanko-admin/internal/admin/session"
)

type mockAuthenticator struct {
	token string
	user  *User
	err   error
}

func (m *mockAuthenticator) Authenticate(_ *http.Request, token string) (*User, error) {
	if token != m.token {
		return nil, ErrUnauthorized
	}
	return m.user, m.err
}

func TestAuthMiddleware(t *testing.T) {
	auth := &mockAuthenticator{
		token: "valid",
		user:  &User{UID: "user-1"},
	}

	handler := HTMX()(Auth(auth, "/login")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := UserFromContext(r.Context()); !ok {
			t.Fatalf("expected user in context")
		}
		w.WriteHeader(http.StatusOK)
	})))

	t.Run("missing token redirects", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusFound {
			t.Fatalf("expected 302, got %d", rr.Code)
		}
		location := rr.Header().Get("Location")
		if location == "" {
			t.Fatalf("expected redirect location header")
		}
		parsed, err := url.Parse(location)
		if err != nil {
			t.Fatalf("invalid redirect url: %v", err)
		}
		if parsed.Path != "/login" {
			t.Fatalf("expected redirect to /login, got %s", parsed.Path)
		}
		q := parsed.Query()
		if got := q.Get("reason"); got != ReasonMissingToken {
			t.Fatalf("expected reason %s, got %s", ReasonMissingToken, got)
		}
		if got := q.Get("next"); got != "/admin" {
			t.Fatalf("expected next=/admin, got %s", got)
		}
	})

	t.Run("htmx unauthorized returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("HX-Request", "true")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
		location := rr.Header().Get("HX-Redirect")
		if location == "" {
			t.Fatalf("expected HX-Redirect header")
		}
		parsed, err := url.Parse(location)
		if err != nil {
			t.Fatalf("invalid HX-Redirect url: %v", err)
		}
		if parsed.Path != "/login" {
			t.Fatalf("expected HX-Redirect path /login, got %s", parsed.Path)
		}
		if parsed.Query().Get("next") != "/admin" {
			t.Fatalf("expected HX-Redirect next=/admin")
		}
	})

	t.Run("valid token passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer valid")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("token from cookie passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: "Authorization", Value: "valid"})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("expired token triggers refresh header", func(t *testing.T) {
		auth.err = NewAuthError(ReasonTokenExpired, errors.New("expired"))
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		req.Header.Set("Authorization", "Bearer valid")
		req.Header.Set("HX-Request", "true")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rr.Code)
		}
		if rr.Header().Get("HX-Refresh") != "true" {
			t.Fatalf("expected HX-Refresh header")
		}
		auth.err = nil
	})
}

func TestCSRFMiddleware(t *testing.T) {
	mw := CSRF(CSRFConfig{CookieName: "csrf", HeaderName: "X-CSRF-Token"})

	t.Run("issues cookie on GET", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rr := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := CSRFTokenFromContext(r.Context())
			if token == "" {
				t.Fatalf("expected token in context")
			}
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		found := false
		for _, c := range rr.Result().Cookies() {
			if c.Name == "csrf" && c.Value != "" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected csrf cookie to be set")
		}
	})

	t.Run("rejects unsafe request without header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: "csrf", Value: "token"})
		rr := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})).ServeHTTP(rr, req)
		if rr.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rr.Code)
		}
	})

	t.Run("allows unsafe request with matching header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/admin", nil)
		req.AddCookie(&http.Cookie{Name: "csrf", Value: "token"})
		req.Header.Set("X-CSRF-Token", "token")
		rr := httptest.NewRecorder()
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})
}

func TestHTMXMiddleware(t *testing.T) {
	base := HTMX()

	t.Run("detects htmx", func(t *testing.T) {
		handler := base(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsHTMXRequest(r.Context()) {
				t.Fatalf("expected htmx request")
			}
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/admin/fragments", nil)
		req.Header.Set("HX-Request", "true")
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("RequireHTMX blocks non-htmx", func(t *testing.T) {
		handler := base(RequireHTMX()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		req := httptest.NewRequest(http.MethodGet, "/admin/fragments", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rr.Code)
		}
	})
}

func TestNoStoreMiddleware(t *testing.T) {
	handler := NoStore()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Cache-Control"); got != "no-store, max-age=0" {
		t.Fatalf("unexpected Cache-Control: %s", got)
	}
	if got := rr.Header().Get("Pragma"); got != "no-cache" {
		t.Fatalf("unexpected Pragma: %s", got)
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestAuthMiddlewareStoresSessionUser(t *testing.T) {
	clock := &sessionTestClock{now: time.Now().UTC()}
	store := newSessionStoreForTest(t, clock)

	auth := &mockAuthenticator{
		token: "valid",
		user: &User{
			UID:          "user-123",
			Email:        "user@example.com",
			Roles:        []string{"admin"},
			FeatureFlags: map[string]bool{"beta": true},
		},
	}

	handler := Session(store)(Auth(auth, "/login")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFromContext(r.Context())
		if !ok {
			t.Fatalf("expected session in context")
		}
		user := sess.User()
		if user == nil || user.UID != "user-123" || user.Email != "user@example.com" {
			t.Fatalf("unexpected session user: %+v", user)
		}
		if !sess.FeatureFlags()["beta"] {
			t.Fatalf("expected beta flag to persist")
		}
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer valid")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	resp := &http.Response{Header: rec.Header()}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie to be issued")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req2.AddCookie(cookies[0])
	sess, err := store.Load(req2)
	if err != nil {
		t.Fatalf("load session error: %v", err)
	}
	if sess.User() == nil || sess.User().UID != "user-123" {
		t.Fatalf("expected persisted user, got %+v", sess.User())
	}
	if !sess.FeatureFlags()["beta"] {
		t.Fatalf("expected persisted feature flags")
	}
}

func TestAuthMiddlewareClearsSessionOnUnauthorized(t *testing.T) {
	clock := &sessionTestClock{now: time.Now().UTC()}
	store := newSessionStoreForTest(t, clock)

	initial := store.New()
	initial.SetUser(&appsession.User{UID: "seed"})

	rec := httptest.NewRecorder()
	if err := store.Save(rec, initial); err != nil {
		t.Fatalf("save seed session: %v", err)
	}
	resp := &http.Response{Header: rec.Header()}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected seed session cookie")
	}

	auth := &mockAuthenticator{token: "valid", user: &User{UID: "user-1"}}
	handler := Session(store)(Auth(auth, "/login")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not be reached")
	})))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(cookies[0])
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusFound {
		t.Fatalf("expected 302 redirect, got %d", rec2.Code)
	}
	setCookie := rec2.Header().Get("Set-Cookie")
	if setCookie == "" || (!strings.Contains(setCookie, "Max-Age=0") && !strings.Contains(setCookie, "Max-Age=-1")) {
		t.Fatalf("expected session cookie cleared, got %q", setCookie)
	}
}
