package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

type authContextKey string

const userContextKey authContextKey = "auth.user"

// User represents the authenticated staff member.
type User struct {
	UID   string
	Email string
	Roles []string
	Token string
}

// Authenticator resolves an incoming Bearer token into a User.
type Authenticator interface {
	Authenticate(r *http.Request, token string) (*User, error)
}

var (
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")
)

// DefaultAuthenticator accepts any non-empty bearer token and is intended for local development.
func DefaultAuthenticator() Authenticator {
	return &passthroughAuthenticator{}
}

// Auth validates incoming requests and either attaches a User to context or redirects to login.
func Auth(authenticator Authenticator, loginPath string) func(http.Handler) http.Handler {
	if authenticator == nil {
		authenticator = DefaultAuthenticator()
	}
	if loginPath == "" {
		loginPath = "/login"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := parseBearerToken(r.Header.Get("Authorization"))
			user, err := authenticator.Authenticate(r, token)
			if err != nil || user == nil {
				handleUnauthorized(w, r, loginPath)
				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserFromContext retrieves the authenticated user if present.
func UserFromContext(ctx context.Context) (*User, bool) {
	user, ok := ctx.Value(userContextKey).(*User)
	return user, ok
}

func parseBearerToken(header string) string {
	if header == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request, loginPath string) {
	if IsHTMXRequest(r.Context()) {
		w.Header().Set("HX-Redirect", loginPath)
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, loginPath, http.StatusFound)
}

type passthroughAuthenticator struct{}

func (p *passthroughAuthenticator) Authenticate(_ *http.Request, token string) (*User, error) {
	if token == "" {
		return nil, ErrUnauthorized
	}
	return &User{
		UID:   token,
		Roles: []string{"admin"},
		Token: token,
	}, nil
}
