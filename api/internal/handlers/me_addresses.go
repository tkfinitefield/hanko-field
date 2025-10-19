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

func (h *MeHandlers) addressRoutes(r chi.Router) {
	r.Get("/", h.listAddresses)
	r.Post("/", h.createAddress)
	r.Route("/{addressID}", func(r chi.Router) {
		r.Put("/", h.updateAddress)
		r.Delete("/", h.deleteAddress)
	})
}

func (h *MeHandlers) listAddresses(w http.ResponseWriter, r *http.Request) {
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

	addresses, err := h.users.ListAddresses(ctx, identity.UID)
	if err != nil {
		writeAddressError(ctx, w, err)
		return
	}

	payload := make([]addressPayload, 0, len(addresses))
	for _, addr := range addresses {
		payload = append(payload, buildAddressPayload(addr))
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *MeHandlers) createAddress(w http.ResponseWriter, r *http.Request) {
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

	req, err := decodeAddressRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cmd := services.UpsertAddressCommand{
		UserID:          identity.UID,
		Address:         req.toDomainAddress(),
		DefaultShipping: req.DefaultShipping,
		DefaultBilling:  req.DefaultBilling,
	}

	saved, err := h.users.UpsertAddress(ctx, cmd)
	if err != nil {
		writeAddressError(ctx, w, err)
		return
	}

	payload := buildAddressPayload(saved)
	w.Header().Set("Location", strings.TrimSuffix(r.URL.Path, "/")+"/"+saved.ID)
	writeJSONResponse(w, http.StatusCreated, payload)
}

func (h *MeHandlers) updateAddress(w http.ResponseWriter, r *http.Request) {
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

	addressID := strings.TrimSpace(chi.URLParam(r, "addressID"))
	if addressID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "address id is required", http.StatusBadRequest))
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

	req, err := decodeAddressRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	cmd := services.UpsertAddressCommand{
		UserID:          identity.UID,
		AddressID:       &addressID,
		Address:         req.toDomainAddress(),
		DefaultShipping: req.DefaultShipping,
		DefaultBilling:  req.DefaultBilling,
	}

	saved, err := h.users.UpsertAddress(ctx, cmd)
	if err != nil {
		writeAddressError(ctx, w, err)
		return
	}

	payload := buildAddressPayload(saved)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *MeHandlers) deleteAddress(w http.ResponseWriter, r *http.Request) {
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

	addressID := strings.TrimSpace(chi.URLParam(r, "addressID"))
	if addressID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "address id is required", http.StatusBadRequest))
		return
	}

	replacementParam := strings.TrimSpace(r.URL.Query().Get("replacement_id"))
	var replacementPtr *string
	if replacementParam != "" {
		replacementPtr = &replacementParam
	}

	err := h.users.DeleteAddress(ctx, services.DeleteAddressCommand{
		UserID:        identity.UID,
		AddressID:     addressID,
		ReplacementID: replacementPtr,
	})
	if err != nil {
		writeAddressError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type addressRequest struct {
	Label           *string `json:"label"`
	Recipient       *string `json:"recipient"`
	Company         *string `json:"company"`
	Line1           *string `json:"line1"`
	Line2           *string `json:"line2"`
	City            *string `json:"city"`
	State           *string `json:"state"`
	PostalCode      *string `json:"postal_code"`
	Country         *string `json:"country"`
	Phone           *string `json:"phone"`
	DefaultShipping *bool   `json:"default_shipping"`
	DefaultBilling  *bool   `json:"default_billing"`
}

func decodeAddressRequest(data []byte) (addressRequest, error) {
	var req addressRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return addressRequest{}, err
	}
	if req.Recipient == nil || strings.TrimSpace(*req.Recipient) == "" {
		return addressRequest{}, errors.New("recipient is required")
	}
	if req.Line1 == nil || strings.TrimSpace(*req.Line1) == "" {
		return addressRequest{}, errors.New("line1 is required")
	}
	if req.City == nil || strings.TrimSpace(*req.City) == "" {
		return addressRequest{}, errors.New("city is required")
	}
	if req.PostalCode == nil || strings.TrimSpace(*req.PostalCode) == "" {
		return addressRequest{}, errors.New("postal_code is required")
	}
	if req.Country == nil || strings.TrimSpace(*req.Country) == "" {
		return addressRequest{}, errors.New("country is required")
	}
	return req, nil
}

func (req addressRequest) toDomainAddress() services.Address {
	addr := services.Address{
		Label:      trimOrEmpty(req.Label),
		Recipient:  strings.TrimSpace(valueOrEmpty(req.Recipient)),
		Company:    trimOrEmpty(req.Company),
		Line1:      strings.TrimSpace(valueOrEmpty(req.Line1)),
		City:       strings.TrimSpace(valueOrEmpty(req.City)),
		PostalCode: strings.TrimSpace(valueOrEmpty(req.PostalCode)),
		Country:    strings.TrimSpace(strings.ToUpper(valueOrEmpty(req.Country))),
	}
	if req.Line2 != nil {
		if trimmed := strings.TrimSpace(*req.Line2); trimmed != "" {
			addr.Line2 = &trimmed
		}
	}
	if req.State != nil {
		if trimmed := strings.TrimSpace(*req.State); trimmed != "" {
			addr.State = &trimmed
		}
	}
	if req.Phone != nil {
		if trimmed := strings.TrimSpace(*req.Phone); trimmed != "" {
			addr.Phone = &trimmed
		}
	}
	if req.DefaultShipping != nil {
		addr.DefaultShipping = *req.DefaultShipping
	}
	if req.DefaultBilling != nil {
		addr.DefaultBilling = *req.DefaultBilling
	}
	return addr
}

type addressPayload struct {
	ID              string  `json:"id"`
	Label           string  `json:"label,omitempty"`
	Recipient       string  `json:"recipient"`
	Company         string  `json:"company,omitempty"`
	Line1           string  `json:"line1"`
	Line2           *string `json:"line2,omitempty"`
	City            string  `json:"city"`
	State           *string `json:"state,omitempty"`
	PostalCode      string  `json:"postal_code"`
	Country         string  `json:"country"`
	Phone           *string `json:"phone,omitempty"`
	DefaultShipping bool    `json:"default_shipping"`
	DefaultBilling  bool    `json:"default_billing"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
}

func buildAddressPayload(addr services.Address) addressPayload {
	return addressPayload{
		ID:              addr.ID,
		Label:           addr.Label,
		Recipient:       addr.Recipient,
		Company:         addr.Company,
		Line1:           addr.Line1,
		Line2:           addr.Line2,
		City:            addr.City,
		State:           addr.State,
		PostalCode:      addr.PostalCode,
		Country:         addr.Country,
		Phone:           addr.Phone,
		DefaultShipping: addr.DefaultShipping,
		DefaultBilling:  addr.DefaultBilling,
		CreatedAt:       formatTime(addr.CreatedAt),
		UpdatedAt:       formatTime(addr.UpdatedAt),
	}
}

func trimOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func writeAddressError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrUserAddressNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("address_not_found", "address not found", http.StatusNotFound))
		return
	case errors.Is(err, services.ErrUserInvalidAddressRecipient),
		errors.Is(err, services.ErrUserInvalidAddressLine1),
		errors.Is(err, services.ErrUserInvalidAddressCity),
		errors.Is(err, services.ErrUserInvalidAddressCountry),
		errors.Is(err, services.ErrUserInvalidAddressPostalCode),
		errors.Is(err, services.ErrUserInvalidAddressPhone):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_address", err.Error(), http.StatusBadRequest))
		return
	case errors.Is(err, services.ErrUserProfileConflict):
		httpx.WriteError(ctx, w, httpx.NewError("address_conflict", err.Error(), http.StatusConflict))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError("address_not_found", "address not found", http.StatusNotFound))
			return
		case repoErr.IsConflict():
			httpx.WriteError(ctx, w, httpx.NewError("address_conflict", "address conflict", http.StatusConflict))
			return
		default:
			httpx.WriteError(ctx, w, httpx.NewError("address_error", err.Error(), http.StatusInternalServerError))
			return
		}
	}

	httpx.WriteError(ctx, w, httpx.NewError("address_error", err.Error(), http.StatusInternalServerError))
}
