package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"

	"github.com/hanko-field/api/internal/handlers"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/config"
	"github.com/hanko-field/api/internal/platform/idempotency"
)

func main() {
	ctx := context.Background()
	cfg, err := config.Load(ctx)
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	logger := log.Default()

	firestoreClient, err := newFirestoreClient(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialise firestore client: %v", err)
	}
	defer func() {
		if err := firestoreClient.Close(); err != nil {
			logger.Printf("firestore: close error: %v", err)
		}
	}()

	idempotencyStore := idempotency.NewFirestoreStore(firestoreClient)
	idempotencyMiddleware := idempotency.Middleware(
		idempotencyStore,
		idempotency.WithHeader(cfg.Idempotency.Header),
		idempotency.WithTTL(cfg.Idempotency.TTL),
		idempotency.WithLogger(logger),
	)

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	var cleanupWG sync.WaitGroup
	var cleanupTicker *time.Ticker
	if cfg.Idempotency.CleanupInterval > 0 {
		cleanupTicker = time.NewTicker(cfg.Idempotency.CleanupInterval)
		cleanupWG.Add(1)
		go func() {
			defer cleanupWG.Done()
			for {
				select {
				case <-cleanupTicker.C:
					runCtx, cancel := context.WithTimeout(cleanupCtx, time.Minute)
					removed, err := idempotencyStore.CleanupExpired(runCtx, time.Now().UTC(), cfg.Idempotency.CleanupBatchSize)
					cancel()
					if err != nil {
						logger.Printf("idempotency cleanup error: %v", err)
						continue
					}
					if removed > 0 {
						logger.Printf("idempotency cleanup removed %d records", removed)
					}
				case <-cleanupCtx.Done():
					return
				}
			}
		}()
	}

	oidcMiddleware := buildOIDCMiddleware(logger, cfg)
	hmacMiddleware := buildHMACMiddleware(logger, cfg)

	var opts []handlers.Option
	opts = append(opts, handlers.WithMiddlewares(idempotencyMiddleware))
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

	go func() {
		log.Printf("hanko-field api listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server error: %v", err)
		}
	}()

	<-shutdown
	log.Println("shutdown signal received; draining requests")

	if cleanupTicker != nil {
		cleanupTicker.Stop()
	}
	cleanupCancel()
	cleanupWG.Wait()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}

func buildOIDCMiddleware(logger *log.Logger, cfg config.Config) func(http.Handler) http.Handler {
	if strings.TrimSpace(cfg.Security.OIDC.JWKSURL) == "" {
		return nil
	}

	cache := auth.NewJWKSCache(cfg.Security.OIDC.JWKSURL, auth.WithJWKSLogger(logger))
	validator := auth.NewOIDCValidator(cache, auth.WithOIDCLogger(logger))

	audience := strings.TrimSpace(cfg.Security.OIDC.Audience)
	if audience == "" {
		logger.Printf("auth: OIDC audience not configured; internal routes will reject requests")
	}
	issuers := cfg.Security.OIDC.Issuers
	if len(issuers) == 0 {
		logger.Printf("auth: OIDC issuers not configured; internal routes will reject requests")
	}

	return validator.RequireOIDC(audience, issuers)
}

func buildHMACMiddleware(logger *log.Logger, cfg config.Config) func(http.Handler) http.Handler {
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
	validator := auth.NewHMACValidator(provider, nonces,
		auth.WithHMACLogger(logger),
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
