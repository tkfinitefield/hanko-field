package middleware

import (
	"net/http"
)

// HTMX marks requests coming from htmx so handlers/middlewares can adapt responses
func HTMX(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		is := r.Header.Get("HX-Request") == "true"
		ctx := WithHTMX(r.Context(), is)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
