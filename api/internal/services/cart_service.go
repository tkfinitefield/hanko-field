package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

var (
	errCartRepositoryRequired = errors.New("cart service: repository is required")
	errCartClockRequired      = errors.New("cart service: clock is required")
)

// ErrCartInvalidInput indicates the caller supplied invalid input.
var ErrCartInvalidInput = errors.New("cart service: invalid input")

// ErrCartUnavailable indicates the cart service cannot fulfil the request due to missing dependencies or backend issues.
var ErrCartUnavailable = errors.New("cart service: unavailable")

// ErrCartNotFound indicates the requested cart does not exist.
var ErrCartNotFound = errors.New("cart service: not found")

// ErrCartConflict indicates the cart could not be updated due to concurrent modifications.
var ErrCartConflict = errors.New("cart service: conflict")

// CartPricer defines the dependency capable of calculating cart totals.
type CartPricer interface {
	Calculate(ctx context.Context, cmd PriceCartCommand) (PriceCartResult, error)
}

// CartServiceDeps wires the repository and pricing dependencies for cart operations.
type CartServiceDeps struct {
	Repository      repositories.CartRepository
	Pricer          CartPricer
	Clock           func() time.Time
	DefaultCurrency string
	Logger          func(context.Context, string, map[string]any)
}

type cartService struct {
	repo     repositories.CartRepository
	pricer   CartPricer
	now      func() time.Time
	currency string
	logger   func(context.Context, string, map[string]any)
}

// NewCartService constructs a CartService enforcing dependency validation.
func NewCartService(deps CartServiceDeps) (CartService, error) {
	if deps.Repository == nil {
		return nil, errCartRepositoryRequired
	}
	if deps.Clock == nil {
		return nil, errCartClockRequired
	}

	defaultCurrency := strings.ToUpper(strings.TrimSpace(deps.DefaultCurrency))
	if defaultCurrency == "" {
		defaultCurrency = "JPY"
	}

	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}

	service := &cartService{
		repo:     deps.Repository,
		pricer:   deps.Pricer,
		now:      func() time.Time { return deps.Clock().UTC() },
		currency: defaultCurrency,
		logger:   logger,
	}
	return service, nil
}

// GetOrCreateCart loads the active cart for the user, creating a new cart when absent.
func (s *cartService) GetOrCreateCart(ctx context.Context, userID string) (Cart, error) {
	if s == nil || s.repo == nil {
		return Cart{}, ErrCartUnavailable
	}

	uid := strings.TrimSpace(userID)
	if uid == "" {
		return Cart{}, ErrCartInvalidInput
	}

	cart, err := s.repo.GetCart(ctx, uid)
	if err != nil {
		if isRepoNotFound(err) {
			defaultCart := s.newCart(uid)
			saved, err := s.repo.UpsertCart(ctx, defaultCart)
			if err != nil {
				return Cart{}, s.translateRepoError(err)
			}
			cart = saved
		} else {
			return Cart{}, s.translateRepoError(err)
		}
	}

	normalised := s.normaliseCart(cart, uid)

	if s.pricer != nil {
		result, err := s.pricer.Calculate(ctx, PriceCartCommand{Cart: normalised})
		if err != nil {
			s.logger(ctx, "cart.pricing_failed", map[string]any{
				"userID": uid,
				"error":  err.Error(),
			})
			return Cart{}, translatePricingError(err)
		}
		estimateCopy := result.Estimate
		normalised.Estimate = &estimateCopy
	} else if normalised.Estimate == nil {
		estimate := naiveCartEstimate(normalised.Items)
		normalised.Estimate = &estimate
	}

	return normalised, nil
}

func (s *cartService) newCart(userID string) domain.Cart {
	now := s.now()
	return domain.Cart{
		ID:       userID,
		UserID:   userID,
		Currency: s.currency,
		Items:    []domain.CartItem{},
		Metadata: map[string]any{},
		UpdatedAt: func() time.Time {
			if now.IsZero() {
				return time.Now().UTC()
			}
			return now
		}(),
	}
}

func (s *cartService) normaliseCart(cart domain.Cart, userID string) domain.Cart {
	if cart.ID = strings.TrimSpace(cart.ID); cart.ID == "" {
		cart.ID = userID
	}
	cart.UserID = strings.TrimSpace(firstNonEmpty(cart.UserID, userID))
	cart.Currency = strings.ToUpper(strings.TrimSpace(firstNonEmpty(cart.Currency, s.currency)))
	if cart.Items == nil {
		cart.Items = []domain.CartItem{}
	}
	if cart.Metadata == nil {
		cart.Metadata = map[string]any{}
	}
	if cart.Estimate != nil && cart.Estimate.Total == 0 && cart.Estimate.Subtotal == 0 && len(cart.Items) > 0 {
		estimate := naiveCartEstimate(cart.Items)
		cart.Estimate = &estimate
	}
	if cart.UpdatedAt.IsZero() {
		cart.UpdatedAt = s.now()
	}
	return cart
}

func (s *cartService) translateRepoError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return ErrCartNotFound
		case repoErr.IsConflict():
			return ErrCartConflict
		case repoErr.IsUnavailable():
			return ErrCartUnavailable
		}
		return ErrCartUnavailable
	}
	return ErrCartUnavailable
}

func translatePricingError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrCartPricingInvalidInput) {
		return ErrCartInvalidInput
	}
	if errors.Is(err, ErrCartPricingCurrencyMismatch) {
		return ErrCartInvalidInput
	}
	return ErrCartUnavailable
}

func naiveCartEstimate(items []domain.CartItem) CartEstimate {
	var subtotal int64
	for _, item := range items {
		if item.Quantity <= 0 {
			continue
		}
		if item.UnitPrice <= 0 {
			continue
		}
		line := item.UnitPrice * int64(item.Quantity)
		if line > 0 {
			subtotal += line
		}
	}
	return CartEstimate{
		Subtotal: subtotal,
		Discount: 0,
		Tax:      0,
		Shipping: 0,
		Total:    subtotal,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// AddOrUpdateItem is not yet implemented.
func (s *cartService) AddOrUpdateItem(ctx context.Context, cmd UpsertCartItemCommand) (Cart, error) {
	return Cart{}, fmt.Errorf("cart service: add or update item not implemented")
}

// RemoveItem is not yet implemented.
func (s *cartService) RemoveItem(ctx context.Context, cmd RemoveCartItemCommand) (Cart, error) {
	return Cart{}, fmt.Errorf("cart service: remove item not implemented")
}

// Estimate is not yet implemented.
func (s *cartService) Estimate(ctx context.Context, userID string) (CartEstimate, error) {
	return CartEstimate{}, fmt.Errorf("cart service: estimate not implemented")
}

// ApplyPromotion is not yet implemented.
func (s *cartService) ApplyPromotion(ctx context.Context, cmd CartPromotionCommand) (Cart, error) {
	return Cart{}, fmt.Errorf("cart service: apply promotion not implemented")
}

// RemovePromotion is not yet implemented.
func (s *cartService) RemovePromotion(ctx context.Context, userID string) (Cart, error) {
	return Cart{}, fmt.Errorf("cart service: remove promotion not implemented")
}

// ClearCart is not yet implemented.
func (s *cartService) ClearCart(ctx context.Context, userID string) error {
	return fmt.Errorf("cart service: clear cart not implemented")
}
