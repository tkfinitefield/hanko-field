package storage

import (
	"context"
	"errors"

	"github.com/hanko-field/api/internal/platform/auth"
)

// ErrPermissionDenied is returned when the caller lacks permission to access the asset.
var ErrPermissionDenied = errors.New("storage: permission denied")

// AuthorizeDownload validates whether the provided identity may access the asset owned by ownerID.
func AuthorizeDownload(identity *auth.Identity, ownerID string, allowAnonymous bool) error {
	if allowAnonymous {
		return nil
	}
	if identity == nil {
		return ErrPermissionDenied
	}
	if ownerID != "" && identity.UID == ownerID {
		return nil
	}
	if identity.HasAnyRole(auth.RoleStaff, auth.RoleAdmin) {
		return nil
	}
	return ErrPermissionDenied
}

// AuthorizeDownloadFromContext extracts the identity from context and validates access.
func AuthorizeDownloadFromContext(ctx context.Context, ownerID string, allowAnonymous bool) (*auth.Identity, error) {
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok && !allowAnonymous {
		return nil, ErrPermissionDenied
	}
	if err := AuthorizeDownload(identity, ownerID, allowAnonymous); err != nil {
		return nil, err
	}
	return identity, nil
}
