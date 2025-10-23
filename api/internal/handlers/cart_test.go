package handlers

import (
	"bytes"
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
	router.Post("/cart:estimate", handler.estimateCart)

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
	router.Post("/cart:estimate", handler.estimateCart)

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

func TestCartHandlersEstimateSuccess(t *testing.T) {
	var captured services.CartEstimateCommand
	service := &stubCartService{
		estimateFunc: func(ctx context.Context, cmd services.CartEstimateCommand) (services.CartEstimateResult, error) {
			captured = cmd
			return services.CartEstimateResult{
				Currency:  "JPY",
				Estimate:  services.CartEstimate{Subtotal: 5000, Discount: 200, Tax: 300, Shipping: 400, Total: 5500},
				Promotion: &services.CartPromotion{Code: "SAVE20", DiscountAmount: 200, Applied: true},
				Breakdown: services.PricingBreakdown{
					Currency: "JPY",
					Subtotal: 5000,
					Discount: 200,
					Tax:      300,
					Shipping: 400,
					Total:    5500,
					Items: []services.ItemPricingBreakdown{{
						ItemID:   "item-1",
						Currency: "JPY",
						Subtotal: 5000,
						Discount: 200,
						Tax:      300,
						Shipping: 400,
						Total:    5500,
					}},
					Discounts: []services.DiscountBreakdown{{Type: "promotion", Amount: 200}},
				},
				Warnings: []services.CartEstimateWarning{services.CartEstimateWarningMissingShippingAddress},
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := NewRouter(WithCartRoutes(handler.Routes), WithAdditionalRoutes(handler.RegisterStandaloneRoutes))

	body := bytes.NewBufferString(`{"shipping_address_id":"addr-1","promotion_code":"save20","bypass_shipping_cache":true}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart:estimate", body)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-estimate"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	if captured.UserID != "user-estimate" {
		t.Fatalf("expected user id captured, got %s", captured.UserID)
	}
	if !captured.BypassShippingCache {
		t.Fatalf("expected bypass shipping cache true")
	}
	if captured.ShippingAddressID == nil || *captured.ShippingAddressID != "addr-1" {
		t.Fatalf("expected shipping address override, got %#v", captured.ShippingAddressID)
	}
	if captured.PromotionCode == nil || *captured.PromotionCode != "save20" {
		t.Fatalf("expected promotion code captured, got %#v", captured.PromotionCode)
	}

	var payload cartEstimateResultPayload
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Currency != "JPY" {
		t.Fatalf("expected currency JPY, got %s", payload.Currency)
	}
	if payload.Total != 5500 {
		t.Fatalf("expected total 5500, got %d", payload.Total)
	}
	if payload.Promotion == nil || !payload.Promotion.Applied {
		t.Fatalf("expected promotion applied in payload, got %#v", payload.Promotion)
	}
	if len(payload.Items) != 1 {
		t.Fatalf("expected one item breakdown, got %d", len(payload.Items))
	}
	if len(payload.Warnings) != 1 || payload.Warnings[0] != string(services.CartEstimateWarningMissingShippingAddress) {
		t.Fatalf("expected missing shipping warning, got %#v", payload.Warnings)
	}
}

func TestCartHandlersEstimateServiceError(t *testing.T) {
	service := &stubCartService{
		estimateFunc: func(ctx context.Context, cmd services.CartEstimateCommand) (services.CartEstimateResult, error) {
			return services.CartEstimateResult{}, services.ErrCartInvalidInput
		},
	}

	handler := NewCartHandlers(nil, service)
	router := NewRouter(WithCartRoutes(handler.Routes), WithAdditionalRoutes(handler.RegisterStandaloneRoutes))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/cart:estimate", bytes.NewBufferString(`{"promotion_code":"bad"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-estimate"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCartHandlersApplyPromotionSuccess(t *testing.T) {
	now := time.Date(2024, 7, 10, 12, 0, 0, 0, time.UTC)
	var captured services.CartPromotionCommand
	service := &stubCartService{
		applyPromoFunc: func(ctx context.Context, cmd services.CartPromotionCommand) (services.Cart, error) {
			captured = cmd
			return services.Cart{
				ID:       "cart-apply",
				UserID:   cmd.UserID,
				Currency: "JPY",
				Promotion: &services.CartPromotion{
					Code:           "SPRING10",
					DiscountAmount: 500,
					Applied:        true,
				},
				Estimate:  &services.CartEstimate{Subtotal: 2000, Discount: 500, Tax: 0, Shipping: 0, Total: 1500},
				UpdatedAt: now,
			}, nil
		},
	}

	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Post("/cart:apply-promo", handler.applyPromotion)

	req := httptest.NewRequest(http.MethodPost, "/cart:apply-promo", strings.NewReader(`{"code":" spring10 ","source":" referral "}`))
	req.Header.Set("Idempotency-Key", " idem-apply ")
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-promo"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}

	var payload cartResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Cart.Promotion == nil || payload.Cart.Promotion.Code != "SPRING10" {
		t.Fatalf("expected promotion SPRING10, got %#v", payload.Cart.Promotion)
	}
	if captured.UserID != "user-promo" {
		t.Fatalf("expected user id user-promo, got %s", captured.UserID)
	}
	if captured.Code != "spring10" {
		t.Fatalf("expected trimmed code spring10, got %s", captured.Code)
	}
	if captured.Source != "referral" {
		t.Fatalf("expected source referral, got %s", captured.Source)
	}
	if captured.IdempotencyKey != "idem-apply" {
		t.Fatalf("expected idempotency key idem-apply, got %s", captured.IdempotencyKey)
	}
}

func TestCartHandlersApplyPromotionInvalidRequest(t *testing.T) {
	handler := NewCartHandlers(nil, &stubCartService{})
	router := chi.NewRouter()
	router.Post("/cart:apply-promo", handler.applyPromotion)

	req := httptest.NewRequest(http.MethodPost, "/cart:apply-promo", strings.NewReader(`{"code":"   "}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-promo"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCartHandlersApplyPromotionServiceError(t *testing.T) {
	service := &stubCartService{
		applyPromoFunc: func(ctx context.Context, cmd services.CartPromotionCommand) (services.Cart, error) {
			return services.Cart{}, services.ErrCartInvalidInput
		},
	}
	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Post("/cart:apply-promo", handler.applyPromotion)

	req := httptest.NewRequest(http.MethodPost, "/cart:apply-promo", strings.NewReader(`{"code":"spring10"}`))
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-promo"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.Code)
	}
}

func TestCartHandlersRemovePromotionSuccess(t *testing.T) {
	now := time.Date(2024, 7, 11, 10, 0, 0, 0, time.UTC)
	service := &stubCartService{
		removePromoFunc: func(ctx context.Context, userID string) (services.Cart, error) {
			if userID != "user-remove" {
				t.Fatalf("expected user id user-remove, got %s", userID)
			}
			return services.Cart{
				ID:        "cart-remove",
				UserID:    userID,
				Currency:  "JPY",
				Estimate:  &services.CartEstimate{Subtotal: 2000, Discount: 0, Tax: 0, Shipping: 0, Total: 2000},
				UpdatedAt: now,
			}, nil
		},
	}
	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Delete("/cart:remove-promo", handler.removePromotion)

	req := httptest.NewRequest(http.MethodDelete, "/cart:remove-promo", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-remove"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.Code)
	}
	var payload cartResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.Cart.Promotion != nil {
		t.Fatalf("expected promotion cleared, got %#v", payload.Cart.Promotion)
	}
}

func TestCartHandlersRemovePromotionNotFound(t *testing.T) {
	service := &stubCartService{
		removePromoFunc: func(ctx context.Context, userID string) (services.Cart, error) {
			return services.Cart{}, services.ErrCartNotFound
		},
	}
	handler := NewCartHandlers(nil, service)
	router := chi.NewRouter()
	router.Delete("/cart:remove-promo", handler.removePromotion)

	req := httptest.NewRequest(http.MethodDelete, "/cart:remove-promo", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), &auth.Identity{UID: "user-remove"}))
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", resp.Code)
	}
}

type stubCartService struct {
	getOrCreateFunc func(ctx context.Context, userID string) (services.Cart, error)
	updateFunc      func(ctx context.Context, cmd services.UpdateCartCommand) (services.Cart, error)
	addOrUpdateFunc func(ctx context.Context, cmd services.UpsertCartItemCommand) (services.Cart, error)
	removeFunc      func(ctx context.Context, cmd services.RemoveCartItemCommand) (services.Cart, error)
	estimateFunc    func(ctx context.Context, cmd services.CartEstimateCommand) (services.CartEstimateResult, error)
	applyPromoFunc  func(ctx context.Context, cmd services.CartPromotionCommand) (services.Cart, error)
	removePromoFunc func(ctx context.Context, userID string) (services.Cart, error)
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

func (s *stubCartService) Estimate(ctx context.Context, cmd services.CartEstimateCommand) (services.CartEstimateResult, error) {
	if s.estimateFunc != nil {
		return s.estimateFunc(ctx, cmd)
	}
	return services.CartEstimateResult{}, errors.New("not implemented")
}

func (s *stubCartService) ApplyPromotion(ctx context.Context, cmd services.CartPromotionCommand) (services.Cart, error) {
	if s.applyPromoFunc != nil {
		return s.applyPromoFunc(ctx, cmd)
	}
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) RemovePromotion(ctx context.Context, userID string) (services.Cart, error) {
	if s.removePromoFunc != nil {
		return s.removePromoFunc(ctx, userID)
	}
	return services.Cart{}, errors.New("not implemented")
}

func (s *stubCartService) ClearCart(ctx context.Context, userID string) error {
	return errors.New("not implemented")
}
