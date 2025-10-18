package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwt "github.com/golang-jwt/jwt/v4"
)

var (
	// ErrJWKSKeyNotFound is returned when the requested key ID is absent from the JWKS document.
	ErrJWKSKeyNotFound = errors.New("auth: jwks key not found")
	// ErrJWKSFetchFailed wraps transport or decoding errors while refreshing JWKS.
	ErrJWKSFetchFailed = errors.New("auth: jwks fetch failed")
)

// Logger captures the minimal logging contract used by the auth package.
type Logger interface {
	Printf(format string, args ...any)
}

// MetricsRecorder records verification outcomes for observability.
type MetricsRecorder interface {
	RecordVerification(ctx context.Context, kind string, success bool, reason string, duration time.Duration)
}

// MetricsRecorderFunc adapts a function to MetricsRecorder.
type MetricsRecorderFunc func(context.Context, string, bool, string, time.Duration)

// RecordVerification implements MetricsRecorder.
func (f MetricsRecorderFunc) RecordVerification(ctx context.Context, kind string, success bool, reason string, duration time.Duration) {
	if f != nil {
		f(ctx, kind, success, reason, duration)
	}
}

const (
	defaultJWKSRefreshInterval = 15 * time.Minute
	defaultJWKSRefreshTimeout  = 5 * time.Second
)

// JWKSCache lazily fetches and caches JSON Web Keys with optional background refresh.
type JWKSCache struct {
	url    string
	client *http.Client
	logger Logger
	now    func() time.Time

	refreshInterval time.Duration
	refreshTimeout  time.Duration

	background bool

	mu       sync.RWMutex
	keys     map[string]jose.JSONWebKey
	expiry   time.Time
	prefetch time.Time

	refreshMu       sync.Mutex
	asyncRefreshing atomic.Bool
}

// JWKSOption customises JWKSCache behaviour.
type JWKSOption func(*JWKSCache)

// NewJWKSCache constructs a JWKS cache for the provided URL.
func NewJWKSCache(url string, opts ...JWKSOption) *JWKSCache {
	cache := &JWKSCache{
		url:             url,
		client:          &http.Client{Timeout: 10 * time.Second},
		logger:          log.Default(),
		now:             time.Now,
		refreshInterval: defaultJWKSRefreshInterval,
		refreshTimeout:  defaultJWKSRefreshTimeout,
		background:      true,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cache)
		}
	}

	return cache
}

// WithJWKSHTTPClient overrides the HTTP client used to fetch JWKS documents.
func WithJWKSHTTPClient(client *http.Client) JWKSOption {
	return func(c *JWKSCache) {
		if client != nil {
			c.client = client
		}
	}
}

// WithJWKSLogger sets a custom logger for JWKS operations.
func WithJWKSLogger(logger Logger) JWKSOption {
	return func(c *JWKSCache) {
		if logger != nil {
			c.logger = logger
		}
	}
}

// WithJWKSRefreshInterval overrides the fallback refresh interval when cache headers are absent.
func WithJWKSRefreshInterval(d time.Duration) JWKSOption {
	return func(c *JWKSCache) {
		if d > 0 {
			c.refreshInterval = d
		}
	}
}

// WithJWKSRefreshTimeout sets the timeout applied to JWKS fetches.
func WithJWKSRefreshTimeout(d time.Duration) JWKSOption {
	return func(c *JWKSCache) {
		if d > 0 {
			c.refreshTimeout = d
		}
	}
}

// WithJWKSClock injects a custom time source (useful for tests).
func WithJWKSClock(now func() time.Time) JWKSOption {
	return func(c *JWKSCache) {
		if now != nil {
			c.now = now
		}
	}
}

// WithoutJWKSBackgroundRefresh disables background refresh scheduling.
func WithoutJWKSBackgroundRefresh() JWKSOption {
	return func(c *JWKSCache) {
		c.background = false
	}
}

// Keyfunc returns a jwt.Keyfunc backed by the cache.
func (c *JWKSCache) Keyfunc(ctx context.Context) jwt.Keyfunc {
	if ctx == nil {
		ctx = context.Background()
	}

	return func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("auth: token missing kid header")
		}

		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("auth: unexpected signing method %v", token.Method)
		}

		return c.Key(ctx, kid)
	}
}

// Key resolves the public key for the provided kid, refreshing the JWKS if required.
func (c *JWKSCache) Key(ctx context.Context, kid string) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	now := c.now()
	if c.needsRefresh(now) {
		if err := c.refresh(ctx); err != nil {
			return nil, err
		}
	}

	if key, ok := c.cachedKey(kid); ok {
		if c.shouldPrefetch(now) {
			c.scheduleRefresh()
		}
		return key, nil
	}

	if err := c.refresh(ctx); err != nil {
		return nil, err
	}

	if key, ok := c.cachedKey(kid); ok {
		return key, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrJWKSKeyNotFound, kid)
}

func (c *JWKSCache) cachedKey(kid string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.keys == nil {
		return nil, false
	}
	jwk, ok := c.keys[kid]
	if !ok {
		return nil, false
	}
	return jwk.Key, true
}

func (c *JWKSCache) needsRefresh(now time.Time) bool {
	c.mu.RLock()
	empty := len(c.keys) == 0
	expiry := c.expiry
	c.mu.RUnlock()
	if empty {
		return true
	}
	if expiry.IsZero() {
		return false
	}
	return !now.Before(expiry)
}

func (c *JWKSCache) shouldPrefetch(now time.Time) bool {
	if !c.background {
		return false
	}
	c.mu.RLock()
	prefetch := c.prefetch
	expiry := c.expiry
	c.mu.RUnlock()
	if prefetch.IsZero() || expiry.IsZero() {
		return false
	}
	if now.After(expiry) {
		return false
	}
	return !now.Before(prefetch)
}

func (c *JWKSCache) scheduleRefresh() {
	if !c.background {
		return
	}
	if !c.asyncRefreshing.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer c.asyncRefreshing.Store(false)
		if err := c.refresh(context.Background()); err != nil && c.logger != nil {
			c.logger.Printf("auth: background jwks refresh failed: %v", err)
		}
	}()
}

func (c *JWKSCache) refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	if ctx == nil {
		ctx = context.Background()
	}
	if c.refreshTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.refreshTimeout)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSFetchFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: unexpected status %d", ErrJWKSFetchFailed, resp.StatusCode)
	}

	var set jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return fmt.Errorf("%w: decode jwks: %v", ErrJWKSFetchFailed, err)
	}

	keys := make(map[string]jose.JSONWebKey, len(set.Keys))
	for _, jwk := range set.Keys {
		if jwk.KeyID == "" {
			continue
		}
		if !jwk.Valid() {
			continue
		}
		keys[jwk.KeyID] = jwk
	}

	if len(keys) == 0 {
		return fmt.Errorf("%w: empty key set", ErrJWKSFetchFailed)
	}

	validity := c.refreshInterval
	if cacheCtl := resp.Header.Get("Cache-Control"); cacheCtl != "" {
		if maxAge := parseMaxAge(cacheCtl); maxAge > 0 {
			validity = maxAge
		}
	}
	if expires := resp.Header.Get("Expires"); expires != "" {
		if ts, err := http.ParseTime(expires); err == nil {
			if delta := ts.Sub(c.now()); delta > 0 {
				validity = delta
			}
		}
	}
	if validity <= 0 {
		validity = defaultJWKSRefreshInterval
	}

	now := c.now()
	expiry := now.Add(validity)
	prefetch := now.Add(validity / 2)
	if !prefetch.Before(expiry) {
		prefetch = now.Add(validity / 2)
	}

	c.mu.Lock()
	c.keys = keys
	c.expiry = expiry
	c.prefetch = prefetch
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Printf("auth: refreshed jwks (%d keys, valid for %s)", len(keys), validity)
	}

	return nil
}

func parseMaxAge(header string) time.Duration {
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(strings.ToLower(part), "max-age=") {
			value := strings.TrimSpace(part[len("max-age="):])
			if seconds, err := time.ParseDuration(value + "s"); err == nil {
				return seconds
			}
			if n, err := parsePositiveInt(value); err == nil {
				return time.Duration(n) * time.Second
			}
		}
	}
	return 0
}

func parsePositiveInt(value string) (int64, error) {
	var n int64
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid digit")
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}

// OIDCValidator validates Google-signed OIDC/IAP tokens using a JWKS cache.
type OIDCValidator struct {
	cache   *JWKSCache
	logger  Logger
	metrics MetricsRecorder
	now     func() time.Time
}

// OIDCOption customises the validator.
type OIDCOption func(*OIDCValidator)

// NewOIDCValidator constructs an OIDCValidator.
func NewOIDCValidator(cache *JWKSCache, opts ...OIDCOption) *OIDCValidator {
	validator := &OIDCValidator{
		cache:  cache,
		logger: log.Default(),
		now:    time.Now,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(validator)
		}
	}

	return validator
}

// WithOIDCLogger overrides the validator logger.
func WithOIDCLogger(logger Logger) OIDCOption {
	return func(v *OIDCValidator) {
		if logger != nil {
			v.logger = logger
		}
	}
}

// WithOIDCMetrics sets the metrics recorder.
func WithOIDCMetrics(recorder MetricsRecorder) OIDCOption {
	return func(v *OIDCValidator) {
		v.metrics = recorder
	}
}

// WithOIDCClock injects a custom clock (primarily for testing).
func WithOIDCClock(now func() time.Time) OIDCOption {
	return func(v *OIDCValidator) {
		if now != nil {
			v.now = now
		}
	}
}

// ServiceIdentity captures details about the authenticated service principal.
type ServiceIdentity struct {
	Subject  string
	Email    string
	Issuer   string
	Audience string

	Token  *jwt.Token
	Claims map[string]any
}

type serviceIdentityContextKey struct{}

// WithServiceIdentity attaches the verified service identity to the request context.
func WithServiceIdentity(ctx context.Context, identity *ServiceIdentity) context.Context {
	if identity == nil {
		return ctx
	}
	return context.WithValue(ctx, serviceIdentityContextKey{}, identity)
}

// ServiceIdentityFromContext retrieves the identity stored by the middleware.
func ServiceIdentityFromContext(ctx context.Context) (*ServiceIdentity, bool) {
	identity, ok := ctx.Value(serviceIdentityContextKey{}).(*ServiceIdentity)
	if !ok || identity == nil {
		return nil, false
	}
	return identity, true
}

// RequireOIDC enforces presence of a valid Google-signed OIDC/IAP token on the request.
func (v *OIDCValidator) RequireOIDC(audience string, issuers []string) func(http.Handler) http.Handler {
	expectedAudience := strings.TrimSpace(audience)
	allowedIssuers := make(map[string]struct{}, len(issuers))
	for _, issuer := range issuers {
		issuer = strings.TrimSpace(issuer)
		if issuer == "" {
			continue
		}
		allowedIssuers[issuer] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := v.now()
			ctx := r.Context()

			if expectedAudience == "" {
				v.record(ctx, false, "audience_not_configured", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "oidc audience not configured")
				return
			}

			tokenStr, source := extractOIDCToken(r)
			if tokenStr == "" {
				v.record(ctx, false, "token_missing", start)
				respondAuthError(w, http.StatusUnauthorized, "unauthenticated", "oidc token missing")
				return
			}

			if v == nil || v.cache == nil {
				v.record(ctx, false, "cache_unavailable", start)
				respondAuthError(w, http.StatusServiceUnavailable, "verification_unavailable", "oidc verification unavailable")
				return
			}

			parser := jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))

			claims := jwt.MapClaims{}
			parsed, err := parser.ParseWithClaims(tokenStr, claims, v.cache.Keyfunc(ctx))
			if err != nil {
				status := http.StatusUnauthorized
				reason := "token_invalid"
				if errors.Is(err, ErrJWKSFetchFailed) {
					status = http.StatusServiceUnavailable
					reason = "jwks_unavailable"
				}
				if v.logger != nil {
					v.logger.Printf("auth: oidc verification failed (%s): %v", reason, err)
				}
				v.record(ctx, false, reason, start)
				respondAuthError(w, status, "invalid_token", "oidc token verification failed")
				return
			}

			issuer, _ := claims["iss"].(string)
			if len(allowedIssuers) > 0 {
				if _, ok := allowedIssuers[issuer]; !ok {
					if v.logger != nil {
						v.logger.Printf("auth: oidc issuer mismatch, got %q", issuer)
					}
					v.record(ctx, false, "issuer_mismatch", start)
					respondAuthError(w, http.StatusUnauthorized, "invalid_token", "oidc issuer mismatch")
					return
				}
			}

			audiences := audienceFromClaims(claims)
			if !containsString(audiences, expectedAudience) {
				if v.logger != nil {
					v.logger.Printf("auth: oidc audience mismatch, expected %q (hdr=%s)", expectedAudience, source)
				}
				v.record(ctx, false, "audience_mismatch", start)
				respondAuthError(w, http.StatusUnauthorized, "invalid_token", "oidc audience mismatch")
				return
			}

			email, _ := claims["email"].(string)
			subject, _ := claims["sub"].(string)

			identity := &ServiceIdentity{
				Subject:  subject,
				Email:    email,
				Issuer:   issuer,
				Audience: expectedAudience,
				Token:    parsed,
				Claims:   cloneClaims(claims),
			}

			v.record(ctx, true, "ok", start)
			next.ServeHTTP(w, r.WithContext(WithServiceIdentity(ctx, identity)))
		})
	}
}

func (v *OIDCValidator) record(ctx context.Context, success bool, reason string, start time.Time) {
	if v == nil {
		return
	}
	if v.metrics == nil {
		return
	}
	duration := v.now().Sub(start)
	v.metrics.RecordVerification(ctx, "oidc", success, reason, duration)
}

func extractOIDCToken(r *http.Request) (token string, source string) {
	if r == nil {
		return "", ""
	}
	if authz := r.Header.Get("Authorization"); authz != "" {
		if bearer, ok := extractBearerToken(authz); ok {
			return bearer, "authorization"
		}
	}
	if assertion := strings.TrimSpace(r.Header.Get("X-Goog-Iap-Jwt-Assertion")); assertion != "" {
		return assertion, "iap"
	}
	return "", ""
}

func audienceFromClaims(claims jwt.MapClaims) []string {
	raw, ok := claims["aud"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case string:
		return []string{strings.TrimSpace(v)}
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				continue
			}
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			out = append(out, str)
		}
		return out
	default:
		return nil
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneClaims(claims jwt.MapClaims) map[string]any {
	out := make(map[string]any, len(claims))
	for key, value := range claims {
		out[key] = value
	}
	return out
}
