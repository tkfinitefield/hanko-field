package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadWithDefaults(t *testing.T) {
	env := map[string]string{
		"API_FIREBASE_PROJECT_ID":   "hf-dev",
		"API_STORAGE_ASSETS_BUCKET": "hanko-assets-dev",
	}

	cfg, err := Load(context.Background(), WithEnvMap(env), WithoutSystemEnv(), WithEnvFile(""))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("unexpected read timeout: %s", cfg.Server.ReadTimeout)
	}
	if cfg.Firestore.ProjectID != "hf-dev" {
		t.Errorf("expected firestore project to default to firebase project, got %s", cfg.Firestore.ProjectID)
	}
	if cfg.RateLimits.DefaultPerMinute != 120 {
		t.Errorf("unexpected default rate limit: %d", cfg.RateLimits.DefaultPerMinute)
	}
	if len(cfg.Webhooks.AllowedHosts) != 0 {
		t.Errorf("expected no allowed hosts, got %v", cfg.Webhooks.AllowedHosts)
	}
	if cfg.Security.Environment != "local" {
		t.Errorf("expected default security environment local, got %s", cfg.Security.Environment)
	}
	if cfg.Security.OIDC.JWKSURL != defaultOIDCJWKSURL {
		t.Errorf("expected default jwks url %s, got %s", defaultOIDCJWKSURL, cfg.Security.OIDC.JWKSURL)
	}
	if len(cfg.Security.OIDC.Issuers) != 2 {
		t.Errorf("expected default issuers, got %v", cfg.Security.OIDC.Issuers)
	}
	if cfg.Security.HMAC.SignatureHeader != defaultHMACSignatureHeader {
		t.Errorf("expected default signature header, got %s", cfg.Security.HMAC.SignatureHeader)
	}
	if cfg.Idempotency.Header != defaultIdempotencyHeader {
		t.Errorf("expected default idempotency header, got %s", cfg.Idempotency.Header)
	}
	if cfg.Idempotency.TTL != defaultIdempotencyTTL {
		t.Errorf("unexpected default idempotency ttl: %s", cfg.Idempotency.TTL)
	}
	if cfg.Idempotency.CleanupInterval != defaultIdempotencyInterval {
		t.Errorf("unexpected default cleanup interval: %s", cfg.Idempotency.CleanupInterval)
	}
	if cfg.Idempotency.CleanupBatchSize != defaultIdempotencyBatchSize {
		t.Errorf("unexpected default cleanup batch size: %d", cfg.Idempotency.CleanupBatchSize)
	}
}

func TestLoadWithOverridesAndSecrets(t *testing.T) {
	env := map[string]string{
		"API_SERVER_PORT":                    "9090",
		"API_SERVER_READ_TIMEOUT":            "20s",
		"API_SERVER_WRITE_TIMEOUT":           "25s",
		"API_SERVER_IDLE_TIMEOUT":            "2m",
		"API_FIREBASE_PROJECT_ID":            "hf-prod",
		"API_FIRESTORE_PROJECT_ID":           "hf-fire",
		"API_STORAGE_ASSETS_BUCKET":          "assets-prod",
		"API_STORAGE_LOGS_BUCKET":            "logs-prod",
		"API_STORAGE_EXPORTS_BUCKET":         "exports-prod",
		"API_PSP_STRIPE_API_KEY":             "secret://stripe/api",
		"API_PSP_STRIPE_WEBHOOK_SECRET":      "secret://stripe/webhook",
		"API_PSP_PAYPAL_CLIENT_ID":           "paypal-client",
		"API_PSP_PAYPAL_SECRET":              "secret://paypal/secret",
		"API_AI_SUGGESTION_ENDPOINT":         "https://ai.example.com",
		"API_AI_AUTH_TOKEN":                  "secret://ai/token",
		"API_WEBHOOK_SIGNING_SECRET":         "secret://webhook/secret",
		"API_WEBHOOK_ALLOWED_HOSTS":          "https://example.com, https://foo.bar",
		"API_RATELIMIT_DEFAULT_PER_MIN":      "150",
		"API_RATELIMIT_AUTH_PER_MIN":         "300",
		"API_RATELIMIT_WEBHOOK_BURST":        "80",
		"API_FEATURE_AISUGGESTIONS":          "true",
		"API_FEATURE_PROMOTIONS":             "false",
		"API_SECURITY_ENVIRONMENT":           "prod",
		"API_SECURITY_OIDC_AUDIENCE":         "https://service.example.com",
		"API_SECURITY_OIDC_ISSUERS":          "https://accounts.google.com, https://cloud.google.com/iap",
		"API_SECURITY_OIDC_JWKS_URL":         "https://example.com/jwks.json",
		"API_SECURITY_HMAC_SECRETS":          "payments/stripe=secret://hmac/stripe,shipping=shipping-secret",
		"API_SECURITY_HMAC_HEADER_SIGNATURE": "X-Custom-Signature",
		"API_SECURITY_HMAC_CLOCK_SKEW":       "3m",
		"API_SECURITY_HMAC_NONCE_TTL":        "10m",
		"API_IDEMPOTENCY_HEADER":             "X-Idem-Key",
		"API_IDEMPOTENCY_TTL":                "48h",
		"API_IDEMPOTENCY_CLEANUP_INTERVAL":   "30m",
		"API_IDEMPOTENCY_CLEANUP_BATCH":      "500",
	}

	secrets := map[string]string{
		"secret://stripe/api":     "stripe-key",
		"secret://stripe/webhook": "stripe-webhook",
		"secret://paypal/secret":  "paypal-secret",
		"secret://ai/token":       "ai-token",
		"secret://webhook/secret": "webhook-secret",
		"secret://hmac/stripe":    "stripe-hmac",
	}

	resolver := SecretResolverFunc(func(_ context.Context, ref string) (string, error) {
		if v, ok := secrets[ref]; ok {
			return v, nil
		}
		return "", &SecretError{Ref: ref, Err: errSecretResolverNotConfigured}
	})

	cfg, err := Load(context.Background(), WithEnvMap(env), WithoutSystemEnv(), WithEnvFile(""), WithSecretResolver(resolver))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Server.Port)
	}
	if cfg.Server.IdleTimeout != 2*time.Minute {
		t.Errorf("unexpected idle timeout: %s", cfg.Server.IdleTimeout)
	}
	if cfg.PSP.StripeAPIKey != "stripe-key" {
		t.Errorf("expected resolved stripe api key, got %s", cfg.PSP.StripeAPIKey)
	}
	if cfg.PSP.PayPalSecret != "paypal-secret" {
		t.Errorf("expected resolved paypal secret, got %s", cfg.PSP.PayPalSecret)
	}
	if len(cfg.Webhooks.AllowedHosts) != 2 {
		t.Fatalf("expected 2 allowed hosts, got %v", cfg.Webhooks.AllowedHosts)
	}
	if !cfg.Features.EnableAISuggestions {
		t.Errorf("expected AISuggestions flag enabled")
	}
	if cfg.Features.EnablePromotions {
		t.Errorf("expected promotions flag disabled")
	}
	if cfg.Security.Environment != "prod" {
		t.Errorf("expected security environment prod, got %s", cfg.Security.Environment)
	}
	if cfg.Security.OIDC.Audience != "https://service.example.com" {
		t.Errorf("unexpected oidc audience %s", cfg.Security.OIDC.Audience)
	}
	if cfg.Security.OIDC.JWKSURL != "https://example.com/jwks.json" {
		t.Errorf("unexpected jwks url %s", cfg.Security.OIDC.JWKSURL)
	}
	if cfg.Security.HMAC.Secrets["payments/stripe"] != "stripe-hmac" {
		t.Errorf("expected resolved stripe hmac secret, got %s", cfg.Security.HMAC.Secrets["payments/stripe"])
	}
	if cfg.Security.HMAC.Secrets["shipping"] != "shipping-secret" {
		t.Errorf("expected shipping secret fallback, got %s", cfg.Security.HMAC.Secrets["shipping"])
	}
	if cfg.Security.HMAC.SignatureHeader != "X-Custom-Signature" {
		t.Errorf("unexpected signature header %s", cfg.Security.HMAC.SignatureHeader)
	}
	if cfg.Security.HMAC.ClockSkew != 3*time.Minute {
		t.Errorf("unexpected clock skew %s", cfg.Security.HMAC.ClockSkew)
	}
	if cfg.Security.HMAC.NonceTTL != 10*time.Minute {
		t.Errorf("unexpected nonce ttl %s", cfg.Security.HMAC.NonceTTL)
	}
	if cfg.Idempotency.Header != "X-Idem-Key" {
		t.Errorf("unexpected idempotency header %s", cfg.Idempotency.Header)
	}
	if cfg.Idempotency.TTL != 48*time.Hour {
		t.Errorf("unexpected idempotency ttl %s", cfg.Idempotency.TTL)
	}
	if cfg.Idempotency.CleanupInterval != 30*time.Minute {
		t.Errorf("unexpected cleanup interval %s", cfg.Idempotency.CleanupInterval)
	}
	if cfg.Idempotency.CleanupBatchSize != 500 {
		t.Errorf("unexpected cleanup batch size %d", cfg.Idempotency.CleanupBatchSize)
	}
}

func TestLoadDotEnvFallback(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.test")
	content := "API_SERVER_PORT=7070\nAPI_FIREBASE_PROJECT_ID=hf-dot\nAPI_STORAGE_ASSETS_BUCKET=assets-dot\n"
	if err := os.WriteFile(envPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write dotenv file: %v", err)
	}

	cfg, err := Load(context.Background(), WithEnvFile(envPath), WithoutSystemEnv())
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "7070" {
		t.Errorf("expected port from dotenv 7070, got %s", cfg.Server.Port)
	}
	if cfg.Firebase.ProjectID != "hf-dot" {
		t.Errorf("expected firebase project from dotenv, got %s", cfg.Firebase.ProjectID)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	_, err := Load(context.Background(), WithEnvMap(map[string]string{}), WithoutSystemEnv(), WithEnvFile(""))
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestLoadSecretResolverError(t *testing.T) {
	env := map[string]string{
		"API_FIREBASE_PROJECT_ID":   "hf-dev",
		"API_STORAGE_ASSETS_BUCKET": "assets",
		"API_PSP_STRIPE_API_KEY":    "secret://missing",
	}

	_, err := Load(context.Background(), WithEnvMap(env), WithoutSystemEnv(), WithEnvFile(""))
	if err == nil {
		t.Fatal("expected secret resolution error, got nil")
	}
	var secretErr *SecretError
	if !errors.As(err, &secretErr) {
		t.Fatalf("expected SecretError, got %T", err)
	}
	if secretErr.Ref != "secret://missing" {
		t.Errorf("unexpected secret ref %s", secretErr.Ref)
	}
}

func TestEnvironmentValuesMergesSources(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env.test")
	content := "API_FIREBASE_PROJECT_ID=dot-project\nAPI_SECRET_FALLBACK_FILE=.dot.local\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatalf("failed writing env file: %v", err)
	}

	t.Setenv("API_FIREBASE_PROJECT_ID", "os-project")
	t.Setenv("API_SECRET_PROJECT_IDS", "prod=project-prod")

	overrides := map[string]string{
		"API_FIREBASE_PROJECT_ID": "override-project",
		"API_SECRET_VERSION_PINS": "secret://stripe/api=5",
	}

	values, err := EnvironmentValues(WithEnvFile(envPath), WithEnvMap(overrides))
	if err != nil {
		t.Fatalf("EnvironmentValues returned error: %v", err)
	}

	if got := values["API_FIREBASE_PROJECT_ID"]; got != "override-project" {
		t.Fatalf("expected override project, got %s", got)
	}
	if got := values["API_SECRET_FALLBACK_FILE"]; got != ".dot.local" {
		t.Fatalf("expected dotenv fallback file, got %s", got)
	}
	if got := values["API_SECRET_PROJECT_IDS"]; got != "prod=project-prod" {
		t.Fatalf("expected system env project map, got %s", got)
	}
	if got := values["API_SECRET_VERSION_PINS"]; got != "secret://stripe/api=5" {
		t.Fatalf("expected override version pin, got %s", got)
	}
}

func TestLoadMissingRequiredSecrets(t *testing.T) {
	env := map[string]string{
		"API_FIREBASE_PROJECT_ID":   "hf-dev",
		"API_STORAGE_ASSETS_BUCKET": "assets",
	}

	_, err := Load(context.Background(),
		WithEnvMap(env),
		WithoutSystemEnv(),
		WithEnvFile(""),
		WithRequiredSecrets("Webhooks.SigningSecret"),
	)
	if err == nil {
		t.Fatal("expected missing secrets error, got nil")
	}
	var missing *MissingSecretsError
	if !errors.As(err, &missing) {
		t.Fatalf("expected MissingSecretsError, got %T", err)
	}
	expectedRedacted := redactSecretName("Webhooks.SigningSecret")
	if got := missing.RedactedNames(); len(got) != 1 || got[0] != expectedRedacted {
		t.Fatalf("unexpected redacted names %v", got)
	}
}

func TestLoadMissingRequiredSecretsPanic(t *testing.T) {
	env := map[string]string{
		"API_FIREBASE_PROJECT_ID":   "hf-dev",
		"API_STORAGE_ASSETS_BUCKET": "assets",
	}

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic when required secrets missing")
		}
		missing, ok := rec.(*MissingSecretsError)
		if !ok {
			t.Fatalf("expected MissingSecretsError panic, got %T", rec)
		}
		if len(missing.Names()) != 1 || missing.Names()[0] != "Webhooks.SigningSecret" {
			t.Fatalf("unexpected missing secrets %v", missing.Names())
		}
	}()

	Load(context.Background(),
		WithEnvMap(env),
		WithoutSystemEnv(),
		WithEnvFile(""),
		WithRequiredSecrets("Webhooks.SigningSecret"),
		WithPanicOnMissingSecrets(),
	)
}

func TestLoadSupportsLegacySecretScheme(t *testing.T) {
	env := map[string]string{
		"API_FIREBASE_PROJECT_ID":    "hf-dev",
		"API_STORAGE_ASSETS_BUCKET":  "assets",
		"API_WEBHOOK_SIGNING_SECRET": "sm://webhook/secret",
	}

	secrets := map[string]string{
		"secret://webhook/secret": "legacy-secret",
	}

	resolver := SecretResolverFunc(func(_ context.Context, ref string) (string, error) {
		if v, ok := secrets[ref]; ok {
			return v, nil
		}
		return "", &SecretError{Ref: ref, Err: errors.New("not found")}
	})

	cfg, err := Load(context.Background(),
		WithEnvMap(env),
		WithoutSystemEnv(),
		WithEnvFile(""),
		WithSecretResolver(resolver),
	)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Webhooks.SigningSecret != "legacy-secret" {
		t.Fatalf("expected legacy secret, got %s", cfg.Webhooks.SigningSecret)
	}
}
