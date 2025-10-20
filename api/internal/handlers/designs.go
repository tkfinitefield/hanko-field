package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

const maxDesignRequestBody = 256 * 1024

// DesignHandlers exposes design creation endpoints for authenticated users.
type DesignHandlers struct {
	authn   *auth.Authenticator
	designs services.DesignService
}

// NewDesignHandlers constructs a new DesignHandlers instance.
func NewDesignHandlers(authn *auth.Authenticator, designs services.DesignService) *DesignHandlers {
	return &DesignHandlers{
		authn:   authn,
		designs: designs,
	}
}

// Routes registers the /designs endpoints.
func (h *DesignHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth())
	}
	r.Post("/", h.createDesign)
}

func (h *DesignHandlers) createDesign(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.designs == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "design service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	reader := http.MaxBytesReader(w, r.Body, maxDesignRequestBody)
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	var payload createDesignRequest
	if err := decoder.Decode(&payload); err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest))
		return
	}
	if decoder.More() {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "invalid request body: extraneous data", http.StatusBadRequest))
		return
	}

	idempotency := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	cmd := payload.toCommand(identity.UID, identity.UID, idempotency)

	design, err := h.designs.CreateDesign(ctx, cmd)
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}

	location := fmt.Sprintf("%s/%s", strings.TrimSuffix(r.URL.Path, "/"), design.ID)
	w.Header().Set("Location", location)

	response := createDesignResponse{
		Design: buildDesignPayload(design),
	}
	writeJSONResponse(w, http.StatusCreated, response)
}

func (h *DesignHandlers) writeDesignError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrDesignInvalidInput):
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
	case errors.Is(err, services.ErrDesignConflict):
		httpx.WriteError(ctx, w, httpx.NewError("design_conflict", "design conflict", http.StatusConflict))
	case errors.Is(err, services.ErrDesignRepositoryUnavailable), errors.Is(err, services.ErrDesignRendererUnavailable):
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "design service unavailable", http.StatusServiceUnavailable))
	default:
		httpx.WriteError(ctx, w, httpx.NewError("internal_error", "internal server error", http.StatusInternalServerError))
	}
}

type createDesignRequest struct {
	Label           *string            `json:"label"`
	Type            string             `json:"type"`
	TextLines       []string           `json:"text_lines"`
	FontID          *string            `json:"font_id"`
	MaterialID      *string            `json:"material_id"`
	TemplateID      *string            `json:"template_id"`
	Locale          *string            `json:"locale"`
	Shape           *string            `json:"shape"`
	SizeMM          *float64           `json:"size_mm"`
	RawName         *string            `json:"raw_name"`
	KanjiValue      *string            `json:"kanji_value"`
	KanjiMappingRef *string            `json:"kanji_mapping_ref"`
	UploadAsset     *assetInputPayload `json:"upload_asset"`
	LogoAsset       *assetInputPayload `json:"logo_asset"`
	Snapshot        map[string]any     `json:"snapshot"`
	Metadata        map[string]any     `json:"metadata"`
}

type assetInputPayload struct {
	AssetID     string `json:"asset_id"`
	Bucket      string `json:"bucket"`
	ObjectPath  string `json:"object_path"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Checksum    string `json:"checksum"`
}

func (p *assetInputPayload) toInput() *services.DesignAssetInput {
	if p == nil {
		return nil
	}
	return &services.DesignAssetInput{
		AssetID:     p.AssetID,
		Bucket:      p.Bucket,
		ObjectPath:  p.ObjectPath,
		FileName:    p.FileName,
		ContentType: p.ContentType,
		SizeBytes:   p.SizeBytes,
		Checksum:    p.Checksum,
	}
}

func (req *createDesignRequest) toCommand(ownerID, actorID, idempotency string) services.CreateDesignCommand {
	cmd := services.CreateDesignCommand{
		OwnerID:        ownerID,
		ActorID:        actorID,
		Type:           services.DesignType(strings.TrimSpace(req.Type)),
		TextLines:      append([]string(nil), req.TextLines...),
		IdempotencyKey: idempotency,
		Snapshot:       cloneMap(req.Snapshot),
		Metadata:       cloneMap(req.Metadata),
	}
	if req.Label != nil {
		cmd.Label = strings.TrimSpace(*req.Label)
	}
	if req.FontID != nil {
		cmd.FontID = strings.TrimSpace(*req.FontID)
	}
	if req.MaterialID != nil {
		cmd.MaterialID = strings.TrimSpace(*req.MaterialID)
	}
	if req.TemplateID != nil {
		cmd.TemplateID = strings.TrimSpace(*req.TemplateID)
	}
	if req.Locale != nil {
		cmd.Locale = strings.TrimSpace(*req.Locale)
	}
	if req.Shape != nil {
		cmd.Shape = strings.TrimSpace(*req.Shape)
	}
	if req.SizeMM != nil {
		cmd.SizeMM = *req.SizeMM
	}
	if req.RawName != nil {
		cmd.RawName = strings.TrimSpace(*req.RawName)
	}
	if req.KanjiValue != nil {
		value := strings.TrimSpace(*req.KanjiValue)
		cmd.KanjiValue = &value
	}
	if req.KanjiMappingRef != nil {
		value := strings.TrimSpace(*req.KanjiMappingRef)
		cmd.KanjiMappingRef = &value
	}
	if input := req.UploadAsset; input != nil {
		cmd.Upload = input.toInput()
	}
	if input := req.LogoAsset; input != nil {
		cmd.Logo = input.toInput()
	}
	return cmd
}

type createDesignResponse struct {
	Design designPayload `json:"design"`
}

type designPayload struct {
	ID               string              `json:"id"`
	Label            string              `json:"label"`
	Type             string              `json:"type"`
	TextLines        []string            `json:"text_lines"`
	FontID           string              `json:"font_id,omitempty"`
	MaterialID       string              `json:"material_id,omitempty"`
	TemplateID       string              `json:"template_id,omitempty"`
	Locale           string              `json:"locale,omitempty"`
	Shape            string              `json:"shape,omitempty"`
	SizeMM           float64             `json:"size_mm,omitempty"`
	Status           string              `json:"status"`
	ThumbnailURL     string              `json:"thumbnail_url,omitempty"`
	CurrentVersionID string              `json:"current_version_id"`
	Assets           designAssetsPayload `json:"assets"`
	Source           designSourcePayload `json:"source"`
	Snapshot         map[string]any      `json:"snapshot,omitempty"`
	CreatedAt        string              `json:"created_at,omitempty"`
	UpdatedAt        string              `json:"updated_at,omitempty"`
}

type designAssetsPayload struct {
	SourcePath  string `json:"source_path,omitempty"`
	VectorPath  string `json:"vector_path,omitempty"`
	PreviewPath string `json:"preview_path,omitempty"`
	PreviewURL  string `json:"preview_url,omitempty"`
}

type designSourcePayload struct {
	Type        string           `json:"type"`
	RawName     string           `json:"raw_name,omitempty"`
	TextLines   []string         `json:"text_lines,omitempty"`
	UploadAsset *assetRefPayload `json:"upload_asset,omitempty"`
	LogoAsset   *assetRefPayload `json:"logo_asset,omitempty"`
}

type assetRefPayload struct {
	AssetID     string `json:"asset_id,omitempty"`
	Bucket      string `json:"bucket,omitempty"`
	ObjectPath  string `json:"object_path,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	Checksum    string `json:"checksum,omitempty"`
}

func buildDesignPayload(design services.Design) designPayload {
	payload := designPayload{
		ID:               design.ID,
		Label:            design.Label,
		Type:             string(design.Type),
		TextLines:        cloneStrings(design.TextLines),
		FontID:           design.FontID,
		MaterialID:       design.MaterialID,
		TemplateID:       design.Template,
		Locale:           design.Locale,
		Shape:            design.Shape,
		SizeMM:           design.SizeMM,
		Status:           string(design.Status),
		ThumbnailURL:     design.ThumbnailURL,
		CurrentVersionID: design.CurrentVersionID,
		Assets: designAssetsPayload{
			SourcePath:  design.Assets.SourcePath,
			VectorPath:  design.Assets.VectorPath,
			PreviewPath: design.Assets.PreviewPath,
			PreviewURL:  design.Assets.PreviewURL,
		},
		Source: designSourcePayload{
			Type:        string(design.Source.Type),
			RawName:     design.Source.RawName,
			TextLines:   cloneStrings(design.Source.TextLines),
			UploadAsset: assetRefPayloadFrom(design.Source.UploadAsset),
			LogoAsset:   assetRefPayloadFrom(design.Source.LogoAsset),
		},
		CreatedAt: formatTime(design.CreatedAt),
		UpdatedAt: formatTime(design.UpdatedAt),
	}
	if len(design.Snapshot) > 0 {
		payload.Snapshot = cloneMap(design.Snapshot)
	}
	return payload
}

func assetRefPayloadFrom(ref *services.DesignAssetReference) *assetRefPayload {
	if ref == nil {
		return nil
	}
	return &assetRefPayload{
		AssetID:     ref.AssetID,
		Bucket:      ref.Bucket,
		ObjectPath:  ref.ObjectPath,
		FileName:    ref.FileName,
		ContentType: ref.ContentType,
		SizeBytes:   ref.SizeBytes,
		Checksum:    ref.Checksum,
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return slices.Clone(values)
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	return maps.Clone(src)
}
