package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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

const maxCartBodySize = 16 * 1024

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
	r.Patch("/", h.patchCart)
	r.Get("/items", h.listItems)
	r.Post("/items", h.createItem)
	r.Put("/items/{itemId}", h.updateItem)
	r.Delete("/items/{itemId}", h.deleteItem)
}

// RegisterStandaloneRoutes mounts cart endpoints that are not nested under /cart.
func (h *CartHandlers) RegisterStandaloneRoutes(r chi.Router) {
	if r == nil {
		return
	}
	group := r
	if h.authn != nil {
		group = group.With(h.authn.RequireFirebaseAuth())
	}
	group.Post("/cart:estimate", h.estimateCart)
	group.Post("/cart:apply-promo", h.applyPromotion)
	group.Delete("/cart:remove-promo", h.removePromotion)
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

func (h *CartHandlers) patchCart(w http.ResponseWriter, r *http.Request) {
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

	body, err := readLimitedBody(r, maxCartBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	updateReq, err := parseUpdateCartRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cmd := services.UpdateCartCommand{UserID: identity.UID}
	if updateReq.currency != nil {
		cmd.Currency = updateReq.currency
	}
	if updateReq.shippingSet {
		cmd.ShippingAddressID = updateReq.shippingAddressID
	}
	if updateReq.billingSet {
		cmd.BillingAddressID = updateReq.billingAddressID
	}
	if updateReq.notesSet {
		cmd.Notes = updateReq.notes
	}
	if updateReq.promotionSet {
		cmd.PromotionHint = updateReq.promotionHint
	}

	expected := updateReq.updatedAt
	if expected == nil {
		if ifUnmodified := strings.TrimSpace(r.Header.Get("If-Unmodified-Since")); ifUnmodified != "" {
			parsed, parseErr := time.Parse(http.TimeFormat, ifUnmodified)
			if parseErr != nil {
				httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "If-Unmodified-Since must be a valid HTTP-date", http.StatusBadRequest))
				return
			}
			expected = &parsed
			updateReq.versionFromHeader = true
		}
	}
	cmd.ExpectedUpdatedAt = expected
	cmd.ExpectedFromHeader = updateReq.versionFromHeader

	updated, err := h.carts.UpdateCart(ctx, cmd)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	payload := cartResponse{Cart: buildCartPayload(updated)}
	setCartResponseHeaders(w, updated)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) estimateCart(w http.ResponseWriter, r *http.Request) {
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

	body, err := readLimitedBody(r, maxCartBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	req, err := parseCartEstimateRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cmd := services.CartEstimateCommand{UserID: identity.UID}
	if req.shippingSet {
		cmd.ShippingAddressID = req.shippingAddressID
	}
	if req.billingSet {
		cmd.BillingAddressID = req.billingAddressID
	}
	if req.promotionSet {
		cmd.PromotionCode = req.promotionCode
	}
	if req.bypassSet {
		cmd.BypassShippingCache = req.bypassShippingCache
	}

	result, err := h.carts.Estimate(ctx, cmd)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setEstimateResponseHeaders(w)
	payload := buildCartEstimatePayload(result)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) applyPromotion(w http.ResponseWriter, r *http.Request) {
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

	body, err := readLimitedBody(r, maxCartBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	var req applyPromotionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON payload", http.StatusBadRequest))
		return
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "code is required", http.StatusBadRequest))
		return
	}

	cmd := services.CartPromotionCommand{
		UserID: identity.UID,
		Code:   code,
	}
	if req.Source != nil {
		cmd.Source = strings.TrimSpace(*req.Source)
	}
	if idempotency := strings.TrimSpace(r.Header.Get("Idempotency-Key")); idempotency != "" {
		cmd.IdempotencyKey = idempotency
	}

	cart, err := h.carts.ApplyPromotion(ctx, cmd)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setCartResponseHeaders(w, cart)
	payload := cartResponse{Cart: buildCartPayload(cart)}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) removePromotion(w http.ResponseWriter, r *http.Request) {
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

	cart, err := h.carts.RemovePromotion(ctx, identity.UID)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setCartResponseHeaders(w, cart)
	payload := cartResponse{Cart: buildCartPayload(cart)}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) listItems(w http.ResponseWriter, r *http.Request) {
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

	setCartResponseHeaders(w, cart)
	payload := cartItemsResponse{Items: buildCartItems(cart.Items)}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) createItem(w http.ResponseWriter, r *http.Request) {
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

	body, err := readLimitedBody(r, maxCartBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	cmd, err := parseCartItemCommand(identity.UID, "", body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cart, err := h.carts.AddOrUpdateItem(ctx, cmd)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setCartResponseHeaders(w, cart)
	payload := cartResponse{Cart: buildCartPayload(cart)}
	writeJSONResponse(w, http.StatusCreated, payload)
}

func (h *CartHandlers) updateItem(w http.ResponseWriter, r *http.Request) {
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

	itemID := strings.TrimSpace(chi.URLParam(r, "itemId"))
	if itemID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "item_id is required", http.StatusBadRequest))
		return
	}

	body, err := readLimitedBody(r, maxCartBodySize)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	cmd, err := parseCartItemCommand(identity.UID, itemID, body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cart, err := h.carts.AddOrUpdateItem(ctx, cmd)
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setCartResponseHeaders(w, cart)
	payload := cartResponse{Cart: buildCartPayload(cart)}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CartHandlers) deleteItem(w http.ResponseWriter, r *http.Request) {
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

	itemID := strings.TrimSpace(chi.URLParam(r, "itemId"))
	if itemID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "item_id is required", http.StatusBadRequest))
		return
	}

	cart, err := h.carts.RemoveItem(ctx, services.RemoveCartItemCommand{UserID: identity.UID, ItemID: itemID})
	if err != nil {
		h.writeCartError(ctx, w, err)
		return
	}

	setCartResponseHeaders(w, cart)
	payload := cartResponse{Cart: buildCartPayload(cart)}
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

func setEstimateResponseHeaders(w http.ResponseWriter) {
	cacheControl := "no-store, no-cache, max-age=0, must-revalidate"
	w.Header().Set("Cache-Control", cacheControl)
	w.Header().Set("Pragma", "no-cache")
}

func buildCartEstimatePayload(result services.CartEstimateResult) cartEstimateResultPayload {
	payload := cartEstimateResultPayload{
		Currency: strings.ToUpper(strings.TrimSpace(result.Currency)),
		Subtotal: result.Estimate.Subtotal,
		Discount: result.Estimate.Discount,
		Tax:      result.Estimate.Tax,
		Shipping: result.Estimate.Shipping,
		Total:    result.Estimate.Total,
	}

	if result.Promotion != nil {
		payload.Promotion = &cartPromotionPayload{
			Code:           strings.ToUpper(strings.TrimSpace(result.Promotion.Code)),
			DiscountAmount: result.Promotion.DiscountAmount,
			Applied:        result.Promotion.Applied,
		}
	}

	if len(result.Breakdown.Items) > 0 {
		items := make([]cartEstimateItemPayload, 0, len(result.Breakdown.Items))
		for _, item := range result.Breakdown.Items {
			entry := cartEstimateItemPayload{
				ItemID:   strings.TrimSpace(item.ItemID),
				Currency: strings.ToUpper(strings.TrimSpace(firstNonEmpty(item.Currency, result.Currency))),
				Subtotal: item.Subtotal,
				Discount: item.Discount,
				Tax:      item.Tax,
				Shipping: item.Shipping,
				Total:    item.Total,
				Metadata: cloneMap(item.Metadata),
			}
			items = append(items, entry)
		}
		payload.Items = items
	} else {
		payload.Items = []cartEstimateItemPayload{}
	}

	if len(result.Breakdown.Discounts) > 0 {
		discounts := make([]cartEstimateDiscountPayload, 0, len(result.Breakdown.Discounts))
		for _, disc := range result.Breakdown.Discounts {
			discounts = append(discounts, cartEstimateDiscountPayload{
				Type:        strings.TrimSpace(disc.Type),
				Code:        strings.TrimSpace(disc.Code),
				Source:      strings.TrimSpace(disc.Source),
				Description: strings.TrimSpace(disc.Description),
				Amount:      disc.Amount,
				Metadata:    cloneMap(disc.Metadata),
			})
		}
		payload.Discounts = discounts
	} else {
		payload.Discounts = []cartEstimateDiscountPayload{}
	}

	if len(result.Breakdown.Taxes) > 0 {
		taxes := make([]cartEstimateTaxPayload, 0, len(result.Breakdown.Taxes))
		for _, tax := range result.Breakdown.Taxes {
			taxes = append(taxes, cartEstimateTaxPayload{
				Name:         strings.TrimSpace(tax.Name),
				Jurisdiction: strings.TrimSpace(tax.Jurisdiction),
				Rate:         tax.Rate,
				Amount:       tax.Amount,
				Metadata:     cloneMap(tax.Metadata),
			})
		}
		payload.Taxes = taxes
	} else {
		payload.Taxes = []cartEstimateTaxPayload{}
	}

	if len(result.Breakdown.ShippingDetails) > 0 {
		ship := make([]cartEstimateShippingPayload, 0, len(result.Breakdown.ShippingDetails))
		for _, detail := range result.Breakdown.ShippingDetails {
			ship = append(ship, cartEstimateShippingPayload{
				ServiceLevel: strings.TrimSpace(detail.ServiceLevel),
				Carrier:      strings.TrimSpace(detail.Carrier),
				Amount:       detail.Amount,
				Currency:     strings.ToUpper(strings.TrimSpace(firstNonEmpty(detail.Currency, result.Currency))),
				EstimateDays: detail.EstimateDays,
				Metadata:     cloneMap(detail.Metadata),
			})
		}
		payload.ShippingDetails = ship
	} else {
		payload.ShippingDetails = []cartEstimateShippingPayload{}
	}

	if len(result.Warnings) > 0 {
		warnings := make([]string, 0, len(result.Warnings))
		for _, warning := range result.Warnings {
			warnings = append(warnings, string(warning))
		}
		payload.Warnings = warnings
	}

	if meta := cloneMap(result.Breakdown.Metadata); meta != nil {
		payload.Metadata = meta
	}

	return payload
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

	if note := strings.TrimSpace(cart.Notes); note != "" {
		payload.Notes = note
	}
	if hint := strings.TrimSpace(cart.PromotionHint); hint != "" {
		payload.PromotionHint = hint
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

type cartItemsResponse struct {
	Items []cartItemPayload `json:"items"`
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
	Notes           string                `json:"notes,omitempty"`
	PromotionHint   string                `json:"promotion_hint,omitempty"`
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

type cartEstimateResultPayload struct {
	Currency        string                        `json:"currency"`
	Subtotal        int64                         `json:"subtotal"`
	Discount        int64                         `json:"discount"`
	Tax             int64                         `json:"tax"`
	Shipping        int64                         `json:"shipping"`
	Total           int64                         `json:"total"`
	Promotion       *cartPromotionPayload         `json:"promotion,omitempty"`
	Items           []cartEstimateItemPayload     `json:"items,omitempty"`
	Discounts       []cartEstimateDiscountPayload `json:"discounts,omitempty"`
	Taxes           []cartEstimateTaxPayload      `json:"taxes,omitempty"`
	ShippingDetails []cartEstimateShippingPayload `json:"shipping_details,omitempty"`
	Warnings        []string                      `json:"warnings,omitempty"`
	Metadata        map[string]any                `json:"metadata,omitempty"`
}

type cartEstimateItemPayload struct {
	ItemID   string         `json:"item_id"`
	Currency string         `json:"currency"`
	Subtotal int64          `json:"subtotal"`
	Discount int64          `json:"discount"`
	Tax      int64          `json:"tax"`
	Shipping int64          `json:"shipping"`
	Total    int64          `json:"total"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type cartEstimateDiscountPayload struct {
	Type        string         `json:"type"`
	Code        string         `json:"code,omitempty"`
	Source      string         `json:"source,omitempty"`
	Description string         `json:"description,omitempty"`
	Amount      int64          `json:"amount"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type cartEstimateTaxPayload struct {
	Name         string         `json:"name"`
	Jurisdiction string         `json:"jurisdiction,omitempty"`
	Rate         float64        `json:"rate,omitempty"`
	Amount       int64          `json:"amount"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type cartEstimateShippingPayload struct {
	ServiceLevel string         `json:"service_level,omitempty"`
	Carrier      string         `json:"carrier,omitempty"`
	Amount       int64          `json:"amount"`
	Currency     string         `json:"currency,omitempty"`
	EstimateDays *int           `json:"estimate_days,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
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

type applyPromotionRequest struct {
	Code   string  `json:"code"`
	Source *string `json:"source,omitempty"`
}

type updateCartRequest struct {
	currency          *string
	shippingAddressID *string
	shippingSet       bool
	billingAddressID  *string
	billingSet        bool
	notes             *string
	notesSet          bool
	promotionHint     *string
	promotionSet      bool
	updatedAt         *time.Time
	versionFromHeader bool
}

type cartEstimateRequest struct {
	shippingAddressID   *string
	shippingSet         bool
	billingAddressID    *string
	billingSet          bool
	promotionCode       *string
	promotionSet        bool
	bypassShippingCache bool
	bypassSet           bool
}

type cartItemRequest struct {
	ProductID     string         `json:"product_id"`
	SKU           string         `json:"sku"`
	Quantity      *int           `json:"quantity"`
	UnitPrice     *int64         `json:"unit_price"`
	Currency      *string        `json:"currency"`
	Customization map[string]any `json:"customization"`
	DesignID      *string        `json:"design_id"`
}

func parseUpdateCartRequest(body []byte) (updateCartRequest, error) {
	var req updateCartRequest
	if len(strings.TrimSpace(string(body))) == 0 {
		return req, errEmptyBody
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return req, errors.New("invalid JSON payload")
	}

	for key, value := range raw {
		switch key {
		case "currency":
			if isJSONNull(value) {
				return req, errors.New("currency must be a string")
			}
			var currency string
			if err := json.Unmarshal(value, &currency); err != nil {
				return req, errors.New("currency must be a string")
			}
			currency = strings.TrimSpace(currency)
			if currency == "" {
				return req, errors.New("currency must not be empty")
			}
			req.currency = &currency
		case "shipping_address_id":
			req.shippingSet = true
			if isJSONNull(value) {
				empty := ""
				req.shippingAddressID = &empty
				continue
			}
			var id string
			if err := json.Unmarshal(value, &id); err != nil {
				return req, errors.New("shipping_address_id must be a string or null")
			}
			trimmed := strings.TrimSpace(id)
			req.shippingAddressID = &trimmed
		case "billing_address_id":
			req.billingSet = true
			if isJSONNull(value) {
				empty := ""
				req.billingAddressID = &empty
				continue
			}
			var id string
			if err := json.Unmarshal(value, &id); err != nil {
				return req, errors.New("billing_address_id must be a string or null")
			}
			trimmed := strings.TrimSpace(id)
			req.billingAddressID = &trimmed
		case "notes":
			req.notesSet = true
			if isJSONNull(value) {
				empty := ""
				req.notes = &empty
				continue
			}
			var note string
			if err := json.Unmarshal(value, &note); err != nil {
				return req, errors.New("notes must be a string or null")
			}
			req.notes = &note
		case "promotion_hint":
			req.promotionSet = true
			if isJSONNull(value) {
				empty := ""
				req.promotionHint = &empty
				continue
			}
			var hint string
			if err := json.Unmarshal(value, &hint); err != nil {
				return req, errors.New("promotion_hint must be a string or null")
			}
			req.promotionHint = &hint
		case "updated_at":
			if isJSONNull(value) {
				req.updatedAt = nil
				continue
			}
			var ts string
			if err := json.Unmarshal(value, &ts); err != nil {
				return req, errors.New("updated_at must be a string")
			}
			parsed, err := parseRFC3339(strings.TrimSpace(ts))
			if err != nil {
				return req, fmt.Errorf("updated_at must be RFC3339 timestamp: %w", err)
			}
			req.updatedAt = &parsed
		default:
			return req, fmt.Errorf("field %q is not editable", key)
		}
	}

	if req.currency == nil && !req.shippingSet && !req.billingSet && !req.notesSet && !req.promotionSet {
		return req, errNoEditableFields
	}

	return req, nil
}

func parseCartEstimateRequest(body []byte) (cartEstimateRequest, error) {
	var req cartEstimateRequest
	if len(body) == 0 || len(strings.TrimSpace(string(body))) == 0 {
		return req, nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return req, errors.New("invalid JSON payload")
	}

	for key, value := range raw {
		switch key {
		case "shipping_address_id":
			req.shippingSet = true
			if isJSONNull(value) {
				empty := ""
				req.shippingAddressID = &empty
				continue
			}
			var id string
			if err := json.Unmarshal(value, &id); err != nil {
				return req, errors.New("shipping_address_id must be a string or null")
			}
			trimmed := strings.TrimSpace(id)
			copy := trimmed
			req.shippingAddressID = &copy
		case "billing_address_id":
			req.billingSet = true
			if isJSONNull(value) {
				empty := ""
				req.billingAddressID = &empty
				continue
			}
			var id string
			if err := json.Unmarshal(value, &id); err != nil {
				return req, errors.New("billing_address_id must be a string or null")
			}
			trimmed := strings.TrimSpace(id)
			copy := trimmed
			req.billingAddressID = &copy
		case "promotion_code":
			req.promotionSet = true
			if isJSONNull(value) {
				empty := ""
				req.promotionCode = &empty
				continue
			}
			var code string
			if err := json.Unmarshal(value, &code); err != nil {
				return req, errors.New("promotion_code must be a string or null")
			}
			trimmed := strings.TrimSpace(code)
			copy := trimmed
			req.promotionCode = &copy
		case "bypass_shipping_cache":
			if isJSONNull(value) {
				return req, errors.New("bypass_shipping_cache must be a boolean")
			}
			var flag bool
			if err := json.Unmarshal(value, &flag); err != nil {
				return req, errors.New("bypass_shipping_cache must be a boolean")
			}
			req.bypassSet = true
			req.bypassShippingCache = flag
		}
	}

	return req, nil
}

func parseCartItemCommand(userID string, itemID string, body []byte) (services.UpsertCartItemCommand, error) {
	if len(strings.TrimSpace(string(body))) == 0 {
		return services.UpsertCartItemCommand{}, errEmptyBody
	}

	var req cartItemRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return services.UpsertCartItemCommand{}, errors.New("invalid JSON payload")
	}

	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		return services.UpsertCartItemCommand{}, errors.New("product_id is required")
	}

	sku := strings.TrimSpace(req.SKU)
	if sku == "" {
		return services.UpsertCartItemCommand{}, errors.New("sku is required")
	}

	if req.Quantity == nil {
		return services.UpsertCartItemCommand{}, errors.New("quantity is required")
	}

	if req.UnitPrice == nil {
		return services.UpsertCartItemCommand{}, errors.New("unit_price is required")
	}

	cmd := services.UpsertCartItemCommand{
		UserID:        userID,
		ProductID:     productID,
		SKU:           sku,
		Quantity:      *req.Quantity,
		UnitPrice:     *req.UnitPrice,
		Customization: req.Customization,
	}

	if req.Currency != nil {
		cmd.Currency = strings.TrimSpace(*req.Currency)
	}

	if strings.TrimSpace(itemID) != "" {
		id := strings.TrimSpace(itemID)
		cmd.ItemID = &id
	}

	if req.DesignID != nil {
		designID := strings.TrimSpace(*req.DesignID)
		if designID != "" {
			cmd.DesignID = &designID
		}
	}

	return cmd, nil
}
