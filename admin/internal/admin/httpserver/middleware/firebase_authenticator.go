package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	firebaseauth "firebase.google.com/go/v4/auth"
)

// ErrTokenExpired is returned when the Firebase token has expired.
var ErrTokenExpired = errors.New("firebase token expired")

// FirebaseTokenVerifier abstracts the Firebase Admin SDK client for testability.
type FirebaseTokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

// FirebaseAuthenticator validates Firebase ID tokens and maps them onto a User.
type FirebaseAuthenticator struct {
	verifier FirebaseTokenVerifier
}

// NewFirebaseAuthenticator constructs an Authenticator backed by the provided verifier.
func NewFirebaseAuthenticator(verifier FirebaseTokenVerifier) *FirebaseAuthenticator {
	if verifier == nil {
		panic("firebase token verifier is required")
	}
	return &FirebaseAuthenticator{verifier: verifier}
}

// Authenticate verifies the supplied ID token using Firebase and builds a User object.
func (f *FirebaseAuthenticator) Authenticate(r *http.Request, token string) (*User, error) {
	if strings.TrimSpace(token) == "" {
		return nil, NewAuthError(ReasonMissingToken, ErrUnauthorized)
	}

	verified, err := f.verifier.VerifyIDToken(r.Context(), token)
	if err != nil {
		switch {
		case firebaseauth.IsIDTokenExpired(err), errors.Is(err, ErrTokenExpired):
			return nil, NewAuthError(ReasonTokenExpired, err)
		default:
			return nil, NewAuthError(ReasonTokenInvalid, err)
		}
	}

	return &User{
		UID:   verified.UID,
		Email: claimString(verified.Claims["email"]),
		Roles: claimStringSlice(verified.Claims["role"], verified.Claims["roles"]),
		Token: token,
	}, nil
}

func claimString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case *string:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(*v)
	default:
		return ""
	}
}

func claimStringSlice(values ...any) []string {
	seen := make(map[string]struct{})
	var result []string

	appendValue := func(val string) {
		val = strings.TrimSpace(val)
		if val == "" {
			return
		}
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}

	for _, value := range values {
		switch v := value.(type) {
		case string:
			appendValue(v)
		case []string:
			for _, item := range v {
				appendValue(item)
			}
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					appendValue(s)
				}
			}
		case map[string]any:
			for key, val := range v {
				if b, ok := val.(bool); ok && b {
					appendValue(key)
				}
			}
		case nil:
			continue
		default:
			if s := claimString(v); s != "" {
				appendValue(s)
			}
		}
	}
	return result
}
