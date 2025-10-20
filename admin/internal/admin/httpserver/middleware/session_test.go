package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appsession "finitefield.org/hanko-admin/internal/admin/session"
)

type sessionTestClock struct {
	now time.Time
}

func (c *sessionTestClock) Now() time.Time {
	return c.now
}

func newSessionStoreForTest(t *testing.T, clock *sessionTestClock) *appsession.Manager {
	t.Helper()
	hashKey := []byte("12345678901234567890123456789012")
	blockKey := []byte("abcdefghijklmnopqrstuvwxyzABCDEF")
	httpOnly := true
	store, err := appsession.NewManager(appsession.Config{
		CookieName:       "test_session",
		HashKey:          hashKey,
		BlockKey:         blockKey,
		CookiePath:       "/admin",
		CookieHTTPOnly:   &httpOnly,
		IdleTimeout:      5 * time.Minute,
		Lifetime:         time.Hour,
		RememberLifetime: 24 * time.Hour,
		Now:              clock.Now,
	})
	if err != nil {
		t.Fatalf("session manager init: %v", err)
	}
	return store
}

func TestSessionMiddlewareLifecycle(t *testing.T) {
	clock := &sessionTestClock{now: time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)}
	store := newSessionStoreForTest(t, clock)

	var ids []string
	handler := Session(store)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, ok := SessionFromContext(r.Context())
		if !ok {
			t.Fatalf("session missing in context")
		}
		ids = append(ids, sess.ID())
		w.WriteHeader(http.StatusOK)
	}))

	req1 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if len(ids) != 1 || ids[0] == "" {
		t.Fatalf("expected initial session id")
	}
	res := &http.Response{Header: rec1.Header()}
	cookies := res.Cookies()
	if len(cookies) == 0 {
		t.Fatalf("expected session cookie on first response, got headers %v", rec1.Header().Values("Set-Cookie"))
	}
	cookie := findCookie(cookies, "test_session")
	if cookie == nil {
		t.Fatalf("cookie list=%v", cookies)
	}

	clock.now = clock.now.Add(2 * time.Minute)
	req2 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req2.AddCookie(cookie)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if len(ids) != 2 {
		t.Fatalf("expected two session observations, got %d", len(ids))
	}
	if ids[1] != ids[0] {
		t.Fatalf("expected same session id between active requests")
	}
	clock.now = clock.now.Add(15 * time.Minute) // exceed idle timeout
	req3 := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req3.AddCookie(cookie)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if len(ids) != 3 {
		t.Fatalf("expected three session observations, got %d", len(ids))
	}
	if ids[2] == ids[1] {
		t.Fatalf("expected new session id after idle timeout")
	}

	if header := rec3.Header().Get("Set-Cookie"); header == "" {
		t.Fatalf("expected refreshed session cookie after timeout")
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
