package idempotency

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"
)

// Status represents the lifecycle state of an idempotency record.
type Status string

const (
	// DefaultTTL is the default duration that idempotency records are retained.
	DefaultTTL = 24 * time.Hour
	// StatusPending indicates that a request has reserved the key but not yet persisted a response.
	StatusPending Status = "pending"
	// StatusCompleted indicates that the response for the key has been stored and can be replayed.
	StatusCompleted Status = "completed"
)

// ReservationState describes the outcome of attempting to reserve an idempotency key.
type ReservationState int

const (
	// ReservationStateNew means no existing reservation was found and the caller may continue processing.
	ReservationStateNew ReservationState = iota
	// ReservationStateCompleted means a previous response was found and should be replayed.
	ReservationStateCompleted
	// ReservationStatePending means another request is currently processing this key.
	ReservationStatePending
)

// Reservation encapsulates the result of reserving a key, including the stored record if available.
type Reservation struct {
	State  ReservationState
	Record Record
}

// Record captures the persisted response metadata for an idempotency key.
type Record struct {
	Key             string
	Fingerprint     string
	Status          Status
	ResponseStatus  int
	ResponseHeaders map[string][]string
	ResponseBody    []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ExpiresAt       time.Time
}

// Response represents the HTTP response that should be stored for future replays.
type Response struct {
	Status  int
	Headers http.Header
	Body    []byte
}

// Store persists idempotency reservations and responses.
type Store interface {
	Reserve(ctx context.Context, key, fingerprint string, now time.Time, ttl time.Duration) (Reservation, error)
	SaveResponse(ctx context.Context, key, fingerprint string, resp Response, now time.Time, ttl time.Duration) error
	Release(ctx context.Context, key, fingerprint string) error
	CleanupExpired(ctx context.Context, now time.Time, limit int) (int, error)
}

var (
	// ErrFingerprintMismatch is returned when an idempotency key is reused with a different request fingerprint.
	ErrFingerprintMismatch = errors.New("idempotency: key reserved for different request fingerprint")
)

func compositeKey(key, fingerprint string) string {
	return sha256Hex([]byte(strings.TrimSpace(key)))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sanitizeHeaders(header http.Header) map[string][]string {
	if len(header) == 0 {
		return nil
	}

	filtered := make(map[string][]string, len(header))
	for name, values := range header {
		canonical := http.CanonicalHeaderKey(name)
		if shouldOmitHeader(canonical) {
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		filtered[canonical] = copied
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func shouldOmitHeader(name string) bool {
	switch strings.ToLower(name) {
	case "content-length", "date", "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade":
		return true
	default:
		return false
	}
}

func headersFromRecord(values map[string][]string) http.Header {
	if len(values) == 0 {
		return http.Header{}
	}

	header := make(http.Header, len(values))
	for name, vals := range values {
		copied := make([]string, len(vals))
		copy(copied, vals)
		header[name] = copied
	}
	return header
}
