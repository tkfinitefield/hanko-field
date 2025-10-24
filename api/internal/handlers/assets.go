package handlers

import (
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

const maxAssetRequestBody = 4 * 1024

// AssetHandlers exposes endpoints for issuing signed asset URLs.
type AssetHandlers struct {
	authn  *auth.Authenticator
	assets services.AssetService
}

// NewAssetHandlers constructs handlers enforcing Firebase authentication.
func NewAssetHandlers(authn *auth.Authenticator, assets services.AssetService) *AssetHandlers {
	return &AssetHandlers{
		authn:  authn,
		assets: assets,
	}
}

// Routes registers the asset endpoints on the provided router.
func (h *AssetHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	group := r
	if h.authn != nil {
		group = group.With(h.authn.RequireFirebaseAuth())
	}
	group.Post("/assets:signed-upload", h.issueSignedUpload)
	group.Post("/assets/{assetId}:signed-download", h.issueSignedDownload)
}

func (h *AssetHandlers) issueSignedUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.assets == nil {
		httpx.WriteError(ctx, w, httpx.NewError("asset_service_unavailable", "asset service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	body, err := readLimitedBody(r, maxAssetRequestBody)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), status))
		return
	}
	if len(body) == 0 {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body is required", http.StatusBadRequest))
		return
	}

	var req signedUploadRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "request body must be valid JSON", http.StatusBadRequest))
		return
	}

	cmd := buildSignedUploadCommand(identity.UID, req)
	response, err := h.assets.IssueSignedUpload(ctx, cmd)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAssetInvalidInput):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		case errors.Is(err, services.ErrAssetRepositoryUnavailable):
			httpx.WriteError(ctx, w, httpx.NewError("asset_service_unavailable", "asset repository unavailable", http.StatusServiceUnavailable))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("asset_service_error", err.Error(), http.StatusBadGateway))
		}
		return
	}

	payload := signedUploadResponse{
		AssetID:   response.AssetID,
		UploadURL: response.URL,
		Method:    response.Method,
		Headers:   response.Headers,
	}
	if !response.ExpiresAt.IsZero() {
		payload.ExpiresAt = response.ExpiresAt.UTC().Format(time.RFC3339)
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

type signedUploadRequest struct {
	Kind            string  `json:"kind"`
	Purpose         string  `json:"purpose"`
	ContentType     string  `json:"content_type,omitempty"`
	MimeTypeSnake   string  `json:"mime_type,omitempty"`
	MimeTypeCamel   string  `json:"mimeType,omitempty"`
	FileName        string  `json:"file_name,omitempty"`
	DesignID        *string `json:"design_id,omitempty"`
	SizeBytes       *int64  `json:"size_bytes,omitempty"`
	DeclaredSizeInt int64   `json:"sizeBytes,omitempty"`
}

type signedUploadResponse struct {
	AssetID   string            `json:"asset_id"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	ExpiresAt string            `json:"expires_at,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}

func buildSignedUploadCommand(actorID string, req signedUploadRequest) services.SignedUploadCommand {
	contentType := firstNonEmptyTrimmed(
		req.ContentType,
		req.MimeTypeSnake,
		req.MimeTypeCamel,
	)

	var designID *string
	if req.DesignID != nil {
		if trimmed := strings.TrimSpace(*req.DesignID); trimmed != "" {
			designID = &trimmed
		}
	}

	size := int64(0)
	switch {
	case req.SizeBytes != nil:
		size = *req.SizeBytes
	case req.DeclaredSizeInt > 0:
		size = req.DeclaredSizeInt
	}

	return services.SignedUploadCommand{
		ActorID:     actorID,
		DesignID:    designID,
		Kind:        req.Kind,
		Purpose:     req.Purpose,
		FileName:    req.FileName,
		ContentType: contentType,
		SizeBytes:   size,
	}
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (h *AssetHandlers) issueSignedDownload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.assets == nil {
		httpx.WriteError(ctx, w, httpx.NewError("asset_service_unavailable", "asset service is unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	assetID := strings.TrimSpace(chi.URLParam(r, "assetId"))
	if assetID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "assetId path parameter is required", http.StatusBadRequest))
		return
	}

	response, err := h.assets.IssueSignedDownload(ctx, services.SignedDownloadCommand{
		ActorID: identity.UID,
		AssetID: assetID,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAssetInvalidInput):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		case errors.Is(err, services.ErrAssetForbidden):
			httpx.WriteError(ctx, w, httpx.NewError("forbidden", "insufficient permissions for asset", http.StatusForbidden))
		case errors.Is(err, services.ErrAssetUnavailable):
			httpx.WriteError(ctx, w, httpx.NewError("asset_unavailable", "asset is not ready for download", http.StatusConflict))
		case errors.Is(err, services.ErrAssetNotFound):
			httpx.WriteError(ctx, w, httpx.NewError("asset_not_found", "asset not found", http.StatusNotFound))
		case errors.Is(err, services.ErrAssetRepositoryUnavailable):
			httpx.WriteError(ctx, w, httpx.NewError("asset_service_unavailable", "asset repository unavailable", http.StatusServiceUnavailable))
		default:
			httpx.WriteError(ctx, w, httpx.NewError("asset_service_error", err.Error(), http.StatusBadGateway))
		}
		return
	}

	payload := signedDownloadResponse{
		URL:    response.URL,
		Method: response.Method,
	}
	if !response.ExpiresAt.IsZero() {
		payload.ExpiresAt = response.ExpiresAt.UTC().Format(time.RFC3339)
	}
	if len(response.Headers) > 0 {
		payload.Headers = response.Headers
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

type signedDownloadResponse struct {
	URL       string            `json:"url"`
	Method    string            `json:"method,omitempty"`
	ExpiresAt string            `json:"expires_at,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}
