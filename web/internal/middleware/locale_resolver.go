package middleware

import (
    "context"
    "net/http"
    "strings"

    "finitefield.org/hanko-web/internal/i18n"
)

// Locale resolves and stores the preferred language in the session and cookie `hl`.
func Locale(bundle *i18n.Bundle) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // make fallback available to request context for helpers
            ctx := context.WithValue(r.Context(), ctxKeyLocaleFB, bundle.Fallback())
            r = r.WithContext(ctx)
            s := GetSession(r)
            // query override
            if q := r.URL.Query().Get("hl"); q != "" {
                q = strings.ToLower(q)
                s.Locale = q
                s.MarkDirty()
                http.SetCookie(w, &http.Cookie{Name: "hl", Value: q, Path: "/"})
            } else if s.Locale == "" {
                // cookie or Accept-Language
                if c, err := r.Cookie("hl"); err == nil && c.Value != "" {
                    s.Locale = strings.ToLower(c.Value)
                    s.MarkDirty()
                } else {
                    s.Locale = bundle.Resolve(r.Header.Get("Accept-Language"))
                    s.MarkDirty()
                }
            }
            // surface Content-Language
            if s.Locale != "" {
                w.Header().Set("Content-Language", s.Locale)
            }
            next.ServeHTTP(w, r)
        })
    }
}

// Lang returns current lang from session or default "ja".
func Lang(r *http.Request) string {
    if s := GetSession(r); s != nil && s.Locale != "" {
        return s.Locale
    }
    if v := r.Context().Value(ctxKeyLocaleFB); v != nil {
        if fb, ok := v.(string); ok && fb != "" {
            return fb
        }
    }
    return "ja"
}
