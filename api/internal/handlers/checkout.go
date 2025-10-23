package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

const maxCheckoutRequestBody = 8 * 1024

// CheckoutHandlers exposes checkout related endpoints for authenticated users.
type CheckoutHandlers struct {
	authn    *auth.Authenticator
	checkout services.CheckoutService
}

// NewCheckoutHandlers constructs checkout handlers guarded by Firebase authentication.
func NewCheckoutHandlers(authn *auth.Authenticator, checkout services.CheckoutService) *CheckoutHandlers {
	return &CheckoutHandlers{
		authn:    authn,
		checkout: checkout,
	}
}

// Routes registers checkout endpoints under the provided router.
func (h *CheckoutHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	group := r
	if h.authn != nil {
		group = group.With(h.authn.RequireFirebaseAuth())
	}
	group.Post("/checkout/session", h.createSession)
	group.Post("/checkout/confirm", h.confirmCheckout)
}

type checkoutSessionRequest struct {
	Provider   string            `json:"provider"`
	SuccessURL string            `json:"successUrl"`
	CancelURL  string            `json:"cancelUrl"`
	Metadata   map[string]string `json:"metadata"`
}

type checkoutSessionResponse struct {
	SessionID    string `json:"sessionId"`
	Provider     string `json:"provider"`
	URL          string `json:"url"`
	ClientSecret string `json:"clientSecret,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
}

type checkoutConfirmRequest struct {
	SessionID       string `json:"sessionId"`
	PaymentIntentID string `json:"paymentIntentId"`
	OrderID         string `json:"orderId"`
}

type checkoutConfirmResponse struct {
	Status  string `json:"status"`
	OrderID string `json:"orderId,omitempty"`
}

func (h *CheckoutHandlers) createSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.checkout == nil {
		httpx.WriteError(ctx, w, httpx.NewError("checkout_unavailable", "checkout service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxCheckoutRequestBody)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), status))
		return
	}

	var req checkoutSessionRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body must be valid JSON", http.StatusBadRequest))
			return
		}
	}

	successURL := strings.TrimSpace(req.SuccessURL)
	cancelURL := strings.TrimSpace(req.CancelURL)
	if successURL == "" || cancelURL == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "successUrl and cancelUrl are required", http.StatusBadRequest))
		return
	}

	metadata := make(map[string]string, len(req.Metadata))
	for k, v := range req.Metadata {
		key := strings.TrimSpace(k)
		value := strings.TrimSpace(v)
		if key == "" || value == "" {
			continue
		}
		metadata[key] = value
	}

	cmd := services.CreateCheckoutSessionCommand{
		UserID:     identity.UID,
		CartID:     identity.UID,
		SuccessURL: successURL,
		CancelURL:  cancelURL,
		PSP:        strings.TrimSpace(req.Provider),
		Metadata:   metadata,
	}

	session, err := h.checkout.CreateCheckoutSession(ctx, cmd)
	if err != nil {
		h.writeCheckoutError(ctx, w, err)
		return
	}

	payload := checkoutSessionResponse{
		SessionID:    session.SessionID,
		Provider:     session.PSP,
		URL:          session.RedirectURL,
		ClientSecret: session.ClientSecret,
	}
	if !session.ExpiresAt.IsZero() {
		payload.ExpiresAt = session.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *CheckoutHandlers) confirmCheckout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.checkout == nil {
		httpx.WriteError(ctx, w, httpx.NewError("checkout_unavailable", "checkout service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxCheckoutRequestBody)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), status))
		return
	}

	var req checkoutConfirmRequest
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body must be valid JSON", http.StatusBadRequest))
			return
		}
	}

	sessionID := strings.TrimSpace(req.SessionID)
	if sessionID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "sessionId is required", http.StatusBadRequest))
		return
	}

	cmd := services.ConfirmCheckoutCommand{
		UserID:          identity.UID,
		SessionID:       sessionID,
		PaymentIntentID: strings.TrimSpace(req.PaymentIntentID),
		OrderID:         strings.TrimSpace(req.OrderID),
	}

	result, err := h.checkout.ConfirmClientCompletion(ctx, cmd)
	if err != nil {
		h.writeCheckoutError(ctx, w, err)
		return
	}

	resp := checkoutConfirmResponse{
		Status:  strings.TrimSpace(result.Status),
		OrderID: strings.TrimSpace(result.OrderID),
	}

	writeJSONResponse(w, http.StatusOK, resp)
}

func (h *CheckoutHandlers) writeCheckoutError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrCheckoutInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrCheckoutCartNotReady):
		httpx.WriteError(ctx, w, httpx.NewError("cart_not_ready", "cart is not ready for checkout", http.StatusConflict))
	case errors.Is(err, services.ErrCheckoutInsufficientStock):
		httpx.WriteError(ctx, w, httpx.NewError("insufficient_stock", "insufficient stock to reserve items", http.StatusConflict))
	case errors.Is(err, services.ErrCheckoutConflict):
		httpx.WriteError(ctx, w, httpx.NewError("checkout_conflict", "cart has changed; refresh and retry", http.StatusConflict))
	case errors.Is(err, services.ErrCheckoutPaymentFailed):
		httpx.WriteError(ctx, w, httpx.NewError("payment_failed", "payment could not be completed", http.StatusBadGateway))
	case errors.Is(err, services.ErrCheckoutUnavailable):
		httpx.WriteError(ctx, w, httpx.NewError("checkout_unavailable", "checkout service unavailable", http.StatusServiceUnavailable))
	default:
		httpx.WriteError(ctx, w, httpx.NewError("checkout_error", "failed to process checkout request", http.StatusInternalServerError))
	}
}
