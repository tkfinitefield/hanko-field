package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	cartCollection = "carts"
)

// CartRepository persists cart headers within Firestore.
type CartRepository struct {
	base     *pfirestore.BaseRepository[cartDocument]
	provider *pfirestore.Provider
}

// NewCartRepository constructs a Firestore-backed cart repository.
func NewCartRepository(provider *pfirestore.Provider) (*CartRepository, error) {
	if provider == nil {
		return nil, errors.New("cart repository requires firestore provider")
	}
	base := pfirestore.NewBaseRepository[cartDocument](provider, cartCollection, nil, nil)
	return &CartRepository{
		base:     base,
		provider: provider,
	}, nil
}

// UpsertCart persists the cart header document using the user ID as document identifier.
func (r *CartRepository) UpsertCart(ctx context.Context, cart domain.Cart, expectedUpdate *time.Time) (domain.Cart, error) {
	if r == nil || r.base == nil {
		return domain.Cart{}, errors.New("cart repository not initialised")
	}

	cartID := strings.TrimSpace(firstCartID(cart))
	if cartID == "" {
		return domain.Cart{}, errors.New("cart repository: cart id is required")
	}

	now := time.Now().UTC()
	if !cart.UpdatedAt.IsZero() {
		now = cart.UpdatedAt.UTC()
	}
	createdAt := cart.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = now
	}

	shippingAddressID := strings.TrimSpace(cart.ShippingAddressID)
	billingAddressID := strings.TrimSpace(cart.BillingAddressID)

	doc := cartDocument{
		Currency:          strings.ToUpper(strings.TrimSpace(cart.Currency)),
		ShippingAddressID: shippingAddressID,
		BillingAddressID:  billingAddressID,
		Notes:             strings.TrimSpace(cart.Notes),
		PromotionHint:     strings.TrimSpace(cart.PromotionHint),
		Metadata:          cloneAnyMap(cart.Metadata),
		ItemsCount:        len(cart.Items),
		UpdatedAt:         now,
		CreatedAt:         createdAt,
	}

	if cart.Promotion != nil {
		doc.Promotion = &cartPromotionDocument{
			Code:           strings.TrimSpace(cart.Promotion.Code),
			DiscountAmount: cart.Promotion.DiscountAmount,
			Applied:        cart.Promotion.Applied,
		}
	}
	if cart.Estimate != nil {
		doc.Estimates = &cartEstimateDocument{
			Subtotal: cart.Estimate.Subtotal,
			Discount: cart.Estimate.Discount,
			Tax:      cart.Estimate.Tax,
			Shipping: cart.Estimate.Shipping,
			Total:    cart.Estimate.Total,
		}
	}

	if expectedUpdate == nil || expectedUpdate.IsZero() {
		result, err := r.base.Set(ctx, cartID, doc)
		if err != nil {
			return domain.Cart{}, err
		}

		saved := cloneCart(cart)
		saved.ID = cartID
		saved.UserID = cartID
		saved.Currency = doc.Currency
		saved.ShippingAddressID = shippingAddressID
		saved.BillingAddressID = billingAddressID
		saved.Notes = doc.Notes
		saved.PromotionHint = doc.PromotionHint
		saved.Metadata = cloneAnyMap(cart.Metadata)
		saved.CreatedAt = doc.CreatedAt
		saved.UpdatedAt = result.UpdateTime
		return saved, nil
	}

	updates := []firestore.Update{
		{Path: "currency", Value: doc.Currency},
		{Path: "itemsCount", Value: doc.ItemsCount},
		{Path: "updatedAt", Value: doc.UpdatedAt},
	}

	appendStringUpdate := func(path string, value string) {
		if strings.TrimSpace(value) == "" {
			updates = append(updates, firestore.Update{Path: path, Value: firestore.Delete})
		} else {
			updates = append(updates, firestore.Update{Path: path, Value: value})
		}
	}

	appendStringUpdate("shippingAddressId", shippingAddressID)
	appendStringUpdate("billingAddressId", billingAddressID)
	appendStringUpdate("notes", doc.Notes)
	appendStringUpdate("promotionHint", doc.PromotionHint)

	if len(doc.Metadata) == 0 {
		updates = append(updates, firestore.Update{Path: "metadata", Value: firestore.Delete})
	} else {
		updates = append(updates, firestore.Update{Path: "metadata", Value: doc.Metadata})
	}

	if doc.Promotion == nil {
		updates = append(updates, firestore.Update{Path: "promo", Value: firestore.Delete})
	} else {
		updates = append(updates, firestore.Update{Path: "promo", Value: doc.Promotion})
	}

	if doc.Estimates == nil {
		updates = append(updates, firestore.Update{Path: "estimates", Value: firestore.Delete})
	} else {
		updates = append(updates, firestore.Update{Path: "estimates", Value: doc.Estimates})
	}

	result, err := r.base.Update(ctx, cartID, updates, firestore.LastUpdateTime(expectedUpdate.UTC()))
	if err != nil {
		return domain.Cart{}, err
	}

	saved := cloneCart(cart)
	saved.ID = cartID
	saved.UserID = cartID
	saved.Currency = doc.Currency
	saved.ShippingAddressID = shippingAddressID
	saved.BillingAddressID = billingAddressID
	saved.Notes = doc.Notes
	saved.PromotionHint = doc.PromotionHint
	saved.Metadata = cloneAnyMap(cart.Metadata)
	saved.CreatedAt = cart.CreatedAt
	saved.UpdatedAt = result.UpdateTime
	return saved, nil
}

// GetCart loads the cart header for the given user ID.
func (r *CartRepository) GetCart(ctx context.Context, userID string) (domain.Cart, error) {
	if r == nil || r.base == nil {
		return domain.Cart{}, errors.New("cart repository not initialised")
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return domain.Cart{}, errors.New("cart repository: user id is required")
	}

	doc, err := r.base.Get(ctx, uid)
	if err != nil {
		return domain.Cart{}, err
	}

	cart := domain.Cart{
		ID:                doc.ID,
		UserID:            doc.ID,
		Currency:          strings.ToUpper(strings.TrimSpace(doc.Data.Currency)),
		ShippingAddressID: strings.TrimSpace(doc.Data.ShippingAddressID),
		BillingAddressID:  strings.TrimSpace(doc.Data.BillingAddressID),
		Items:             []domain.CartItem{},
		Notes:             strings.TrimSpace(doc.Data.Notes),
		PromotionHint:     strings.TrimSpace(doc.Data.PromotionHint),
		Metadata:          cloneAnyMap(doc.Data.Metadata),
		UpdatedAt: func() time.Time {
			if !doc.UpdateTime.IsZero() {
				return doc.UpdateTime
			}
			return doc.Data.UpdatedAt
		}(),
		CreatedAt: func() time.Time {
			if !doc.Data.CreatedAt.IsZero() {
				return doc.Data.CreatedAt
			}
			return doc.UpdateTime
		}(),
	}

	if doc.Data.Promotion != nil {
		cart.Promotion = &domain.CartPromotion{
			Code:           doc.Data.Promotion.Code,
			DiscountAmount: doc.Data.Promotion.DiscountAmount,
			Applied:        doc.Data.Promotion.Applied,
		}
	}
	if doc.Data.Estimates != nil {
		cart.Estimate = &domain.CartEstimate{
			Subtotal: doc.Data.Estimates.Subtotal,
			Discount: doc.Data.Estimates.Discount,
			Tax:      doc.Data.Estimates.Tax,
			Shipping: doc.Data.Estimates.Shipping,
			Total:    doc.Data.Estimates.Total,
		}
	}

	return cart, nil
}

// ReplaceItems is not yet implemented.
func (r *CartRepository) ReplaceItems(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
	return domain.Cart{}, errors.New("cart repository: replace items not implemented")
}

func firstCartID(cart domain.Cart) string {
	if strings.TrimSpace(cart.ID) != "" {
		return strings.TrimSpace(cart.ID)
	}
	return strings.TrimSpace(cart.UserID)
}

func cloneCart(cart domain.Cart) domain.Cart {
	dup := cart
	if cart.Items != nil {
		dup.Items = make([]domain.CartItem, len(cart.Items))
		copy(dup.Items, cart.Items)
	}
	if cart.Metadata != nil {
		dup.Metadata = cloneAnyMap(cart.Metadata)
	}
	return dup
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]any, len(values))
	for k, v := range values {
		out[k] = v
	}
	return out
}

type cartDocument struct {
	Currency          string                 `firestore:"currency"`
	ShippingAddressID string                 `firestore:"shippingAddressId,omitempty"`
	BillingAddressID  string                 `firestore:"billingAddressId,omitempty"`
	Notes             string                 `firestore:"notes,omitempty"`
	PromotionHint     string                 `firestore:"promotionHint,omitempty"`
	Promotion         *cartPromotionDocument `firestore:"promo,omitempty"`
	Estimates         *cartEstimateDocument  `firestore:"estimates,omitempty"`
	Metadata          map[string]any         `firestore:"metadata,omitempty"`
	ItemsCount        int                    `firestore:"itemsCount"`
	CreatedAt         time.Time              `firestore:"createdAt"`
	UpdatedAt         time.Time              `firestore:"updatedAt"`
}

type cartPromotionDocument struct {
	Code           string `firestore:"code"`
	DiscountAmount int64  `firestore:"discountAmount"`
	Applied        bool   `firestore:"applied"`
}

type cartEstimateDocument struct {
	Subtotal int64 `firestore:"subtotal"`
	Discount int64 `firestore:"discount"`
	Tax      int64 `firestore:"tax"`
	Shipping int64 `firestore:"shipping"`
	Total    int64 `firestore:"total"`
}

var _ repositories.CartRepository = (*CartRepository)(nil)
