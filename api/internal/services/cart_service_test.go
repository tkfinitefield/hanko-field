package services

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

func strPtr(v string) *string {
	return &v
}

func TestCartServiceGetOrCreateCartReturnsExisting(t *testing.T) {
	now := time.Date(2024, 5, 10, 12, 0, 0, 0, time.UTC)
	estimate := CartEstimate{Subtotal: 2000, Total: 2000}

	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			if userID != "user-123" {
				t.Fatalf("unexpected user id %q", userID)
			}
			return domain.Cart{
				ID:       "cart-user-123",
				UserID:   "user-123",
				Currency: "jpy",
				Items: []domain.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 2, UnitPrice: 500},
				},
				Estimate:  &domain.CartEstimate{Subtotal: 999, Total: 999},
				UpdatedAt: now.Add(-time.Hour),
			}, nil
		},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if len(cmd.Cart.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(cmd.Cart.Items))
			}
			return PriceCartResult{Estimate: estimate}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Pricer:          pricer,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	ctx := context.Background()
	cart, err := service.GetOrCreateCart(ctx, " user-123 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cart.ID != "cart-user-123" {
		t.Fatalf("expected cart id cart-user-123, got %q", cart.ID)
	}
	if cart.Currency != "JPY" {
		t.Fatalf("expected currency uppercased JPY, got %q", cart.Currency)
	}
	if cart.Estimate == nil {
		t.Fatalf("expected estimate")
	}
	if cart.Estimate.Total != estimate.Total {
		t.Fatalf("expected estimate total %d, got %d", estimate.Total, cart.Estimate.Total)
	}
}

func TestCartServiceGetOrCreateCartLazyCreates(t *testing.T) {
	now := time.Date(2024, 5, 11, 9, 30, 0, 0, time.UTC)
	var upserted domain.Cart

	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{}, &repositoryErrorStub{notFound: true}
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			upserted = cart
			return cart, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "usd",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	ctx := context.Background()
	cart, err := service.GetOrCreateCart(ctx, "guest-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if upserted.ID != "guest-5" {
		t.Fatalf("expected upserted cart id guest-5, got %q", upserted.ID)
	}
	if cart.ID != "guest-5" {
		t.Fatalf("expected cart id guest-5, got %q", cart.ID)
	}
	if cart.Currency != "USD" {
		t.Fatalf("expected default currency USD, got %q", cart.Currency)
	}
	if len(cart.Items) != 0 {
		t.Fatalf("expected empty items")
	}
	if cart.Estimate == nil {
		t.Fatalf("expected estimate fallback")
	}
	if cart.Estimate.Total != 0 {
		t.Fatalf("expected zero total")
	}
	if cart.UpdatedAt.IsZero() {
		t.Fatalf("expected updated at set")
	}
}

func TestCartServiceGetOrCreateCartInvalidUser(t *testing.T) {
	service, err := NewCartService(CartServiceDeps{
		Repository:      &stubCartRepository{},
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.GetOrCreateCart(context.Background(), "  ")
	if !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
}

func TestCartServiceGetOrCreateCartPricingError(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:       userID,
				UserID:   userID,
				Currency: "JPY",
				Items: []domain.CartItem{
					{ID: "item", SKU: "SKU", ProductID: "prod", Quantity: 1, UnitPrice: 100},
				},
			}, nil
		},
	}
	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			return PriceCartResult{}, ErrCartPricingInvalidInput
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository: repo,
		Pricer:     pricer,
		Clock:      time.Now,
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.GetOrCreateCart(context.Background(), "user-1")
	if !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
}

func TestCartServiceUpdateCartCurrencyAndNotes(t *testing.T) {
	now := time.Date(2024, 6, 1, 8, 0, 0, 0, time.UTC)
	existingUpdated := now.Add(-time.Minute * 5)
	createdAt := now.Add(-time.Hour)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:        "cart-1",
				UserID:    "user-1",
				Currency:  "JPY",
				Notes:     "old",
				Items:     []domain.CartItem{{ID: "item-1", ProductID: "prod-1", SKU: "SKU-1", Quantity: 1, UnitPrice: 500}},
				Estimate:  &domain.CartEstimate{Subtotal: 500, Total: 500},
				CreatedAt: createdAt,
				UpdatedAt: existingUpdated,
			}, nil
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			if expected == nil {
				t.Fatalf("expected optimistic lock timestamp")
			}
			if !expected.Equal(existingUpdated.UTC()) {
				t.Fatalf("unexpected expected timestamp %v", expected)
			}
			if cart.Currency != "USD" {
				t.Fatalf("expected currency USD got %s", cart.Currency)
			}
			if cart.Notes != "gift" {
				t.Fatalf("expected notes trimmed to gift got %q", cart.Notes)
			}
			if cart.PromotionHint != "welcome" {
				t.Fatalf("expected promotion hint welcome got %q", cart.PromotionHint)
			}
			cart.UpdatedAt = now
			return cart, nil
		},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if cmd.Cart.Currency != "USD" {
				t.Fatalf("expected pricing with updated currency, got %s", cmd.Cart.Currency)
			}
			return PriceCartResult{Estimate: CartEstimate{Subtotal: 500, Total: 500}}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Pricer:          pricer,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cmd := UpdateCartCommand{
		UserID:            "user-1",
		Currency:          strPtr("usd"),
		Notes:             strPtr("  gift "),
		PromotionHint:     strPtr(" welcome "),
		ExpectedUpdatedAt: &existingUpdated,
	}

	updated, err := service.UpdateCart(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error updating cart: %v", err)
	}
	if updated.Currency != "USD" {
		t.Fatalf("expected currency USD got %s", updated.Currency)
	}
	if updated.Notes != "gift" {
		t.Fatalf("expected trimmed notes gift got %q", updated.Notes)
	}
	if updated.PromotionHint != "welcome" {
		t.Fatalf("expected promotion hint welcome got %q", updated.PromotionHint)
	}
	if updated.UpdatedAt != now {
		t.Fatalf("expected updated at %v got %v", now, updated.UpdatedAt)
	}
}

func TestCartServiceUpdateCartAddresses(t *testing.T) {
	now := time.Date(2024, 6, 2, 9, 0, 0, 0, time.UTC)
	existingUpdated := now.Add(-time.Minute * 2)
	address := Address{ID: "addr-1", Recipient: "Foo", Line1: "1-2-3", City: "Tokyo", PostalCode: "100-0001", Country: "JP"}
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:        "cart-1",
				UserID:    "user-1",
				Currency:  "JPY",
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: existingUpdated,
			}, nil
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			cart.UpdatedAt = now
			return cart, nil
		},
	}
	addresses := &stubAddressProvider{
		listFunc: func(ctx context.Context, userID string) ([]Address, error) {
			if userID != "user-1" {
				t.Fatalf("unexpected user id %s", userID)
			}
			return []Address{address}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
		Addresses:       addresses,
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cmd := UpdateCartCommand{
		UserID:            "user-1",
		ShippingAddressID: strPtr("addr-1"),
		ExpectedUpdatedAt: &existingUpdated,
	}

	updated, err := service.UpdateCart(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error updating cart: %v", err)
	}
	if updated.ShippingAddress == nil || updated.ShippingAddress.ID != "addr-1" {
		t.Fatalf("expected shipping address assigned, got %#v", updated.ShippingAddress)
	}

	cmdInvalid := UpdateCartCommand{
		UserID:            "user-1",
		ShippingAddressID: strPtr("missing"),
		ExpectedUpdatedAt: &existingUpdated,
	}

	_, err = service.UpdateCart(context.Background(), cmdInvalid)
	if !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput got %v", err)
	}
}

func TestCartServiceUpdateCartConflictingTimestamp(t *testing.T) {
	now := time.Now().UTC()
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:        "cart",
				UserID:    userID,
				Currency:  "JPY",
				UpdatedAt: now,
			}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository: repo,
		Clock:      time.Now,
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cmd := UpdateCartCommand{UserID: "user-1", Currency: strPtr("eur")}
	_, err = service.UpdateCart(context.Background(), cmd)
	if !errors.Is(err, ErrCartConflict) {
		t.Fatalf("expected ErrCartConflict got %v", err)
	}
}

type stubCartRepository struct {
	getFunc     func(ctx context.Context, userID string) (domain.Cart, error)
	upsertFunc  func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error)
	replaceFunc func(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error)
}

func (s *stubCartRepository) GetCart(ctx context.Context, userID string) (domain.Cart, error) {
	if s.getFunc != nil {
		return s.getFunc(ctx, userID)
	}
	return domain.Cart{}, errors.New("not implemented")
}

func (s *stubCartRepository) UpsertCart(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
	if s.upsertFunc != nil {
		return s.upsertFunc(ctx, cart, expected)
	}
	return cart, nil
}

func (s *stubCartRepository) ReplaceItems(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
	if s.replaceFunc != nil {
		return s.replaceFunc(ctx, userID, items)
	}
	return domain.Cart{}, errors.New("not implemented")
}

type stubCartPricer struct {
	calculateFunc func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error)
}

func (s *stubCartPricer) Calculate(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
	if s.calculateFunc != nil {
		return s.calculateFunc(ctx, cmd)
	}
	return PriceCartResult{}, nil
}

type repositoryErrorStub struct {
	notFound    bool
	conflict    bool
	unavailable bool
}

func (e *repositoryErrorStub) Error() string {
	return "repository error"
}

func (e *repositoryErrorStub) IsNotFound() bool {
	return e.notFound
}

func (e *repositoryErrorStub) IsConflict() bool {
	return e.conflict
}

func (e *repositoryErrorStub) IsUnavailable() bool {
	return e.unavailable
}

type stubAddressProvider struct {
	listFunc func(ctx context.Context, userID string) ([]Address, error)
}

func (s *stubAddressProvider) ListAddresses(ctx context.Context, userID string) ([]Address, error) {
	if s.listFunc != nil {
		return s.listFunc(ctx, userID)
	}
	return nil, nil
}
