package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type stubTokenVerifier struct {
	token    *firebaseauth.Token
	err      error
	received string
}

func (s *stubTokenVerifier) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	s.received = idToken
	if s.err != nil {
		return nil, s.err
	}
	return s.token, nil
}

type stubUserGetter struct {
	record  *firebaseauth.UserRecord
	calls   int
	lastUID string
}

func (s *stubUserGetter) GetUser(ctx context.Context, uid string) (*firebaseauth.UserRecord, error) {
	s.calls++
	s.lastUID = uid
	return s.record, nil
}

func TestRequireFirebaseAuth_AllowsValidToken(t *testing.T) {
	verifier := &stubTokenVerifier{
		token: &firebaseauth.Token{
			UID: "uid-123",
			Claims: map[string]interface{}{
				"role":   []interface{}{"staff", "admin"},
				"locale": "ja-JP",
				"email":  "user@example.com",
			},
		},
	}
	userGetter := &stubUserGetter{record: &firebaseauth.UserRecord{UserInfo: &firebaseauth.UserInfo{UID: "uid-123", Email: "user@example.com"}}}

	authn := NewAuthenticator(verifier, WithUserGetter(userGetter))

	handlerCalled := false
	handler := authn.RequireFirebaseAuth(RoleStaff)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true

		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatalf("expected identity in context")
		}
		if identity.UID != "uid-123" {
			t.Fatalf("unexpected uid: %s", identity.UID)
		}
		if !identity.HasRole(RoleStaff) {
			t.Fatalf("expected staff role, got %v", identity.Roles)
		}
		if identity.Locale != "ja-JP" {
			t.Fatalf("expected locale ja-JP, got %s", identity.Locale)
		}
		if identity.Email != "user@example.com" {
			t.Fatalf("expected email user@example.com, got %s", identity.Email)
		}

		// Ensure the user loader is lazy and memoized.
		loaded, err := identity.User(r.Context())
		if err != nil {
			t.Fatalf("unexpected user load error: %v", err)
		}
		loadedAgain, err := identity.User(r.Context())
		if err != nil {
			t.Fatalf("unexpected second user load error: %v", err)
		}
		if loaded != loadedAgain {
			t.Fatalf("expected cached user record")
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token-value")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rr.Code)
	}
	if !handlerCalled {
		t.Fatalf("expected handler to be called")
	}
	if verifier.received != "token-value" {
		t.Fatalf("expected verifier to receive token-value, got %s", verifier.received)
	}
	if userGetter.calls != 1 {
		t.Fatalf("expected single user fetch, got %d", userGetter.calls)
	}
	if userGetter.lastUID != "uid-123" {
		t.Fatalf("expected user loader to receive uid-123, got %s", userGetter.lastUID)
	}
}

func TestRequireFirebaseAuth_ExpiredToken(t *testing.T) {
	verifier := &stubTokenVerifier{err: ErrTokenExpired}
	authn := NewAuthenticator(verifier)

	handler := authn.RequireFirebaseAuth(RoleUser)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not execute on expired token")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON body: %v", err)
	}
	if body["error"] != "token_expired" {
		t.Fatalf("expected token_expired error, got %v", body["error"])
	}
}

func TestRequireFirebaseAuth_MissingRoleUsesFallback(t *testing.T) {
	verifier := &stubTokenVerifier{
		token: &firebaseauth.Token{
			UID:    "uid-456",
			Claims: map[string]interface{}{},
		},
	}

	authn := NewAuthenticator(verifier)

	handler := authn.RequireFirebaseAuth()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r.Context())
		if !ok {
			t.Fatalf("expected identity in context")
		}
		if len(identity.Roles) != 1 || identity.Roles[0] != RoleUser {
			t.Fatalf("expected fallback role %q, got %v", RoleUser, identity.Roles)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer missing-role-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
