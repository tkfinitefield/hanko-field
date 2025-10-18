package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type mapSecretProvider map[string]string

func (m mapSecretProvider) GetSecret(_ context.Context, name string) (string, error) {
	if secret, ok := m[name]; ok {
		return secret, nil
	}
	return "", fmt.Errorf("secret %s not found", name)
}

func TestRequireHMAC_Success(t *testing.T) {
	t.Helper()

	const secretName = "webhooks/stripe"
	secretValue := "super-secret"

	provider := mapSecretProvider{secretName: secretValue}
	store := NewInMemoryNonceStore()

	metrics := &recordingMetrics{}
	now := time.Now().UTC().Truncate(time.Second)

	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
		WithHMACMetrics(metrics),
	)

	body := []byte(`{"event":"payment_intent.succeeded"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments/stripe", bytes.NewReader(body))

	timestamp := now.UTC().Format(time.RFC3339)
	nonce := "nonce-123"

	canonical := buildCanonicalString(req, body, timestamp, nonce)
	signature := computeHMAC([]byte(secretValue), canonical)

	req.Header.Set(defaultSignatureHeader, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set(defaultTimestampHeader, timestamp)
	req.Header.Set(defaultNonceHeader, nonce)

	rr := httptest.NewRecorder()

	middleware := validator.RequireHMAC(secretName)
	middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta, ok := HMACMetadataFromContext(r.Context())
		if !ok {
			t.Fatalf("expected hmac metadata in context")
		}
		if meta.SecretName != secretName {
			t.Fatalf("unexpected secret name %q", meta.SecretName)
		}
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d", rr.Code)
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()
	if len(metrics.records) != 1 || !metrics.records[0].success {
		t.Fatalf("expected success metric, got %+v", metrics.records)
	}
}

func TestRequireHMAC_ReplayRejected(t *testing.T) {
	const secretName = "webhooks/paypal"
	secretValue := "another-secret"

	provider := mapSecretProvider{secretName: secretValue}
	store := NewInMemoryNonceStore()

	metrics := &recordingMetrics{}
	now := time.Now().UTC().Truncate(time.Second)

	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
		WithHMACMetrics(metrics),
	)

	body := []byte(`{"status":"completed"}`)
	timestamp := now.UTC().Format(time.RFC3339)
	nonce := "nonce-replay"

	makeRequest := func() *http.Request {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/payments/paypal", bytes.NewReader(body))
		signature := computeHMAC([]byte(secretValue), buildCanonicalString(req, body, timestamp, nonce))
		req.Header.Set(defaultSignatureHeader, base64.StdEncoding.EncodeToString(signature))
		req.Header.Set(defaultTimestampHeader, timestamp)
		req.Header.Set(defaultNonceHeader, nonce)
		return req
	}

	handler := validator.RequireHMAC(secretName)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest())
	if rr.Code != http.StatusOK {
		t.Fatalf("expected first request to succeed, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, makeRequest())
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected replay to be rejected with 401, got %d", rr.Code)
	}
}

func TestRequireHMAC_SignatureMismatch(t *testing.T) {
	const secretName = "webhooks/shipping"
	secretValue := "shipping-secret"

	provider := mapSecretProvider{secretName: secretValue}
	store := NewInMemoryNonceStore()
	now := time.Now().UTC().Truncate(time.Second)

	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
	)

	originalBody := []byte(`{"shipment":"in_transit"}`)
	timestamp := now.UTC().Format(time.RFC3339)
	nonce := "nonce-ship"

	req := httptest.NewRequest(http.MethodPost, "/webhooks/shipping/jp-post", bytes.NewReader([]byte(`{"shipment":"delivered"}`)))

	signature := computeHMAC([]byte(secretValue), buildCanonicalString(httptest.NewRequest(http.MethodPost, "/webhooks/shipping/jp-post", bytes.NewReader(originalBody)), originalBody, timestamp, nonce))

	req.Header.Set(defaultSignatureHeader, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set(defaultTimestampHeader, timestamp)
	req.Header.Set(defaultNonceHeader, nonce)

	rr := httptest.NewRecorder()
	validator.RequireHMAC(secretName)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not be invoked on signature mismatch")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on signature mismatch, got %d", rr.Code)
	}
}

func TestRequireHMAC_TimestampSkewRejected(t *testing.T) {
	const secretName = "webhooks/ai"
	secretValue := "ai-secret"

	provider := mapSecretProvider{secretName: secretValue}
	store := NewInMemoryNonceStore()

	now := time.Now().UTC().Truncate(time.Second)
	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
	)

	body := []byte(`{"job":"complete"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/ai/worker", bytes.NewReader(body))

	timestamp := now.Add(-10 * time.Minute).UTC().Format(time.RFC3339)
	nonce := "nonce-old"
	signature := computeHMAC([]byte(secretValue), buildCanonicalString(req, body, timestamp, nonce))

	req.Header.Set(defaultSignatureHeader, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set(defaultTimestampHeader, timestamp)
	req.Header.Set(defaultNonceHeader, nonce)

	rr := httptest.NewRecorder()
	validator.RequireHMAC(secretName)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not be called when timestamp is skewed")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 on timestamp skew, got %d", rr.Code)
	}
}

func TestRequireHMAC_SecretUnavailable(t *testing.T) {
	provider := SecretProviderFunc(func(context.Context, string) (string, error) {
		return "", fmt.Errorf("secret unavailable")
	})
	store := NewInMemoryNonceStore()
	now := time.Now().UTC().Truncate(time.Second)

	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/test", bytes.NewReader(nil))
	rr := httptest.NewRecorder()

	validator.RequireHMAC("missing/secret")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not run when secret unavailable")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when secret unavailable, got %d", rr.Code)
	}
}

func TestRequireHMACResolver(t *testing.T) {
	const secretName = "payments/stripe"
	secretValue := "resolver-secret"

	provider := mapSecretProvider{secretName: secretValue}
	store := NewInMemoryNonceStore()
	now := time.Now().UTC().Truncate(time.Second)

	validator := NewHMACValidator(provider, store,
		WithHMACLogger(noopLogger{}),
		WithHMACClock(func() time.Time { return now }),
	)

	body := []byte(`{"event":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/payments/stripe", bytes.NewReader(body))
	timestamp := now.Format(time.RFC3339)
	nonce := "resolver-nonce"
	signature := computeHMAC([]byte(secretValue), buildCanonicalString(req, body, timestamp, nonce))
	req.Header.Set(defaultSignatureHeader, base64.StdEncoding.EncodeToString(signature))
	req.Header.Set(defaultTimestampHeader, timestamp)
	req.Header.Set(defaultNonceHeader, nonce)

	resolver := func(r *http.Request) (string, bool) {
		return secretName, true
	}

	rr := httptest.NewRecorder()
	validator.RequireHMACResolver(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from resolver middleware, got %d", rr.Code)
	}

	// Unknown provider should fail fast.
	unknown := httptest.NewRecorder()
	validator.RequireHMACResolver(func(*http.Request) (string, bool) {
		return "", false
	})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatalf("handler should not run for unknown provider")
	})).ServeHTTP(unknown, httptest.NewRequest(http.MethodPost, "/webhooks/unknown", nil))

	if unknown.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unknown provider, got %d", unknown.Code)
	}
}
