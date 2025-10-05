package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	jwt "github.com/golang-jwt/jwt/v4"
)

type noopLogger struct{}

func (noopLogger) Printf(string, ...any) {}

type recordingMetrics struct {
	mu      sync.Mutex
	records []verificationRecord
}

type verificationRecord struct {
	kind    string
	success bool
	reason  string
}

func (m *recordingMetrics) RecordVerification(_ context.Context, kind string, success bool, reason string, _ time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records = append(m.records, verificationRecord{kind: kind, success: success, reason: reason})
}

func TestJWKSCache_KeyCachesKeys(t *testing.T) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwk := jose.JSONWebKey{
		Key:       &key.PublicKey,
		KeyID:     "key1",
		Algorithm: jwt.SigningMethodRS256.Alg(),
		Use:       "sig",
	}

	var mu sync.Mutex
	var requests int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=3600")
		if err := json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}); err != nil {
			t.Fatalf("encode jwks: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	cache := NewJWKSCache(server.URL,
		WithJWKSLogger(noopLogger{}),
		WithJWKSClock(func() time.Time { return time.Unix(1_000_000, 0) }),
	)

	ctx := context.Background()
	got, err := cache.Key(ctx, "key1")
	if err != nil {
		t.Fatalf("cache.Key: %v", err)
	}

	if _, ok := got.(*rsa.PublicKey); !ok {
		t.Fatalf("expected *rsa.PublicKey, got %T", got)
	}

	if got, err = cache.Key(ctx, "key1"); err != nil {
		t.Fatalf("cache.Key second call: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if requests != 1 {
		t.Fatalf("expected single JWKS fetch, got %d", requests)
	}
}

func TestOIDCRequireOIDC_Success(t *testing.T) {
	validator, metrics, token := setupOIDCTest(t, func(claims jwt.MapClaims) {
		claims["aud"] = []string{"https://example.com"}
		claims["iss"] = "https://accounts.google.com"
	})

	middleware := validator.RequireOIDC("https://example.com", []string{"https://accounts.google.com"})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := ServiceIdentityFromContext(r.Context()); !ok {
			t.Fatalf("expected service identity in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if len(metrics.records) != 1 {
		t.Fatalf("expected 1 metric record, got %d", len(metrics.records))
	}
	record := metrics.records[0]
	if !record.success || record.reason != "ok" {
		t.Fatalf("unexpected metric record: %+v", record)
	}
}

func TestOIDCRequireOIDC_AudienceMismatch(t *testing.T) {
	validator, metrics, token := setupOIDCTest(t, func(claims jwt.MapClaims) {
		claims["aud"] = []string{"https://example.com"}
		claims["iss"] = "https://accounts.google.com"
	})

	middleware := validator.RequireOIDC("https://service.internal", []string{"https://accounts.google.com"})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.records[len(metrics.records)-1].reason != "audience_mismatch" {
		t.Fatalf("expected audience_mismatch metric, got %+v", metrics.records)
	}
}

func TestOIDCRequireOIDC_UsesIAPHeader(t *testing.T) {
	validator, _, token := setupOIDCTest(t, func(claims jwt.MapClaims) {
		claims["aud"] = []string{"/projects/123/global/backendServices/456"}
		claims["iss"] = "https://cloud.google.com/iap"
	})

	middleware := validator.RequireOIDC("/projects/123/global/backendServices/456", []string{"https://cloud.google.com/iap"})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("X-Goog-Iap-Jwt-Assertion", token)

	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}
}

func TestOIDCRequireOIDC_JWKSUnavailable(t *testing.T) {
	validator, metrics, token := setupOIDCTest(t, func(claims jwt.MapClaims) {
		claims["aud"] = []string{"https://example.com"}
		claims["iss"] = "https://accounts.google.com"
	})

	// Override cache URL to point to a server returning 500 responses.
	validator.cache.url = "http://127.0.0.1:65535/invalid"

	middleware := validator.RequireOIDC("https://example.com", []string{"https://accounts.google.com"})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/internal/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not be called")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if metrics.records[len(metrics.records)-1].reason != "jwks_unavailable" {
		t.Fatalf("expected jwks_unavailable metric, got %+v", metrics.records)
	}
}

func setupOIDCTest(t *testing.T, mutateClaims func(jwt.MapClaims)) (*OIDCValidator, *recordingMetrics, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	jwk := jose.JSONWebKey{
		Key:       &key.PublicKey,
		KeyID:     "svc-key",
		Algorithm: jwt.SigningMethodRS256.Alg(),
		Use:       "sig",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "max-age=600")
		if err := json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{jwk}}); err != nil {
			t.Fatalf("encode jwks: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	metrics := &recordingMetrics{}

	now := time.Unix(1_700_000_000, 0)
	originalTimeFunc := jwt.TimeFunc
	jwt.TimeFunc = func() time.Time { return now }
	t.Cleanup(func() { jwt.TimeFunc = originalTimeFunc })
	validator := NewOIDCValidator(NewJWKSCache(server.URL,
		WithJWKSLogger(noopLogger{}),
		WithJWKSClock(func() time.Time { return now }),
	),
		WithOIDCLogger(noopLogger{}),
		WithOIDCMetrics(metrics),
		WithOIDCClock(func() time.Time { return now }),
	)

	claims := jwt.MapClaims{
		"aud":   []string{"https://example.com"},
		"iss":   "https://accounts.google.com",
		"sub":   "service-account@example.com",
		"email": "svc@example.com",
		"exp":   float64(now.Add(time.Hour).Unix()),
		"iat":   float64(now.Unix()),
	}
	if mutateClaims != nil {
		mutateClaims(claims)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "svc-key"
	signed, err := token.SignedString(key)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	return validator, metrics, signed
}
