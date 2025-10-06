package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"github.com/hanko-field/api/internal/handlers"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/config"
	"github.com/hanko-field/api/internal/platform/idempotency"
	"github.com/hanko-field/api/internal/platform/observability"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

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

	firestoreClient, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		logger.Fatal("failed to initialise firestore client", zap.Error(err))
	}
	defer func() {
		if err := firestoreClient.Close(); err != nil {
			logger.Warn("firestore close error", zap.Error(err))
		}
	}()

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

	var opts []handlers.Option
	opts = append(opts, handlers.WithMiddlewares(middlewares...))
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
