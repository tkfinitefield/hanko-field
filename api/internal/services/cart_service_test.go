package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
)

func strPtr(v string) *string {
	return &v
}

func containsWarning(warnings []CartEstimateWarning, target CartEstimateWarning) bool {
	for _, w := range warnings {
		if w == target {
			return true
		}
	}
	return false
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

func TestCartServiceAddOrUpdateItemCreatesNew(t *testing.T) {
	now := time.Date(2024, 7, 5, 10, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: []domain.CartItem{}, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Minute)}, nil
		},
		replaceFunc: func(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
			if userID != "user-new" {
				t.Fatalf("unexpected user id %q", userID)
			}
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			if items[0].Quantity != 2 {
				t.Fatalf("expected quantity 2, got %d", items[0].Quantity)
			}
			if items[0].Customization["color"] != "red" {
				t.Fatalf("expected customization color red, got %#v", items[0].Customization)
			}
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: items, UpdatedAt: now.Add(time.Minute)}, nil
		},
	}

	invCalled := false
	inventory := &stubInventoryAvailability{
		validateFunc: func(ctx context.Context, lines []InventoryLine) error {
			invCalled = true
			if len(lines) != 1 {
				t.Fatalf("expected 1 inventory line, got %d", len(lines))
			}
			if lines[0].Quantity != 2 {
				t.Fatalf("expected quantity 2, got %d", lines[0].Quantity)
			}
			if lines[0].ProductID != "prod-1" {
				t.Fatalf("expected product prod-1, got %s", lines[0].ProductID)
			}
			return nil
		},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if len(cmd.Cart.Items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(cmd.Cart.Items))
			}
			return PriceCartResult{Estimate: CartEstimate{Subtotal: 1000, Total: 1000}}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Pricer:          pricer,
		Availability:    inventory,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
		IDGenerator:     func() string { return "item-generated" },
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cart, err := service.AddOrUpdateItem(context.Background(), UpsertCartItemCommand{
		UserID:        "user-new",
		ProductID:     "prod-1",
		SKU:           "SKU-1",
		Quantity:      2,
		UnitPrice:     500,
		Currency:      "JPY",
		Customization: map[string]any{"color": "red"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !invCalled {
		t.Fatalf("expected inventory validation to be invoked")
	}
	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].ID != "item-generated" {
		t.Fatalf("expected generated item id, got %s", cart.Items[0].ID)
	}
	if cart.Estimate == nil || cart.Estimate.Total != 1000 {
		t.Fatalf("expected estimate total 1000, got %#v", cart.Estimate)
	}
}

func TestCartServiceAddOrUpdateItemMerges(t *testing.T) {
	now := time.Date(2024, 7, 6, 9, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:       userID,
				UserID:   userID,
				Currency: "JPY",
				Items: []domain.CartItem{{
					ID:               "item-1",
					ProductID:        "prod-merge",
					SKU:              "SKU-M",
					Quantity:         1,
					UnitPrice:        500,
					Currency:         "JPY",
					RequiresShipping: true,
					Customization:    map[string]any{"size": "M"},
				}},
				UpdatedAt: now.Add(-time.Hour),
			}, nil
		},
		replaceFunc: func(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
			if len(items) != 1 {
				t.Fatalf("expected 1 item, got %d", len(items))
			}
			if items[0].Quantity != 4 {
				t.Fatalf("expected merged quantity 4, got %d", items[0].Quantity)
			}
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: items, UpdatedAt: now.Add(time.Minute)}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
		IDGenerator:     func() string { return "unused" },
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cart, err := service.AddOrUpdateItem(context.Background(), UpsertCartItemCommand{
		UserID:        "user-merge",
		ProductID:     "prod-merge",
		SKU:           "SKU-M",
		Quantity:      3,
		UnitPrice:     500,
		Currency:      "JPY",
		Customization: map[string]any{"size": "M"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cart.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(cart.Items))
	}
	if cart.Items[0].Quantity != 4 {
		t.Fatalf("expected merged quantity 4, got %d", cart.Items[0].Quantity)
	}
}

func TestCartServiceAddOrUpdateItemDesignOwnership(t *testing.T) {
	now := time.Date(2024, 7, 7, 12, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: []domain.CartItem{}}, nil
		},
	}

	designs := &stubDesignFinder{
		findFunc: func(ctx context.Context, designID string) (domain.Design, error) {
			return domain.Design{ID: designID, OwnerID: "other-user"}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Designs:         designs,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.AddOrUpdateItem(context.Background(), UpsertCartItemCommand{
		UserID:    "user-owner",
		ProductID: "prod",
		SKU:       "SKU",
		Quantity:  1,
		UnitPrice: 500,
		Currency:  "JPY",
		DesignID:  strPtr("design-1"),
	})
	if err == nil || !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
}

func TestCartServiceRemoveItem(t *testing.T) {
	now := time.Date(2024, 7, 8, 8, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: []domain.CartItem{{ID: "item-1", ProductID: "prod", SKU: "SKU", Quantity: 1, UnitPrice: 500, Currency: "JPY"}}}, nil
		},
		replaceFunc: func(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
			if len(items) != 0 {
				t.Fatalf("expected items cleared, got %d", len(items))
			}
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", Items: []domain.CartItem{}}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cart, err := service.RemoveItem(context.Background(), RemoveCartItemCommand{UserID: "user-rm", ItemID: "item-1"})
	if err != nil {
		t.Fatalf("unexpected error removing item: %v", err)
	}
	if len(cart.Items) != 0 {
		t.Fatalf("expected cart to be empty, got %d", len(cart.Items))
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

func TestCartServiceUpdateCartHeaderPrecision(t *testing.T) {
	now := time.Date(2024, 6, 3, 10, 0, 0, 900_000_000, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:        "cart-precision",
				UserID:    userID,
				Currency:  "JPY",
				UpdatedAt: now,
			}, nil
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			if expected == nil {
				t.Fatalf("expected optimistic lock timestamp")
			}
			if !expected.Equal(now.UTC()) {
				t.Fatalf("expected expected timestamp %v got %v", now, *expected)
			}
			cart.UpdatedAt = now.Add(time.Second)
			return cart, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository: repo,
		Clock:      func() time.Time { return now.Add(time.Second) },
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	truncated := now.Truncate(time.Second)
	cmd := UpdateCartCommand{
		UserID:             "user-1",
		Currency:           strPtr("usd"),
		ExpectedUpdatedAt:  &truncated,
		ExpectedFromHeader: true,
	}

	updated, err := service.UpdateCart(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error updating cart: %v", err)
	}
	if updated.Currency != "USD" {
		t.Fatalf("expected currency USD got %s", updated.Currency)
	}
}

func TestCartServiceEstimateSuccess(t *testing.T) {
	now := time.Date(2024, 8, 1, 12, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			if userID != "user-estimate" {
				t.Fatalf("unexpected user id %s", userID)
			}
			return domain.Cart{
				ID:                "cart-estimate",
				UserID:            userID,
				Currency:          "jpy",
				ShippingAddressID: "addr-1",
				Promotion:         &domain.CartPromotion{Code: "sakura10"},
				Items: []domain.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "sku-1", Quantity: 2, UnitPrice: 1500, Currency: "JPY", RequiresShipping: true},
				},
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			}, nil
		},
	}

	addrProvider := &stubAddressProvider{
		listFunc: func(ctx context.Context, userID string) ([]Address, error) {
			return []Address{{
				ID:         "addr-1",
				Recipient:  "Hanako",
				Line1:      "1-2-3",
				City:       "Tokyo",
				PostalCode: "1000001",
				Country:    "JP",
			}}, nil
		},
	}

	breakdown := PricingBreakdown{
		Currency:  "JPY",
		Subtotal:  3000,
		Discount:  300,
		Tax:       270,
		Shipping:  450,
		Total:     3420,
		Discounts: []DiscountBreakdown{{Type: "promotion", Amount: 300}},
		Items: []ItemPricingBreakdown{{
			ItemID:   "item-1",
			Currency: "JPY",
			Subtotal: 3000,
			Discount: 300,
			Tax:      270,
			Shipping: 450,
			Total:    3420,
		}},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if cmd.PromotionCode == nil || *cmd.PromotionCode != "SAKURA10" {
				t.Fatalf("expected promotion code SAKURA10, got %#v", cmd.PromotionCode)
			}
			if cmd.ShippingAddress == nil || strings.TrimSpace(cmd.ShippingAddress.ID) != "addr-1" {
				t.Fatalf("expected shipping address override, got %#v", cmd.ShippingAddress)
			}
			return PriceCartResult{
				Breakdown: breakdown,
				Estimate:  CartEstimate{Subtotal: 3000, Discount: 300, Tax: 270, Shipping: 450, Total: 3420},
			}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Pricer:          pricer,
		Addresses:       addrProvider,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cmd := CartEstimateCommand{UserID: "user-estimate", ShippingAddressID: strPtr("addr-1")}
	result, err := service.Estimate(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Currency != "JPY" {
		t.Fatalf("expected currency JPY, got %s", result.Currency)
	}
	if result.Estimate.Total != 3420 {
		t.Fatalf("expected total 3420, got %d", result.Estimate.Total)
	}
	if result.Promotion == nil || !result.Promotion.Applied {
		t.Fatalf("expected promotion applied, got %#v", result.Promotion)
	}
	if result.Promotion.DiscountAmount != 300 {
		t.Fatalf("expected promotion discount 300, got %d", result.Promotion.DiscountAmount)
	}
	if result.Promotion.Code != "SAKURA10" {
		t.Fatalf("expected promotion code uppercased, got %s", result.Promotion.Code)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %#v", result.Warnings)
	}
}

func TestCartServiceEstimateWarnings(t *testing.T) {
	now := time.Date(2024, 8, 2, 9, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:        userID,
				UserID:    userID,
				Currency:  "JPY",
				Items:     []domain.CartItem{{ID: "item-2", ProductID: "prod", SKU: "sku", Quantity: 1, UnitPrice: 1200, Currency: "JPY", RequiresShipping: true}},
				CreatedAt: now.Add(-time.Hour),
				UpdatedAt: now,
			}, nil
		},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if cmd.PromotionCode == nil || *cmd.PromotionCode != "WINTER20" {
				t.Fatalf("expected promotion code WINTER20, got %#v", cmd.PromotionCode)
			}
			return PriceCartResult{
				Breakdown: PricingBreakdown{Currency: "JPY", Subtotal: 1200, Total: 1200},
				Estimate:  CartEstimate{Subtotal: 1200, Total: 1200},
			}, nil
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

	cmd := CartEstimateCommand{UserID: "user-warn", PromotionCode: strPtr("winter20")}
	result, err := service.Estimate(context.Background(), cmd)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 2 {
		t.Fatalf("expected two warnings, got %#v", result.Warnings)
	}
	if !containsWarning(result.Warnings, CartEstimateWarningMissingShippingAddress) {
		t.Fatalf("expected missing shipping warning, got %#v", result.Warnings)
	}
	if !containsWarning(result.Warnings, CartEstimateWarningPromotionNotApplied) {
		t.Fatalf("expected promotion warning, got %#v", result.Warnings)
	}
	if result.Promotion != nil {
		t.Fatalf("expected no promotion returned, got %#v", result.Promotion)
	}
}

func TestCartServiceEstimateEmptyCart(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY"}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.Estimate(context.Background(), CartEstimateCommand{UserID: "user-empty"})
	if err == nil || !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
}

func TestCartServiceApplyPromotionSuccess(t *testing.T) {
	now := time.Date(2024, 7, 15, 9, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			if userID != "user-apply" {
				t.Fatalf("unexpected user id %s", userID)
			}
			return domain.Cart{
				ID:       "cart-apply",
				UserID:   userID,
				Currency: "JPY",
				Items: []domain.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "sku-1", Quantity: 2, UnitPrice: 1000, Currency: "JPY"},
				},
				Metadata:  map[string]any{"existing": "value"},
				CreatedAt: now.Add(-2 * time.Hour),
				UpdatedAt: now.Add(-time.Minute),
			}, nil
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			if expected == nil {
				t.Fatalf("expected optimistic lock timestamp")
			}
			if cart.Promotion == nil || !strings.EqualFold(cart.Promotion.Code, "SPRING10") {
				t.Fatalf("expected promotion SPRING10, got %#v", cart.Promotion)
			}
			if cart.Estimate == nil || cart.Estimate.Discount != 500 {
				t.Fatalf("expected estimate discount 500, got %#v", cart.Estimate)
			}
			saved := cart
			saved.UpdatedAt = now
			return saved, nil
		},
	}

	promotions := &stubPromotionService{
		validateFunc: func(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
			if cmd.Code != "SPRING10" {
				t.Fatalf("expected code SPRING10, got %s", cmd.Code)
			}
			if cmd.UserID == nil || *cmd.UserID != "user-apply" {
				t.Fatalf("expected user id user-apply, got %#v", cmd.UserID)
			}
			if cmd.CartID == nil || *cmd.CartID != "cart-apply" {
				t.Fatalf("expected cart id cart-apply, got %#v", cmd.CartID)
			}
			return PromotionValidationResult{Code: "spring10", Eligible: true, DiscountAmount: 500, Reason: "seasonal"}, nil
		},
	}

	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if cmd.PromotionCode == nil || *cmd.PromotionCode != "SPRING10" {
				t.Fatalf("expected promotion code SPRING10, got %#v", cmd.PromotionCode)
			}
			return PriceCartResult{
				Estimate: CartEstimate{Subtotal: 2000, Discount: 500, Tax: 100, Shipping: 0, Total: 1600},
				Breakdown: PricingBreakdown{
					Currency: "JPY",
					Discounts: []DiscountBreakdown{
						{Type: "promotion", Code: "SPRING10", Amount: 500},
					},
				},
			}, nil
		},
	}

	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Pricer:          pricer,
		Promotions:      promotions,
		Clock:           func() time.Time { return now },
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	cart, err := service.ApplyPromotion(context.Background(), CartPromotionCommand{
		UserID:         " user-apply ",
		Code:           " spring10 ",
		Source:         " user ",
		IdempotencyKey: " idem-apply ",
	})
	if err != nil {
		t.Fatalf("unexpected error applying promotion: %v", err)
	}

	if cart.Promotion == nil {
		t.Fatalf("expected promotion info returned")
	}
	if cart.Promotion.Code != "SPRING10" {
		t.Fatalf("expected promotion code SPRING10, got %s", cart.Promotion.Code)
	}
	if !cart.Promotion.Applied {
		t.Fatalf("expected promotion marked applied")
	}
	if cart.Promotion.DiscountAmount != 500 {
		t.Fatalf("expected discount amount 500, got %d", cart.Promotion.DiscountAmount)
	}
	if cart.Estimate == nil || cart.Estimate.Discount != 500 {
		t.Fatalf("expected estimate discount 500, got %#v", cart.Estimate)
	}
	if cart.Metadata == nil {
		t.Fatalf("expected metadata map")
	}
	if v := cart.Metadata["existing"]; v != "value" {
		t.Fatalf("expected existing metadata preserved, got %#v", v)
	}
	if v := cart.Metadata["promotion_source"]; v != "user" {
		t.Fatalf("expected promotion source metadata user, got %#v", v)
	}
	if v := cart.Metadata["promotion_idempotency_key"]; v != "idem-apply" {
		t.Fatalf("expected idempotency metadata idem-apply, got %#v", v)
	}
	if v := cart.Metadata["promotion_reason"]; v != "seasonal" {
		t.Fatalf("expected promotion reason seasonal, got %#v", v)
	}
}

func TestCartServiceApplyPromotionIneligible(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", UpdatedAt: time.Now()}, nil
		},
	}
	promotions := &stubPromotionService{
		validateFunc: func(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
			return PromotionValidationResult{Code: cmd.Code, Eligible: false, Reason: "expired"}, nil
		},
	}
	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Promotions:      promotions,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.ApplyPromotion(context.Background(), CartPromotionCommand{UserID: "user-1", Code: "BADCODE"})
	if err == nil || !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "expired") {
		t.Fatalf("expected error mentioning expired, got %v", err)
	}
}

func TestCartServiceApplyPromotionUnavailable(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", UpdatedAt: time.Now()}, nil
		},
	}
	promotions := &stubPromotionService{
		validateFunc: func(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
			return PromotionValidationResult{}, errors.New("service down")
		},
	}
	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Promotions:      promotions,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.ApplyPromotion(context.Background(), CartPromotionCommand{UserID: "user-1", Code: "PROMO"})
	if err == nil || !errors.Is(err, ErrCartUnavailable) {
		t.Fatalf("expected ErrCartUnavailable, got %v", err)
	}
}

func TestCartServiceApplyPromotionInvalidCode(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{ID: userID, UserID: userID, Currency: "JPY", UpdatedAt: time.Now()}, nil
		},
	}
	promotions := &stubPromotionService{
		validateFunc: func(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
			return PromotionValidationResult{}, ErrPromotionInvalidCode
		},
	}
	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Promotions:      promotions,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.ApplyPromotion(context.Background(), CartPromotionCommand{UserID: "user-1", Code: "bad"})
	if err == nil || !errors.Is(err, ErrCartInvalidInput) {
		t.Fatalf("expected ErrCartInvalidInput, got %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "invalid") {
		t.Fatalf("expected invalid message, got %v", err)
	}
}

func TestCartServiceRemovePromotionSuccess(t *testing.T) {
	now := time.Date(2024, 7, 20, 14, 0, 0, 0, time.UTC)
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{
				ID:       userID,
				UserID:   userID,
				Currency: "JPY",
				Items: []domain.CartItem{
					{ID: "item-1", ProductID: "prod-1", SKU: "sku-1", Quantity: 1, UnitPrice: 1500, Currency: "JPY"},
				},
				Promotion: &domain.CartPromotion{Code: "WINTER20", DiscountAmount: 200, Applied: true},
				Metadata: map[string]any{
					"promotion_source":          "user",
					"promotion_idempotency_key": "idem-old",
					"other":                     "keep",
				},
				UpdatedAt: now.Add(-time.Minute),
				CreatedAt: now.Add(-time.Hour),
			}, nil
		},
		upsertFunc: func(ctx context.Context, cart domain.Cart, expected *time.Time) (domain.Cart, error) {
			if cart.Promotion != nil {
				t.Fatalf("expected promotion removed before persistence")
			}
			if cart.Metadata != nil {
				if _, exists := cart.Metadata["promotion_idempotency_key"]; exists {
					t.Fatalf("expected idempotency metadata removed, got %#v", cart.Metadata)
				}
			}
			cart.UpdatedAt = now
			return cart, nil
		},
	}
	pricer := &stubCartPricer{
		calculateFunc: func(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error) {
			if cmd.PromotionCode != nil {
				t.Fatalf("expected no promotion code when removing")
			}
			return PriceCartResult{
				Estimate: CartEstimate{Subtotal: 1500, Discount: 0, Tax: 120, Shipping: 0, Total: 1620},
			}, nil
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

	cart, err := service.RemovePromotion(context.Background(), " user-remove ")
	if err != nil {
		t.Fatalf("unexpected error removing promotion: %v", err)
	}
	if cart.Promotion != nil {
		t.Fatalf("expected promotion cleared, got %#v", cart.Promotion)
	}
	if cart.Metadata == nil {
		t.Fatalf("expected metadata map preserved")
	}
	if _, exists := cart.Metadata["promotion_source"]; exists {
		t.Fatalf("expected promotion metadata removed, got %#v", cart.Metadata)
	}
	if v := cart.Metadata["other"]; v != "keep" {
		t.Fatalf("expected other metadata kept, got %#v", v)
	}
	if cart.Estimate == nil || cart.Estimate.Discount != 0 {
		t.Fatalf("expected estimate updated without discount, got %#v", cart.Estimate)
	}
}

func TestCartServiceRemovePromotionNotFound(t *testing.T) {
	repo := &stubCartRepository{
		getFunc: func(ctx context.Context, userID string) (domain.Cart, error) {
			return domain.Cart{}, &repositoryErrorStub{notFound: true}
		},
	}
	service, err := NewCartService(CartServiceDeps{
		Repository:      repo,
		Clock:           time.Now,
		DefaultCurrency: "JPY",
	})
	if err != nil {
		t.Fatalf("unexpected error constructing cart service: %v", err)
	}

	_, err = service.RemovePromotion(context.Background(), "missing")
	if err == nil || !errors.Is(err, ErrCartNotFound) {
		t.Fatalf("expected ErrCartNotFound, got %v", err)
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

type stubPromotionService struct {
	validateFunc func(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error)
}

func (s *stubPromotionService) GetPublicPromotion(context.Context, string) (PromotionPublic, error) {
	return PromotionPublic{}, errors.New("not implemented")
}

func (s *stubPromotionService) ValidatePromotion(ctx context.Context, cmd ValidatePromotionCommand) (PromotionValidationResult, error) {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, cmd)
	}
	return PromotionValidationResult{}, nil
}

func (s *stubPromotionService) ListPromotions(context.Context, PromotionListFilter) (domain.CursorPage[Promotion], error) {
	return domain.CursorPage[Promotion]{}, errors.New("not implemented")
}

func (s *stubPromotionService) CreatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	return Promotion{}, errors.New("not implemented")
}

func (s *stubPromotionService) UpdatePromotion(context.Context, UpsertPromotionCommand) (Promotion, error) {
	return Promotion{}, errors.New("not implemented")
}

func (s *stubPromotionService) DeletePromotion(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubPromotionService) ListPromotionUsage(context.Context, PromotionUsageFilter) (domain.CursorPage[PromotionUsage], error) {
	return domain.CursorPage[PromotionUsage]{}, errors.New("not implemented")
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

type stubDesignFinder struct {
	findFunc func(ctx context.Context, designID string) (domain.Design, error)
}

func (s *stubDesignFinder) FindByID(ctx context.Context, designID string) (domain.Design, error) {
	if s.findFunc != nil {
		return s.findFunc(ctx, designID)
	}
	return domain.Design{}, errors.New("not implemented")
}

type stubInventoryAvailability struct {
	validateFunc func(ctx context.Context, lines []InventoryLine) error
}

func (s *stubInventoryAvailability) ValidateAvailability(ctx context.Context, lines []InventoryLine) error {
	if s.validateFunc != nil {
		return s.validateFunc(ctx, lines)
	}
	return nil
}
