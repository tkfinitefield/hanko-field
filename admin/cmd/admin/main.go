package main

import (
	"context"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	firebase "firebase.google.com/go/v4"

	"finitefield.org/hanko-admin/internal/admin/httpserver"
	"finitefield.org/hanko-admin/internal/admin/httpserver/middleware"
)

func main() {
	rootCtx := context.Background()
	cfg := httpserver.Config{
		Address:       getEnv("ADMIN_HTTP_ADDR", ":8080"),
		BasePath:      getEnv("ADMIN_BASE_PATH", "/admin"),
		Authenticator: buildAuthenticator(rootCtx),
		Session: httpserver.SessionConfig{
			CookieName:       getEnv("ADMIN_SESSION_COOKIE_NAME", ""),
			CookieDomain:     os.Getenv("ADMIN_SESSION_COOKIE_DOMAIN"),
			CookieSecure:     getEnvBool("ADMIN_SESSION_COOKIE_SECURE", false),
			CookieHTTPOnly:   boolPtr(true),
			IdleTimeout:      getEnvDuration("ADMIN_SESSION_IDLE_TIMEOUT", 0),
			Lifetime:         getEnvDuration("ADMIN_SESSION_LIFETIME", 0),
			RememberLifetime: getEnvDuration("ADMIN_SESSION_REMEMBER_LIFETIME", 0),
			HashKey:          getEnvBytes("ADMIN_SESSION_HASH_KEY"),
			BlockKey:         getEnvBytes("ADMIN_SESSION_BLOCK_KEY"),
		},
	}
	cfg.Session.CookiePath = cfg.BasePath

	srv := httpserver.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server failed: %v", err)
		}
	}()

	log.Printf("admin server listening on %s (base path %s)", cfg.Address, cfg.BasePath)

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		cancel()
		stop()
		os.Exit(1)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		val, err := strconv.ParseBool(v)
		if err != nil {
			log.Printf("invalid boolean for %s: %v", key, err)
			return fallback
		}
		return val
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			log.Printf("invalid duration for %s: %v", key, err)
			return fallback
		}
		return d
	}
	return fallback
}

func getEnvBytes(key string) []byte {
	val := os.Getenv(key)
	if val == "" {
		return nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(val); err == nil {
		return decoded
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(val); err == nil {
		return decoded
	}
	if decoded, err := base64.URLEncoding.DecodeString(val); err == nil {
		return decoded
	}
	if decoded, err := base64.RawURLEncoding.DecodeString(val); err == nil {
		return decoded
	}
	log.Printf("using literal value for %s; base64 decode failed", key)
	return []byte(val)
}

func boolPtr(v bool) *bool {
	return &v
}

func buildAuthenticator(ctx context.Context) middleware.Authenticator {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		log.Printf("FIREBASE_PROJECT_ID not set; using passthrough authenticator")
		return nil
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	})
	if err != nil {
		log.Fatalf("failed to initialise Firebase app: %v", err)
	}

	client, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("failed to initialise Firebase auth client: %v", err)
	}

	log.Printf("Firebase authenticator enabled (project=%s)", projectID)
	return middleware.NewFirebaseAuthenticator(client)
}
