package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"

	appsession "finitefield.org/hanko-admin/internal/admin/session"
)

type sessionContextKey string

const requestSessionKey sessionContextKey = "admin.session"

// SessionStore abstracts the session manager for middleware integration.
type SessionStore interface {
	Load(*http.Request) (*appsession.Session, error)
	New() *appsession.Session
	Save(http.ResponseWriter, *appsession.Session) error
	Destroy(http.ResponseWriter)
}

// Session attaches the decoded session to the request context and persists
// changes back to the client cookie.
func Session(store SessionStore) func(http.Handler) http.Handler {
	if store == nil {
		panic("session store is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sess, err := store.Load(r)
			if errors.Is(err, appsession.ErrExpired) {
				log.Printf("session expired: resetting")
				store.Destroy(w)
				sess = store.New()
			} else if err != nil || sess == nil {
				if err != nil {
					log.Printf("session load failed: %v", err)
				}
				sess = store.New()
			}

			ctx := context.WithValue(r.Context(), requestSessionKey, sess)
			rr := r.WithContext(ctx)

			next.ServeHTTP(w, rr)

			if err := store.Save(w, sess); err != nil {
				log.Printf("session save failed: %v", err)
			}
		})
	}
}

// SessionFromContext retrieves the session attached to this request.
func SessionFromContext(ctx context.Context) (*appsession.Session, bool) {
	if ctx == nil {
		return nil, false
	}
	sess, ok := ctx.Value(requestSessionKey).(*appsession.Session)
	return sess, ok && sess != nil
}
