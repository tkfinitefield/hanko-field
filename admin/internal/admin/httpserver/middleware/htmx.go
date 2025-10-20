package middleware

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const htmxContextKey contextKey = "htmx.info"

// HTMXInfo captures request metadata from HX-* headers.
type HTMXInfo struct {
	IsHTMX         bool
	IsBoosted      bool
	CurrentURL     string
	Target         string
	TriggerID      string
	TriggerName    string
	HistoryRestore bool
}

// HTMX returns middleware that inspects HX-* headers and annotates the context.
func HTMX() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := HTMXInfo{
				IsHTMX:         strings.EqualFold(r.Header.Get("HX-Request"), "true"),
				IsBoosted:      strings.EqualFold(r.Header.Get("HX-Boosted"), "true"),
				CurrentURL:     r.Header.Get("HX-Current-URL"),
				Target:         r.Header.Get("HX-Target"),
				TriggerID:      r.Header.Get("HX-Trigger"),
				TriggerName:    r.Header.Get("HX-Trigger-Name"),
				HistoryRestore: strings.EqualFold(r.Header.Get("HX-History-Restore-Request"), "true"),
			}

			ctx := context.WithValue(r.Context(), htmxContextKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// HTMXInfoFromContext retrieves HTMX metadata; returns zero value if absent.
func HTMXInfoFromContext(ctx context.Context) HTMXInfo {
	val, ok := ctx.Value(htmxContextKey).(HTMXInfo)
	if !ok {
		return HTMXInfo{}
	}
	return val
}

// IsHTMXRequest returns true when the current request was initiated by htmx.
func IsHTMXRequest(ctx context.Context) bool {
	return HTMXInfoFromContext(ctx).IsHTMX
}

// RequireHTMX ensures the request originated from htmx; otherwise returns 404 to
// avoid exposing fragment routes to direct navigation.
func RequireHTMX() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !IsHTMXRequest(r.Context()) {
				http.NotFound(w, r)
				return
			}
			w.Header().Add("Vary", "HX-Request")
			next.ServeHTTP(w, r)
		})
	}
}
