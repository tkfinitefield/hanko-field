package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hanko-field/api/internal/handlers"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/config"
	"github.com/hanko-field/api/internal/platform/idempotency"
	"github.com/hanko-field/api/internal/platform/observability"
	"github.com/hanko-field/api/internal/platform/secrets"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

func main() {
	ctx := context.Background()
	startedAt := time.Now().UTC()

	baseLogger, err := observability.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = baseLogger.Sync()
	}()

	logger := baseLogger.Named("api")
	ctx = observability.WithLogger(ctx, logger)

	envValues, err := config.EnvironmentValues()
	if err != nil {
		logger.Fatal("failed to read environment values", zap.Error(err))
	}

	fetcher, err := newSecretFetcher(ctx, logger, envValues)
	if err != nil {
		logger.Fatal("failed to initialise secret fetcher", zap.Error(err))
	}
	defer func() {
		if err := fetcher.Close(); err != nil {
			logger.Warn("secret fetcher close error", zap.Error(err))
		}
	}()

	requiredSecrets := requiredSecretNames(envValues)
	cfg, err := config.Load(ctx,
		config.WithSecretResolver(config.SecretResolverFunc(fetcher.Resolve)),
		config.WithRequiredSecrets(requiredSecrets...),
	)
	if err != nil {
		var missing *config.MissingSecretsError
		if errors.As(err, &missing) {
			logger.Fatal("missing required secrets", zap.Strings("secrets", missing.RedactedNames()))
		}
		logger.Fatal("failed to load configuration", zap.Error(err))
	}

	buildInfo := buildInfoFromEnv(envValues, cfg, startedAt)

	firestoreClient, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to initialise firestore client", zap.Error(err))
	}
	defer func() {
		if err := firestoreClient.Close(); err != nil {
			logger.Warn("firestore close error", zap.Error(err))
		}
	}()

	systemService, err := newSystemService(ctx, firestoreClient, fetcher, buildInfo)
	if err != nil {
		logger.Warn("health: system service init failed", zap.Error(err))
	}

	idempotencyStore := idempotency.NewFirestoreStore(firestoreClient)
	idempotencyMiddleware := idempotency.Middleware(
		idempotencyStore,
		idempotency.WithHeader(cfg.Idempotency.Header),
		idempotency.WithTTL(cfg.Idempotency.TTL),
		idempotency.WithLogger(observability.NewPrintfAdapter(logger.Named("idempotency"))),
	)

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	var cleanupWG sync.WaitGroup
	var cleanupTicker *time.Ticker
	if cfg.Idempotency.CleanupInterval > 0 {
		cleanupTicker = time.NewTicker(cfg.Idempotency.CleanupInterval)
		cleanupWG.Add(1)
		go func() {
			defer cleanupWG.Done()
			cleanupLogger := logger.Named("idempotency")
			for {
				select {
				case <-cleanupTicker.C:
					runCtx, cancel := context.WithTimeout(cleanupCtx, time.Minute)
					removed, err := idempotencyStore.CleanupExpired(runCtx, time.Now().UTC(), cfg.Idempotency.CleanupBatchSize)
					cancel()
					if err != nil {
						cleanupLogger.Error("idempotency cleanup error", zap.Error(err))
						continue
					}
					if removed > 0 {
						cleanupLogger.Info("idempotency cleanup removed records", zap.Int("count", removed))
					}
				case <-cleanupCtx.Done():
					return
				}
			}
		}()
	}

	oidcMiddleware := buildOIDCMiddleware(logger.Named("auth"), cfg)
	hmacMiddleware := buildHMACMiddleware(logger.Named("auth"), cfg)

	projectID := traceProjectID(cfg)
	middlewares := []func(http.Handler) http.Handler{
		observability.InjectLoggerMiddleware(logger.Named("http")),
		observability.TraceMiddleware(projectID),
		observability.RecoveryMiddleware(logger.Named("http")),
		observability.RequestLoggerMiddleware(projectID),
		idempotencyMiddleware,
	}

	healthHandlers := handlers.NewHealthHandlers(
		handlers.WithHealthBuildInfo(buildInfo),
		handlers.WithHealthSystemService(systemService),
	)

	var opts []handlers.Option
	opts = append(opts, handlers.WithMiddlewares(middlewares...))
	opts = append(opts, handlers.WithHealthHandlers(healthHandlers))
	publicHandlers := handlers.NewPublicHandlers()
	opts = append(opts, handlers.WithPublicRoutes(publicHandlers.Routes))
	if oidcMiddleware != nil {
		opts = append(opts, handlers.WithInternalMiddlewares(oidcMiddleware))
	}
	if hmacMiddleware != nil {
		opts = append(opts, handlers.WithWebhookMiddlewares(hmacMiddleware))
	}

	router := handlers.NewRouter(opts...)
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	serverLogger := logger.Named("http").With(zap.String("addr", server.Addr))
	go func() {
		serverLogger.Info("hanko-field api listening")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverLogger.Fatal("http server error", zap.Error(err))
		}
	}()

	<-shutdown
	logger.Info("shutdown signal received; draining requests")

	if cleanupTicker != nil {
		cleanupTicker.Stop()
	}
	cleanupCancel()
	cleanupWG.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	}
}

func buildInfoFromEnv(env map[string]string, cfg config.Config, started time.Time) services.BuildInfo {
	version := strings.TrimSpace(env["API_BUILD_VERSION"])
	if version == "" {
		version = "dev"
	}
	commit := strings.TrimSpace(env["API_BUILD_COMMIT_SHA"])
	if commit == "" {
		commit = "unknown"
	}
	environment := strings.TrimSpace(cfg.Security.Environment)
	if environment == "" {
		environment = "local"
	}
	return services.BuildInfo{
		Version:     version,
		CommitSHA:   commit,
		Environment: environment,
		StartedAt:   started,
	}
}

func newSystemService(ctx context.Context, client *firestore.Client, fetcher *secrets.Fetcher, build services.BuildInfo) (services.SystemService, error) {
	checks := make([]repositories.DependencyCheck, 0, 4)
	if client != nil {
		c := client
		checks = append(checks, repositories.DependencyCheck{
			Name:    "firestore",
			Timeout: 1500 * time.Millisecond,
			Check: func(ctx context.Context) error {
				iter := c.Collections(ctx)
				_, err := iter.Next()
				if errors.Is(err, iterator.Done) {
					return nil
				}
				return err
			},
		})
	}
	if fetcher != nil {
		const secretHealthReference = "secret://system/healthz?version=latest"
		checks = append(checks, repositories.DependencyCheck{
			Name:    "secretManager",
			Timeout: time.Second,
			Check: func(ctx context.Context) error {
				_, err := fetcher.Resolve(ctx, secretHealthReference)
				if err == nil {
					return nil
				}
				if st, ok := status.FromError(err); ok {
					switch st.Code() {
					case codes.NotFound:
						return nil
					}
				}
				return err
			},
		})
	}
	if len(checks) == 0 {
		return nil, errors.New("health: no dependency checks configured")
	}
	repo, err := repositories.NewDependencyHealthRepository(checks)
	if err != nil {
		return nil, err
	}
	return services.NewSystemService(services.SystemServiceDeps{
		HealthRepository: repo,
		Clock:            time.Now,
		Build:            build,
	})
}

func buildOIDCMiddleware(logger *zap.Logger, cfg config.Config) func(http.Handler) http.Handler {
	if strings.TrimSpace(cfg.Security.OIDC.JWKSURL) == "" {
		return nil
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	adapter := observability.NewPrintfAdapter(logger)
	cache := auth.NewJWKSCache(cfg.Security.OIDC.JWKSURL, auth.WithJWKSLogger(adapter))
	validator := auth.NewOIDCValidator(cache, auth.WithOIDCLogger(adapter))

	audience := strings.TrimSpace(cfg.Security.OIDC.Audience)
	if audience == "" {
		logger.Warn("auth: OIDC audience not configured; internal routes will reject requests")
	}
	issuers := cfg.Security.OIDC.Issuers
	if len(issuers) == 0 {
		logger.Warn("auth: OIDC issuers not configured; internal routes will reject requests")
	}

	return validator.RequireOIDC(audience, issuers)
}

func buildHMACMiddleware(logger *zap.Logger, cfg config.Config) func(http.Handler) http.Handler {
	secrets := make(map[string]string)
	for key, value := range cfg.Security.HMAC.Secrets {
		if strings.TrimSpace(value) == "" {
			continue
		}
		secrets[strings.ToLower(key)] = value
	}
	if cfg.Webhooks.SigningSecret != "" {
		if _, ok := secrets["default"]; !ok {
			secrets["default"] = cfg.Webhooks.SigningSecret
		}
	}
	if len(secrets) == 0 {
		return nil
	}

	provider := staticSecretProvider{secrets: secrets}
	nonces := auth.NewInMemoryNonceStore()
	adapter := observability.NewPrintfAdapter(logger)
	validator := auth.NewHMACValidator(provider, nonces,
		auth.WithHMACLogger(adapter),
		auth.WithHMACHeaders(cfg.Security.HMAC.SignatureHeader, cfg.Security.HMAC.TimestampHeader, cfg.Security.HMAC.NonceHeader),
		auth.WithHMACClockSkew(cfg.Security.HMAC.ClockSkew),
		auth.WithHMACNonceTTL(cfg.Security.HMAC.NonceTTL),
	)

	resolver := webhookSecretResolver(secrets)
	return validator.RequireHMACResolver(resolver)
}

type staticSecretProvider struct {
	secrets map[string]string
}

func (p staticSecretProvider) GetSecret(_ context.Context, name string) (string, error) {
	if len(p.secrets) == 0 {
		return "", errors.New("auth: hmac secrets not configured")
	}
	key := strings.ToLower(strings.TrimSpace(name))
	if key == "" {
		return "", errors.New("auth: secret name required")
	}
	if secret, ok := p.secrets[key]; ok && secret != "" {
		return secret, nil
	}
	return "", errors.New("auth: secret not found")
}

func webhookSecretResolver(secrets map[string]string) func(*http.Request) (string, bool) {
	return func(r *http.Request) (string, bool) {
		path := r.URL.Path
		if idx := strings.Index(path, "/webhooks/"); idx >= 0 {
			path = path[idx+len("/webhooks/"):]
		}
		path = strings.Trim(path, "/")
		if path == "" {
			if secret, ok := secrets["default"]; ok && secret != "" {
				return "default", true
			}
			return "", false
		}

		segments := strings.Split(path, "/")
		candidates := make([]string, 0, 3)
		if len(segments) >= 2 {
			candidates = append(candidates, strings.ToLower(strings.Join(segments[:2], "/")))
		}
		if len(segments) >= 1 {
			candidates = append(candidates, strings.ToLower(segments[0]))
		}
		candidates = append(candidates, "default")

		for _, candidate := range candidates {
			if secret, ok := secrets[candidate]; ok && secret != "" {
				return candidate, true
			}
		}
		return "", false
	}
}

func newFirestoreClient(ctx context.Context, cfg config.Config) (*firestore.Client, error) {
	projectID := strings.TrimSpace(cfg.Firestore.ProjectID)
	if projectID == "" {
		return nil, fmt.Errorf("firestore project id not configured")
	}

	if host := strings.TrimSpace(cfg.Firestore.EmulatorHost); host != "" {
		if err := os.Setenv("FIRESTORE_EMULATOR_HOST", host); err != nil {
			return nil, fmt.Errorf("failed to set FIRESTORE_EMULATOR_HOST: %w", err)
		}
	}

	var opts []option.ClientOption
	if credentials := strings.TrimSpace(cfg.Firebase.CredentialsFile); credentials != "" {
		opts = append(opts, option.WithCredentialsFile(credentials))
	}

	return firestore.NewClient(ctx, projectID, opts...)
}

func traceProjectID(cfg config.Config) string {
	if id := strings.TrimSpace(cfg.Firebase.ProjectID); id != "" {
		return id
	}
	return strings.TrimSpace(cfg.Firestore.ProjectID)
}

func newSecretFetcher(ctx context.Context, logger *zap.Logger, env map[string]string) (*secrets.Fetcher, error) {
	lookup := func(key string) string {
		if env == nil {
			return ""
		}
		if value, ok := env[key]; ok {
			return strings.TrimSpace(value)
		}
		return ""
	}

	envLabel := strings.ToLower(lookup("API_SECURITY_ENVIRONMENT"))
	if envLabel == "" {
		envLabel = "local"
	}
	projectMap := secretProjectMapFromEnv(env)
	defaultProject := lookup("API_SECRET_DEFAULT_PROJECT_ID")
	if defaultProject == "" {
		defaultProject = lookup("API_FIREBASE_PROJECT_ID")
	}
	fallbackPath := lookup("API_SECRET_FALLBACK_FILE")
	if fallbackPath == "" {
		fallbackPath = ".secrets.local"
	}
	versionPins := secretVersionPinsFromEnv(env)
	credentialsFile := lookup("API_FIREBASE_CREDENTIALS_FILE")

	opts := []secrets.Option{
		secrets.WithEnvironment(envLabel),
		secrets.WithLogger(logger.Named("secrets")),
		secrets.WithFallbackFile(fallbackPath),
	}
	if len(projectMap) > 0 {
		opts = append(opts, secrets.WithProjectMap(projectMap))
	}
	if defaultProject != "" {
		opts = append(opts, secrets.WithDefaultProject(defaultProject))
	}
	if len(versionPins) > 0 {
		opts = append(opts, secrets.WithVersionPins(versionPins))
	}
	if credentialsFile != "" {
		opts = append(opts, secrets.WithClientOptions(option.WithCredentialsFile(credentialsFile)))
	}

	return secrets.NewFetcher(ctx, opts...)
}

func requiredSecretNames(env map[string]string) []string {
	required := []string{
		"PSP.StripeAPIKey",
		"PSP.StripeWebhookSecret",
		"Webhooks.SigningSecret",
	}

	hmacRaw := ""
	if env != nil {
		hmacRaw = strings.TrimSpace(env["API_SECURITY_HMAC_SECRETS"])
		if secret := strings.TrimSpace(env["API_PSP_PAYPAL_SECRET"]); secret != "" {
			required = append(required, "PSP.PayPalSecret")
		}
	}
	for _, key := range parseHMACSecretKeys(hmacRaw) {
		required = append(required, fmt.Sprintf("Security.HMAC.Secrets[%s]", key))
	}

	return uniqueStrings(required)
}

func secretProjectMapFromEnv(env map[string]string) map[string]string {
	raw := ""
	if env != nil {
		raw = env["API_SECRET_PROJECT_IDS"]
	}
	raw = strings.TrimSpace(raw)
	projects := make(map[string]string)
	if raw == "" {
		return projects
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
		envLabel := strings.ToLower(strings.TrimSpace(parts[0]))
		project := strings.TrimSpace(parts[1])
		if envLabel == "" || project == "" {
			continue
		}
		projects[envLabel] = project
	}
	return projects
}

func secretVersionPinsFromEnv(env map[string]string) map[string]string {
	raw := ""
	if env != nil {
		raw = env["API_SECRET_VERSION_PINS"]
	}
	raw = strings.TrimSpace(raw)
	pins := make(map[string]string)
	if raw == "" {
		return pins
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
		ref := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])
		if ref == "" || version == "" {
			continue
		}
		var prefix string
		if idx := strings.Index(ref, ":"); idx > 0 {
			schemeSplit := strings.Index(ref, "://")
			if schemeSplit == -1 || idx < schemeSplit {
				prefix = strings.ToLower(strings.TrimSpace(ref[:idx])) + ":"
				ref = strings.TrimSpace(ref[idx+1:])
			}
		}
		if strings.HasPrefix(ref, "sm://") {
			ref = "secret://" + strings.TrimPrefix(ref, "sm://")
		} else if !strings.HasPrefix(ref, "secret://") {
			ref = "secret://" + ref
		}
		ref = prefix + ref
		pins[ref] = version
	}
	return pins
}

func parseHMACSecretKeys(raw string) []string {
	values := parseKeyValueList(raw)
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, strings.ToLower(key))
	}
	sort.Strings(keys)
	return keys
}

func parseKeyValueList(raw string) map[string]string {
	result := make(map[string]string)
	if strings.TrimSpace(raw) == "" {
		return result
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
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			continue
		}
		result[key] = value
	}
	return result
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
