package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultSignatureHeader = "X-Signature"
	defaultTimestampHeader = "X-Signature-Timestamp"
	defaultNonceHeader     = "X-Signature-Nonce"

	defaultClockSkew = 5 * time.Minute
	defaultNonceTTL  = 5 * time.Minute
)

// SecretProvider resolves shared secrets used for HMAC validation.
type SecretProvider interface {
	GetSecret(ctx context.Context, name string) (string, error)
}

// SecretProviderFunc adapts a function to the SecretProvider interface.
type SecretProviderFunc func(context.Context, string) (string, error)

// GetSecret implements SecretProvider.
func (f SecretProviderFunc) GetSecret(ctx context.Context, name string) (string, error) {
	if f == nil {
		return "", errors.New("auth: secret provider not configured")
	}
	return f(ctx, name)
}

// NonceStore tracks unique nonces for replay prevention.
type NonceStore interface {
	// UseNonce records the nonce if it has not been seen before within the scope. The boolean indicates
	// whether the nonce was stored (true) or already existed (false).
	UseNonce(ctx context.Context, scope, nonce string, expiry time.Time) (bool, error)
}

// InMemoryNonceStore offers an in-memory nonce registry suitable for tests and local development.
type InMemoryNonceStore struct {
	mu     sync.Mutex
	nonces map[string]time.Time
}

// NewInMemoryNonceStore constructs the store.
func NewInMemoryNonceStore() *InMemoryNonceStore {
	return &InMemoryNonceStore{nonces: make(map[string]time.Time)}
}

// UseNonce records the nonce until the provided expiry, rejecting replays until then.
func (s *InMemoryNonceStore) UseNonce(_ context.Context, scope, nonce string, expiry time.Time) (bool, error) {
	if scope == "" || nonce == "" {
		return false, errors.New("auth: scope and nonce are required")
	}

	key := scope + "::" + nonce

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for k, exp := range s.nonces {
		if exp.Before(now) {
			delete(s.nonces, k)
		}
	}

	if expiry.Before(now) {
		return false, errors.New("auth: nonce expiry is in the past")
	}

	if existing, ok := s.nonces[key]; ok && existing.After(now) {
		return false, nil
	}

	s.nonces[key] = expiry
	return true, nil
}

// HMACValidator verifies signed requests from trusted integrations (webhooks, internal services).
type HMACValidator struct {
	provider SecretProvider
	nonces   NonceStore

	logger  Logger
	metrics MetricsRecorder
	now     func() time.Time

	signatureHeader string
	timestampHeader string
	nonceHeader     string

	clockSkew time.Duration
	nonceTTL  time.Duration

	secretCache sync.Map
}

// HMACOption customises the validator.
type HMACOption func(*HMACValidator)

// NewHMACValidator builds a validator using the given secret provider and nonce store.
func NewHMACValidator(provider SecretProvider, nonces NonceStore, opts ...HMACOption) *HMACValidator {
	validator := &HMACValidator{
		provider:        provider,
		nonces:          nonces,
		logger:          log.Default(),
		now:             time.Now,
		signatureHeader: defaultSignatureHeader,
		timestampHeader: defaultTimestampHeader,
		nonceHeader:     defaultNonceHeader,
		clockSkew:       defaultClockSkew,
		nonceTTL:        defaultNonceTTL,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(validator)
		}
	}

	return validator
}

// WithHMACLogger overrides the validator logger.
func WithHMACLogger(logger Logger) HMACOption {
	return func(v *HMACValidator) {
		if logger != nil {
			v.logger = logger
		}
	}
}

// WithHMACMetrics sets the metrics recorder.
func WithHMACMetrics(metrics MetricsRecorder) HMACOption {
	return func(v *HMACValidator) {
		v.metrics = metrics
	}
}

// WithHMACClock injects a custom clock, primarily for tests.
func WithHMACClock(now func() time.Time) HMACOption {
	return func(v *HMACValidator) {
		if now != nil {
			v.now = now
		}
	}
}

// WithHMACHeaders customises the header names used by the middleware.
func WithHMACHeaders(signature, timestamp, nonce string) HMACOption {
	return func(v *HMACValidator) {
		if signature != "" {
			v.signatureHeader = signature
		}
		if timestamp != "" {
			v.timestampHeader = timestamp
		}
		if nonce != "" {
			v.nonceHeader = nonce
		}
	}
}

// WithHMACClockSkew adjusts the accepted timestamp skew.
func WithHMACClockSkew(d time.Duration) HMACOption {
	return func(v *HMACValidator) {
		if d > 0 {
			v.clockSkew = d
		}
	}
}

// WithHMACNonceTTL customises the nonce retention duration.
func WithHMACNonceTTL(d time.Duration) HMACOption {
	return func(v *HMACValidator) {
		if d > 0 {
			v.nonceTTL = d
		}
	}
}

// HMACMetadata describes the verification context for downstream handlers.
type HMACMetadata struct {
	SecretName   string
	Timestamp    time.Time
	Nonce        string
	Signature    []byte
	RawSignature string
}

type hmacContextKey struct{}

// WithHMACMetadata stores the metadata on the context.
func WithHMACMetadata(ctx context.Context, meta *HMACMetadata) context.Context {
	if meta == nil {
		return ctx
	}
	return context.WithValue(ctx, hmacContextKey{}, meta)
}

// HMACMetadataFromContext retrieves metadata from the context.
func HMACMetadataFromContext(ctx context.Context) (*HMACMetadata, bool) {
	meta, ok := ctx.Value(hmacContextKey{}).(*HMACMetadata)
	if !ok || meta == nil {
		return nil, false
	}
	return meta, true
}

// RequireHMAC enforces the presence of a valid HMAC signature on the request.
func (v *HMACValidator) RequireHMAC(secretName string) func(http.Handler) http.Handler {
	scopedSecret := strings.TrimSpace(secretName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := v.now()
			ctx := r.Context()

			if scopedSecret == "" {
				v.record(ctx, false, "secret_not_configured", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "hmac secret not configured")
				return
			}

			secret, err := v.loadSecret(ctx, scopedSecret)
			if err != nil {
				if v.logger != nil {
					v.logger.Printf("auth: hmac secret lookup failed: %v", err)
				}
				v.record(ctx, false, "secret_unavailable", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "hmac secret unavailable")
				return
			}

			signatureValue := strings.TrimSpace(r.Header.Get(v.signatureHeader))
			if signatureValue == "" {
				v.record(ctx, false, "signature_missing", start)
				respondAuthError(w, http.StatusUnauthorized, "signature_missing", "signature header missing")
				return
			}

			timestampValue := strings.TrimSpace(r.Header.Get(v.timestampHeader))
			if timestampValue == "" {
				v.record(ctx, false, "timestamp_missing", start)
				respondAuthError(w, http.StatusUnauthorized, "timestamp_missing", "signature timestamp missing")
				return
			}

			timestamp, err := parseSignatureTimestamp(timestampValue)
			if err != nil {
				v.record(ctx, false, "timestamp_invalid", start)
				respondAuthError(w, http.StatusUnauthorized, "timestamp_invalid", "signature timestamp invalid")
				return
			}

			if skew := v.now().Sub(timestamp); skew > v.clockSkew || skew < -v.clockSkew {
				v.record(ctx, false, "timestamp_skew", start)
				respondAuthError(w, http.StatusUnauthorized, "timestamp_skew", "signature timestamp outside allowed window")
				return
			}

			nonce := strings.TrimSpace(r.Header.Get(v.nonceHeader))
			if nonce == "" {
				v.record(ctx, false, "nonce_missing", start)
				respondAuthError(w, http.StatusUnauthorized, "nonce_missing", "signature nonce missing")
				return
			}

			bodyBytes, err := readAndRestoreBody(r)
			if err != nil {
				v.record(ctx, false, "body_unreadable", start)
				respondAuthError(w, http.StatusBadRequest, "invalid_body", "unable to read body for signature verification")
				return
			}

			canonical := buildCanonicalString(r, bodyBytes, timestampValue, nonce)
			signature, err := decodeSignature(signatureValue)
			if err != nil {
				v.record(ctx, false, "signature_invalid", start)
				respondAuthError(w, http.StatusUnauthorized, "signature_invalid", "signature encoding invalid")
				return
			}

			expected := computeHMAC(secret, canonical)
			if !hmac.Equal(signature, expected) {
				v.record(ctx, false, "signature_mismatch", start)
				respondAuthError(w, http.StatusUnauthorized, "signature_mismatch", "signature verification failed")
				return
			}

			if v.nonces == nil {
				v.record(ctx, false, "nonce_store_unavailable", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "nonce store unavailable")
				return
			}

			ttl := timestamp.Add(v.nonceTTL)
			if ttlBeforeNow := ttl.Before(v.now()); ttlBeforeNow {
				ttl = v.now().Add(v.nonceTTL)
			}

			stored, err := v.nonces.UseNonce(ctx, scopedSecret, nonce, ttl)
			if err != nil {
				if v.logger != nil {
					v.logger.Printf("auth: nonce store error: %v", err)
				}
				v.record(ctx, false, "nonce_store_error", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "nonce storage error")
				return
			}

			if !stored {
				v.record(ctx, false, "nonce_replay", start)
				respondAuthError(w, http.StatusUnauthorized, "nonce_replay", "duplicate signature nonce")
				return
			}

			meta := &HMACMetadata{
				SecretName:   scopedSecret,
				Timestamp:    timestamp,
				Nonce:        nonce,
				Signature:    signature,
				RawSignature: signatureValue,
			}

			v.record(ctx, true, "ok", start)
			next.ServeHTTP(w, r.WithContext(WithHMACMetadata(ctx, meta)))
		})
	}
}

// RequireHMACResolver selects a secret dynamically using the supplied resolver.
func (v *HMACValidator) RequireHMACResolver(resolver func(*http.Request) (string, bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolver == nil {
				start := v.now()
				v.record(r.Context(), false, "secret_not_configured", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "hmac secret resolver not configured")
				return
			}

			secretName, ok := resolver(r)
			if !ok || strings.TrimSpace(secretName) == "" {
				start := v.now()
				v.record(r.Context(), false, "provider_unknown", start)
				respondAuthError(w, http.StatusUnauthorized, "unknown_provider", "webhook provider not recognised")
				return
			}

			v.RequireHMAC(secretName)(next).ServeHTTP(w, r)
		})
	}
}

func (v *HMACValidator) record(ctx context.Context, success bool, reason string, start time.Time) {
	if v == nil || v.metrics == nil {
		return
	}
	duration := v.now().Sub(start)
	v.metrics.RecordVerification(ctx, "hmac", success, reason, duration)
}

func (v *HMACValidator) loadSecret(ctx context.Context, name string) ([]byte, error) {
	if v == nil || v.provider == nil {
		return nil, errors.New("auth: secret provider not configured")
	}

	if cached, ok := v.secretCache.Load(name); ok {
		if secret, ok := cached.([]byte); ok && len(secret) > 0 {
			return secret, nil
		}
	}

	raw, err := v.provider.GetSecret(ctx, name)
	if err != nil {
		return nil, err
	}

	secret := []byte(raw)
	if len(secret) == 0 {
		return nil, errors.New("auth: secret is empty")
	}

	v.secretCache.Store(name, secret)
	return secret, nil
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	defer r.Body.Close()

	buf, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(bytes.NewReader(buf))
	return buf, nil
}

func decodeSignature(value string) ([]byte, error) {
	if value == "" {
		return nil, errors.New("auth: empty signature")
	}
	if decoded, err := base64.StdEncoding.DecodeString(value); err == nil {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(value); err == nil {
		return decoded, nil
	}
	return nil, errors.New("auth: signature must be base64 or hex encoded")
}

func parseSignatureTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("auth: timestamp empty")
	}

	value = strings.TrimSpace(value)
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts.UTC(), nil
	}

	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UTC(), nil
	}

	if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.Unix(seconds, 0).UTC(), nil
	}

	return time.Time{}, fmt.Errorf("auth: unable to parse timestamp %q", value)
}

func buildCanonicalString(r *http.Request, body []byte, timestamp, nonce string) []byte {
	method := strings.ToUpper(r.Method)
	path := r.URL.EscapedPath()
	if path == "" {
		path = "/"
	}

	hash := sha256.Sum256(body)
	canonical := strings.Join([]string{
		method,
		path,
		timestamp,
		nonce,
		hex.EncodeToString(hash[:]),
	}, "\n")
	return []byte(canonical)
}

func computeHMAC(secret []byte, message []byte) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(message)
	return mac.Sum(nil)
}
