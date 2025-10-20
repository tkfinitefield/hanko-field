package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

func (h *MeHandlers) favoriteRoutes(r chi.Router) {
	r.Get("/", h.listFavorites)
	r.Put("/{designID}", h.addFavorite)
	r.Delete("/{designID}", h.removeFavorite)
}

func (h *MeHandlers) listFavorites(w http.ResponseWriter, r *http.Request) {
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

	pager := services.Pagination{}
	if sizeRaw := strings.TrimSpace(r.URL.Query().Get("page_size")); sizeRaw != "" {
		if size, err := strconv.Atoi(sizeRaw); err == nil && size > 0 {
			pager.PageSize = size
		}
	}
	pager.PageToken = strings.TrimSpace(r.URL.Query().Get("page_token"))

	page, err := h.users.ListFavorites(ctx, identity.UID, pager)
	if err != nil {
		writeFavoriteError(ctx, w, err)
		return
	}

	payload := make([]favoritePayload, 0, len(page.Items))
	for _, fav := range page.Items {
		payload = append(payload, buildFavoritePayload(fav))
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *MeHandlers) addFavorite(w http.ResponseWriter, r *http.Request) {
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

	designID := strings.TrimSpace(chi.URLParam(r, "designID"))
	if designID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "design id is required", http.StatusBadRequest))
		return
	}

	err := h.users.ToggleFavorite(ctx, services.ToggleFavoriteCommand{
		UserID:   identity.UID,
		DesignID: designID,
		Mark:     true,
	})
	if err != nil {
		writeFavoriteError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *MeHandlers) removeFavorite(w http.ResponseWriter, r *http.Request) {
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

	designID := strings.TrimSpace(chi.URLParam(r, "designID"))
	if designID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "design id is required", http.StatusBadRequest))
		return
	}

	err := h.users.ToggleFavorite(ctx, services.ToggleFavoriteCommand{
		UserID:   identity.UID,
		DesignID: designID,
		Mark:     false,
	})
	if err != nil {
		writeFavoriteError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type favoritePayload struct {
	DesignID string                  `json:"design_id"`
	AddedAt  string                  `json:"added_at"`
	Design   *favoriteDesignMetadata `json:"design,omitempty"`
}

type favoriteDesignMetadata struct {
	ID        string         `json:"id"`
	OwnerID   string         `json:"owner_id,omitempty"`
	Status    string         `json:"status,omitempty"`
	Template  string         `json:"template,omitempty"`
	Locale    string         `json:"locale,omitempty"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Snapshot  map[string]any `json:"snapshot,omitempty"`
}

func buildFavoritePayload(fav services.FavoriteDesign) favoritePayload {
	payload := favoritePayload{
		DesignID: fav.DesignID,
		AddedAt:  formatTime(fav.AddedAt),
	}
	if fav.Design != nil {
		design := fav.Design
		payload.Design = &favoriteDesignMetadata{
			ID:        design.ID,
			OwnerID:   design.OwnerID,
			Status:    string(design.Status),
			Template:  design.Template,
			Locale:    design.Locale,
			UpdatedAt: formatTime(design.UpdatedAt),
		}
		if design.Snapshot != nil && len(design.Snapshot) > 0 {
			payload.Design.Snapshot = design.Snapshot
		}
	}
	return payload
}

func writeFavoriteError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, services.ErrUserFavoriteLimitExceeded):
		httpx.WriteError(ctx, w, httpx.NewError("favorite_limit", "favorite limit reached", http.StatusConflict))
		return
	case errors.Is(err, services.ErrUserFavoriteDesignNotFound):
		httpx.WriteError(ctx, w, httpx.NewError("design_not_found", "design not found", http.StatusNotFound))
		return
	case errors.Is(err, services.ErrUserFavoriteDesignForbidden):
		httpx.WriteError(ctx, w, httpx.NewError("design_forbidden", "design cannot be favorited", http.StatusForbidden))
		return
	}

	httpx.WriteError(ctx, w, httpx.NewError("favorite_error", err.Error(), http.StatusInternalServerError))
}
