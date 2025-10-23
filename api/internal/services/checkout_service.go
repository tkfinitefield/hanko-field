package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/payments"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	checkoutReservationReason         = "checkout"
	checkoutReleaseReasonPaymentFail  = "checkout_payment_failed"
	checkoutReleaseReasonPersistError = "checkout_persist_failed"
	defaultCheckoutReservationTTL     = 15 * time.Minute
)

var (
	// ErrCheckoutInvalidInput indicates the caller supplied invalid input parameters.
	ErrCheckoutInvalidInput = errors.New("checkout: invalid input")
	// ErrCheckoutUnavailable indicates checkout dependencies are currently unavailable.
	ErrCheckoutUnavailable = errors.New("checkout: unavailable")
	// ErrCheckoutCartNotReady indicates the cart is missing required data for checkout.
	ErrCheckoutCartNotReady = errors.New("checkout: cart not ready")
	// ErrCheckoutInsufficientStock indicates stock could not be reserved for the cart items.
	ErrCheckoutInsufficientStock = errors.New("checkout: insufficient stock")
	// ErrCheckoutConflict indicates a concurrent modification prevented completing checkout.
	ErrCheckoutConflict = errors.New("checkout: conflict")
	// ErrCheckoutPaymentFailed indicates the PSP session could not be created.
	ErrCheckoutPaymentFailed = errors.New("checkout: payment failed")
)

// checkoutSessionManager abstracts payments.Manager for easier testing.
type checkoutSessionManager interface {
	CreateCheckoutSession(ctx context.Context, paymentCtx payments.PaymentContext, req payments.CheckoutSessionRequest) (payments.CheckoutSession, error)
}

// CheckoutServiceDeps wires the dependencies required by the checkout service.
type CheckoutServiceDeps struct {
	Carts          repositories.CartRepository
	Inventory      InventoryService
	Payments       checkoutSessionManager
	Clock          func() time.Time
	Logger         func(ctx context.Context, event string, fields map[string]any)
	ReservationTTL time.Duration
}

type checkoutService struct {
	carts          repositories.CartRepository
	inventory      InventoryService
	payments       checkoutSessionManager
	now            func() time.Time
	logger         func(ctx context.Context, event string, fields map[string]any)
	reservationTTL time.Duration
}

// NewCheckoutService constructs a CheckoutService validating required dependencies.
func NewCheckoutService(deps CheckoutServiceDeps) (CheckoutService, error) {
	if deps.Carts == nil {
		return nil, errors.New("checkout service: cart repository is required")
	}
	if deps.Payments == nil {
		return nil, errors.New("checkout service: payment manager is required")
	}
	if deps.Inventory == nil {
		return nil, errors.New("checkout service: inventory service is required")
	}

	clock := deps.Clock
	if clock == nil {
		clock = time.Now
	}
	logger := deps.Logger
	if logger == nil {
		logger = func(context.Context, string, map[string]any) {}
	}
	ttl := deps.ReservationTTL
	if ttl <= 0 {
		ttl = defaultCheckoutReservationTTL
	}

	return &checkoutService{
		carts:     deps.Carts,
		inventory: deps.Inventory,
		payments:  deps.Payments,
		now: func() time.Time {
			return clock().UTC()
		},
		logger:         logger,
		reservationTTL: ttl,
	}, nil
}

// CreateCheckoutSession validates cart readiness, reserves stock, creates a PSP session, and records metadata.
func (s *checkoutService) CreateCheckoutSession(ctx context.Context, cmd CreateCheckoutSessionCommand) (CheckoutSession, error) {
	if s == nil || s.carts == nil || s.payments == nil {
		return CheckoutSession{}, ErrCheckoutUnavailable
	}

	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return CheckoutSession{}, ErrCheckoutInvalidInput
	}

	cartID := strings.TrimSpace(cmd.CartID)
	successURL := strings.TrimSpace(cmd.SuccessURL)
	cancelURL := strings.TrimSpace(cmd.CancelURL)
	if successURL == "" || cancelURL == "" {
		return CheckoutSession{}, ErrCheckoutInvalidInput
	}

	cart, err := s.carts.GetCart(ctx, userID)
	if err != nil {
		return CheckoutSession{}, s.translateCartError(err)
	}
	cart = normaliseCheckoutCart(cart, userID)

	if cartID != "" && !strings.EqualFold(cart.ID, cartID) {
		return CheckoutSession{}, ErrCheckoutInvalidInput
	}
	if err := validateCheckoutCart(cart); err != nil {
		return CheckoutSession{}, err
	}

	idempotencyKey := s.checkoutIdempotencyKey(cmd, cart)
	reservation, reserved, err := s.reserveStockIfNeeded(ctx, cart, userID, idempotencyKey)
	if err != nil {
		return CheckoutSession{}, err
	}

	session, err := s.createPSPSession(ctx, cmd, cart, successURL, cancelURL, idempotencyKey, reservation)
	if err != nil {
		if reserved && reservation.ID != "" {
			s.releaseReservation(ctx, reservation.ID, checkoutReleaseReasonPaymentFail)
		}
		return CheckoutSession{}, err
	}

	if err := s.persistCheckoutMetadata(ctx, cart, reservation, session, idempotencyKey); err != nil {
		if reserved && reservation.ID != "" {
			s.releaseReservation(ctx, reservation.ID, checkoutReleaseReasonPersistError)
		}
		return CheckoutSession{}, err
	}

	return CheckoutSession{
		SessionID:    session.ID,
		PSP:          session.Provider,
		ClientSecret: session.ClientSecret,
		RedirectURL:  session.RedirectURL,
		ExpiresAt:    session.ExpiresAt.UTC(),
	}, nil
}

// ConfirmClientCompletion is a placeholder until the confirmation flow is implemented.
func (s *checkoutService) ConfirmClientCompletion(context.Context, ConfirmCheckoutCommand) error {
	return ErrCheckoutUnavailable
}

func (s *checkoutService) reserveStockIfNeeded(ctx context.Context, cart domain.Cart, userID string, idempotencyKey string) (InventoryReservation, bool, error) {
	lines := extractInventoryLines(cart.Items)
	if len(lines) == 0 {
		return InventoryReservation{}, false, nil
	}
	reservation, err := s.inventory.ReserveStocks(ctx, InventoryReserveCommand{
		OrderID:        "",
		UserID:         userID,
		Lines:          lines,
		TTL:            s.reservationTTL,
		Reason:         checkoutReservationReason,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrInventoryInvalidInput):
			return InventoryReservation{}, false, ErrCheckoutInvalidInput
		case errors.Is(err, ErrInventoryInsufficientStock):
			return InventoryReservation{}, false, ErrCheckoutInsufficientStock
		default:
			s.logger(ctx, "checkout.reserve_failed", map[string]any{
				"userID": userID,
				"error":  err.Error(),
			})
			return InventoryReservation{}, false, ErrCheckoutUnavailable
		}
	}
	return reservation, true, nil
}

func (s *checkoutService) createPSPSession(ctx context.Context, cmd CreateCheckoutSessionCommand, cart domain.Cart, successURL, cancelURL, idempotencyKey string, reservation InventoryReservation) (payments.CheckoutSession, error) {
	currency := strings.ToUpper(strings.TrimSpace(cart.Currency))
	amount, err := cartTotal(cart)
	if err != nil {
		return payments.CheckoutSession{}, err
	}

	paymentCtx := payments.PaymentContext{
		PreferredProvider: strings.TrimSpace(cmd.PSP),
		Currency:          currency,
		Metadata:          copyStringMap(cmd.Metadata),
	}

	metadata := s.buildPaymentMetadata(cmd.Metadata, cart, reservation, idempotencyKey)
	req := payments.CheckoutSessionRequest{
		Amount:         amount,
		Currency:       currency,
		SuccessURL:     successURL,
		CancelURL:      cancelURL,
		Metadata:       metadata,
		IdempotencyKey: idempotencyKey,
		Items:          buildCheckoutLineItems(cart, amount),
		AllowPromotion: cart.Promotion != nil && cart.Promotion.Applied,
	}

	if locale := strings.TrimSpace(metadataValue(cmd.Metadata, "locale")); locale != "" {
		req.Locale = locale
	}

	session, err := s.payments.CreateCheckoutSession(ctx, paymentCtx, req)
	if err != nil {
		if errors.Is(err, payments.ErrUnsupportedProvider) {
			return payments.CheckoutSession{}, ErrCheckoutInvalidInput
		}
		s.logger(ctx, "checkout.payment_session_failed", map[string]any{
			"userID":   cart.UserID,
			"cartID":   cart.ID,
			"provider": paymentCtx.PreferredProvider,
			"error":    err.Error(),
		})
		return payments.CheckoutSession{}, ErrCheckoutPaymentFailed
	}
	return session, nil
}

func (s *checkoutService) persistCheckoutMetadata(ctx context.Context, cart domain.Cart, reservation InventoryReservation, session payments.CheckoutSession, idempotencyKey string) error {
	originalUpdated := cart.UpdatedAt
	metadata := cloneAnyMap(cart.Metadata)
	if metadata == nil {
		metadata = make(map[string]any)
	}
	checkoutMeta := map[string]any{
		"sessionId":      session.ID,
		"provider":       session.Provider,
		"clientSecret":   session.ClientSecret,
		"redirectUrl":    session.RedirectURL,
		"intentId":       session.IntentID,
		"expiresAt":      session.ExpiresAt.UTC(),
		"idempotencyKey": idempotencyKey,
		"updatedAt":      s.now(),
	}
	if reservation.ID != "" {
		checkoutMeta["reservationId"] = reservation.ID
		checkoutMeta["reservationExpiresAt"] = reservation.ExpiresAt.UTC()
	}
	if existing, ok := metadata["checkout"].(map[string]any); ok {
		for k, v := range existing {
			if _, exists := checkoutMeta[k]; !exists {
				checkoutMeta[k] = v
			}
		}
	}
	metadata["checkout"] = checkoutMeta
	cart.Metadata = metadata
	cart.UpdatedAt = s.now()

	if _, err := s.carts.UpsertCart(ctx, cart, &originalUpdated); err != nil {
		return s.translateCartError(err)
	}
	return nil
}

func (s *checkoutService) releaseReservation(ctx context.Context, reservationID string, reason string) {
	if s.inventory == nil || strings.TrimSpace(reservationID) == "" {
		return
	}
	_, err := s.inventory.ReleaseReservation(ctx, InventoryReleaseCommand{
		ReservationID: reservationID,
		Reason:        reason,
	})
	if err != nil {
		s.logger(ctx, "checkout.release_failed", map[string]any{
			"reservationId": reservationID,
			"reason":        reason,
			"error":         err.Error(),
		})
	}
}

func (s *checkoutService) checkoutIdempotencyKey(cmd CreateCheckoutSessionCommand, cart domain.Cart) string {
	if key := metadataValue(cmd.Metadata, "idempotency_key"); key != "" {
		return key
	}
	if key := metadataValue(cmd.Metadata, "idempotencyKey"); key != "" {
		return key
	}
	base := fmt.Sprintf("%s|%s|%s|%d", strings.ToLower(strings.TrimSpace(cmd.PSP)), cart.ID, cart.UpdatedAt.UTC().Format(time.RFC3339Nano), cartTotalFallback(cart))
	sum := sha256.Sum256([]byte(base))
	return hex.EncodeToString(sum[:])
}

func (s *checkoutService) translateCartError(err error) error {
	if err == nil {
		return nil
	}
	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			return ErrCheckoutCartNotReady
		case repoErr.IsConflict():
			return ErrCheckoutConflict
		case repoErr.IsUnavailable():
			return ErrCheckoutUnavailable
		default:
			return ErrCheckoutUnavailable
		}
	}
	return ErrCheckoutUnavailable
}

func normaliseCheckoutCart(cart domain.Cart, userID string) domain.Cart {
	cart.ID = strings.TrimSpace(firstNonEmpty(cart.ID, cart.UserID, userID))
	cart.UserID = strings.TrimSpace(firstNonEmpty(cart.UserID, userID, cart.ID))
	cart.Currency = strings.ToUpper(strings.TrimSpace(cart.Currency))
	if cart.Currency == "" {
		cart.Currency = "JPY"
	}
	if cart.Items == nil {
		cart.Items = []domain.CartItem{}
	}
	if cart.Metadata == nil {
		cart.Metadata = map[string]any{}
	}
	if cart.CreatedAt.IsZero() {
		cart.CreatedAt = time.Now().UTC()
	}
	if cart.UpdatedAt.IsZero() {
		cart.UpdatedAt = time.Now().UTC()
	}
	return cart
}

func validateCheckoutCart(cart domain.Cart) error {
	if len(cart.Items) == 0 {
		return ErrCheckoutCartNotReady
	}

	if requiresShipping(cart.Items) && strings.TrimSpace(cart.ShippingAddressID) == "" {
		return ErrCheckoutCartNotReady
	}

	if cart.Estimate == nil || cart.Estimate.Total <= 0 {
		// Allow carts where estimate is missing but totals can be derived.
		if cartTotalFallback(cart) <= 0 {
			return ErrCheckoutCartNotReady
		}
	}

	if cart.Promotion != nil && !cart.Promotion.Applied {
		return ErrCheckoutCartNotReady
	}

	return nil
}

func cartTotal(cart domain.Cart) (int64, error) {
	if cart.Estimate != nil && cart.Estimate.Total > 0 {
		return cart.Estimate.Total, nil
	}
	total := cartTotalFallback(cart)
	if total <= 0 {
		return 0, ErrCheckoutCartNotReady
	}
	return total, nil
}

func cartTotalFallback(cart domain.Cart) int64 {
	var total int64
	for _, item := range cart.Items {
		if item.Quantity <= 0 || item.UnitPrice <= 0 {
			continue
		}
		total += item.UnitPrice * int64(item.Quantity)
	}
	return total
}

func extractInventoryLines(items []domain.CartItem) []InventoryLine {
	lines := make([]InventoryLine, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		if sku == "" || item.Quantity <= 0 {
			continue
		}
		lines = append(lines, InventoryLine{
			ProductID: strings.TrimSpace(item.ProductID),
			SKU:       sku,
			Quantity:  item.Quantity,
		})
	}
	return lines
}

func buildCheckoutLineItems(cart domain.Cart, total int64) []payments.CheckoutLineItem {
	items := make([]payments.CheckoutLineItem, 0, len(cart.Items))
	var itemTotal int64
	for _, item := range cart.Items {
		if item.Quantity <= 0 || item.UnitPrice <= 0 {
			continue
		}
		name := ""
		description := ""
		if item.Metadata != nil {
			if label, ok := item.Metadata["name"].(string); ok {
				name = strings.TrimSpace(label)
			}
			if desc, ok := item.Metadata["description"].(string); ok {
				description = strings.TrimSpace(desc)
			}
		}
		if name == "" {
			if item.ProductID != "" {
				name = item.ProductID
			} else if item.SKU != "" {
				name = item.SKU
			} else {
				name = "Item"
			}
		}
		items = append(items, payments.CheckoutLineItem{
			Name:        name,
			Description: description,
			SKU:         strings.TrimSpace(item.SKU),
			Quantity:    int64(item.Quantity),
			Amount:      item.UnitPrice,
			Currency:    strings.ToUpper(strings.TrimSpace(firstNonEmpty(item.Currency, cart.Currency))),
		})
		itemTotal += item.UnitPrice * int64(item.Quantity)
	}

	if total > 0 && itemTotal == total && len(items) > 0 {
		return items
	}
	return []payments.CheckoutLineItem{
		{
			Name:     "Order",
			Quantity: 1,
			Amount:   total,
			Currency: strings.ToUpper(strings.TrimSpace(firstNonEmpty(cart.Currency))),
		},
	}
}

func (s *checkoutService) buildPaymentMetadata(cmdMeta map[string]string, cart domain.Cart, reservation InventoryReservation, idempotencyKey string) map[string]string {
	meta := map[string]string{
		"cart_id":        cart.ID,
		"user_id":        cart.UserID,
		"idempotencyKey": idempotencyKey,
	}
	if reservation.ID != "" {
		meta["reservation_id"] = reservation.ID
	}
	for k, v := range cmdMeta {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		meta[k] = v
	}
	return meta
}

func metadataValue(meta map[string]string, key string) string {
	if len(meta) == 0 {
		return ""
	}
	return strings.TrimSpace(meta[key])
}

func copyStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	return maps.Clone(values)
}
