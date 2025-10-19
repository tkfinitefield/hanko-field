package services

import "errors"

var (
	// ErrPromotionRepositoryMissing indicates the promotion repository dependency is absent.
	ErrPromotionRepositoryMissing = errors.New("promotion service: repository is not configured")
	// ErrPromotionInvalidCode signals the supplied promotion code is missing or invalid.
	ErrPromotionInvalidCode = errors.New("promotion service: invalid promotion code")
	// ErrPromotionNotFound indicates no promotion exists for the provided code.
	ErrPromotionNotFound = errors.New("promotion service: promotion not found")
	// ErrPromotionUnavailable indicates the promotion exists but is not exposed to the public channel.
	ErrPromotionUnavailable = errors.New("promotion service: promotion unavailable")
	// ErrPromotionOperationUnsupported marks operations that have not been implemented yet.
	ErrPromotionOperationUnsupported = errors.New("promotion service: operation unsupported")
)
