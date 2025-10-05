package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEnvFile               = ".env"
	defaultPort                  = "8080"
	defaultReadTimeout           = 15 * time.Second
	defaultWriteTimeout          = 30 * time.Second
	defaultIdleTimeout           = 120 * time.Second
	defaultRateLimitDefault      = 120
	defaultRateLimitAuth         = 240
	defaultRateLimitWebhookBurst = 60
	defaultSecurityEnvironment   = "local"
	defaultOIDCJWKSURL           = "https://www.googleapis.com/oauth2/v3/certs"
	defaultSecurityIssuer        = "https://accounts.google.com"
	defaultSecurityIAPIssuer     = "https://cloud.google.com/iap"
	defaultHMACSignatureHeader   = "X-Signature"
	defaultHMACTimestampHeader   = "X-Signature-Timestamp"
	defaultHMACNonceHeader       = "X-Signature-Nonce"
	defaultHMACClockSkew         = 5 * time.Minute
	defaultHMACNonceTTL          = 5 * time.Minute
)

// Config captures all runtime configuration organised by concern.
type Config struct {
	Server     ServerConfig
	Firebase   FirebaseConfig
	Firestore  FirestoreConfig
	Storage    StorageConfig
	PSP        PSPConfig
	AI         AIConfig
	Webhooks   WebhookConfig
	RateLimits RateLimitConfig
	Features   FeatureFlags
	Security   SecurityConfig
}

// ServerConfig configures HTTP server parameters.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// FirebaseConfig stores Firebase project settings.
type FirebaseConfig struct {
	ProjectID       string
	CredentialsFile string
}

// FirestoreConfig stores database parameters.
type FirestoreConfig struct {
	ProjectID    string
	EmulatorHost string
}

// StorageConfig lists bucket names used by the application.
type StorageConfig struct {
	AssetsBucket  string
	LogsBucket    string
	ExportsBucket string
}

// PSPConfig collects secrets for payment providers.
type PSPConfig struct {
	StripeAPIKey        string
	StripeWebhookSecret string
	PayPalClientID      string
	PayPalSecret        string
}

// AIConfig defines endpoints and credentials for AI workers.
type AIConfig struct {
	SuggestionEndpoint string
	AuthToken          string
}

// WebhookConfig contains webhook security parameters.
type WebhookConfig struct {
	SigningSecret string
	AllowedHosts  []string
}

// RateLimitConfig controls request throttling.
type RateLimitConfig struct {
	DefaultPerMinute       int
	AuthenticatedPerMinute int
	WebhookBurst           int
}

// FeatureFlags toggle optional behaviour without redeploying.
type FeatureFlags struct {
	EnableAISuggestions bool
	EnablePromotions    bool
}

// SecurityConfig groups server-to-server authentication settings.
type SecurityConfig struct {
	Environment string
	OIDC        OIDCConfig
	HMAC        HMACConfig
}

// OIDCConfig controls Google-signed token verification.
type OIDCConfig struct {
	JWKSURL   string
	Audience  string
	Audiences map[string]string
	Issuers   []string
}

// HMACConfig captures webhook signing expectations.
type HMACConfig struct {
	Secrets         map[string]string
	SignatureHeader string
	TimestampHeader string
	NonceHeader     string
	ClockSkew       time.Duration
	NonceTTL        time.Duration
}

// SecretResolver resolves references to external secrets (e.g. Secret Manager URIs).
type SecretResolver interface {
	ResolveSecret(ctx context.Context, ref string) (string, error)
}

// SecretResolverFunc adapts ordinary functions to SecretResolver.
type SecretResolverFunc func(context.Context, string) (string, error)

// ResolveSecret resolves the secret using the wrapped function.
func (f SecretResolverFunc) ResolveSecret(ctx context.Context, ref string) (string, error) {
	return f(ctx, ref)
}

// ValidationError is returned when required configuration fields are missing or invalid.
type ValidationError struct {
	fields []string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("config validation failed: missing or invalid fields [%s]", strings.Join(e.fields, ", "))
}

// Fields returns a copy of the missing/invalid field list.
func (e *ValidationError) Fields() []string {
	out := make([]string, len(e.fields))
	copy(out, e.fields)
	return out
}

// SecretError describes failures while resolving a secret reference.
type SecretError struct {
	Ref string
	Err error
}

// Error implements the error interface.
func (e *SecretError) Error() string {
	return fmt.Sprintf("secret resolution failed for ref %q: %v", e.Ref, e.Err)
}

// Unwrap exposes the underlying error.
func (e *SecretError) Unwrap() error { return e.Err }

var errSecretResolverNotConfigured = errors.New("secret resolver not configured")

// Option customises Load behaviour.
type Option func(*loaderOptions)

type loaderOptions struct {
	envFile      string
	envMap       map[string]string
	useSystemEnv bool
	secret       SecretResolver
}

// WithEnvFile overrides the .env file path used for local overrides.
func WithEnvFile(path string) Option {
	return func(o *loaderOptions) {
		o.envFile = path
	}
}

// WithEnvMap injects an explicit key/value map for environment lookups. Values in the map
// take precedence over system environment variables.
func WithEnvMap(values map[string]string) Option {
	return func(o *loaderOptions) {
		o.envMap = values
	}
}

// WithoutSystemEnv disables reading from os.Getenv, relying only on provided maps and .env files.
func WithoutSystemEnv() Option {
	return func(o *loaderOptions) {
		o.useSystemEnv = false
	}
}

// WithSecretResolver sets a custom secret resolver used for sm:// references.
func WithSecretResolver(resolver SecretResolver) Option {
	return func(o *loaderOptions) {
		o.secret = resolver
	}
}

// Load assembles the application configuration by combining defaults, .env overrides,
// environment variables, and optional secret manager lookups.
func Load(ctx context.Context, opts ...Option) (Config, error) {
	options := loaderOptions{
		envFile:      defaultEnvFile,
		useSystemEnv: true,
		secret: SecretResolverFunc(func(ctx context.Context, ref string) (string, error) {
			return "", &SecretError{Ref: ref, Err: errSecretResolverNotConfigured}
		}),
	}

	for _, opt := range opts {
		opt(&options)
	}

	dotEnvValues, err := loadDotEnv(options.envFile)
	if err != nil {
		return Config{}, err
	}

	lookup := func(key string) (string, bool) {
		if options.envMap != nil {
			if value, ok := options.envMap[key]; ok {
				return value, true
			}
		}
		if options.useSystemEnv {
			if value, ok := os.LookupEnv(key); ok {
				return value, true
			}
		}
		if dotEnvValues != nil {
			if value, ok := dotEnvValues[key]; ok {
				return value, true
			}
		}
		return "", false
	}

	cfg := Config{
		Server: ServerConfig{
			Port:         stringWithDefault(lookup, "API_SERVER_PORT", defaultPort),
			ReadTimeout:  durationWithDefault(lookup, "API_SERVER_READ_TIMEOUT", defaultReadTimeout),
			WriteTimeout: durationWithDefault(lookup, "API_SERVER_WRITE_TIMEOUT", defaultWriteTimeout),
			IdleTimeout:  durationWithDefault(lookup, "API_SERVER_IDLE_TIMEOUT", defaultIdleTimeout),
		},
		Firebase: FirebaseConfig{
			ProjectID:       stringWithDefault(lookup, "API_FIREBASE_PROJECT_ID", ""),
			CredentialsFile: stringWithDefault(lookup, "API_FIREBASE_CREDENTIALS_FILE", ""),
		},
		Firestore: FirestoreConfig{
			ProjectID:    stringWithDefault(lookup, "API_FIRESTORE_PROJECT_ID", ""),
			EmulatorHost: stringWithDefault(lookup, "API_FIRESTORE_EMULATOR_HOST", ""),
		},
		Storage: StorageConfig{
			AssetsBucket:  stringWithDefault(lookup, "API_STORAGE_ASSETS_BUCKET", ""),
			LogsBucket:    stringWithDefault(lookup, "API_STORAGE_LOGS_BUCKET", ""),
			ExportsBucket: stringWithDefault(lookup, "API_STORAGE_EXPORTS_BUCKET", ""),
		},
		PSP: PSPConfig{
			StripeAPIKey:        stringWithDefault(lookup, "API_PSP_STRIPE_API_KEY", ""),
			StripeWebhookSecret: stringWithDefault(lookup, "API_PSP_STRIPE_WEBHOOK_SECRET", ""),
			PayPalClientID:      stringWithDefault(lookup, "API_PSP_PAYPAL_CLIENT_ID", ""),
			PayPalSecret:        stringWithDefault(lookup, "API_PSP_PAYPAL_SECRET", ""),
		},
		AI: AIConfig{
			SuggestionEndpoint: stringWithDefault(lookup, "API_AI_SUGGESTION_ENDPOINT", ""),
			AuthToken:          stringWithDefault(lookup, "API_AI_AUTH_TOKEN", ""),
		},
		Webhooks: WebhookConfig{
			SigningSecret: stringWithDefault(lookup, "API_WEBHOOK_SIGNING_SECRET", ""),
			AllowedHosts:  csvWithDefault(lookup, "API_WEBHOOK_ALLOWED_HOSTS"),
		},
		RateLimits: RateLimitConfig{
			DefaultPerMinute:       intWithDefault(lookup, "API_RATELIMIT_DEFAULT_PER_MIN", defaultRateLimitDefault),
			AuthenticatedPerMinute: intWithDefault(lookup, "API_RATELIMIT_AUTH_PER_MIN", defaultRateLimitAuth),
			WebhookBurst:           intWithDefault(lookup, "API_RATELIMIT_WEBHOOK_BURST", defaultRateLimitWebhookBurst),
		},
		Features: FeatureFlags{
			EnableAISuggestions: boolWithDefault(lookup, "API_FEATURE_AISUGGESTIONS", false),
			EnablePromotions:    boolWithDefault(lookup, "API_FEATURE_PROMOTIONS", true),
		},
		Security: SecurityConfig{
			Environment: strings.ToLower(stringWithDefault(lookup, "API_SECURITY_ENVIRONMENT", defaultSecurityEnvironment)),
			OIDC: OIDCConfig{
				JWKSURL:   stringWithDefault(lookup, "API_SECURITY_OIDC_JWKS_URL", defaultOIDCJWKSURL),
				Audience:  stringWithDefault(lookup, "API_SECURITY_OIDC_AUDIENCE", ""),
				Audiences: mapWithDefault(lookup, "API_SECURITY_OIDC_AUDIENCES"),
				Issuers:   csvWithDefault(lookup, "API_SECURITY_OIDC_ISSUERS"),
			},
			HMAC: HMACConfig{
				Secrets:         mapWithDefault(lookup, "API_SECURITY_HMAC_SECRETS"),
				SignatureHeader: stringWithDefault(lookup, "API_SECURITY_HMAC_HEADER_SIGNATURE", defaultHMACSignatureHeader),
				TimestampHeader: stringWithDefault(lookup, "API_SECURITY_HMAC_HEADER_TIMESTAMP", defaultHMACTimestampHeader),
				NonceHeader:     stringWithDefault(lookup, "API_SECURITY_HMAC_HEADER_NONCE", defaultHMACNonceHeader),
				ClockSkew:       durationWithDefault(lookup, "API_SECURITY_HMAC_CLOCK_SKEW", defaultHMACClockSkew),
				NonceTTL:        durationWithDefault(lookup, "API_SECURITY_HMAC_NONCE_TTL", defaultHMACNonceTTL),
			},
		},
	}

	// Firestore project defaults to Firebase project when unspecified.
	if cfg.Firestore.ProjectID == "" {
		cfg.Firestore.ProjectID = cfg.Firebase.ProjectID
	}

	if len(cfg.Security.OIDC.Issuers) == 0 {
		cfg.Security.OIDC.Issuers = []string{defaultSecurityIssuer, defaultSecurityIAPIssuer}
	}

	envKey := strings.ToLower(cfg.Security.Environment)
	if cfg.Security.OIDC.Audience == "" && cfg.Security.OIDC.Audiences != nil {
		if audience, ok := cfg.Security.OIDC.Audiences[envKey]; ok {
			cfg.Security.OIDC.Audience = audience
		}
	}

	for key, value := range cfg.Security.HMAC.Secrets {
		resolved, err := resolveSecret(ctx, value, options.secret)
		if err != nil {
			return Config{}, err
		}
		cfg.Security.HMAC.Secrets[key] = resolved
	}

	// Resolve secrets when values reference Secret Manager.
	secretFields := []*string{
		&cfg.PSP.StripeAPIKey,
		&cfg.PSP.StripeWebhookSecret,
		&cfg.PSP.PayPalSecret,
		&cfg.AI.AuthToken,
		&cfg.Webhooks.SigningSecret,
	}
	for _, field := range secretFields {
		resolved, err := resolveSecret(ctx, *field, options.secret)
		if err != nil {
			return Config{}, err
		}
		*field = resolved
	}

	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func resolveSecret(ctx context.Context, value string, resolver SecretResolver) (string, error) {
	if value == "" {
		return value, nil
	}
	if !strings.HasPrefix(value, "sm://") {
		return value, nil
	}
	if resolver == nil {
		return "", &SecretError{Ref: value, Err: errSecretResolverNotConfigured}
	}
	secret, err := resolver.ResolveSecret(ctx, value)
	if err != nil {
		return "", &SecretError{Ref: value, Err: err}
	}
	return secret, nil
}

func validateConfig(cfg Config) error {
	var missing []string

	if cfg.Server.Port == "" {
		missing = append(missing, "Server.Port")
	}
	if cfg.Firebase.ProjectID == "" {
		missing = append(missing, "Firebase.ProjectID")
	}
	if cfg.Firestore.ProjectID == "" {
		missing = append(missing, "Firestore.ProjectID")
	}
	if cfg.Storage.AssetsBucket == "" {
		missing = append(missing, "Storage.AssetsBucket")
	}

	if len(missing) > 0 {
		return &ValidationError{fields: missing}
	}
	return nil
}

func loadDotEnv(path string) (map[string]string, error) {
	if path == "" {
		return nil, nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	file, err := os.Open(absPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("config: unable to read %s: %w", absPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	values := make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			continue
		}
		value = strings.Trim(value, "\"'")
		values[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("config: failed parsing %s: %w", absPath, err)
	}
	return values, nil
}

func stringWithDefault(lookup func(string) (string, bool), key, fallback string) string {
	if value, ok := lookup(key); ok && value != "" {
		return value
	}
	return fallback
}

func durationWithDefault(lookup func(string) (string, bool), key string, fallback time.Duration) time.Duration {
	if value, ok := lookup(key); ok && value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return fallback
}

func intWithDefault(lookup func(string) (string, bool), key string, fallback int) int {
	if value, ok := lookup(key); ok && value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func boolWithDefault(lookup func(string) (string, bool), key string, fallback bool) bool {
	if value, ok := lookup(key); ok && value != "" {
		switch strings.ToLower(value) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}

func csvWithDefault(lookup func(string) (string, bool), key string) []string {
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func mapWithDefault(lookup func(string) (string, bool), key string) map[string]string {
	values := make(map[string]string)
	raw, ok := lookup(key)
	if !ok || strings.TrimSpace(raw) == "" {
		return values
	}
	entries := strings.Split(raw, ",")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(parts[0]))
		secret := strings.TrimSpace(parts[1])
		if name == "" || secret == "" {
			continue
		}
		values[name] = secret
	}
	return values
}
