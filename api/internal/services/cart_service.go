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

const (
	maxCartNotesLength     = 2000
	maxPromotionHintLength = 120
)

type addressProvider interface {
	ListAddresses(ctx context.Context, userID string) ([]Address, error)
}

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
	Addresses       addressProvider
	Clock           func() time.Time
	DefaultCurrency string
	Logger          func(context.Context, string, map[string]any)
}

type cartService struct {
	repo      repositories.CartRepository
	pricer    CartPricer
	addresses addressProvider
	now       func() time.Time
	currency  string
	logger    func(context.Context, string, map[string]any)
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
		repo:      deps.Repository,
		pricer:    deps.Pricer,
		addresses: deps.Addresses,
		now:       func() time.Time { return deps.Clock().UTC() },
		currency:  defaultCurrency,
		logger:    logger,
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
			saved, err := s.repo.UpsertCart(ctx, defaultCart, nil)
			if err != nil {
				return Cart{}, s.translateRepoError(err)
			}
			cart = saved
		} else {
			return Cart{}, s.translateRepoError(err)
		}
	}

	normalised := s.normaliseCart(cart, uid)
	_ = s.hydrateCartAddresses(ctx, &normalised, false)

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

func (s *cartService) UpdateCart(ctx context.Context, cmd UpdateCartCommand) (Cart, error) {
	if s == nil || s.repo == nil {
		return Cart{}, ErrCartUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return Cart{}, ErrCartInvalidInput
	}

	if cmd.Currency == nil && cmd.ShippingAddressID == nil && cmd.BillingAddressID == nil && cmd.Notes == nil && cmd.PromotionHint == nil {
		return Cart{}, ErrCartInvalidInput
	}

	cart, err := s.repo.GetCart(ctx, userID)
	exists := true
	if err != nil {
		if isRepoNotFound(err) {
			cart = s.newCart(userID)
			exists = false
		} else {
			return Cart{}, s.translateRepoError(err)
		}
	}

	cart = s.normaliseCart(cart, userID)
	previousUpdatedAt := cart.UpdatedAt

    if exists {
        if cmd.ExpectedUpdatedAt == nil || cmd.ExpectedUpdatedAt.IsZero() {
            return Cart{}, ErrCartConflict
        }
        expected := cmd.ExpectedUpdatedAt.UTC()
        previous := previousUpdatedAt.UTC()
        if cmd.ExpectedFromHeader {
            expected = expected.Truncate(time.Second)
            previous = previous.Truncate(time.Second)
        }
        if previous.IsZero() || !previous.Equal(expected) {
            return Cart{}, ErrCartConflict
        }
    }

	var (
		addressBook     map[string]Address
		addressBookErr  error
		addressBookOnce bool
	)

	loadAddress := func(id string) (*Address, error) {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return nil, nil
		}
		if !addressBookOnce {
			addressBookOnce = true
			addressBook, addressBookErr = s.loadAddressBook(ctx, userID)
		}
		if addressBookErr != nil {
			return nil, addressBookErr
		}
		addr, ok := addressBook[trimmed]
		if !ok {
			return nil, ErrCartInvalidInput
		}
		dup := addr
		return &dup, nil
	}

	if cmd.ShippingAddressID == nil && strings.TrimSpace(cart.ShippingAddressID) != "" {
		addr, err := loadAddress(cart.ShippingAddressID)
		if err != nil {
			return Cart{}, err
		}
		cart.ShippingAddress = cloneCartAddress(addr)
	}

	if cmd.BillingAddressID == nil && strings.TrimSpace(cart.BillingAddressID) != "" {
		addr, err := loadAddress(cart.BillingAddressID)
		if err != nil {
			return Cart{}, err
		}
		cart.BillingAddress = cloneCartAddress(addr)
	}

	if cmd.ShippingAddressID != nil {
		input := strings.TrimSpace(*cmd.ShippingAddressID)
		if input == "" {
			cart.ShippingAddressID = ""
			cart.ShippingAddress = nil
		} else {
			addr, err := loadAddress(input)
			if err != nil {
				return Cart{}, err
			}
			cart.ShippingAddressID = strings.TrimSpace(addr.ID)
			cart.ShippingAddress = cloneCartAddress(addr)
		}
	}

	if cmd.BillingAddressID != nil {
		input := strings.TrimSpace(*cmd.BillingAddressID)
		if input == "" {
			cart.BillingAddressID = ""
			cart.BillingAddress = nil
		} else {
			addr, err := loadAddress(input)
			if err != nil {
				return Cart{}, err
			}
			cart.BillingAddressID = strings.TrimSpace(addr.ID)
			cart.BillingAddress = cloneCartAddress(addr)
		}
	}

	if cmd.Currency != nil {
		currency := strings.ToUpper(strings.TrimSpace(*cmd.Currency))
		if err := validateCurrencyCode(currency); err != nil {
			return Cart{}, err
		}
		cart.Currency = currency
	}

	if cmd.Notes != nil {
		note := strings.TrimSpace(*cmd.Notes)
		if len(note) > maxCartNotesLength {
			return Cart{}, fmt.Errorf("%w: notes must be %d characters or fewer", ErrCartInvalidInput, maxCartNotesLength)
		}
		cart.Notes = note
	}

	if cmd.PromotionHint != nil {
		hint := strings.TrimSpace(*cmd.PromotionHint)
		if len(hint) > maxPromotionHintLength {
			return Cart{}, fmt.Errorf("%w: promotion_hint must be %d characters or fewer", ErrCartInvalidInput, maxPromotionHintLength)
		}
		cart.PromotionHint = hint
	}

	cart.UpdatedAt = s.now()
	if cart.CreatedAt.IsZero() {
		cart.CreatedAt = cart.UpdatedAt
	}

	if s.pricer != nil {
		result, err := s.pricer.Calculate(ctx, PriceCartCommand{Cart: cart})
		if err != nil {
			s.logger(ctx, "cart.pricing_failed", map[string]any{
				"userID": userID,
				"error":  err.Error(),
			})
			return Cart{}, translatePricingError(err)
		}
		estimateCopy := result.Estimate
		cart.Estimate = &estimateCopy
	} else if cart.Estimate == nil {
		estimate := naiveCartEstimate(cart.Items)
		cart.Estimate = &estimate
	}

	var expected *time.Time
	if exists {
		if previousUpdatedAt.IsZero() {
			return Cart{}, ErrCartConflict
		}
		ts := previousUpdatedAt.UTC()
		expected = &ts
	}

	saved, err := s.repo.UpsertCart(ctx, cart, expected)
	if err != nil {
		return Cart{}, s.translateRepoError(err)
	}

	saved = s.normaliseCart(saved, userID)

	if strings.TrimSpace(saved.ShippingAddressID) != "" {
		addr, addrErr := loadAddress(saved.ShippingAddressID)
		if addrErr != nil {
			if errors.Is(addrErr, ErrCartInvalidInput) {
				saved.ShippingAddressID = ""
				saved.ShippingAddress = nil
			} else {
				return Cart{}, addrErr
			}
		} else {
			saved.ShippingAddress = cloneCartAddress(addr)
		}
	}

	if strings.TrimSpace(saved.BillingAddressID) != "" {
		addr, addrErr := loadAddress(saved.BillingAddressID)
		if addrErr != nil {
			if errors.Is(addrErr, ErrCartInvalidInput) {
				saved.BillingAddressID = ""
				saved.BillingAddress = nil
			} else {
				return Cart{}, addrErr
			}
		} else {
			saved.BillingAddress = cloneCartAddress(addr)
		}
	}

	return saved, nil
}

func (s *cartService) newCart(userID string) domain.Cart {
	now := s.now()
	return domain.Cart{
		ID:        userID,
		UserID:    userID,
		Currency:  s.currency,
		Items:     []domain.CartItem{},
		Metadata:  map[string]any{},
		CreatedAt: now,
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
	cart.Notes = strings.TrimSpace(cart.Notes)
	cart.PromotionHint = strings.TrimSpace(cart.PromotionHint)
	cart.ShippingAddressID = strings.TrimSpace(cart.ShippingAddressID)
	cart.BillingAddressID = strings.TrimSpace(cart.BillingAddressID)
	if cart.ShippingAddressID == "" {
		cart.ShippingAddress = nil
	}
	if cart.BillingAddressID == "" {
		cart.BillingAddress = nil
	}
	if cart.Estimate != nil && cart.Estimate.Total == 0 && cart.Estimate.Subtotal == 0 && len(cart.Items) > 0 {
		estimate := naiveCartEstimate(cart.Items)
		cart.Estimate = &estimate
	}
	if cart.CreatedAt.IsZero() {
		cart.CreatedAt = s.now()
	}
	if cart.UpdatedAt.IsZero() {
		cart.UpdatedAt = s.now()
	}
	return cart
}

func (s *cartService) hydrateCartAddresses(ctx context.Context, cart *domain.Cart, strict bool) error {
	if cart == nil {
		return nil
	}
	needShipping := strings.TrimSpace(cart.ShippingAddressID) != ""
	needBilling := strings.TrimSpace(cart.BillingAddressID) != ""
	if !needShipping && !needBilling {
		cart.ShippingAddress = nil
		cart.BillingAddress = nil
		return nil
	}

	book, err := s.loadAddressBook(ctx, cart.UserID)
	if err != nil {
		if strict {
			return err
		}
		s.logger(ctx, "cart.address_lookup_failed", map[string]any{
			"userID": cart.UserID,
			"error":  err.Error(),
		})
		return nil
	}

	if needShipping {
		addr, ok := book[strings.TrimSpace(cart.ShippingAddressID)]
		if !ok {
			if strict {
				return ErrCartInvalidInput
			}
			cart.ShippingAddressID = ""
			cart.ShippingAddress = nil
		} else {
			cart.ShippingAddress = cloneCartAddress(&addr)
		}
	}

	if needBilling {
		addr, ok := book[strings.TrimSpace(cart.BillingAddressID)]
		if !ok {
			if strict {
				return ErrCartInvalidInput
			}
			cart.BillingAddressID = ""
			cart.BillingAddress = nil
		} else {
			cart.BillingAddress = cloneCartAddress(&addr)
		}
	}

	return nil
}

func (s *cartService) loadAddressBook(ctx context.Context, userID string) (map[string]Address, error) {
	if s.addresses == nil {
		return nil, ErrCartUnavailable
	}
	addresses, err := s.addresses.ListAddresses(ctx, userID)
	if err != nil {
		if errors.Is(err, errUserIDRequired) {
			return nil, ErrCartInvalidInput
		}
		return nil, ErrCartUnavailable
	}

	book := make(map[string]Address, len(addresses))
	for _, addr := range addresses {
		id := strings.TrimSpace(addr.ID)
		if id != "" {
			book[id] = addr
		}
	}
	return book, nil
}

func cloneCartAddress(addr *Address) *Address {
	if addr == nil {
		return nil
	}
	dup := *addr
	return &dup
}

func validateCurrencyCode(code string) error {
	if len(code) != 3 {
		return fmt.Errorf("%w: currency must be a 3-letter ISO code", ErrCartInvalidInput)
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return fmt.Errorf("%w: currency must be a 3-letter ISO code", ErrCartInvalidInput)
		}
	}
	return nil
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
