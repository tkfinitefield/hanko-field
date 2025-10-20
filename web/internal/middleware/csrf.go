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
		// Tie token to session: use per-session token from session data
		s := GetSession(r)
		token := s.CSRFToken
		if token == "" { // initialize if missing
			token = newCSRFToken()
			s.CSRFToken = token
			s.MarkDirty()
		}

		// Ensure client has cookie with the same token (double submit cookie)
		needSet := true
		if c, err := r.Cookie(csrfCookieName); err == nil && c.Value == token {
			needSet = false
		}
		if needSet {
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

		// For unsafe methods, verify header AND (optionally) cookie token
        if !isSafeMethod(r.Method) {
            // Skip CSRF check for programmatic clients sending Authorization Bearer (non-browser)
            if auth := r.Header.Get("Authorization"); auth == "" || !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
                hdr := r.Header.Get("X-CSRF-Token")
                if hdr == "" || hdr != token {
                    writeError(w, r, http.StatusForbidden, "invalid CSRF token")
                    return
                }
                if c, err := r.Cookie(csrfCookieName); err != nil || c.Value != token {
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
