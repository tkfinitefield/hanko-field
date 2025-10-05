package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	firebaseauth "firebase.google.com/go/v4/auth"
)

const (
	defaultRoleClaim     = "role"
	defaultLocaleClaim   = "locale"
	defaultEmailClaim    = "email"
	defaultFallbackRole  = RoleUser
	defaultVerifyTimeout = 5 * time.Second
)

var (
	// ErrTokenExpired signals that the provided Firebase ID token has expired.
	ErrTokenExpired = errors.New("auth: firebase id token expired")
	// ErrTokenInvalid signals that the provided Firebase ID token is invalid for other reasons.
	ErrTokenInvalid = errors.New("auth: firebase id token invalid")
)

// TokenVerifier verifies Firebase ID tokens.
type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

// UserGetter retrieves Firebase user information.
type UserGetter interface {
	GetUser(ctx context.Context, uid string) (*firebaseauth.UserRecord, error)
}

// Authenticator wires Firebase token verification into HTTP middleware.
type Authenticator struct {
	verifier TokenVerifier
	users    UserGetter

	roleClaim   string
	localeClaim string
	emailClaim  string

	fallbackRole string
	timeout      time.Duration
}

// Option customises Authenticator behaviour.
type Option func(*Authenticator)

// WithUserGetter enables lazy user record loading via Firebase Admin APIs.
func WithUserGetter(getter UserGetter) Option {
	return func(a *Authenticator) {
		a.users = getter
	}
}

// WithRoleClaim overrides the custom claim used for role extraction.
func WithRoleClaim(claim string) Option {
	return func(a *Authenticator) {
		claim = strings.TrimSpace(claim)
		if claim != "" {
			a.roleClaim = claim
		}
	}
}

// WithLocaleClaim overrides the claim used to populate Identity.Locale.
func WithLocaleClaim(claim string) Option {
	return func(a *Authenticator) {
		claim = strings.TrimSpace(claim)
		if claim != "" {
			a.localeClaim = claim
		}
	}
}

// WithEmailClaim overrides the claim used to populate Identity.Email.
func WithEmailClaim(claim string) Option {
	return func(a *Authenticator) {
		claim = strings.TrimSpace(claim)
		if claim != "" {
			a.emailClaim = claim
		}
	}
}

// WithFallbackRole sets the default role when no custom claim is present.
func WithFallbackRole(role string) Option {
	return func(a *Authenticator) {
		role = normaliseRole(role)
		if role != "" {
			a.fallbackRole = role
		}
	}
}

// WithVerificationTimeout sets the timeout used when verifying tokens and loading users.
func WithVerificationTimeout(d time.Duration) Option {
	return func(a *Authenticator) {
		if d > 0 {
			a.timeout = d
		}
	}
}

// NewAuthenticator constructs a Firebase Authenticator for middleware composition.
func NewAuthenticator(verifier TokenVerifier, opts ...Option) *Authenticator {
	a := &Authenticator{
		verifier:     verifier,
		roleClaim:    defaultRoleClaim,
		localeClaim:  defaultLocaleClaim,
		emailClaim:   defaultEmailClaim,
		fallbackRole: defaultFallbackRole,
		timeout:      defaultVerifyTimeout,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}

	return a
}

// RequireFirebaseAuth verifies the Authorization bearer token and ensures allowed roles.
func (a *Authenticator) RequireFirebaseAuth(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		role = normaliseRole(role)
		if role == "" {
			continue
		}
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				respondAuthError(w, http.StatusUnauthorized, "unauthenticated", "authorization header missing or invalid")
				return
			}
			if a == nil || a.verifier == nil {
				respondAuthError(w, http.StatusUnauthorized, "unauthenticated", "authorization service unavailable")
				return
			}

			ctx, cancel := a.contextWithTimeout(r.Context())
			if cancel != nil {
				defer cancel()
			}

			token, err := a.verifier.VerifyIDToken(ctx, tokenStr)
			if err != nil {
				respondVerificationError(w, err)
				return
			}

			identity := &Identity{
				UID:    token.UID,
				Email:  claimAsString(token.Claims, a.emailClaim),
				Locale: claimAsString(token.Claims, a.localeClaim),
				Roles:  rolesFromClaims(token.Claims, a.roleClaim),
				token:  token,
			}

			if identity.Email == "" {
				// Fallback to the standard email claim if the custom claim was overridden.
				identity.Email = claimAsString(token.Claims, defaultEmailClaim)
			}
			if identity.Locale == "" {
				identity.Locale = claimAsString(token.Claims, defaultLocaleClaim)
			}
			if len(identity.Roles) == 0 && a.fallbackRole != "" {
				identity.Roles = []string{a.fallbackRole}
			}

			if len(identity.Roles) == 0 {
				respondAuthError(w, http.StatusUnauthorized, "missing_role", "no roles associated with identity")
				return
			}

			if len(allowed) > 0 && !hasAllowedRole(identity.Roles, allowed) {
				respondAuthError(w, http.StatusUnauthorized, "insufficient_role", "identity does not have required role")
				return
			}

			if a.users != nil {
				identity.userLoader = func(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
					if uid == "" {
						uid = identity.UID
					}
					ctx, cancel := a.contextWithTimeout(ctx)
					if cancel != nil {
						defer cancel()
					}
					return a.users.GetUser(ctx, uid)
				}
			}

			ctx = WithIdentity(r.Context(), identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (a *Authenticator) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if a == nil || a.timeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, a.timeout)
}

func hasAllowedRole(identityRoles []string, allowed map[string]struct{}) bool {
	for _, role := range identityRoles {
		if _, ok := allowed[normaliseRole(role)]; ok {
			return true
		}
	}
	return false
}

func rolesFromClaims(claims map[string]interface{}, key string) []string {
	raw, ok := claims[key]
	if !ok {
		return nil
	}

	switch v := raw.(type) {
	case string:
		role := normaliseRole(v)
		if role == "" {
			return nil
		}
		return []string{role}
	case []interface{}:
		return uniqueRolesFromInterfaces(v)
	case []string:
		out := make([]string, 0, len(v))
		seen := make(map[string]struct{}, len(v))
		for _, item := range v {
			role := normaliseRole(item)
			if role == "" {
				continue
			}
			if _, exists := seen[role]; exists {
				continue
			}
			seen[role] = struct{}{}
			out = append(out, role)
		}
		return out
	case map[string]interface{}:
		out := make([]string, 0, len(v))
		for key, value := range v {
			boolVal, ok := value.(bool)
			if !ok || !boolVal {
				continue
			}
			role := normaliseRole(key)
			if role == "" {
				continue
			}
			out = append(out, role)
		}
		return out
	default:
		return nil
	}
}

func uniqueRolesFromInterfaces(values []interface{}) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		str, ok := value.(string)
		if !ok {
			continue
		}
		role := normaliseRole(str)
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		out = append(out, role)
	}
	return out
}

func claimAsString(claims map[string]interface{}, key string) string {
	raw, ok := claims[key]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func normaliseRole(role string) string {
	return strings.ToLower(strings.TrimSpace(role))
}

func extractBearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}

	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 {
		return "", false
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}

	token := strings.TrimSpace(parts[1])
	if token == "" {
		return "", false
	}

	return token, true
}

func respondAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   code,
		"message": message,
		"status":  status,
	})
}

func respondVerificationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTokenExpired), firebaseauth.IsIDTokenExpired(err):
		respondAuthError(w, http.StatusUnauthorized, "token_expired", "firebase id token expired")
	case errors.Is(err, ErrTokenInvalid), firebaseauth.IsIDTokenInvalid(err):
		respondAuthError(w, http.StatusUnauthorized, "invalid_token", "firebase id token invalid")
	default:
		respondAuthError(w, http.StatusUnauthorized, "invalid_token", "firebase id token verification failed")
	}
}
