package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
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
	}

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
		log.Printf("failed to initialise Firebase app: %v", err)
		return nil
	}

	client, err := app.Auth(ctx)
	if err != nil {
		log.Printf("failed to initialise Firebase auth client: %v", err)
		return nil
	}

	log.Printf("Firebase authenticator enabled (project=%s)", projectID)
	return middleware.NewFirebaseAuthenticator(client)
}
