package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"path"
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
				sess.SetFeatureFlags(user.FeatureFlags)
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

// ContextWithUser returns a new context containing the provided user. Primarily used in tests.
func ContextWithUser(ctx context.Context, user *User) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, userContextKey, user)
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
	c, err := r.Cookie("Authorization")
	if err != nil {
		return ""
	}
	val := strings.TrimSpace(c.Value)
	if val == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(val), "bearer ") {
		return strings.TrimSpace(val[7:])
	}
	return val
}

func handleUnauthorized(w http.ResponseWriter, r *http.Request, loginPath, reason string) {
	if strings.TrimSpace(loginPath) == "" {
		loginPath = "/login"
	}
	if reason == "" {
		reason = ReasonTokenInvalid
	}

	next := extractNextParam(r, loginPath)
	target := buildLoginRedirectURL(loginPath, reason, next)

	if IsHTMXRequest(r.Context()) {
		if reason == ReasonTokenExpired {
			w.Header().Set("HX-Refresh", "true")
		} else {
			w.Header().Set("HX-Redirect", target)
		}
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, target, http.StatusFound)
}

func extractNextParam(r *http.Request, loginPath string) string {
	if r == nil || r.URL == nil {
		return ""
	}

	base := deriveBasePath(loginPath)
	candidate := sanitizeRedirectCandidate(base, r.URL.RequestURI())
	if candidate == "" {
		pathOnly := r.URL.Path
		if pathOnly != "" {
			candidate = sanitizeRedirectCandidate(base, pathOnly)
			if candidate != "" && r.URL.RawQuery != "" {
				candidate += "?" + r.URL.RawQuery
			}
		}
	}

	if candidate == "" {
		return ""
	}

	if loginPath != "" && samePath(pathComponent(candidate), loginPath) {
		return ""
	}

	return candidate
}

func buildLoginRedirectURL(loginPath, reason, next string) string {
	parsed, err := url.Parse(loginPath)
	if err != nil {
		if reason == "" && next == "" {
			return loginPath
		}
		params := url.Values{}
		if reason != "" {
			params.Set("reason", reason)
		}
		if next != "" {
			params.Set("next", next)
		}
		if strings.Contains(loginPath, "?") {
			return loginPath + "&" + params.Encode()
		}
		return loginPath + "?" + params.Encode()
	}

	q := parsed.Query()
	if reason != "" {
		q.Set("reason", reason)
	}
	if next != "" {
		q.Set("next", next)
	}
	parsed.RawQuery = q.Encode()
	return parsed.String()
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	trim := func(p string) string {
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		for len(p) > 1 && strings.HasSuffix(p, "/") {
			p = strings.TrimSuffix(p, "/")
		}
		return p
	}
	return trim(a) == trim(b)
}

func sanitizeRedirectCandidate(basePath, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "" || parsed.Host != "" {
		return ""
	}

	pathPart := parsed.Path
	if pathPart == "" {
		pathPart = "/"
	}

	unescaped, err := url.PathUnescape(pathPart)
	if err != nil {
		return ""
	}
	if strings.Contains(unescaped, "\\") {
		return ""
	}

	cleaned := path.Clean(unescaped)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	if strings.HasPrefix(cleaned, "//") {
		return ""
	}

	normalisedBase := normalizeRedirectBase(basePath)
	if normalisedBase != "/" && !hasPathPrefix(cleaned, normalisedBase) {
		return ""
	}

	result := cleaned
	if parsed.RawQuery != "" {
		result += "?" + parsed.RawQuery
	}
	if parsed.Fragment != "" {
		result += "#" + parsed.Fragment
	}
	return result
}

func deriveBasePath(loginPath string) string {
	loginPath = strings.TrimSpace(loginPath)
	if loginPath == "" {
		return "/"
	}
	if !strings.HasPrefix(loginPath, "/") {
		loginPath = "/" + loginPath
	}
	base := path.Dir(loginPath)
	if base == "." || base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 && strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}
	return base
}

func normalizeRedirectBase(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if len(base) > 1 && strings.HasSuffix(base, "/") {
		base = strings.TrimRight(base, "/")
	}
	return base
}

func hasPathPrefix(candidate, base string) bool {
	if base == "/" {
		return strings.HasPrefix(candidate, "/")
	}
	if !strings.HasPrefix(candidate, base) {
		return false
	}
	if len(candidate) == len(base) {
		return true
	}
	return candidate[len(base)] == '/'
}

func pathComponent(raw string) string {
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return parsed.Path
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
