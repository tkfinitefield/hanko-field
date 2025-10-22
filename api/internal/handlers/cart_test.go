package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/services"
)

func TestCartHandlersGetCartSuccess(t *testing.T) {
	now := time.Date(2024, 5, 12, 10, 0, 0, 0, time.UTC)
	updated := now.Add(2 * time.Minute)
	itemUpdated := now.Add(3 * time.Minute)

	service := &stubCartService{
		getOrCreateFunc: func(ctx context.Context, userID string) (services.Cart, error) {
			if userID != "user-7" {
				t.Fatalf("unexpected user id %q", userID)
			}
			return services.Cart{
				ID:       "cart-user-7",
				UserID:   "user-7",
				Currency: "jpy",
				Items: []services.CartItem{
					{
						ID:               "item-1",
						ProductID:        "prod-1",
						SKU:              "SKU-1",
						Quantity:         1,
						UnitPrice:        1200,
						Currency:         "JPY",
						RequiresShipping: true,
						Customization:    map[string]any{"color": "red"},
						Metadata:         map[string]any{"note": "gift"},
						Estimates:        map[string]int64{"tax": 120},
						AddedAt:          now,
						UpdatedAt:        &itemUpdated,
					},
				},
				BillingAddress: &services.Address{
					ID:         "addr-1",
					PostalCode: "100-0001",
					Country:    "JP",
				},
				Promotion: &services.CartPromotion{
					Code:           "SPRING24",
					DiscountAmount: 200,
					Applied:        true,
				},
				Estimate: &services.CartEstimate{
					Subtotal: 1200,
					Discount: 200,
					Tax:      100,
					Shipping: 0,
					Total:    1100,
				},
				Metadata:  map[string]any{"channel": "app"},
				UpdatedAt: updated,
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)

	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-7"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	cacheControl := rr.Header().Get("Cache-Control")
	if !strings.Contains(cacheControl, "no-store") {
		t.Fatalf("expected Cache-Control no-store, got %q", cacheControl)
	}
	if etag := rr.Header().Get("ETag"); etag == "" {
		t.Fatalf("expected ETag header")
	}
	if lastModified := rr.Header().Get("Last-Modified"); lastModified == "" {
		t.Fatalf("expected Last-Modified header")
	}

	var resp cartResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Cart.ID != "cart-user-7" {
		t.Fatalf("expected cart id cart-user-7, got %q", resp.Cart.ID)
	}
	if resp.Cart.Currency != "JPY" {
		t.Fatalf("expected currency JPY, got %q", resp.Cart.Currency)
	}
	if resp.Cart.ItemsCount != 1 || len(resp.Cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", resp.Cart.ItemsCount)
	}
	if resp.Cart.Estimate == nil || resp.Cart.Estimate.Total != 1100 {
		t.Fatalf("expected estimate total 1100, got %#v", resp.Cart.Estimate)
	}
}

func TestCartHandlersGetCartServiceUnavailable(t *testing.T) {
	handler := NewCartHandlers(nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	rr := httptest.NewRecorder()
	handler.getCart(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rr.Code)
	}
}

func TestCartHandlersGetCartUnauthenticated(t *testing.T) {
	handler := NewCartHandlers(nil, &stubCartService{})
	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	rr := httptest.NewRecorder()
	handler.getCart(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

func TestCartHandlersGetCartInvalidInput(t *testing.T) {
	service := &stubCartService{
		getOrCreateFunc: func(ctx context.Context, userID string) (services.Cart, error) {
			return services.Cart{}, services.ErrCartInvalidInput
		},
	}
	handler := NewCartHandlers(nil, service)
	req := httptest.NewRequest(http.MethodGet, "/cart", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	rr := httptest.NewRecorder()
	handler.getCart(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

type stubCartService struct {
	getOrCreateFunc func(ctx context.Context, userID string) (services.Cart, error)
}

func (s *stubCartService) GetOrCreateCart(ctx context.Context, userID string) (services.Cart, error) {
	if s.getOrCreateFunc != nil {
		return s.getOrCreateFunc(ctx, userID)
	}
	return services.Cart{}, services.ErrCartUnavailable
}

func (s *stubCartService) AddOrUpdateItem(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error) {
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) RemoveItem(ctx context.Context, cmd services.RemoveCartItemCommand) (services.Cart, error) {
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) Estimate(ctx context.Context, userID string) (services.CartEstimate, error) {
	return services.CartEstimate{}, errors.New("not implemented")
}

func (s *stubCartService) ApplyPromotion(ctx context.Context, cmd services.CartPromotionCommand) (services.Cart, error) {
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) RemovePromotion(ctx context.Context, userID string) (services.Cart, error) {
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) ClearCart(ctx context.Context, userID string) error {
	return errors.New("not implemented")
}
