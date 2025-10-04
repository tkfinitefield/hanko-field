package handlers

import (
	"log"
	"net/http"
	"time"
)

// NewRouter constructs the HTTP router with lightweight middleware suitable for local development.
func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)

	return requestLogger(mux)
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)
		log.Printf("%s %s %dms", r.Method, r.URL.Path, duration.Milliseconds())
	})
}
