package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	firebase "firebase.google.com/go/v4"
	firebaseauth "firebase.google.com/go/v4/auth"
	"github.com/hanko-field/api/internal/platform/config"
	"google.golang.org/api/option"
)

// FirebaseVerifier coordinates Firebase Admin SDK initialisation for token verification.
type FirebaseVerifier struct {
	client  *firebaseauth.Client
	timeout time.Duration
}

// FirebaseOption customises FirebaseVerifier instances.
type FirebaseOption func(*FirebaseVerifier)

// WithFirebaseTimeout overrides the timeout used for Admin SDK calls.
func WithFirebaseTimeout(d time.Duration) FirebaseOption {
	return func(v *FirebaseVerifier) {
		if d > 0 {
			v.timeout = d
		}
	}
}

// NewFirebaseVerifier constructs a FirebaseVerifier backed by the Admin SDK.
func NewFirebaseVerifier(ctx context.Context, cfg config.FirebaseConfig, opts ...FirebaseOption) (*FirebaseVerifier, error) {
	if cfg.ProjectID == "" {
		return nil, errors.New("firebase project id is required")
	}

	var clientOpts []option.ClientOption
	if cfg.CredentialsFile != "" {
		clientOpts = append(clientOpts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: cfg.ProjectID}, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("initialise firebase app: %w", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialise firebase auth client: %w", err)
	}

	verifier := &FirebaseVerifier{
		client:  authClient,
		timeout: defaultVerifyTimeout,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(verifier)
		}
	}

	return verifier, nil
}

// VerifyIDToken forwards verification to the underlying Firebase client using a bounded context.
func (v *FirebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	if v == nil || v.client == nil {
		return nil, errors.New("firebase verifier not initialised")
	}

	ctx, cancel := v.contextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	return v.client.VerifyIDToken(ctx, idToken)
}

// GetUser loads a Firebase user record for the given UID, respecting the configured timeout.
func (v *FirebaseVerifier) GetUser(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
	if v == nil || v.client == nil {
		return nil, errors.New("firebase verifier not initialised")
	}

	ctx, cancel := v.contextWithTimeout(ctx)
	if cancel != nil {
		defer cancel()
	}

	return v.client.GetUser(ctx, uid)
}

func (v *FirebaseVerifier) contextWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if v == nil || v.timeout <= 0 {
		return ctx, nil
	}
	return context.WithTimeout(ctx, v.timeout)
}
