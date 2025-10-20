package middleware

import "net/http"

// VaryLocale sets Vary header for Accept-Language on dynamic responses
func VaryLocale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// append to existing Vary if any
		w.Header().Add("Vary", "Accept-Language")
		next.ServeHTTP(w, r)
	})
}
