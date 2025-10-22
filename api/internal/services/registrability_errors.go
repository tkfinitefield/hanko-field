package services

import "errors"

var (
	// ErrRegistrabilityUnavailable indicates the external registrability service is unavailable.
	ErrRegistrabilityUnavailable = errors.New("registrability: unavailable")
	// ErrRegistrabilityRateLimited indicates the external service rejected the request due to rate limits.
	ErrRegistrabilityRateLimited = errors.New("registrability: rate limited")
	// ErrRegistrabilityInvalidInput indicates the payload provided to the registrability service was invalid.
	ErrRegistrabilityInvalidInput = errors.New("registrability: invalid input")
)
