package middleware

import (
	"context"
	"net/http"
	"strings"
)

type requestInfoKeyType int

const requestInfoKey requestInfoKeyType = iota

// RequestInfo holds lightweight request metadata exposed to templates.
type RequestInfo struct {
	Path     string
	BasePath string
	Method   string
}

// RequestInfoMiddleware annotates the context with the current request path and base path.
func RequestInfoMiddleware(basePath string) func(http.Handler) http.Handler {
	base := normaliseBase(basePath)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := &RequestInfo{
				Path:     r.URL.Path,
				Method:   r.Method,
				BasePath: base,
			}
			ctx := context.WithValue(r.Context(), requestInfoKey, info)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestInfoFromContext returns the request metadata stored by RequestInfoMiddleware.
func RequestInfoFromContext(ctx context.Context) (*RequestInfo, bool) {
	info, ok := ctx.Value(requestInfoKey).(*RequestInfo)
	return info, ok && info != nil
}

// RequestPathFromContext returns the request path or empty string when unavailable.
func RequestPathFromContext(ctx context.Context) string {
	if info, ok := RequestInfoFromContext(ctx); ok {
		return info.Path
	}
	return ""
}

// BasePathFromContext returns the resolved admin base path or "/" when unavailable.
func BasePathFromContext(ctx context.Context) string {
	if info, ok := RequestInfoFromContext(ctx); ok && info.BasePath != "" {
		return info.BasePath
	}
	return "/"
}

func normaliseBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if base != "/" {
		base = strings.TrimRight(base, "/")
		if base == "" {
			return "/"
		}
	}
	return base
}
