package middleware

import (
    "net/http"
    "os"
    "strings"
)

// Auth inspects Authorization header (development helper) and hydrates user context from session.
// In production, integrate Firebase token verification and populate session on successful login.
func Auth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        env := strings.ToLower(os.Getenv("HANKO_WEB_ENV"))
        // Development helper: "Authorization: Bearer debug:<uid>"
        if env != "prod" {
            if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
                token := strings.TrimPrefix(auth, "Bearer ")
                if strings.HasPrefix(token, "debug:") {
                    uid := strings.TrimPrefix(token, "debug:")
                    s := GetSession(r)
                    if s.UserID != uid {
                        s.UserID = uid
                        s.MarkDirty()
                    }
                }
            }
        }
        next.ServeHTTP(w, r)
    })
}

