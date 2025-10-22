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
	"github.com/hanko-field/api/internal/services"
)

const maxNameMappingRequestBody = 16 * 1024

// NameMappingHandlers exposes endpoints for kanji conversion mappings.
type NameMappingHandlers struct {
	authn  *auth.Authenticator
	mapsvc services.NameMappingService
}

// NewNameMappingHandlers constructs a name mapping handler set.
func NewNameMappingHandlers(authn *auth.Authenticator, svc services.NameMappingService) *NameMappingHandlers {
	return &NameMappingHandlers{
		authn:  authn,
		mapsvc: svc,
	}
}

// Routes registers the name mapping endpoints beneath /name-mappings.
func (h *NameMappingHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}

	route := r
	if h.authn != nil {
		route = route.With(h.authn.RequireFirebaseAuth())
	}

	route.Post("/name-mappings:convert", h.convert)
	route.Post("/name-mappings/{mappingId}:select", h.selectCandidate)
}

func (h *NameMappingHandlers) convert(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mapsvc == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "name mapping service not available", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxNameMappingRequestBody)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}
	if len(body) == 0 {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body is required", http.StatusBadRequest))
		return
	}

	var req convertNameRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON payload", http.StatusBadRequest))
		return
	}

	cmd := services.NameConversionCommand{
		UserID:       identity.UID,
		Latin:        req.Latin,
		Locale:       req.Locale,
		Gender:       req.Gender,
		Context:      req.Context,
		ForceRefresh: req.ForceRefresh,
	}

	mapping, err := h.mapsvc.ConvertName(ctx, cmd)
	if err != nil {
		writeNameMappingError(ctx, w, err)
		return
	}

	payload := nameMappingResponse{
		Mapping: buildNameMappingPayload(mapping),
	}
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *NameMappingHandlers) selectCandidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.mapsvc == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "name mapping service not available", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	mappingID := strings.TrimSpace(chi.URLParam(r, "mappingId"))
	if mappingID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "mapping_id is required", http.StatusBadRequest))
		return
	}

	body, err := readLimitedBody(r, maxNameMappingRequestBody)
	if err != nil {
		switch {
		case errors.Is(err, errBodyTooLarge):
			httpx.WriteError(ctx, w, httpx.NewError("payload_too_large", "request body exceeds allowed size", http.StatusRequestEntityTooLarge))
		case errors.Is(err, errEmptyBody):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body is required", http.StatusBadRequest))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		}
		return
	}

	var req selectNameMappingRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid JSON payload", http.StatusBadRequest))
		return
	}

	selectedID := strings.TrimSpace(req.Selected)
	if selectedID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "selected candidate is required", http.StatusBadRequest))
		return
	}

	cmd := services.NameMappingSelectCommand{
		UserID:        identity.UID,
		MappingID:     mappingID,
		CandidateID:   selectedID,
		AllowOverride: req.AllowOverride,
	}

	mapping, err := h.mapsvc.SelectCandidate(ctx, cmd)
	if err != nil {
		writeNameMappingError(ctx, w, err)
		return
	}

	payload := nameMappingResponse{
		Mapping: buildNameMappingPayload(mapping),
	}
	writeJSONResponse(w, http.StatusOK, payload)
}

type convertNameRequest struct {
	Latin        string            `json:"latin"`
	Locale       string            `json:"locale"`
	Gender       string            `json:"gender"`
	Context      map[string]string `json:"context"`
	ForceRefresh bool              `json:"force_refresh"`
}

type selectNameMappingRequest struct {
	Selected      string `json:"selected"`
	AllowOverride bool   `json:"allow_override"`
}

type nameMappingResponse struct {
	Mapping nameMappingPayload `json:"mapping"`
}

type nameMappingPayload struct {
	ID                string             `json:"id"`
	Latin             string             `json:"latin"`
	Locale            string             `json:"locale"`
	Gender            string             `json:"gender,omitempty"`
	Status            string             `json:"status"`
	Source            string             `json:"source,omitempty"`
	Context           map[string]string  `json:"context,omitempty"`
	Candidates        []candidatePayload `json:"candidates"`
	SelectedCandidate *candidatePayload  `json:"selected_candidate,omitempty"`
	SelectedAt        string             `json:"selected_at,omitempty"`
	ExpiresAt         string             `json:"expires_at,omitempty"`
	CreatedAt         string             `json:"created_at"`
	UpdatedAt         string             `json:"updated_at"`
}

type candidatePayload struct {
	ID    string   `json:"id"`
	Kanji string   `json:"kanji"`
	Kana  []string `json:"kana,omitempty"`
	Score float64  `json:"score"`
	Notes string   `json:"notes,omitempty"`
}

func buildNameMappingPayload(mapping services.NameMapping) nameMappingPayload {
	result := nameMappingPayload{
		ID:        mapping.ID,
		Latin:     mapping.Input.Latin,
		Locale:    mapping.Input.Locale,
		Gender:    mapping.Input.Gender,
		Status:    string(mapping.Status),
		Source:    mapping.Source,
		Context:   cloneStringMap(mapping.Input.Context),
		ExpiresAt: formatTime(pointerTime(mapping.ExpiresAt)),
		CreatedAt: formatTime(mapping.CreatedAt),
		UpdatedAt: formatTime(mapping.UpdatedAt),
	}
	candidates := make([]candidatePayload, 0, len(mapping.Candidates))
	for _, cand := range mapping.Candidates {
		candidates = append(candidates, candidatePayload{
			ID:    cand.ID,
			Kanji: cand.Kanji,
			Kana:  cloneStringSlice(cand.Kana),
			Score: cand.Score,
			Notes: cand.Notes,
		})
	}
	result.Candidates = candidates
	if mapping.SelectedCandidate != nil {
		selected := candidatePayload{
			ID:    mapping.SelectedCandidate.ID,
			Kanji: mapping.SelectedCandidate.Kanji,
			Kana:  cloneStringSlice(mapping.SelectedCandidate.Kana),
			Score: mapping.SelectedCandidate.Score,
			Notes: mapping.SelectedCandidate.Notes,
		}
		result.SelectedCandidate = &selected
	}
	result.SelectedAt = formatTime(pointerTime(mapping.SelectedAt))
	return result
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func writeNameMappingError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, services.ErrNameMappingInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrNameMappingUnsupportedLocale):
		httpx.WriteError(ctx, w, httpx.NewError("unsupported_locale", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrNameMappingUnauthorized):
		httpx.WriteError(ctx, w, httpx.NewError("forbidden", err.Error(), http.StatusForbidden))
	case errors.Is(err, services.ErrNameMappingNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("not_found", err.Error(), http.StatusNotFound))
	case errors.Is(err, services.ErrNameMappingConflict):
		httpx.WriteError(ctx, w, httpx.NewError("conflict", err.Error(), http.StatusConflict))
	case errors.Is(err, services.ErrNameMappingUnavailable):
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "name mapping service temporarily unavailable", http.StatusServiceUnavailable))
	default:
		httpx.WriteError(ctx, w, httpx.NewError("name_mapping_error", "failed to convert name", http.StatusInternalServerError))
	}
}
