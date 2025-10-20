package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

const csrfCookieName = "csrf_token"

// CSRF issues a CSRF cookie and verifies modifying requests carry the token in header
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ensure token cookie exists (non-HttpOnly so client can read for htmx header)
		token := ""
		if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
			token = c.Value
		} else {
			token = newCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: false,
				Secure:   sessionSecure,
				SameSite: http.SameSiteLaxMode,
				Expires:  time.Now().Add(24 * time.Hour),
			})
		}

		// For unsafe methods, verify header
		if !isSafeMethod(r.Method) {
			// Skip CSRF check for programmatic clients sending Authorization Bearer (non-browser)
			if auth := r.Header.Get("Authorization"); auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
				hdr := r.Header.Get("X-CSRF-Token")
				if hdr == "" || hdr != token {
					writeError(w, r, http.StatusForbidden, "invalid CSRF token")
					return
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func newCSRFToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func isSafeMethod(m string) bool {
	switch m {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
