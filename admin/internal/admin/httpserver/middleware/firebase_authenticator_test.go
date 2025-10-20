package middleware

import (
	"context"
	"errors"
	"net/http"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type stubFirebaseVerifier struct {
	token *firebaseauth.Token
	err   error
}

func (s *stubFirebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	return s.token, s.err
}

func TestFirebaseAuthenticatorSuccess(t *testing.T) {
	verifier := &stubFirebaseVerifier{
		token: &firebaseauth.Token{
			UID: "user-123",
			Claims: map[string]interface{}{
				"email": "admin@example.com",
				"role":  []interface{}{"admin", "ops"},
			},
		},
	}

	auth := NewFirebaseAuthenticator(verifier)
	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	user, err := auth.Authenticate(req, "good-token")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.UID != "user-123" {
		t.Fatalf("unexpected uid %s", user.UID)
	}
	if user.Email != "admin@example.com" {
		t.Fatalf("unexpected email %s", user.Email)
	}
	if len(user.Roles) != 2 || user.Roles[0] != "admin" || user.Roles[1] != "ops" {
		t.Fatalf("unexpected roles %#v", user.Roles)
	}
}

func TestFirebaseAuthenticatorHandlesExpiredToken(t *testing.T) {
	verifier := &stubFirebaseVerifier{
		err: ErrTokenExpired,
	}
	auth := NewFirebaseAuthenticator(verifier)

	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	_, err := auth.Authenticate(req, "expired")
	if err == nil {
		t.Fatalf("expected error")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected auth error")
	}
	if authErr.Reason != ReasonTokenExpired {
		t.Fatalf("expected token_expired, got %s", authErr.Reason)
	}
}

func TestFirebaseAuthenticatorRejectsMissingToken(t *testing.T) {
	auth := NewFirebaseAuthenticator(&stubFirebaseVerifier{})
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	_, err := auth.Authenticate(req, "  ")
	if err == nil {
		t.Fatalf("expected error")
	}
	var authErr *AuthError
	if !errors.As(err, &authErr) {
		t.Fatalf("expected auth error")
	}
	if authErr.Reason != ReasonMissingToken {
		t.Fatalf("expected missing_token, got %s", authErr.Reason)
	}
}
