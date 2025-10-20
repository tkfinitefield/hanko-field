package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

func (h *MeHandlers) paymentMethodRoutes(r chi.Router) {
	r.Get("/", h.listPaymentMethods)
	r.Post("/", h.createPaymentMethod)
	r.Route("/{paymentMethodID}", func(r chi.Router) {
		r.Delete("/", h.deletePaymentMethod)
	})
}

func (h *MeHandlers) listPaymentMethods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.users == nil {
		httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	methods, err := h.users.ListPaymentMethods(ctx, identity.UID)
	if err != nil {
		writePaymentMethodError(ctx, w, err)
		return
	}

	payload := make([]paymentMethodPayload, 0, len(methods))
	for _, method := range methods {
		payload = append(payload, buildPaymentMethodPayload(method))
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *MeHandlers) createPaymentMethod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.users == nil {
		httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxProfileBodySize)
	if err != nil {
		status := http.StatusBadRequest
		code := "invalid_request"
		if errors.Is(err, errBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
			code = "payload_too_large"
		}
		httpx.WriteError(ctx, w, httpx.NewError(code, err.Error(), status))
		return
	}

	req, err := decodePaymentMethodRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cmd := services.AddPaymentMethodCommand{
		UserID:      identity.UID,
		Provider:    req.Provider,
		Token:       req.Token,
		MakeDefault: req.MakeDefault,
	}

	method, err := h.users.AddPaymentMethod(ctx, cmd)
	if err != nil {
		writePaymentMethodError(ctx, w, err)
		return
	}

	payload := buildPaymentMethodPayload(method)
	w.Header().Set("Location", strings.TrimSuffix(r.URL.Path, "/")+"/"+payload.ID)
	writeJSONResponse(w, http.StatusCreated, payload)
}

func (h *MeHandlers) deletePaymentMethod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.users == nil {
		httpx.WriteError(ctx, w, httpx.NewError("profile_service_unavailable", "profile service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	paymentMethodID := strings.TrimSpace(chi.URLParam(r, "paymentMethodID"))
	if paymentMethodID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "payment method id is required", http.StatusBadRequest))
		return
	}

	err := h.users.RemovePaymentMethod(ctx, services.RemovePaymentMethodCommand{
		UserID:          identity.UID,
		PaymentMethodID: paymentMethodID,
	})
	if err != nil {
		writePaymentMethodError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type paymentMethodRequest struct {
	Provider    string `json:"provider"`
	Token       string `json:"token"`
	MakeDefault bool   `json:"make_default"`
}

func decodePaymentMethodRequest(body []byte) (paymentMethodRequest, error) {
	if len(body) == 0 {
		return paymentMethodRequest{}, errors.New("request body is required")
	}

	var req paymentMethodRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return paymentMethodRequest{}, err
	}

	req.Provider = strings.TrimSpace(req.Provider)
	req.Token = strings.TrimSpace(req.Token)
	if req.Provider == "" {
		return paymentMethodRequest{}, errors.New("provider is required")
	}
	if req.Token == "" {
		return paymentMethodRequest{}, errors.New("token is required")
	}

	return req, nil
}

type paymentMethodPayload struct {
	ID        string `json:"id"`
	Provider  string `json:"provider"`
	Token     string `json:"token"`
	Brand     string `json:"brand,omitempty"`
	Last4     string `json:"last4,omitempty"`
	ExpMonth  int    `json:"exp_month,omitempty"`
	ExpYear   int    `json:"exp_year,omitempty"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func buildPaymentMethodPayload(method services.PaymentMethod) paymentMethodPayload {
	return paymentMethodPayload{
		ID:        method.ID,
		Provider:  method.Provider,
		Token:     method.Token,
		Brand:     method.Brand,
		Last4:     method.Last4,
		ExpMonth:  method.ExpMonth,
		ExpYear:   method.ExpYear,
		IsDefault: method.IsDefault,
		CreatedAt: formatTime(method.CreatedAt),
		UpdatedAt: formatTime(method.UpdatedAt),
	}
}

func writePaymentMethodError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, services.ErrUserPaymentMethodNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("payment_method_not_found", "payment method not found", http.StatusNotFound))
		return
	case errors.Is(err, services.ErrUserPaymentMethodDuplicate):
		httpx.WriteError(ctx, w, httpx.NewError("payment_method_conflict", "payment method already exists", http.StatusConflict))
		return
	case errors.Is(err, services.ErrUserPaymentMethodInUse):
		httpx.WriteError(ctx, w, httpx.NewError("payment_method_in_use", "payment method cannot be removed while invoices are outstanding", http.StatusConflict))
		return
	case errors.Is(err, services.ErrUserPaymentProviderRequired),
		errors.Is(err, services.ErrUserPaymentTokenRequired):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_payment_method", err.Error(), http.StatusBadRequest))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError("payment_method_not_found", "payment method not found", http.StatusNotFound))
			return
		case repoErr.IsConflict():
			httpx.WriteError(ctx, w, httpx.NewError("payment_method_conflict", "payment method conflict", http.StatusConflict))
			return
		default:
			httpx.WriteError(ctx, w, httpx.NewError("payment_method_error", err.Error(), http.StatusInternalServerError))
			return
		}
	}

	httpx.WriteError(ctx, w, httpx.NewError("payment_method_error", err.Error(), http.StatusInternalServerError))
}
