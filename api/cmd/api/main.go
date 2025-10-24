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
	cloudstorage "cloud.google.com/go/storage"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hanko-field/api/internal/handlers"
	"github.com/hanko-field/api/internal/payments"
	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/config"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/platform/idempotency"
	"github.com/hanko-field/api/internal/platform/observability"
	"github.com/hanko-field/api/internal/platform/secrets"
	platformstorage "github.com/hanko-field/api/internal/platform/storage"
	"github.com/hanko-field/api/internal/repositories"
	firestoreRepo "github.com/hanko-field/api/internal/repositories/firestore"
	"github.com/hanko-field/api/internal/services"

	"github.com/oklog/ulid/v2"
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

	firestoreProvider := pfirestore.NewProvider(cfg.Firestore)
	firestoreClient, err := firestoreProvider.Client(ctx)
	if err != nil {
		logger.Fatal("failed to initialise firestore client", zap.Error(err))
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := firestoreProvider.Close(closeCtx); err != nil {
			logger.Warn("firestore close error", zap.Error(err))
		}
	}()

	storageClient, err := cloudstorage.NewClient(ctx)
	if err != nil {
		logger.Fatal("failed to initialise storage client", zap.Error(err))
	}
	defer func() {
		if err := storageClient.Close(); err != nil {
			logger.Warn("storage close error", zap.Error(err))
		}
	}()

	assetCopier, err := platformstorage.NewCopier(storageClient)
	if err != nil {
		logger.Fatal("failed to initialise storage copier", zap.Error(err))
	}

	signerKey := strings.TrimSpace(cfg.Storage.SignedURLKey)
	if signerKey == "" {
		logger.Fatal("storage signer key is required")
	}
	signer, err := platformstorage.NewServiceAccountSignerFromJSON([]byte(signerKey))
	if err != nil {
		logger.Fatal("failed to parse storage signer key", zap.Error(err))
	}
	signedURLClient, err := platformstorage.NewClient(signer)
	if err != nil {
		logger.Fatal("failed to initialise signed url client", zap.Error(err))
	}

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

	firebaseVerifier, err := auth.NewFirebaseVerifier(ctx, cfg.Firebase)
	if err != nil {
		logger.Fatal("failed to initialise firebase verifier", zap.Error(err))
	}
	authenticator := auth.NewAuthenticator(firebaseVerifier, auth.WithUserGetter(firebaseVerifier))

	userRepo, err := firestoreRepo.NewUserRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise user repository", zap.Error(err))
	}
	addressRepo, err := firestoreRepo.NewAddressRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise address repository", zap.Error(err))
	}
	favoriteRepo, err := firestoreRepo.NewFavoriteRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise favorite repository", zap.Error(err))
	}
	cartRepo, err := firestoreRepo.NewCartRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise cart repository", zap.Error(err))
	}
	inventoryRepo, err := firestoreRepo.NewInventoryRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise inventory repository", zap.Error(err))
	}
	nameMappingRepo, err := firestoreRepo.NewNameMappingRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise name mapping repository", zap.Error(err))
	}
	registrabilityRepo, err := firestoreRepo.NewDesignRegistrabilityRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise registrability repository", zap.Error(err))
	}
	designRepo, err := firestoreRepo.NewDesignRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise design repository", zap.Error(err))
	}
	designVersionRepo, err := firestoreRepo.NewDesignVersionRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise design version repository", zap.Error(err))
	}
	designFinder := newFirestoreDesignFinder(firestoreProvider)
	paymentRepo, err := firestoreRepo.NewPaymentMethodRepository(firestoreProvider)
	if err != nil {
		logger.Fatal("failed to initialise payment method repository", zap.Error(err))
	}
	assetRepo, err := firestoreRepo.NewAssetRepository(firestoreProvider, signedURLClient, cfg.Storage.AssetsBucket)
	if err != nil {
		logger.Fatal("failed to initialise asset repository", zap.Error(err))
	}

	if strings.TrimSpace(cfg.PSP.StripeAPIKey) == "" {
		logger.Fatal("stripe api key is required for payment method management")
	}
	stripeVerifier, err := payments.NewStripePaymentMethodVerifier(payments.StripeProviderConfig{
		APIKey: cfg.PSP.StripeAPIKey,
	})
	if err != nil {
		logger.Fatal("failed to initialise stripe payment verifier", zap.Error(err))
	}
	paymentVerifier := &providerPaymentVerifier{stripe: stripeVerifier}

	inventoryService, err := services.NewInventoryService(services.InventoryServiceDeps{
		Inventory: inventoryRepo,
		Clock:     time.Now,
	})
	if err != nil {
		logger.Fatal("failed to initialise inventory service", zap.Error(err))
	}

	paymentsLogger := logger.Named("payments")
	stripeProvider, err := payments.NewStripeProvider(payments.StripeProviderConfig{
		APIKey: cfg.PSP.StripeAPIKey,
		Logger: func(ctx context.Context, event string, fields map[string]any) {
			zFields := make([]zap.Field, 0, len(fields)+1)
			zFields = append(zFields, zap.String("event", event))
			for k, v := range fields {
				zFields = append(zFields, zap.Any(k, v))
			}
			paymentsLogger.Debug("stripe log", zFields...)
		},
		Clock: time.Now,
	})
	if err != nil {
		logger.Fatal("failed to initialise stripe payment provider", zap.Error(err))
	}

	paymentManager, err := payments.NewManager(map[string]payments.Provider{
		"stripe": stripeProvider,
	})
	if err != nil {
		logger.Fatal("failed to initialise payment manager", zap.Error(err))
	}

	userService, err := services.NewUserService(services.UserServiceDeps{
		Users:           userRepo,
		Addresses:       addressRepo,
		PaymentMethods:  paymentRepo,
		PaymentVerifier: paymentVerifier,
		Favorites:       favoriteRepo,
		Designs:         designFinder,
		Audit:           nil,
		Firebase:        firebaseVerifier,
		Clock:           time.Now,
	})
	if err != nil {
		logger.Fatal("failed to initialise user service", zap.Error(err))
	}
	meHandlers := handlers.NewMeHandlers(authenticator, userService)
	assetService, err := services.NewAssetService(services.AssetServiceDeps{
		Repository: assetRepo,
		Clock:      time.Now,
	})
	if err != nil {
		logger.Fatal("failed to initialise asset service", zap.Error(err))
	}
	assetHandlers := handlers.NewAssetHandlers(authenticator, assetService)

	cartLogger := logger.Named("cart")
	cartService, err := services.NewCartService(services.CartServiceDeps{
		Repository:      cartRepo,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
		Logger: func(ctx context.Context, event string, fields map[string]any) {
			zFields := make([]zap.Field, 0, len(fields)+1)
			zFields = append(zFields, zap.String("event", event))
			for k, v := range fields {
				zFields = append(zFields, zap.Any(k, v))
			}
			cartLogger.Debug("cart log", zFields...)
		},
	})
	if err != nil {
		logger.Fatal("failed to initialise cart service", zap.Error(err))
	}
	cartHandlers := handlers.NewCartHandlers(authenticator, cartService)

	checkoutLogger := logger.Named("checkout")
	checkoutWorkflowDispatcher := services.CheckoutWorkflowDispatcherFunc(func(ctx context.Context, payload services.CheckoutWorkflowPayload) (string, error) {
		workflowID := ulid.Make().String()
		fields := map[string]any{
			"workflowId":  workflowID,
			"userId":      strings.TrimSpace(payload.UserID),
			"cartId":      strings.TrimSpace(payload.CartID),
			"sessionId":   strings.TrimSpace(payload.SessionID),
			"intentId":    strings.TrimSpace(payload.PaymentIntentID),
			"orderId":     strings.TrimSpace(payload.OrderID),
			"status":      strings.TrimSpace(payload.Status),
			"reservation": strings.TrimSpace(payload.ReservationID),
		}
		checkoutLogger.Debug("checkout workflow dispatched", zap.Any("payload", fields))
		return workflowID, nil
	})
	checkoutService, err := services.NewCheckoutService(services.CheckoutServiceDeps{
		Carts:     cartRepo,
		Inventory: inventoryService,
		Payments:  paymentManager,
		Workflow:  checkoutWorkflowDispatcher,
		Clock:     time.Now,
		Logger: func(ctx context.Context, event string, fields map[string]any) {
			zFields := make([]zap.Field, 0, len(fields)+1)
			zFields = append(zFields, zap.String("event", event))
			for k, v := range fields {
				zFields = append(zFields, zap.Any(k, v))
			}
			checkoutLogger.Debug("checkout log", zFields...)
		},
	})
	if err != nil {
		logger.Fatal("failed to initialise checkout service", zap.Error(err))
	}
	checkoutHandlers := handlers.NewCheckoutHandlers(authenticator, checkoutService)

	nameMappingLogger := logger.Named("name_mapping")
	nameMappingService, err := services.NewNameMappingService(services.NameMappingServiceDeps{
		Repository: nameMappingRepo,
		Users:      userRepo,
		Clock:      time.Now,
		Logger: func(_ context.Context, event string, fields map[string]any) {
			zFields := make([]zap.Field, 0, len(fields)+1)
			zFields = append(zFields, zap.String("event", event))
			for k, v := range fields {
				zFields = append(zFields, zap.Any(k, v))
			}
			nameMappingLogger.Debug("name mapping log", zFields...)
		},
	})
	if err != nil {
		logger.Fatal("failed to initialise name mapping service", zap.Error(err))
	}
	nameMappingHandlers := handlers.NewNameMappingHandlers(authenticator, nameMappingService)

	registrabilityEvaluator := services.NewHeuristicRegistrabilityEvaluator(time.Now)

	designService, err := services.NewDesignService(services.DesignServiceDeps{
		Designs:             designRepo,
		Versions:            designVersionRepo,
		AssetCopier:         assetCopier,
		AssetsBucket:        cfg.Storage.AssetsBucket,
		Clock:               time.Now,
		Registrability:      registrabilityEvaluator,
		RegistrabilityCache: registrabilityRepo,
	})
	if err != nil {
		logger.Fatal("failed to initialise design service", zap.Error(err))
	}
	designHandlers := handlers.NewDesignHandlers(authenticator, designService)
	reviewHandlers := handlers.NewReviewHandlers(authenticator, nil)
	orderHandlers := handlers.NewOrderHandlers(authenticator, nil)

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
	opts = append(opts, handlers.WithMeRoutes(meHandlers.Routes))
	opts = append(opts, handlers.WithDesignRoutes(designHandlers.Routes))
	opts = append(opts, handlers.WithReviewRoutes(reviewHandlers.Routes))
	opts = append(opts, handlers.WithOrderRoutes(orderHandlers.Routes))
	opts = append(opts, handlers.WithNameMappingRoutes(nameMappingHandlers.Routes))
	opts = append(opts, handlers.WithCartRoutes(cartHandlers.Routes))
	opts = append(opts, handlers.WithAdditionalRoutes(cartHandlers.RegisterStandaloneRoutes))
	opts = append(opts, handlers.WithAdditionalRoutes(assetHandlers.Routes))
	opts = append(opts, handlers.WithAdditionalRoutes(checkoutHandlers.Routes))
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
		"Storage.SignerKey",
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

type providerPaymentVerifier struct {
	stripe *payments.StripePaymentMethodVerifier
}

func (p *providerPaymentVerifier) VerifyPaymentMethod(ctx context.Context, provider string, token string) (services.PaymentMethodMetadata, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "stripe":
		if p.stripe == nil {
			return services.PaymentMethodMetadata{}, fmt.Errorf("stripe payment verifier unavailable")
		}
		details, err := p.stripe.Lookup(ctx, token)
		if err != nil {
			return services.PaymentMethodMetadata{}, err
		}
		return services.PaymentMethodMetadata{
			Token:    details.Token,
			Brand:    details.Brand,
			Last4:    details.Last4,
			ExpMonth: details.ExpMonth,
			ExpYear:  details.ExpYear,
		}, nil
	default:
		return services.PaymentMethodMetadata{}, fmt.Errorf("unsupported payment provider %q", provider)
	}
}

type firestoreDesignFinder struct {
	provider *pfirestore.Provider
}

func newFirestoreDesignFinder(provider *pfirestore.Provider) services.DesignFinder {
	if provider == nil {
		return nil
	}
	return &firestoreDesignFinder{provider: provider}
}

func (f *firestoreDesignFinder) FindByID(ctx context.Context, designID string) (services.Design, error) {
	if f == nil || f.provider == nil {
		return services.Design{}, fmt.Errorf("design finder not configured")
	}
	trimmed := strings.TrimSpace(designID)
	if trimmed == "" {
		return services.Design{}, &favoriteNotFoundError{err: errors.New("design id required")}
	}
	client, err := f.provider.Client(ctx)
	if err != nil {
		return services.Design{}, err
	}
	snap, err := client.Collection("designs").Doc(trimmed).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return services.Design{}, &favoriteNotFoundError{err: err}
		}
		return services.Design{}, err
	}

	data := snap.Data()
	ownerRef, _ := data["ownerRef"].(string)
	statusValue, _ := data["status"].(string)
	locale, _ := data["locale"].(string)
	template := ""
	if style, ok := data["style"].(map[string]any); ok {
		if ref, ok := style["templateRef"].(string); ok {
			template = ref
		}
	}

	var snapshot map[string]any
	if raw, ok := data["snapshot"].(map[string]any); ok && len(raw) > 0 {
		snapshot = raw
	}

	updatedAt := snap.UpdateTime
	if ts, ok := data["updatedAt"].(time.Time); ok && !ts.IsZero() {
		updatedAt = ts
	}

	design := services.Design{
		ID:        trimmed,
		OwnerID:   extractOwnerID(ownerRef),
		Status:    services.DesignStatus(strings.TrimSpace(statusValue)),
		Template:  template,
		Locale:    locale,
		Snapshot:  snapshot,
		UpdatedAt: updatedAt,
	}
	return design, nil
}

type favoriteNotFoundError struct {
	err error
}

func (e *favoriteNotFoundError) Error() string       { return e.err.Error() }
func (e *favoriteNotFoundError) Unwrap() error       { return e.err }
func (e *favoriteNotFoundError) IsNotFound() bool    { return true }
func (e *favoriteNotFoundError) IsConflict() bool    { return false }
func (e *favoriteNotFoundError) IsUnavailable() bool { return false }

func extractOwnerID(ref string) string {
	trimmed := strings.TrimSpace(ref)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.HasPrefix(trimmed, "users/") {
		return trimmed[len("users/"):]
	}
	return trimmed
}
