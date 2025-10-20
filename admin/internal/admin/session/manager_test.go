package session

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type fixedClock struct {
	current time.Time
}

func (c *fixedClock) Now() time.Time {
	return c.current
}

func newTestManager(t *testing.T) (*Manager, *fixedClock) {
	t.Helper()

	hashKey := []byte("12345678901234567890123456789012")
	blockKey := []byte("abcdefghijklmnopqrstuv0123456789")
	clock := &fixedClock{current: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)}
	httpOnly := true
	mgr, err := NewManager(Config{
		CookieName:       "test_session",
		HashKey:          hashKey,
		BlockKey:         blockKey,
		CookiePath:       "/",
		CookieHTTPOnly:   &httpOnly,
		IdleTimeout:      10 * time.Minute,
		Lifetime:         2 * time.Hour,
		RememberLifetime: 48 * time.Hour,
		Now:              clock.Now,
	})
	if err != nil {
		t.Fatalf("NewManager error: %v", err)
	}
	return mgr, clock
}

func TestManager_NewSessionLifecycle(t *testing.T) {
	mgr, clock := newTestManager(t)

	req := httptest.NewRequest("GET", "/admin", nil)
	sess, err := mgr.Load(req)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if sess == nil {
		t.Fatalf("expected session")
	}
	if sess.ID() == "" {
		t.Fatalf("expected session ID")
	}
	if !sess.CreatedAt().Equal(clock.current) {
		t.Fatalf("unexpected CreatedAt: %v", sess.CreatedAt())
	}

	user := &User{UID: "user-1", Email: "test@example.com", Roles: []string{"admin"}}
	sess.SetUser(user)
	if sess.User().UID != "user-1" {
		t.Fatalf("expected user to be stored")
	}
	sess.SetFeatureFlag("beta", true)
	sess.SetRememberMe(true)
	token, err := sess.EnsureCSRFToken()
	if err != nil || token == "" {
		t.Fatalf("expected csrf token: %v", err)
	}

	rec := httptest.NewRecorder()
	if err := mgr.Save(rec, sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	httpSessCookie := findCookie(rec.Result().Cookies(), "test_session")
	if httpSessCookie == nil {
		t.Fatalf("expected session cookie to be set")
	}

	clock.current = clock.current.Add(5 * time.Minute)
	req2 := httptest.NewRequest("GET", "/admin", nil)
	req2.AddCookie(httpSessCookie)
	sess2, err := mgr.Load(req2)
	if err != nil {
		t.Fatalf("Load existing error: %v", err)
	}
	if sess2.User().Email != "test@example.com" {
		t.Fatalf("expected user to persist")
	}
	if !sess2.RememberMe() {
		t.Fatalf("expected remember-me flag")
	}
	if got := sess2.FeatureFlags()["beta"]; !got {
		t.Fatalf("expected beta flag true, got %v", got)
	}
	if sess2.CSRFToken() != token {
		t.Fatalf("expected csrf token to persist")
	}
}

func TestManager_IdleTimeout(t *testing.T) {
	mgr, clock := newTestManager(t)
	req := httptest.NewRequest("GET", "/admin", nil)
	sess, err := mgr.Load(req)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	rec := httptest.NewRecorder()
	if err := mgr.Save(rec, sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	cookie := findCookie(rec.Result().Cookies(), "test_session")

	clock.current = clock.current.Add(20 * time.Minute)
	req2 := httptest.NewRequest("GET", "/admin", nil)
	req2.AddCookie(cookie)
	if _, err := mgr.Load(req2); !errors.Is(err, ErrExpired) {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
}

func TestManager_Destroy(t *testing.T) {
	mgr, _ := newTestManager(t)
	req := httptest.NewRequest("GET", "/admin", nil)
	sess, _ := mgr.Load(req)
	rec := httptest.NewRecorder()
	sess.Destroy()
	if err := mgr.Save(rec, sess); err != nil {
		t.Fatalf("Save error: %v", err)
	}
	cookie := findCookie(rec.Result().Cookies(), "test_session")
	if cookie == nil || cookie.MaxAge != -1 {
		t.Fatalf("expected session cookie cleared")
	}
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}
