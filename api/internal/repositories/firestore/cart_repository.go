package firestore

import (
	"context"
	"errors"
	"strings"
	"time"

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
func (r *CartRepository) UpsertCart(ctx context.Context, cart domain.Cart) (domain.Cart, error) {
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

	doc := cartDocument{
		Currency:   strings.ToUpper(strings.TrimSpace(cart.Currency)),
		Metadata:   cloneAnyMap(cart.Metadata),
		ItemsCount: len(cart.Items),
		UpdatedAt:  now,
		CreatedAt:  now,
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

	result, err := r.base.Set(ctx, cartID, doc)
	if err != nil {
		return domain.Cart{}, err
	}

	saved := cloneCart(cart)
	saved.ID = cartID
	saved.UserID = cartID
	saved.Currency = doc.Currency
	saved.Metadata = cloneAnyMap(cart.Metadata)
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
		ID:       doc.ID,
		UserID:   doc.ID,
		Currency: strings.ToUpper(strings.TrimSpace(doc.Data.Currency)),
		Items:    []domain.CartItem{},
		Metadata: cloneAnyMap(doc.Data.Metadata),
		UpdatedAt: func() time.Time {
			if !doc.UpdateTime.IsZero() {
				return doc.UpdateTime
			}
			return doc.Data.UpdatedAt
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
	Currency   string                 `firestore:"currency"`
	Promotion  *cartPromotionDocument `firestore:"promo,omitempty"`
	Estimates  *cartEstimateDocument  `firestore:"estimates,omitempty"`
	Metadata   map[string]any         `firestore:"metadata,omitempty"`
	ItemsCount int                    `firestore:"itemsCount"`
	CreatedAt  time.Time              `firestore:"createdAt"`
	UpdatedAt  time.Time              `firestore:"updatedAt"`
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
