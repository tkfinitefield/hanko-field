package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net/http"
	"time"
)

type csrfContextKey string

const csrfTokenContextKey csrfContextKey = "csrf.token"

// CSRFConfig controls cookie/header behaviour.
type CSRFConfig struct {
	CookieName string
	CookiePath string
	HeaderName string
	MaxAge     time.Duration
	Secure     bool
}

// CSRF attaches double-submit cookie protection. Safe methods (GET/HEAD/OPTIONS) ensure a token is issued;
// unsafe methods validate the incoming header matches the cookie value.
func CSRF(cfg CSRFConfig) func(http.Handler) http.Handler {
	cookieName := cfg.CookieName
	if cookieName == "" {
		cookieName = "admin_csrf"
	}
	headerName := cfg.HeaderName
	if headerName == "" {
		headerName = "X-CSRF-Token"
	}
	cookiePath := cfg.CookiePath
	if cookiePath == "" {
		cookiePath = "/"
	}
	maxAge := cfg.MaxAge
	if maxAge == 0 {
		maxAge = 24 * time.Hour
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, err := ensureCSRFToken(w, r, cookieName, cookiePath, maxAge, cfg.Secure)
			if err != nil {
				http.Error(w, "csrf token error", http.StatusInternalServerError)
				return
			}

			if isUnsafeMethod(r.Method) {
				submitted := r.Header.Get(headerName)
				if submitted == "" || submitted != token {
					http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
					return
				}
			}

			ctx := context.WithValue(r.Context(), csrfTokenContextKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFTokenFromContext returns the token issued for the current request (to embed in forms or meta tags).
func CSRFTokenFromContext(ctx context.Context) string {
	if token, ok := ctx.Value(csrfTokenContextKey).(string); ok {
		return token
	}
	return ""
}

func ensureCSRFToken(w http.ResponseWriter, r *http.Request, cookieName, cookiePath string, maxAge time.Duration, secure bool) (string, error) {
	if c, err := r.Cookie(cookieName); err == nil && c.Value != "" {
		return c.Value, nil
	}

	token, err := generateToken(32)
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     cookiePath,
		HttpOnly: true,
		Secure:   secure || r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(maxAge.Seconds()),
	})

	return token, nil
}

func generateToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return false
	default:
		return true
	}
}
