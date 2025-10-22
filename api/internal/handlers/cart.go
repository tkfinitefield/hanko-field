package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

// CartHandlers exposes authenticated cart endpoints for the current user.
type CartHandlers struct {
	authn *auth.Authenticator
	carts services.CartService
}

// NewCartHandlers constructs handlers enforcing Firebase authentication before invoking the cart service.
func NewCartHandlers(authn *auth.Authenticator, carts services.CartService) *CartHandlers {
	return &CartHandlers{
		authn: authn,
		carts: carts,
	}
}

// Routes wires the /cart endpoints onto the provided router.
func (h *CartHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth())
	}
	r.Get("/", h.getCart)
}

func (h *CartHandlers) getCart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.carts == nil {
		httpx.WriteError(ctx, w, httpx.NewError("cart_service_unavailable", "cart service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	cart, err := h.carts.GetOrCreateCart(ctx, identity.UID)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	payload := cartResponse{Cart: buildCartPayload(cart)}
	setCartResponseHeaders(w, cart)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) writeCartError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, services.ErrCartInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrCartNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("cart_not_found", "cart not found", http.StatusNotFound))
	case errors.Is(err, services.ErrCartConflict):
		httpx.WriteError(ctx, w, httpx.NewError("cart_conflict", "cart has been modified; refresh and retry", http.StatusConflict))
	case errors.Is(err, services.ErrCartUnavailable):
		httpx.WriteError(ctx, w, httpx.NewError("cart_service_unavailable", "cart service is unavailable", http.StatusServiceUnavailable))
	case errors.Is(err, services.ErrCartPricingInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_cart_state", err.Error(), http.StatusBadRequest))
	default:
		httpx.WriteError(ctx, w, httpx.NewError("cart_error", "failed to fetch cart", http.StatusInternalServerError))
	}
}

func setCartResponseHeaders(w http.ResponseWriter, cart services.Cart) {
	cacheControl := "no-store, no-cache, max-age=0, must-revalidate"
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Pragma", "no-cache")
	if !cart.UpdatedAt.IsZero() {
		w.Header().Set("Last-Modified", cart.UpdatedAt.UTC().Format(http.TimeFormat))
	}
	if etag := buildCartETag(cart); etag != "" {
		w.Header().Set("ETag", etag)
	}
}

func buildCartPayload(cart services.Cart) cartPayload {
	payload := cartPayload{
		ID:         strings.TrimSpace(cart.ID),
		UserID:     strings.TrimSpace(cart.UserID),
		Currency:   strings.ToUpper(strings.TrimSpace(cart.Currency)),
		ItemsCount: len(cart.Items),
		Items:      buildCartItems(cart.Items),
		Metadata:   cloneMap(cart.Metadata),
	}

	if cart.Promotion != nil {
		payload.Promotion = &cartPromotionPayload{
			Code:           strings.TrimSpace(cart.Promotion.Code),
			DiscountAmount: cart.Promotion.DiscountAmount,
			Applied:        cart.Promotion.Applied,
		}
	}
	if cart.Estimate != nil {
		payload.Estimate = &cartEstimatePayload{
			Subtotal: cart.Estimate.Subtotal,
			Discount: cart.Estimate.Discount,
			Tax:      cart.Estimate.Tax,
			Shipping: cart.Estimate.Shipping,
			Total:    cart.Estimate.Total,
		}
	}
	if cart.ShippingAddress != nil {
		addr := buildAddressPayload(*cart.ShippingAddress)
		payload.ShippingAddress = &addr
	}
	if cart.BillingAddress != nil {
		addr := buildAddressPayload(*cart.BillingAddress)
		payload.BillingAddress = &addr
	}
	if cart.Metadata == nil {
		payload.Metadata = nil
	}
	if !cart.UpdatedAt.IsZero() {
		payload.UpdatedAt = formatTime(cart.UpdatedAt)
	}

	return payload
}

func buildCartItems(items []services.CartItem) []cartItemPayload {
	if len(items) == 0 {
		return []cartItemPayload{}
	}

	payload := make([]cartItemPayload, 0, len(items))
	for _, item := range items {
		entry := cartItemPayload{
			ID:               strings.TrimSpace(item.ID),
			ProductID:        strings.TrimSpace(item.ProductID),
			SKU:              strings.TrimSpace(item.SKU),
			Quantity:         item.Quantity,
			UnitPrice:        item.UnitPrice,
			Currency:         strings.ToUpper(strings.TrimSpace(item.Currency)),
			WeightGrams:      item.WeightGrams,
			TaxCode:          strings.TrimSpace(item.TaxCode),
			RequiresShipping: item.RequiresShipping,
			Customization:    cloneMap(item.Customization),
			Metadata:         cloneMap(item.Metadata),
			Estimates:        cloneInt64Map(item.Estimates),
		}
		if item.DesignRef != nil {
			entry.DesignRef = cloneStringPointer(item.DesignRef)
		}
		if !item.AddedAt.IsZero() {
			entry.AddedAt = formatTime(item.AddedAt)
		}
		if item.UpdatedAt != nil && !item.UpdatedAt.IsZero() {
			entry.UpdatedAt = formatTime(*item.UpdatedAt)
		}
		payload = append(payload, entry)
	}
	return payload
}

func buildCartETag(cart services.Cart) string {
	if strings.TrimSpace(cart.ID) == "" || cart.UpdatedAt.IsZero() {
		return ""
	}
	input := fmt.Sprintf("%s:%d", strings.TrimSpace(cart.ID), cart.UpdatedAt.UTC().UnixNano())
	sum := sha256.Sum256([]byte(input))
	token := hex.EncodeToString(sum[:8])
	return fmt.Sprintf(`W/"%s"`, token)
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

type cartResponse struct {
	Cart cartPayload `json:"cart"`
}

type cartPayload struct {
	ID              string                `json:"id"`
	UserID          string                `json:"user_id"`
	Currency        string                `json:"currency"`
	ItemsCount      int                   `json:"items_count"`
	Items           []cartItemPayload     `json:"items"`
	Promotion       *cartPromotionPayload `json:"promotion,omitempty"`
	Estimate        *cartEstimatePayload  `json:"estimate,omitempty"`
	ShippingAddress *addressPayload       `json:"shipping_address,omitempty"`
	BillingAddress  *addressPayload       `json:"billing_address,omitempty"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
	UpdatedAt       string                `json:"updated_at,omitempty"`
}

type cartPromotionPayload struct {
	Code           string `json:"code"`
	DiscountAmount int64  `json:"discount_amount"`
	Applied        bool   `json:"applied"`
}

type cartEstimatePayload struct {
	Subtotal int64 `json:"subtotal"`
	Discount int64 `json:"discount"`
	Tax      int64 `json:"tax"`
	Shipping int64 `json:"shipping"`
	Total    int64 `json:"total"`
}

type cartItemPayload struct {
	ID               string           `json:"id"`
	ProductID        string           `json:"product_id"`
	SKU              string           `json:"sku"`
	DesignRef        *string          `json:"design_ref,omitempty"`
	Quantity         int              `json:"quantity"`
	UnitPrice        int64            `json:"unit_price"`
	Currency         string           `json:"currency"`
	WeightGrams      int              `json:"weight_grams,omitempty"`
	TaxCode          string           `json:"tax_code,omitempty"`
	RequiresShipping bool             `json:"requires_shipping"`
	Customization    map[string]any   `json:"customization,omitempty"`
	Metadata         map[string]any   `json:"metadata,omitempty"`
	Estimates        map[string]int64 `json:"estimates,omitempty"`
	AddedAt          string           `json:"added_at,omitempty"`
	UpdatedAt        string           `json:"updated_at,omitempty"`
}
