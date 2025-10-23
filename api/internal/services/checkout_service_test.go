package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/payments"
)

func TestCheckoutServiceCreateSessionSuccess(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 10, 9, 30, 0, 0, time.UTC)
	cartUpdated := now.Add(-10 * time.Minute)

	cart := domain.Cart{
		ID:                "cart-user-1",
		UserID:            "user-1",
		Currency:          "jpy",
		ShippingAddressID: "addr-1",
		Items: []domain.CartItem{
			{
				ID:               "item-1",
				ProductID:        "prod-1",
				SKU:              "SKU-1",
				Quantity:         2,
				UnitPrice:        1500,
				Currency:         "JPY",
				RequiresShipping: true,
				Metadata:         map[string]any{"name": "Ribbon", "description": "Red ribbon"},
			},
		},
		Estimate: &domain.CartEstimate{
			Subtotal: 3000,
			Discount: 0,
			Tax:      300,
			Shipping: 0,
			Total:    3300,
		},
		UpdatedAt: cartUpdated,
		CreatedAt: cartUpdated.Add(-time.Hour),
		Metadata:  map[string]any{},
	}

	var saved domain.Cart
	cartRepo := &stubCartRepository{
		getFunc: func(context.Context, string) (domain.Cart, error) {
			return cart, nil
		},
		upsertFunc: func(ctx context.Context, c domain.Cart, expected *time.Time) (domain.Cart, error) {
			saved = c
			if expected == nil || !expected.Equal(cartUpdated) {
				t.Fatalf("expected optimistic lock %v, got %v", cartUpdated, expected)
			}
			return c, nil
		},
	}

	var reservedCmd InventoryReserveCommand
	inventory := &stubCheckoutInventory{
		reserveFunc: func(ctx context.Context, cmd InventoryReserveCommand) (InventoryReservation, error) {
			reservedCmd = cmd
			if len(cmd.Lines) != 1 {
				t.Fatalf("expected 1 inventory line, got %d", len(cmd.Lines))
			}
			if cmd.Lines[0].Quantity != 2 || cmd.Lines[0].SKU != "SKU-1" {
				t.Fatalf("unexpected lines %#v", cmd.Lines)
			}
			return InventoryReservation{
				ID:        "sr_123",
				UserRef:   "/users/user-1",
				ExpiresAt: now.Add(20 * time.Minute),
			}, nil
		},
	}

	var paymentReq payments.CheckoutSessionRequest
	paymentMgr := &stubCheckoutPayments{
		createFunc: func(ctx context.Context, pCtx payments.PaymentContext, req payments.CheckoutSessionRequest) (payments.CheckoutSession, error) {
			if !strings.EqualFold(pCtx.PreferredProvider, "stripe") {
				t.Fatalf("unexpected provider %s", pCtx.PreferredProvider)
			}
			paymentReq = req
			if req.Amount != 3300 {
				t.Fatalf("expected amount 3300, got %d", req.Amount)
			}
			if req.Currency != "JPY" {
				t.Fatalf("expected currency JPY, got %s", req.Currency)
			}
			if len(req.Items) != 1 {
				t.Fatalf("expected 1 line item, got %d", len(req.Items))
			}
			if req.Items[0].Amount != 3300 {
				t.Fatalf("expected line item amount 3300, got %d", req.Items[0].Amount)
			}
			if req.Items[0].Name != "Order" {
				t.Fatalf("expected aggregated line item name Order, got %s", req.Items[0].Name)
			}
			return payments.CheckoutSession{
				ID:           "sess_123",
				Provider:     "stripe",
				ClientSecret: "sec_abc",
				RedirectURL:  "https://checkout.example/sess_123",
				IntentID:     "pi_123",
				ExpiresAt:    now.Add(30 * time.Minute),
			}, nil
		},
	}

	service, err := NewCheckoutService(CheckoutServiceDeps{
		Carts:          cartRepo,
		Inventory:      inventory,
		Payments:       paymentMgr,
		Clock:          func() time.Time { return now },
		ReservationTTL: 25 * time.Minute,
	})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	session, err := service.CreateCheckoutSession(ctx, CreateCheckoutSessionCommand{
		UserID:     "user-1",
		CartID:     "cart-user-1",
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
		PSP:        "stripe",
		Metadata: map[string]string{
			"locale": "ja-JP",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if session.SessionID != "sess_123" {
		t.Fatalf("expected session ID sess_123, got %s", session.SessionID)
	}
	if !strings.EqualFold(session.PSP, "stripe") {
		t.Fatalf("expected provider stripe, got %s", session.PSP)
	}
	if session.ClientSecret != "sec_abc" {
		t.Fatalf("expected client secret stored")
	}
	if reservedCmd.Reason != checkoutReservationReason {
		t.Fatalf("expected reservation reason checkout, got %s", reservedCmd.Reason)
	}
	if paymentReq.Metadata["reservation_id"] != "sr_123" {
		t.Fatalf("expected payment metadata reservation id, got %#v", paymentReq.Metadata)
	}

	meta, ok := saved.Metadata["checkout"].(map[string]any)
	if !ok {
		t.Fatalf("expected checkout metadata stored, got %#v", saved.Metadata)
	}
	if meta["sessionId"] != "sess_123" {
		t.Fatalf("expected sessionId stored, got %#v", meta["sessionId"])
	}
	if meta["reservationId"] != "sr_123" {
		t.Fatalf("expected reservationId stored, got %#v", meta["reservationId"])
	}
	if meta["intentId"] != "pi_123" {
		t.Fatalf("expected intentId stored, got %#v", meta["intentId"])
	}
}

func TestCheckoutServiceCreateSessionUsesLineItemsWhenTotalsMatch(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 11, 9, 0, 0, 0, time.UTC)
	cart := domain.Cart{
		ID:                "cart-user-2",
		UserID:            "user-2",
		Currency:          "JPY",
		ShippingAddressID: "addr-2",
		Items: []domain.CartItem{
			{
				ID:               "item-1",
				ProductID:        "prod-1",
				SKU:              "SKU-1",
				Quantity:         2,
				UnitPrice:        1500,
				Currency:         "JPY",
				RequiresShipping: true,
				Metadata:         map[string]any{"name": "Ribbon"},
			},
		},
		Estimate: &domain.CartEstimate{
			Subtotal: 3000,
			Discount: 0,
			Tax:      0,
			Shipping: 0,
			Total:    3000,
		},
		UpdatedAt: now.Add(-time.Minute),
		Metadata:  map[string]any{},
	}

	cartRepo := &stubCartRepository{
		getFunc: func(context.Context, string) (domain.Cart, error) {
			return cart, nil
		},
		upsertFunc: func(ctx context.Context, c domain.Cart, expected *time.Time) (domain.Cart, error) {
			return c, nil
		},
	}

	inventory := &stubCheckoutInventory{
		reserveFunc: func(context.Context, InventoryReserveCommand) (InventoryReservation, error) {
			return InventoryReservation{ID: "sr_match"}, nil
		},
	}

	var paymentReq payments.CheckoutSessionRequest
	paymentMgr := &stubCheckoutPayments{
		createFunc: func(_ context.Context, _ payments.PaymentContext, req payments.CheckoutSessionRequest) (payments.CheckoutSession, error) {
			paymentReq = req
			return payments.CheckoutSession{
				ID:        "sess_match",
				Provider:  "stripe",
				ExpiresAt: now.Add(30 * time.Minute),
			}, nil
		},
	}

	service, err := NewCheckoutService(CheckoutServiceDeps{
		Carts:          cartRepo,
		Inventory:      inventory,
		Payments:       paymentMgr,
		Clock:          func() time.Time { return now },
		ReservationTTL: 20 * time.Minute,
	})
	if err != nil {
		t.Fatalf("unexpected error creating service: %v", err)
	}

	_, err = service.CreateCheckoutSession(ctx, CreateCheckoutSessionCommand{
		UserID:     "user-2",
		CartID:     "cart-user-2",
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
		PSP:        "stripe",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(paymentReq.Items) != 1 {
		t.Fatalf("expected detailed line items, got %d", len(paymentReq.Items))
	}
	item := paymentReq.Items[0]
	if item.Amount != 1500 {
		t.Fatalf("expected unit amount 1500, got %d", item.Amount)
	}
	if item.Quantity != 2 {
		t.Fatalf("expected quantity 2, got %d", item.Quantity)
	}
	if item.Name != "Ribbon" {
		t.Fatalf("expected item name Ribbon, got %s", item.Name)
	}
}
func TestCheckoutServiceCreateSessionCartNotReady(t *testing.T) {
	cartRepo := &stubCartRepository{
		getFunc: func(context.Context, string) (domain.Cart, error) {
			return domain.Cart{
				ID:       "cart-1",
				UserID:   "user-1",
				Currency: "JPY",
				Items: []domain.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 1, UnitPrice: 1200, RequiresShipping: true},
				},
				Estimate:  nil,
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	service, err := NewCheckoutService(CheckoutServiceDeps{
		Carts:          cartRepo,
		Inventory:      &stubCheckoutInventory{},
		Payments:       &stubCheckoutPayments{},
		Clock:          time.Now,
		ReservationTTL: 15 * time.Minute,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = service.CreateCheckoutSession(context.Background(), CreateCheckoutSessionCommand{
		UserID:     "user-1",
		CartID:     "cart-1",
		SuccessURL: "https://example/success",
		CancelURL:  "https://example/cancel",
	})
	if !errors.Is(err, ErrCheckoutCartNotReady) {
		t.Fatalf("expected cart not ready error, got %v", err)
	}
}

func TestCheckoutServiceCreateSessionInsufficientStock(t *testing.T) {
	cart := domain.Cart{
		ID:                "cart-1",
		UserID:            "user-1",
		Currency:          "JPY",
		ShippingAddressID: "addr-1",
		Items: []domain.CartItem{
			{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 1, UnitPrice: 1000, RequiresShipping: true},
		},
		Estimate:  &domain.CartEstimate{Total: 1000},
		UpdatedAt: time.Now(),
	}
	cartRepo := &stubCartRepository{
		getFunc: func(context.Context, string) (domain.Cart, error) {
			return cart, nil
		},
	}

	inventory := &stubCheckoutInventory{
		reserveFunc: func(context.Context, InventoryReserveCommand) (InventoryReservation, error) {
			return InventoryReservation{}, ErrInventoryInsufficientStock
		},
	}

	service, err := NewCheckoutService(CheckoutServiceDeps{
		Carts:          cartRepo,
		Inventory:      inventory,
		Payments:       &stubCheckoutPayments{},
		Clock:          time.Now,
		ReservationTTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = service.CreateCheckoutSession(context.Background(), CreateCheckoutSessionCommand{
		UserID:     "user-1",
		CartID:     "cart-1",
		SuccessURL: "https://example/success",
		CancelURL:  "https://example/cancel",
	})
	if !errors.Is(err, ErrCheckoutInsufficientStock) {
		t.Fatalf("expected insufficient stock error, got %v", err)
	}
}

func TestCheckoutServiceCreatesSessionReleasesReservationOnPaymentFailure(t *testing.T) {
	now := time.Now()
	cart := domain.Cart{
		ID:                "cart-1",
		UserID:            "user-1",
		Currency:          "JPY",
		ShippingAddressID: "addr-1",
		Items: []domain.CartItem{
			{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 1, UnitPrice: 1000, RequiresShipping: true},
		},
		Estimate:  &domain.CartEstimate{Total: 1000},
		UpdatedAt: now,
	}

	cartRepo := &stubCartRepository{
		getFunc: func(context.Context, string) (domain.Cart, error) {
			return cart, nil
		},
		upsertFunc: func(ctx context.Context, c domain.Cart, expected *time.Time) (domain.Cart, error) {
			return c, nil
		},
	}

	released := ""
	releasedReason := ""

	inventory := &stubCheckoutInventory{
		reserveFunc: func(context.Context, InventoryReserveCommand) (InventoryReservation, error) {
			return InventoryReservation{ID: "sr_123"}, nil
		},
		releaseFunc: func(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error) {
			released = cmd.ReservationID
			releasedReason = cmd.Reason
			return InventoryReservation{}, nil
		},
	}

	service, err := NewCheckoutService(CheckoutServiceDeps{
		Carts:     cartRepo,
		Inventory: inventory,
		Payments: &stubCheckoutPayments{
			createFunc: func(context.Context, payments.PaymentContext, payments.CheckoutSessionRequest) (payments.CheckoutSession, error) {
				return payments.CheckoutSession{}, errors.New("psp timeout")
			},
		},
		Clock:          time.Now,
		ReservationTTL: 10 * time.Minute,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = service.CreateCheckoutSession(context.Background(), CreateCheckoutSessionCommand{
		UserID:     "user-1",
		CartID:     "cart-1",
		SuccessURL: "https://example/success",
		CancelURL:  "https://example/cancel",
	})
	if !errors.Is(err, ErrCheckoutPaymentFailed) {
		t.Fatalf("expected payment failed error, got %v", err)
	}
	if released != "sr_123" {
		t.Fatalf("expected reservation release, got %s", released)
	}
	if releasedReason != checkoutReleaseReasonPaymentFail {
		t.Fatalf("expected release reason payment failure, got %s", releasedReason)
	}
}

type stubCheckoutInventory struct {
	reserveFunc func(ctx context.Context, cmd InventoryReserveCommand) (InventoryReservation, error)
	releaseFunc func(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error)
}

func (s *stubCheckoutInventory) ReserveStocks(ctx context.Context, cmd InventoryReserveCommand) (InventoryReservation, error) {
	if s.reserveFunc != nil {
		return s.reserveFunc(ctx, cmd)
	}
	return InventoryReservation{}, nil
}

func (s *stubCheckoutInventory) CommitReservation(context.Context, InventoryCommitCommand) (InventoryReservation, error) {
	return InventoryReservation{}, errors.New("not implemented")
}

func (s *stubCheckoutInventory) ReleaseReservation(ctx context.Context, cmd InventoryReleaseCommand) (InventoryReservation, error) {
	if s.releaseFunc != nil {
		return s.releaseFunc(ctx, cmd)
	}
	return InventoryReservation{}, nil
}

func (s *stubCheckoutInventory) ListLowStock(context.Context, InventoryLowStockFilter) (domain.CursorPage[InventorySnapshot], error) {
	return domain.CursorPage[InventorySnapshot]{}, errors.New("not implemented")
}

type stubCheckoutPayments struct {
	createFunc func(ctx context.Context, paymentCtx payments.PaymentContext, req payments.CheckoutSessionRequest) (payments.CheckoutSession, error)
}

func (s *stubCheckoutPayments) CreateCheckoutSession(ctx context.Context, paymentCtx payments.PaymentContext, req payments.CheckoutSessionRequest) (payments.CheckoutSession, error) {
	if s.createFunc != nil {
		return s.createFunc(ctx, paymentCtx, req)
	}
	return payments.CheckoutSession{}, errors.New("not implemented")
}
