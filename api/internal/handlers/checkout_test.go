package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestCheckoutHandlersCreateSessionSuccess(t *testing.T) {
	router := chi.NewRouter()
	var captured services.CreateCheckoutSessionCommand
	service := &stubCheckoutService{
		createFunc: func(ctx context.Context, cmd services.CreateCheckoutSessionCommand) (services.CheckoutSession, error) {
			captured = cmd
			if cmd.UserID != "user-1" {
				t.Fatalf("expected user id user-1, got %s", cmd.UserID)
			}
			if cmd.SuccessURL != "https://example.com/success" {
				t.Fatalf("unexpected success url %s", cmd.SuccessURL)
			}
			return services.CheckoutSession{
				SessionID:    "sess_123",
				PSP:          "stripe",
				ClientSecret: "sec_abc",
				RedirectURL:  "https://checkout.example/sess_123",
				ExpiresAt:    time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	handler := NewCheckoutHandlers(nil, service)
	handler.Routes(router)

	payload := `{"provider":"stripe","successUrl":"https://example.com/success","cancelUrl":"https://example.com/cancel","metadata":{"locale":"ja-JP"}}`
	req := httptest.NewRequest(http.MethodPost, "/checkout/session", bytes.NewBufferString(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp checkoutSessionResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.SessionID != "sess_123" {
		t.Fatalf("expected session id sess_123, got %s", resp.SessionID)
	}
	if resp.ClientSecret != "sec_abc" {
		t.Fatalf("expected client secret returned")
	}
	if captured.Metadata["locale"] != "ja-JP" {
		t.Fatalf("expected metadata propagated, got %#v", captured.Metadata)
	}
}

func TestCheckoutHandlersCreateSessionUnauthenticated(t *testing.T) {
	router := chi.NewRouter()
	handler := NewCheckoutHandlers(nil, &stubCheckoutService{})
	handler.Routes(router)

	req := httptest.NewRequest(http.MethodPost, "/checkout/session", bytes.NewBufferString(`{"successUrl":"https://example.com/success","cancelUrl":"https://example.com/cancel"}`))
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestCheckoutHandlersCreateSessionMapsServiceErrors(t *testing.T) {
	router := chi.NewRouter()
	handler := NewCheckoutHandlers(nil, &stubCheckoutService{
		createFunc: func(context.Context, services.CreateCheckoutSessionCommand) (services.CheckoutSession, error) {
			return services.CheckoutSession{}, services.ErrCheckoutInsufficientStock
		},
	})
	handler.Routes(router)

	payload := `{"successUrl":"https://example.com/success","cancelUrl":"https://example.com/cancel"}`
	req := httptest.NewRequest(http.MethodPost, "/checkout/session", bytes.NewBufferString(payload))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}

	var errResp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if errResp["error"] != "insufficient_stock" {
		t.Fatalf("expected error code insufficient_stock, got %#v", errResp["error"])
	}
}

type stubCheckoutService struct {
	createFunc  func(ctx context.Context, cmd services.CreateCheckoutSessionCommand) (services.CheckoutSession, error)
	confirmFunc func(ctx context.Context, cmd services.ConfirmCheckoutCommand) error
}

func (s *stubCheckoutService) CreateCheckoutSession(ctx context.Context, cmd services.CreateCheckoutSessionCommand) (services.CheckoutSession, error) {
	if s.createFunc != nil {
		return s.createFunc(ctx, cmd)
	}
	return services.CheckoutSession{}, errors.New("not implemented")
}

func (s *stubCheckoutService) ConfirmClientCompletion(ctx context.Context, cmd services.ConfirmCheckoutCommand) error {
	if s.confirmFunc != nil {
		return s.confirmFunc(ctx, cmd)
	}
	return errors.New("not implemented")
}
