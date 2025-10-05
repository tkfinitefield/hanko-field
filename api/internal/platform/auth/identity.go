package auth

import (
	"context"
	"errors"
	"strings"
	"sync"

	firebaseauth "firebase.google.com/go/v4/auth"
)

// Role constants used throughout the API when checking authorisation boundaries.
const (
	RoleUser  = "user"
	RoleStaff = "staff"
	RoleAdmin = "admin"
)

// ErrUserLoaderUnavailable indicates that the identity was created without a user loader.
var ErrUserLoaderUnavailable = errors.New("auth: user loader not configured")

// Identity captures the authenticated principal details extracted from a Firebase ID token.
type Identity struct {
	UID    string
	Email  string
	Roles  []string
	Locale string

	token *firebaseauth.Token

	userLoader UserLoader
	once       sync.Once
	userRecord *firebaseauth.UserRecord
	userErr    error
}

// Token exposes the decoded Firebase ID token associated with this identity.
func (i *Identity) Token() *firebaseauth.Token {
	if i == nil {
		return nil
	}
	return i.token
}

// HasRole reports whether the identity includes the requested role (case-insensitive).
func (i *Identity) HasRole(role string) bool {
	if i == nil {
		return false
	}
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return false
	}
	for _, r := range i.Roles {
		if strings.EqualFold(r, role) {
			return true
		}
	}
	return false
}

// HasAnyRole reports whether the identity includes any of the provided roles.
func (i *Identity) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if i.HasRole(role) {
			return true
		}
	}
	return false
}

// User resolves the Firebase user profile using the injected loader on first access.
func (i *Identity) User(ctx context.Context) (*firebaseauth.UserRecord, error) {
	if i == nil {
		return nil, ErrUserLoaderUnavailable
	}
	if i.userLoader == nil {
		return nil, ErrUserLoaderUnavailable
	}

	i.once.Do(func() {
		i.userRecord, i.userErr = i.userLoader(ctx, i.UID)
	})

	return i.userRecord, i.userErr
}

type contextKey string

const identityContextKey contextKey = "github.com/hanko-field/api/internal/platform/auth/identity"

// WithIdentity stores the identity within the context for downstream handlers.
func WithIdentity(ctx context.Context, identity *Identity) context.Context {
	return context.WithValue(ctx, identityContextKey, identity)
}

// IdentityFromContext retrieves the identity previously stored in context.
func IdentityFromContext(ctx context.Context) (*Identity, bool) {
	identity, ok := ctx.Value(identityContextKey).(*Identity)
	if !ok || identity == nil {
		return nil, false
	}
	return identity, true
}

// UserLoader fetches the Firebase user profile corresponding to a UID.
type UserLoader func(ctx context.Context, uid string) (*firebaseauth.UserRecord, error)
