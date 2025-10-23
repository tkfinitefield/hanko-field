package firestore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domain "github.com/hanko-field/api/internal/domain"
	pfirestore "github.com/hanko-field/api/internal/platform/firestore"
	"github.com/hanko-field/api/internal/repositories"
)

const (
	cartCollection      = "carts"
	cartItemsCollection = "items"
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

	docRef, err := r.base.DocumentRef(ctx, uid)
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

	itemsIter := docRef.Collection(cartItemsCollection).OrderBy("addedAt", firestore.Asc).Documents(ctx)
	defer itemsIter.Stop()

	items := make([]domain.CartItem, 0, doc.Data.ItemsCount)
	for {
		snap, err := itemsIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return domain.Cart{}, fmt.Errorf("cart repository: fetch items: %w", err)
		}
		var itemDoc cartItemDocument
		if err := snap.DataTo(&itemDoc); err != nil {
			return domain.Cart{}, fmt.Errorf("cart repository: decode item %s: %w", snap.Ref.ID, err)
		}
		items = append(items, itemDoc.toDomain(snap.Ref.ID))
	}
	cart.Items = items

	return cart, nil
}

func (r *CartRepository) ReplaceItems(ctx context.Context, userID string, items []domain.CartItem) (domain.Cart, error) {
	if r == nil || r.provider == nil {
		return domain.Cart{}, errors.New("cart repository not initialised")
	}
	uid := strings.TrimSpace(userID)
	if uid == "" {
		return domain.Cart{}, errors.New("cart repository: user id is required")
	}

	client, err := r.provider.Client(ctx)
	if err != nil {
		return domain.Cart{}, err
	}

	cartRef := client.Collection(cartCollection).Doc(uid)
	itemsRef := cartRef.Collection(cartItemsCollection)
	now := time.Now().UTC()

	newDocs := make(map[string]cartItemDocument, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			return domain.Cart{}, errors.New("cart repository: item id is required")
		}
		doc := newCartItemDocument(item)
		if doc.AddedAt.IsZero() {
			doc.AddedAt = now
		} else {
			doc.AddedAt = doc.AddedAt.UTC()
		}
		if doc.UpdatedAt != nil {
			ts := doc.UpdatedAt.UTC()
			doc.UpdatedAt = &ts
		}
		newDocs[id] = doc
	}

	err = r.provider.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		if _, err := tx.Get(cartRef); err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
		}

		existing := make(map[string]struct{})
		iter := tx.Documents(itemsRef.Query)
		snaps, err := iter.GetAll()
		if err != nil {
			return err
		}
		for _, snap := range snaps {
			existing[snap.Ref.ID] = struct{}{}
		}

		for id, doc := range newDocs {
			ref := itemsRef.Doc(id)
			if err := tx.Set(ref, doc); err != nil {
				return err
			}
			delete(existing, id)
		}

		for id := range existing {
			if err := tx.Delete(itemsRef.Doc(id)); err != nil {
				return err
			}
		}

		updates := []firestore.Update{
			{Path: "itemsCount", Value: len(newDocs)},
			{Path: "updatedAt", Value: now},
		}
		if err := tx.Update(cartRef, updates); err != nil {
			if status.Code(err) == codes.NotFound {
				return tx.Set(cartRef, map[string]any{
					"itemsCount": len(newDocs),
					"updatedAt":  now,
				}, firestore.MergeAll)
			}
			return err
		}

		return nil
	})
	if err != nil {
		return domain.Cart{}, err
	}

	return r.GetCart(ctx, uid)
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

func cloneInt64Map(values map[string]int64) map[string]int64 {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int64, len(values))
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

type cartItemDocument struct {
	ProductRef       string           `firestore:"productRef"`
	DesignRef        *string          `firestore:"designRef,omitempty"`
	SKU              string           `firestore:"sku"`
	Quantity         int              `firestore:"quantity"`
	UnitPrice        int64            `firestore:"unitPrice"`
	Currency         string           `firestore:"currency"`
	WeightGrams      int              `firestore:"weightGrams,omitempty"`
	TaxCode          string           `firestore:"taxCode,omitempty"`
	RequiresShipping bool             `firestore:"requiresShipping"`
	Customization    map[string]any   `firestore:"customization,omitempty"`
	Metadata         map[string]any   `firestore:"metadata,omitempty"`
	Estimates        map[string]int64 `firestore:"estimates,omitempty"`
	AddedAt          time.Time        `firestore:"addedAt"`
	UpdatedAt        *time.Time       `firestore:"updatedAt,omitempty"`
}

func newCartItemDocument(item domain.CartItem) cartItemDocument {
	doc := cartItemDocument{
		ProductRef:       productDocPath(item.ProductID),
		SKU:              strings.TrimSpace(item.SKU),
		Quantity:         item.Quantity,
		UnitPrice:        item.UnitPrice,
		Currency:         strings.ToUpper(strings.TrimSpace(item.Currency)),
		WeightGrams:      item.WeightGrams,
		TaxCode:          strings.TrimSpace(item.TaxCode),
		RequiresShipping: item.RequiresShipping,
		Customization:    cloneAnyMap(item.Customization),
		Metadata:         cloneAnyMap(item.Metadata),
		Estimates:        cloneInt64Map(item.Estimates),
		AddedAt:          item.AddedAt.UTC(),
	}
	if len(doc.Customization) == 0 {
		doc.Customization = nil
	}
	if len(doc.Metadata) == 0 {
		doc.Metadata = nil
	}
	if len(doc.Estimates) == 0 {
		doc.Estimates = nil
	}
	if item.DesignRef != nil && strings.TrimSpace(*item.DesignRef) != "" {
		ref := strings.TrimSpace(*item.DesignRef)
		doc.DesignRef = &ref
	}
	if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
		ts := item.UpdatedAt.UTC()
		doc.UpdatedAt = &ts
	}
	return doc
}

func (doc cartItemDocument) toDomain(id string) domain.CartItem {
	item := domain.CartItem{
		ID:               strings.TrimSpace(id),
		ProductID:        extractProductID(doc.ProductRef),
		SKU:              strings.TrimSpace(doc.SKU),
		Quantity:         doc.Quantity,
		UnitPrice:        doc.UnitPrice,
		Currency:         strings.ToUpper(strings.TrimSpace(doc.Currency)),
		WeightGrams:      doc.WeightGrams,
		TaxCode:          strings.TrimSpace(doc.TaxCode),
		RequiresShipping: doc.RequiresShipping,
		Customization:    cloneAnyMap(doc.Customization),
		Metadata:         cloneAnyMap(doc.Metadata),
		Estimates:        cloneInt64Map(doc.Estimates),
		AddedAt:          doc.AddedAt.UTC(),
	}
	if doc.DesignRef != nil && strings.TrimSpace(*doc.DesignRef) != "" {
		ref := strings.TrimSpace(*doc.DesignRef)
		item.DesignRef = &ref
	}
	if doc.UpdatedAt != nil && !doc.UpdatedAt.IsZero() {
		ts := doc.UpdatedAt.UTC()
		item.UpdatedAt = &ts
	}
	return item
}

func productDocPath(productID string) string {
	trimmed := strings.TrimSpace(productID)
	if trimmed == "" {
		return ""
	}
	for strings.HasPrefix(trimmed, "/") {
		trimmed = strings.TrimPrefix(trimmed, "/")
	}
	if !strings.HasPrefix(trimmed, "products/") {
		trimmed = "products/" + trimmed
	}
	return "/" + trimmed
}

func extractProductID(ref string) string {
	trimmed := strings.TrimSpace(ref)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	const prefix = "products/"
	if strings.HasPrefix(trimmed, prefix) {
		return trimmed[len(prefix):]
	}
	return trimmed
}

var _ repositories.CartRepository = (*CartRepository)(nil)
