package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

const maxDesignRequestBody = 256 * 1024
const (
	defaultDesignPageSize = 20
	maxDesignPageSize     = 100
)

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
	r.Get("/", h.listDesigns)
	r.Post("/", h.createDesign)
	r.Get("/{designID}", h.getDesign)
	r.Put("/{designID}", h.updateDesign)
	r.Delete("/{designID}", h.deleteDesign)
}

func (h *DesignHandlers) listDesigns(w http.ResponseWriter, r *http.Request) {
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

	ownerID := strings.TrimSpace(identity.UID)
	requestedOwner := firstNonEmpty(
		strings.TrimSpace(r.URL.Query().Get("user")),
		strings.TrimSpace(r.URL.Query().Get("user_id")),
	)
	if requestedOwner != "" && !strings.EqualFold(requestedOwner, ownerID) {
		if !identity.HasAnyRole(auth.RoleStaff, auth.RoleAdmin) {
			httpx.WriteError(ctx, w, httpx.NewError("forbidden", "insufficient permissions", http.StatusForbidden))
			return
		}
		ownerID = requestedOwner
	}

	statusFilters := parseFilterValues(r.URL.Query()["status"])
	typeFilters := parseFilterValues(r.URL.Query()["type"])

	var updatedAfter *time.Time
	if updatedRaw := strings.TrimSpace(r.URL.Query().Get("updatedAfter")); updatedRaw != "" {
		parsed, err := parseTimeParam(updatedRaw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", fmt.Sprintf("invalid updatedAfter: %v", err), http.StatusBadRequest))
			return
		}
		updatedAfter = &parsed
	}

	pageSize := defaultDesignPageSize
	if sizeRaw := strings.TrimSpace(r.URL.Query().Get("page_size")); sizeRaw != "" {
		size, err := strconv.Atoi(sizeRaw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "page_size must be an integer", http.StatusBadRequest))
			return
		}
		if size < 0 {
			size = defaultDesignPageSize
		}
		if size > maxDesignPageSize {
			size = maxDesignPageSize
		}
		if size == 0 {
			pageSize = defaultDesignPageSize
		} else {
			pageSize = size
		}
	}

	filter := services.DesignListFilter{
		OwnerID:      ownerID,
		Status:       statusFilters,
		Types:        typeFilters,
		UpdatedAfter: updatedAfter,
		Pagination: services.Pagination{
			PageSize:  pageSize,
			PageToken: strings.TrimSpace(r.URL.Query().Get("page_token")),
		},
	}

	page, err := h.designs.ListDesigns(ctx, filter)
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}

	items := make([]designPayload, 0, len(page.Items))
	for _, design := range page.Items {
		items = append(items, buildDesignPayload(design))
	}

	response := designListResponse{
		Items:         items,
		NextPageToken: page.NextPageToken,
	}
	writeJSONResponse(w, http.StatusOK, response)
}

func (h *DesignHandlers) getDesign(w http.ResponseWriter, r *http.Request) {
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

	designID := strings.TrimSpace(chi.URLParam(r, "designID"))
	if designID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "design id is required", http.StatusBadRequest))
		return
	}

	includeHistory := false
	if flagRaw := strings.TrimSpace(r.URL.Query().Get("includeHistory")); flagRaw != "" {
		value, err := strconv.ParseBool(flagRaw)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "includeHistory must be a boolean", http.StatusBadRequest))
			return
		}
		includeHistory = value
	}

	opts := services.DesignReadOptions{}
	if includeHistory {
		opts.IncludeVersions = true
	}

	design, err := h.designs.GetDesign(ctx, designID, opts)
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}

	ownerID := strings.TrimSpace(identity.UID)
	if ownerID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	if !strings.EqualFold(design.OwnerID, ownerID) && !identity.HasAnyRole(auth.RoleStaff, auth.RoleAdmin) {
		httpx.WriteError(ctx, w, httpx.NewError("design_not_found", "design not found", http.StatusNotFound))
		return
	}

	payload := buildDesignPayload(design)
	if includeHistory && len(design.Versions) > 0 {
		payload.Versions = make([]designVersionPayload, 0, len(design.Versions))
		for _, version := range design.Versions {
			payload.Versions = append(payload.Versions, buildDesignVersionPayload(version))
		}
	}

	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *DesignHandlers) updateDesign(w http.ResponseWriter, r *http.Request) {
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

	designID := strings.TrimSpace(chi.URLParam(r, "designID"))
	if designID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "design id is required", http.StatusBadRequest))
		return
	}

	existing, err := h.designs.GetDesign(ctx, designID, services.DesignReadOptions{})
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}
	if !strings.EqualFold(existing.OwnerID, identity.UID) && !identity.HasAnyRole(auth.RoleStaff, auth.RoleAdmin) {
		httpx.WriteError(ctx, w, httpx.NewError("design_not_found", "design not found", http.StatusNotFound))
		return
	}

	body, err := readLimitedBody(r, maxDesignRequestBody)
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

	req, err := decodeUpdateDesignRequest(body)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	var expectedUpdatedAt *time.Time
	if header := strings.TrimSpace(r.Header.Get("If-Unmodified-Since")); header != "" {
		ts, err := parseTimeParam(header)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "If-Unmodified-Since must be RFC3339 timestamp", http.StatusBadRequest))
			return
		}
		expectedUpdatedAt = &ts
	}

	cmd := services.UpdateDesignCommand{
		DesignID:          designID,
		UpdatedBy:         identity.UID,
		Label:             req.Label,
		Status:            req.Status,
		ThumbnailURL:      req.ThumbnailURL,
		Snapshot:          cloneMap(req.Snapshot),
		ExpectedUpdatedAt: expectedUpdatedAt,
	}

	updated, err := h.designs.UpdateDesign(ctx, cmd)
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}

	payload := buildDesignPayload(updated)
	writeJSONResponse(w, http.StatusOK, payload)
}

func (h *DesignHandlers) deleteDesign(w http.ResponseWriter, r *http.Request) {
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

	designID := strings.TrimSpace(chi.URLParam(r, "designID"))
	if designID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "design id is required", http.StatusBadRequest))
		return
	}

	existing, err := h.designs.GetDesign(ctx, designID, services.DesignReadOptions{})
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}
	if !strings.EqualFold(existing.OwnerID, identity.UID) && !identity.HasAnyRole(auth.RoleStaff, auth.RoleAdmin) {
		httpx.WriteError(ctx, w, httpx.NewError("design_not_found", "design not found", http.StatusNotFound))
		return
	}

	var expectedUpdatedAt *time.Time
	if header := strings.TrimSpace(r.Header.Get("If-Unmodified-Since")); header != "" {
		ts, err := parseTimeParam(header)
		if err != nil {
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "If-Unmodified-Since must be RFC3339 timestamp", http.StatusBadRequest))
			return
		}
		expectedUpdatedAt = &ts
	}

	err = h.designs.DeleteDesign(ctx, services.DeleteDesignCommand{
		DesignID:          designID,
		RequestedBy:       identity.UID,
		SoftDelete:        true,
		ExpectedUpdatedAt: expectedUpdatedAt,
	})
	if err != nil {
		h.writeDesignError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
	ID               string                 `json:"id"`
	Label            string                 `json:"label"`
	Type             string                 `json:"type"`
	TextLines        []string               `json:"text_lines"`
	FontID           string                 `json:"font_id,omitempty"`
	MaterialID       string                 `json:"material_id,omitempty"`
	TemplateID       string                 `json:"template_id,omitempty"`
	Locale           string                 `json:"locale,omitempty"`
	Shape            string                 `json:"shape,omitempty"`
	SizeMM           float64                `json:"size_mm,omitempty"`
	Status           string                 `json:"status"`
	ThumbnailURL     string                 `json:"thumbnail_url,omitempty"`
	CurrentVersionID string                 `json:"current_version_id"`
	Assets           designAssetsPayload    `json:"assets"`
	Source           designSourcePayload    `json:"source"`
	Snapshot         map[string]any         `json:"snapshot,omitempty"`
	CreatedAt        string                 `json:"created_at,omitempty"`
	UpdatedAt        string                 `json:"updated_at,omitempty"`
	Versions         []designVersionPayload `json:"versions,omitempty"`
}

type designVersionPayload struct {
	ID        string         `json:"id"`
	Version   int            `json:"version"`
	Snapshot  map[string]any `json:"snapshot,omitempty"`
	CreatedAt string         `json:"created_at,omitempty"`
	CreatedBy string         `json:"created_by,omitempty"`
}

type designListResponse struct {
	Items         []designPayload `json:"items"`
	NextPageToken string          `json:"next_page_token,omitempty"`
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

func buildDesignVersionPayload(version services.DesignVersion) designVersionPayload {
	payload := designVersionPayload{
		ID:      version.ID,
		Version: version.Version,
	}
	if len(version.Snapshot) > 0 {
		payload.Snapshot = cloneMap(version.Snapshot)
	}
	if !version.CreatedAt.IsZero() {
		payload.CreatedAt = formatTime(version.CreatedAt)
	}
	if strings.TrimSpace(version.CreatedBy) != "" {
		payload.CreatedBy = strings.TrimSpace(version.CreatedBy)
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

func parseFilterValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	filters := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.ToLower(strings.TrimSpace(part))
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			filters = append(filters, trimmed)
		}
	}
	return filters
}

func parseTimeParam(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, errors.New("timestamp is empty")
	}
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts.UTC(), nil
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("must be RFC3339 timestamp")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

type updateDesignRequest struct {
	Label        *string        `json:"label"`
	Status       *string        `json:"status"`
	ThumbnailURL *string        `json:"thumbnail_url"`
	Snapshot     map[string]any `json:"snapshot"`
}

func decodeUpdateDesignRequest(body []byte) (updateDesignRequest, error) {
	if len(body) == 0 {
		return updateDesignRequest{}, errors.New("request body is required")
	}
	var req updateDesignRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return updateDesignRequest{}, err
	}
	if req.Status != nil {
		value := strings.TrimSpace(*req.Status)
		req.Status = &value
	}
	if req.Label != nil {
		value := strings.TrimSpace(*req.Label)
		req.Label = &value
	}
	if req.ThumbnailURL != nil {
		value := strings.TrimSpace(*req.ThumbnailURL)
		req.ThumbnailURL = &value
	}
	if req.Snapshot != nil && len(req.Snapshot) == 0 {
		req.Snapshot = nil
	}
	return req, nil
}
