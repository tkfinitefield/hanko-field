package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"

	appsession "finitefield.org/hanko-admin/internal/admin/session"
)

type authContextKey string

const userContextKey authContextKey = "auth.user"

// User represents the authenticated staff member.
type User struct {
	UID          string
	Email        string
	Roles        []string
	Token        string
	FeatureFlags map[string]bool
}

// Authenticator resolves an incoming Bearer token into a User.
type Authenticator interface {
	Authenticate(r *http.Request, token string) (*User, error)
}

var (
	// ErrUnauthorized is returned when authentication fails.
	ErrUnauthorized = errors.New("unauthorized")
)

// AuthError contains reason codes for failed authentication attempts.
type AuthError struct {
	Reason string
	Err    error
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	if e.Err == nil {
		return e.Reason
	}
	return e.Reason + ": " + e.Err.Error()
}

// Unwrap returns the underlying error.
func (e *AuthError) Unwrap() error {
	return e.Err
}

// NewAuthError constructs an AuthError with the provided reason.
func NewAuthError(reason string, err error) error {
	return &AuthError{Reason: reason, Err: err}
}

const (
	// ReasonMissingToken indicates an auth attempt without credentials.
	ReasonMissingToken = "missing_token"
	// ReasonTokenInvalid indicates a malformed or invalid token.
	ReasonTokenInvalid = "token_invalid"
	// ReasonTokenExpired indicates an expired token which may be recoverable.
	ReasonTokenExpired = "token_expired"
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
			if token == "" {
				token = cookieToken(r)
			}
			if strings.TrimSpace(token) == "" {
				log.Printf("auth failure: reason=%s error=%v", ReasonMissingToken, ErrUnauthorized)
				destroySession(r.Context())
				handleUnauthorized(w, r, loginPath, ReasonMissingToken)
				return
			}

			user, err := authenticator.Authenticate(r, token)
			if err != nil || user == nil {
				reason := ReasonTokenInvalid
				var authErr *AuthError
				if errors.As(err, &authErr) {
					if authErr.Reason != "" {
						reason = authErr.Reason
					}
					err = authErr.Err
				}
				if err == nil {
					err = ErrUnauthorized
				}
				log.Printf("auth failure: reason=%s error=%v", reason, err)
				destroySession(r.Context())
				handleUnauthorized(w, r, loginPath, reason)
				return
			}

			if sess, ok := SessionFromContext(r.Context()); ok {
				sess.SetUser(&appsession.User{
					UID:   user.UID,
					Email: user.Email,
					Roles: append([]string(nil), user.Roles...),
				})
				if len(user.FeatureFlags) > 0 {
					sess.SetFeatureFlags(user.FeatureFlags)
				}
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

func cookieToken(r *http.Request) string {
	candidates := []string{"Authorization", "__session", "idToken", "IDToken"}
	for _, name := range candidates {
		c, err := r.Cookie(name)
		if err != nil {
			continue
		}
		val := strings.TrimSpace(c.Value)
		if val == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(val), "bearer ") {
			return strings.TrimSpace(val[7:])
		}
		return val
	}
	return ""
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request, loginPath, reason string) {
	if reason == "" {
		reason = ReasonTokenInvalid
	}

	if IsHTMXRequest(r.Context()) {
		if reason == ReasonTokenExpired {
			w.Header().Set("HX-Refresh", "true")
		} else {
			w.Header().Set("HX-Redirect", loginPath)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	redirectURL := loginPath
	if reason == ReasonTokenExpired {
		if u, err := url.Parse(loginPath); err == nil {
			q := u.Query()
			q.Set("reason", "expired")
			u.RawQuery = q.Encode()
			redirectURL = u.String()
		}
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func destroySession(ctx context.Context) {
	if sess, ok := SessionFromContext(ctx); ok && sess != nil {
		sess.Destroy()
	}
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
