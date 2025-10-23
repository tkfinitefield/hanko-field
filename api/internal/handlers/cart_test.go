package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

func TestCartHandlersPatchCartSuccess(t *testing.T) {
	updatedAt := time.Date(2024, 6, 10, 12, 0, 0, 0, time.UTC)
	service := &stubCartService{
		updateFunc: func(ctx context.Context, cmd services.UpdateCartCommand) (services.Cart, error) {
			if cmd.UserID != "user-22" {
				t.Fatalf("unexpected user id %q", cmd.UserID)
			}
			if cmd.Currency == nil || *cmd.Currency != "usd" {
				t.Fatalf("expected currency pointer usd, got %#v", cmd.Currency)
			}
			if cmd.ShippingAddressID == nil || *cmd.ShippingAddressID != "addr-1" {
				t.Fatalf("expected shipping address id addr-1, got %#v", cmd.ShippingAddressID)
			}
			if cmd.BillingAddressID == nil || *cmd.BillingAddressID != "" {
				t.Fatalf("expected billing address cleared, got %#v", cmd.BillingAddressID)
			}
			if cmd.Notes == nil || *cmd.Notes != "  gift " {
				t.Fatalf("expected notes pointer, got %#v", cmd.Notes)
			}
			if cmd.PromotionHint == nil || *cmd.PromotionHint != "vip" {
				t.Fatalf("expected promotion hint vip, got %#v", cmd.PromotionHint)
			}
			if cmd.ExpectedUpdatedAt == nil || !cmd.ExpectedUpdatedAt.Equal(updatedAt) {
				t.Fatalf("expected updated_at %v, got %#v", updatedAt, cmd.ExpectedUpdatedAt)
			}
			return services.Cart{
				ID:            "cart-22",
				UserID:        cmd.UserID,
				Currency:      "USD",
				Notes:         "gift",
				PromotionHint: "vip",
				UpdatedAt:     updatedAt.Add(time.Minute),
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	body := fmt.Sprintf(`{"currency":"usd","shipping_address_id":"addr-1","billing_address_id":null,"notes":"  gift ","promotion_hint":"vip","updated_at":"%s"}`, updatedAt.Format(time.RFC3339))
	req := httptest.NewRequest(http.MethodPatch, "/cart", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-22"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	var resp cartResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Cart.Currency != "USD" {
		t.Fatalf("expected currency USD, got %s", resp.Cart.Currency)
	}
	if resp.Cart.Notes != "gift" {
		t.Fatalf("expected notes gift, got %q", resp.Cart.Notes)
	}
	if resp.Cart.PromotionHint != "vip" {
		t.Fatalf("expected promotion hint vip, got %q", resp.Cart.PromotionHint)
	}
}

func TestCartHandlersPatchCartInvalidBody(t *testing.T) {
	handler := NewCartHandlers(nil, &stubCartService{})
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	req := httptest.NewRequest(http.MethodPatch, "/cart", strings.NewReader(`{"updated_at":"2024-01-01T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
}

func TestCartHandlersPatchCartConflict(t *testing.T) {
	service := &stubCartService{
		updateFunc: func(ctx context.Context, cmd services.UpdateCartCommand) (services.Cart, error) {
			return services.Cart{}, services.ErrCartConflict
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	req := httptest.NewRequest(http.MethodPatch, "/cart", strings.NewReader(`{"currency":"usd","updated_at":"2024-01-01T00:00:00Z"}`))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-9"}))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", rr.Code)
	}
}

func TestCartHandlersListItemsSuccess(t *testing.T) {
	service := &stubCartService{
		getOrCreateFunc: func(ctx context.Context, userID string) (services.Cart, error) {
			return services.Cart{
				ID:       "cart-1",
				UserID:   userID,
				Currency: "JPY",
				Items: []services.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 1, UnitPrice: 500, Currency: "JPY"},
					{ID: "item-2", ProductID: "prod-2", SKU: "SKU-2", Quantity: 2, UnitPrice: 800, Currency: "JPY"},
				},
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/cart/items", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload cartItemsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(payload.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(payload.Items))
	}
}

func TestCartHandlersCreateItemSuccess(t *testing.T) {
	var captured services.UpsertCartItemCommand
	service := &stubCartService{
		addOrUpdateFunc: func(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error) {
			captured = cmd
			return services.Cart{
				ID:       "cart-1",
				UserID:   cmd.UserID,
				Currency: "JPY",
				Items:    []services.CartItem{{ID: "item-1", ProductID: cmd.ProductID, SKU: cmd.SKU, Quantity: cmd.Quantity, UnitPrice: cmd.UnitPrice, Currency: "JPY"}},
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	body := `{"product_id":"prod-1","sku":"SKU-1","quantity":2,"unit_price":500,"currency":"JPY","customization":{"color":"red"}}`
	req := httptest.NewRequest(http.MethodPost, "/cart/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-1"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", resp.Code)
	}
	if captured.UserID != "user-1" || captured.ProductID != "prod-1" || captured.Quantity != 2 {
		t.Fatalf("unexpected command captured %#v", captured)
	}
	if captured.Customization["color"] != "red" {
		t.Fatalf("expected customization captured")
	}
}

func TestCartHandlersUpdateItemSuccess(t *testing.T) {
	var captured services.UpsertCartItemCommand
	service := &stubCartService{
		addOrUpdateFunc: func(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error) {
			captured = cmd
			return services.Cart{ID: "cart-1", UserID: cmd.UserID, Currency: "JPY", Items: []services.CartItem{{ID: "item-9", ProductID: cmd.ProductID, SKU: cmd.SKU, Quantity: cmd.Quantity, UnitPrice: cmd.UnitPrice, Currency: "JPY"}}}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	body := `{"product_id":"prod-9","sku":"SKU-9","quantity":5,"unit_price":900}`
	req := httptest.NewRequest(http.MethodPut, "/cart/items/item-9", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-9"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if captured.ItemID == nil || *captured.ItemID != "item-9" {
		t.Fatalf("expected item id captured, got %#v", captured.ItemID)
	}
	if captured.Quantity != 5 {
		t.Fatalf("expected quantity 5, got %d", captured.Quantity)
	}
}

func TestCartHandlersDeleteItemSuccess(t *testing.T) {
	var captured services.RemoveCartItemCommand
	service := &stubCartService{
		removeFunc: func(ctx context.Context, cmd services.RemoveCartItemCommand) (services.Cart, error) {
			captured = cmd
			return services.Cart{ID: cmd.UserID, UserID: cmd.UserID, Currency: "JPY", Items: []services.CartItem{}}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Route("/cart", handler.Routes)

	req := httptest.NewRequest(http.MethodDelete, "/cart/items/item-3", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-3"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if captured.ItemID != "item-3" {
		t.Fatalf("expected captured item id item-3, got %s", captured.ItemID)
	}
}

type stubCartService struct {
	getOrCreateFunc func(ctx context.Context, userID string) (services.Cart, error)
	updateFunc      func(ctx context.Context, cmd services.UpdateCartCommand) (services.Cart, error)
	addOrUpdateFunc func(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error)
	removeFunc      func(ctx context.Context, cmd services.RemoveCartItemCommand) (services.Cart, error)
}

func (s *stubCartService) GetOrCreateCart(ctx context.Context, userID string) (services.Cart, error) {
	if s.getOrCreateFunc != nil {
		return s.getOrCreateFunc(ctx, userID)
	}
	return services.Cart{}, services.ErrCartUnavailable
}

func (s *stubCartService) UpdateCart(ctx context.Context, cmd services.UpdateCartCommand) (services.Cart, error) {
	if s.updateFunc != nil {
		return s.updateFunc(ctx, cmd)
	}
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) AddOrUpdateItem(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error) {
	if s.addOrUpdateFunc != nil {
		return s.addOrUpdateFunc(ctx, cmd)
	}
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) RemoveItem(ctx context.Context, cmd services.RemoveCartItemCommand) (services.Cart, error) {
	if s.removeFunc != nil {
		return s.removeFunc(ctx, cmd)
	}
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
