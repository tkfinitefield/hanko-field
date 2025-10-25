package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/hanko-field/api/internal/platform/auth"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/services"
)

const maxCatalogRequestBody = 256 * 1024

// AdminCatalogHandlers exposes admin catalog CRUD endpoints.
type AdminCatalogHandlers struct {
	authn   *auth.Authenticator
	catalog services.CatalogService
}

// NewAdminCatalogHandlers constructs admin catalog handlers.
func NewAdminCatalogHandlers(authn *auth.Authenticator, catalog services.CatalogService) *AdminCatalogHandlers {
	return &AdminCatalogHandlers{authn: authn, catalog: catalog}
}

// Routes registers admin catalog endpoints.
func (h *AdminCatalogHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	if h.authn != nil {
		r.Use(h.authn.RequireFirebaseAuth(auth.RoleAdmin))
	}
	r.Route("/catalog", func(rt chi.Router) {
		rt.Post("/templates", h.createTemplate)
		rt.Put("/templates/{templateID}", h.updateTemplate)
		rt.Delete("/templates/{templateID}", h.deleteTemplate)
		rt.Post("/fonts", h.createFont)
		rt.Put("/fonts/{fontID}", h.updateFont)
		rt.Delete("/fonts/{fontID}", h.deleteFont)
		rt.Post("/materials", h.createMaterial)
		rt.Put("/materials/{materialID}", h.updateMaterial)
		rt.Delete("/materials/{materialID}", h.deleteMaterial)
		rt.Post("/products", h.createProduct)
		rt.Put("/products/{productID}", h.updateProduct)
		rt.Delete("/products/{productID}", h.deleteProduct)
	})
}

func (h *AdminCatalogHandlers) createTemplate(w http.ResponseWriter, r *http.Request) {
	h.saveTemplate(w, r, "")
}

func (h *AdminCatalogHandlers) updateTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := chi.URLParam(r, "templateID")
	h.saveTemplate(w, r, templateID)
}

func (h *AdminCatalogHandlers) saveTemplate(w http.ResponseWriter, r *http.Request, templateID string) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}

	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}

	payload, err := decodeAdminTemplateRequest(r, templateID)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	result, err := h.catalog.UpsertTemplate(ctx, services.UpsertTemplateCommand{
		Template: payload,
		ActorID:  identity.UID,
	})
	if err != nil {
		writeCatalogError(ctx, w, err, "template")
		return
	}

	response := newAdminTemplateResponse(result)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AdminCatalogHandlers) createFont(w http.ResponseWriter, r *http.Request) {
	h.saveFont(w, r, "")
}

func (h *AdminCatalogHandlers) updateFont(w http.ResponseWriter, r *http.Request) {
	fontID := chi.URLParam(r, "fontID")
	h.saveFont(w, r, fontID)
}

func (h *AdminCatalogHandlers) saveFont(w http.ResponseWriter, r *http.Request, fontID string) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	font, err := decodeAdminFontRequest(r, fontID)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	result, err := h.catalog.UpsertFont(ctx, services.UpsertFontCommand{Font: font, ActorID: identity.UID})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCatalogInvalidInput):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
			return
		case errors.Is(err, services.ErrCatalogFontConflict):
			httpx.WriteError(ctx, w, httpx.NewError("font_conflict", err.Error(), http.StatusConflict))
			return
		default:
			writeCatalogError(ctx, w, err, "font")
			return
		}
	}
	response := newAdminFontResponse(result)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AdminCatalogHandlers) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "template id is required", http.StatusBadRequest))
		return
	}
	err := h.catalog.DeleteTemplate(ctx, services.DeleteTemplateCommand{
		TemplateID: templateID,
		ActorID:    identity.UID,
	})
	if err != nil {
		writeCatalogError(ctx, w, err, "template")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminCatalogHandlers) deleteFont(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	fontID := strings.TrimSpace(chi.URLParam(r, "fontID"))
	if fontID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "font id is required", http.StatusBadRequest))
		return
	}
	if err := h.catalog.DeleteFont(ctx, fontID); err != nil {
		switch {
		case errors.Is(err, services.ErrCatalogInvalidInput):
			httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
			return
		case errors.Is(err, services.ErrCatalogFontInUse):
			httpx.WriteError(ctx, w, httpx.NewError("font_in_use", err.Error(), http.StatusConflict))
			return
		default:
			writeCatalogError(ctx, w, err, "font")
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminCatalogHandlers) createMaterial(w http.ResponseWriter, r *http.Request) {
	h.saveMaterial(w, r, "")
}

func (h *AdminCatalogHandlers) updateMaterial(w http.ResponseWriter, r *http.Request) {
	materialID := chi.URLParam(r, "materialID")
	h.saveMaterial(w, r, materialID)
}

func (h *AdminCatalogHandlers) saveMaterial(w http.ResponseWriter, r *http.Request, materialID string) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	material, err := decodeAdminMaterialRequest(r, materialID)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	result, err := h.catalog.UpsertMaterial(ctx, services.UpsertMaterialCommand{Material: material, ActorID: identity.UID})
	if err != nil {
		writeCatalogError(ctx, w, err, "material")
		return
	}
	response := newAdminMaterialResponse(result)
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(response)
}

func (h *AdminCatalogHandlers) deleteMaterial(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	materialID := strings.TrimSpace(chi.URLParam(r, "materialID"))
	if materialID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "material id is required", http.StatusBadRequest))
		return
	}
	if err := h.catalog.DeleteMaterial(ctx, services.DeleteMaterialCommand{MaterialID: materialID, ActorID: identity.UID}); err != nil {
		writeCatalogError(ctx, w, err, "material")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminCatalogHandlers) createProduct(w http.ResponseWriter, r *http.Request) {
	h.saveProduct(w, r, "")
}

func (h *AdminCatalogHandlers) updateProduct(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "productID")
	h.saveProduct(w, r, productID)
}

func (h *AdminCatalogHandlers) saveProduct(w http.ResponseWriter, r *http.Request, productID string) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	product, err := decodeAdminProductRequest(r, productID)
	if err != nil {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	result, err := h.catalog.UpsertProduct(ctx, services.UpsertProductCommand{
		Product: product,
		ActorID: identity.UID,
	})
	if err != nil {
		writeCatalogError(ctx, w, err, "product")
		return
	}
	response := newAdminProductResponse(result)
	status := http.StatusOK
	if r.Method == http.MethodPost {
		status = http.StatusCreated
	}
	writeJSON(w, status, response)
}

func (h *AdminCatalogHandlers) deleteProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if h.catalog == nil {
		httpx.WriteError(ctx, w, httpx.NewError("service_unavailable", "catalog service unavailable", http.StatusServiceUnavailable))
		return
	}
	identity, ok := auth.IdentityFromContext(ctx)
	if !ok || identity == nil || strings.TrimSpace(identity.UID) == "" {
		httpx.WriteError(ctx, w, httpx.NewError("unauthenticated", "authentication required", http.StatusUnauthorized))
		return
	}
	productID := strings.TrimSpace(chi.URLParam(r, "productID"))
	if productID == "" {
		httpx.WriteError(ctx, w, httpx.NewError("invalid_request", "product id is required", http.StatusBadRequest))
		return
	}
	if err := h.catalog.DeleteProduct(ctx, services.DeleteProductCommand{ProductID: productID, ActorID: identity.UID}); err != nil {
		writeCatalogError(ctx, w, err, "product")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAdminTemplateRequest(r *http.Request, overrideID string) (services.Template, error) {
	limited := io.LimitReader(r.Body, maxCatalogRequestBody)
	defer r.Body.Close()
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	var req adminTemplateRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return services.Template{}, errors.New("request body required")
		}
		return services.Template{}, fmt.Errorf("invalid request body: %w", err)
	}

	template, err := req.toTemplate()
	if err != nil {
		return services.Template{}, err
	}
	if strings.TrimSpace(overrideID) != "" {
		template.ID = strings.TrimSpace(overrideID)
	}
	return template, nil
}

type adminTemplateRequest struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description"`
	Category         string                     `json:"category"`
	Style            string                     `json:"style"`
	Tags             []string                   `json:"tags"`
	PreviewImagePath string                     `json:"preview_image_path"`
	SVGPath          string                     `json:"svg_path"`
	IsPublished      bool                       `json:"is_published"`
	PublishedAt      *string                    `json:"published_at"`
	Draft            *adminTemplateDraftPayload `json:"draft"`
}

func (r adminTemplateRequest) toTemplate() (services.Template, error) {
	tpl := services.Template{}
	if strings.TrimSpace(r.ID) != "" {
		tpl.ID = strings.TrimSpace(r.ID)
	}
	tpl.Name = r.Name
	tpl.Description = r.Description
	tpl.Category = r.Category
	tpl.Style = r.Style
	tpl.Tags = append([]string(nil), r.Tags...)
	tpl.PreviewImagePath = r.PreviewImagePath
	tpl.SVGPath = r.SVGPath
	tpl.IsPublished = r.IsPublished
	if r.PublishedAt != nil {
		parsed, err := parseTimePointer(r.PublishedAt)
		if err != nil {
			return services.Template{}, err
		}
		tpl.PublishedAt = parsed
	}
	if r.Draft != nil {
		tpl.Draft = r.Draft.toModel()
	}
	return tpl, nil
}

type adminTemplateDraftPayload struct {
	Notes            string         `json:"notes,omitempty"`
	PreviewImagePath string         `json:"preview_image_path,omitempty"`
	PreviewSVGPath   string         `json:"preview_svg_path,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	UpdatedAt        string         `json:"updated_at,omitempty"`
	UpdatedBy        string         `json:"updated_by,omitempty"`
}

func (p *adminTemplateDraftPayload) toModel() services.TemplateDraft {
	if p == nil {
		return services.TemplateDraft{}
	}
	draft := services.TemplateDraft{
		Notes:            p.Notes,
		PreviewImagePath: p.PreviewImagePath,
		PreviewSVGPath:   p.PreviewSVGPath,
		Metadata:         p.Metadata,
	}
	if p.UpdatedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, p.UpdatedAt); err == nil {
			draft.UpdatedAt = parsed
		}
	}
	draft.UpdatedBy = p.UpdatedBy
	return draft
}

type adminTemplateResponse struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Description      string                     `json:"description,omitempty"`
	Category         string                     `json:"category,omitempty"`
	Style            string                     `json:"style,omitempty"`
	Tags             []string                   `json:"tags,omitempty"`
	PreviewImagePath string                     `json:"preview_image_path,omitempty"`
	SVGPath          string                     `json:"svg_path,omitempty"`
	IsPublished      bool                       `json:"is_published"`
	PublishedAt      string                     `json:"published_at,omitempty"`
	CreatedAt        string                     `json:"created_at"`
	UpdatedAt        string                     `json:"updated_at"`
	Version          int                        `json:"version"`
	Draft            *adminTemplateDraftPayload `json:"draft,omitempty"`
}

func newAdminTemplateResponse(template services.Template) adminTemplateResponse {
	resp := adminTemplateResponse{
		ID:               template.ID,
		Name:             template.Name,
		Description:      template.Description,
		Category:         template.Category,
		Style:            template.Style,
		Tags:             template.Tags,
		PreviewImagePath: template.PreviewImagePath,
		SVGPath:          template.SVGPath,
		IsPublished:      template.IsPublished,
		CreatedAt:        formatTimestamp(template.CreatedAt),
		UpdatedAt:        formatTimestamp(template.UpdatedAt),
		Version:          template.Version,
	}
	if !template.PublishedAt.IsZero() {
		resp.PublishedAt = formatTimestamp(template.PublishedAt)
	}
	if draft := templateDraftToPayload(template.Draft); draft != nil {
		resp.Draft = draft
	}
	return resp
}

func templateDraftToPayload(draft services.TemplateDraft) *adminTemplateDraftPayload {
	if isTemplateDraftEmpty(draft) {
		return nil
	}
	payload := &adminTemplateDraftPayload{
		Notes:            draft.Notes,
		PreviewImagePath: draft.PreviewImagePath,
		PreviewSVGPath:   draft.PreviewSVGPath,
		Metadata:         draft.Metadata,
		UpdatedBy:        draft.UpdatedBy,
	}
	if !draft.UpdatedAt.IsZero() {
		payload.UpdatedAt = formatTimestamp(draft.UpdatedAt)
	}
	return payload
}

func parseTimePointer(value *string) (time.Time, error) {
	if value == nil {
		return time.Time{}, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timestamp %q: %w", trimmed, err)
	}
	return parsed, nil
}

func decodeAdminFontRequest(r *http.Request, overrideID string) (services.FontSummary, error) {
	limited := io.LimitReader(r.Body, maxCatalogRequestBody)
	defer r.Body.Close()
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	var req adminFontRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return services.FontSummary{}, errors.New("request body required")
		}
		return services.FontSummary{}, fmt.Errorf("invalid request body: %w", err)
	}

	font := services.FontSummary{
		ID:               strings.TrimSpace(req.ID),
		DisplayName:      req.DisplayName,
		Family:           req.Family,
		Weight:           strings.ToLower(strings.TrimSpace(req.Weight)),
		Scripts:          copyStringSlice(req.Scripts),
		PreviewImagePath: req.PreviewImagePath,
		LetterSpacing:    req.LetterSpacing,
		IsPremium:        req.IsPremium,
		SupportedWeights: copyStringSlice(req.SupportedWeights),
		IsPublished:      req.IsPublished,
	}
	if req.License != nil {
		font.License = services.FontLicense{
			Name:          req.License.Name,
			URL:           req.License.URL,
			AllowedUsages: copyStringSlice(req.License.AllowedUsages),
		}
	}
	if strings.TrimSpace(overrideID) != "" {
		font.ID = strings.TrimSpace(overrideID)
	}
	return font, nil
}

func decodeAdminMaterialRequest(r *http.Request, overrideID string) (services.MaterialSummary, error) {
	limited := io.LimitReader(r.Body, maxCatalogRequestBody)
	defer r.Body.Close()
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()
	var req adminMaterialRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return services.MaterialSummary{}, errors.New("request body required")
		}
		return services.MaterialSummary{}, fmt.Errorf("invalid request body: %w", err)
	}
	material := services.MaterialSummary{
		ID:               strings.TrimSpace(req.ID),
		Name:             req.Name,
		Description:      req.Description,
		Category:         req.Category,
		Grain:            req.Grain,
		Color:            req.Color,
		IsAvailable:      req.IsAvailable,
		LeadTimeDays:     req.LeadTimeDays,
		PreviewImagePath: req.PreviewImagePath,
		DefaultLocale:    req.DefaultLocale,
	}
	if strings.TrimSpace(overrideID) != "" {
		material.ID = strings.TrimSpace(overrideID)
	}
	if material.ID == "" {
		return services.MaterialSummary{}, errors.New("material id is required")
	}
	if len(req.Translations) > 0 {
		material.Translations = make(map[string]services.MaterialTranslation, len(req.Translations))
		for key, value := range req.Translations {
			material.Translations[key] = services.MaterialTranslation{
				Locale:      value.Locale,
				Name:        value.Name,
				Description: value.Description,
			}
		}
	}
	if req.Procurement != nil {
		material.Procurement = services.MaterialProcurement{
			SupplierRef:  req.Procurement.SupplierRef,
			SupplierName: req.Procurement.SupplierName,
			ContactEmail: req.Procurement.ContactEmail,
			ContactPhone: req.Procurement.ContactPhone,
			Currency:     req.Procurement.Currency,
			Notes:        req.Procurement.Notes,
		}
		if req.Procurement.LeadTimeDays != nil {
			material.Procurement.LeadTimeDays = *req.Procurement.LeadTimeDays
		}
		if req.Procurement.MinimumOrderQuantity != nil {
			material.Procurement.MinimumOrderQuantity = *req.Procurement.MinimumOrderQuantity
		}
		if req.Procurement.UnitCostCents != nil {
			material.Procurement.UnitCostCents = *req.Procurement.UnitCostCents
		}
	}
	if req.Inventory != nil {
		material.Inventory = services.MaterialInventory{
			SKU:       req.Inventory.SKU,
			Warehouse: req.Inventory.Warehouse,
		}
		if req.Inventory.SafetyStock != nil {
			material.Inventory.SafetyStock = *req.Inventory.SafetyStock
		}
		if req.Inventory.ReorderPoint != nil {
			material.Inventory.ReorderPoint = *req.Inventory.ReorderPoint
		}
		if req.Inventory.ReorderQuantity != nil {
			material.Inventory.ReorderQuantity = *req.Inventory.ReorderQuantity
		}
	}
	return material, nil
}

type adminFontRequest struct {
	ID               string                   `json:"id"`
	DisplayName      string                   `json:"display_name"`
	Family           string                   `json:"family"`
	Weight           string                   `json:"weight"`
	Scripts          []string                 `json:"scripts"`
	PreviewImagePath string                   `json:"preview_image_path"`
	LetterSpacing    float64                  `json:"letter_spacing"`
	IsPremium        bool                     `json:"is_premium"`
	SupportedWeights []string                 `json:"supported_weights"`
	License          *adminFontLicensePayload `json:"license"`
	IsPublished      bool                     `json:"is_published"`
}

type adminFontResponse struct {
	ID               string                  `json:"id"`
	Slug             string                  `json:"slug,omitempty"`
	DisplayName      string                  `json:"display_name"`
	Family           string                  `json:"family"`
	Weight           string                  `json:"weight"`
	Scripts          []string                `json:"scripts"`
	PreviewImagePath string                  `json:"preview_image_path"`
	LetterSpacing    float64                 `json:"letter_spacing"`
	IsPremium        bool                    `json:"is_premium"`
	SupportedWeights []string                `json:"supported_weights"`
	License          adminFontLicensePayload `json:"license"`
	IsPublished      bool                    `json:"is_published"`
	CreatedAt        string                  `json:"created_at"`
	UpdatedAt        string                  `json:"updated_at"`
}

type adminFontLicensePayload struct {
	Name          string   `json:"name"`
	URL           string   `json:"url"`
	AllowedUsages []string `json:"allowed_usages"`
}

func newAdminFontResponse(font services.FontSummary) adminFontResponse {
	resp := adminFontResponse{
		ID:               font.ID,
		Slug:             fallbackNonEmpty(font.Slug, font.ID),
		DisplayName:      font.DisplayName,
		Family:           font.Family,
		Weight:           font.Weight,
		Scripts:          copyStringSlice(font.Scripts),
		PreviewImagePath: font.PreviewImagePath,
		LetterSpacing:    font.LetterSpacing,
		IsPremium:        font.IsPremium,
		SupportedWeights: copyStringSlice(font.SupportedWeights),
		License: adminFontLicensePayload{
			Name:          font.License.Name,
			URL:           strings.TrimSpace(font.License.URL),
			AllowedUsages: copyStringSlice(font.License.AllowedUsages),
		},
		IsPublished: font.IsPublished,
		CreatedAt:   formatTimestamp(font.CreatedAt),
		UpdatedAt:   formatTimestamp(font.UpdatedAt),
	}
	return resp
}

type adminMaterialRequest struct {
	ID               string                                     `json:"id"`
	Name             string                                     `json:"name"`
	Description      string                                     `json:"description"`
	Category         string                                     `json:"category"`
	Grain            string                                     `json:"grain"`
	Color            string                                     `json:"color"`
	IsAvailable      bool                                       `json:"is_available"`
	LeadTimeDays     int                                        `json:"lead_time_days"`
	PreviewImagePath string                                     `json:"preview_image_path"`
	DefaultLocale    string                                     `json:"default_locale"`
	Translations     map[string]adminMaterialTranslationPayload `json:"translations"`
	Procurement      *adminMaterialProcurementPayload           `json:"procurement"`
	Inventory        *adminMaterialInventoryPayload             `json:"inventory"`
}

type adminMaterialTranslationPayload struct {
	Locale      string `json:"locale"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type adminMaterialProcurementPayload struct {
	SupplierRef          string `json:"supplier_ref"`
	SupplierName         string `json:"supplier_name"`
	ContactEmail         string `json:"contact_email"`
	ContactPhone         string `json:"contact_phone"`
	LeadTimeDays         *int   `json:"lead_time_days"`
	MinimumOrderQuantity *int   `json:"minimum_order_quantity"`
	UnitCostCents        *int64 `json:"unit_cost_cents"`
	Currency             string `json:"currency"`
	Notes                string `json:"notes"`
}

type adminMaterialInventoryPayload struct {
	SKU             string `json:"sku"`
	SafetyStock     *int   `json:"safety_stock"`
	ReorderPoint    *int   `json:"reorder_point"`
	ReorderQuantity *int   `json:"reorder_quantity"`
	Warehouse       string `json:"warehouse"`
}

type adminMaterialResponse struct {
	ID               string                                     `json:"id"`
	Name             string                                     `json:"name"`
	Description      string                                     `json:"description,omitempty"`
	Category         string                                     `json:"category,omitempty"`
	Grain            string                                     `json:"grain,omitempty"`
	Color            string                                     `json:"color,omitempty"`
	IsAvailable      bool                                       `json:"is_available"`
	LeadTimeDays     int                                        `json:"lead_time_days"`
	PreviewImagePath string                                     `json:"preview_image_path,omitempty"`
	DefaultLocale    string                                     `json:"default_locale,omitempty"`
	Translations     map[string]adminMaterialTranslationPayload `json:"translations,omitempty"`
	Procurement      *adminMaterialProcurementResponse          `json:"procurement,omitempty"`
	Inventory        *adminMaterialInventoryResponse            `json:"inventory,omitempty"`
	CreatedAt        string                                     `json:"created_at"`
	UpdatedAt        string                                     `json:"updated_at"`
}

type adminMaterialProcurementResponse struct {
	SupplierRef          string `json:"supplier_ref,omitempty"`
	SupplierName         string `json:"supplier_name,omitempty"`
	ContactEmail         string `json:"contact_email,omitempty"`
	ContactPhone         string `json:"contact_phone,omitempty"`
	LeadTimeDays         int    `json:"lead_time_days,omitempty"`
	MinimumOrderQuantity int    `json:"minimum_order_quantity,omitempty"`
	UnitCostCents        int64  `json:"unit_cost_cents,omitempty"`
	Currency             string `json:"currency,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

type adminMaterialInventoryResponse struct {
	SKU             string `json:"sku,omitempty"`
	SafetyStock     int    `json:"safety_stock,omitempty"`
	ReorderPoint    int    `json:"reorder_point,omitempty"`
	ReorderQuantity int    `json:"reorder_quantity,omitempty"`
	Warehouse       string `json:"warehouse,omitempty"`
}

func newAdminMaterialResponse(material services.MaterialSummary) adminMaterialResponse {
	resp := adminMaterialResponse{
		ID:               material.ID,
		Name:             material.Name,
		Description:      material.Description,
		Category:         material.Category,
		Grain:            material.Grain,
		Color:            material.Color,
		IsAvailable:      material.IsAvailable,
		LeadTimeDays:     material.LeadTimeDays,
		PreviewImagePath: material.PreviewImagePath,
		DefaultLocale:    material.DefaultLocale,
		CreatedAt:        formatTimestamp(material.CreatedAt),
		UpdatedAt:        formatTimestamp(material.UpdatedAt),
	}
	if len(material.Translations) > 0 {
		resp.Translations = make(map[string]adminMaterialTranslationPayload, len(material.Translations))
		for key, value := range material.Translations {
			resp.Translations[key] = adminMaterialTranslationPayload{
				Locale:      value.Locale,
				Name:        value.Name,
				Description: value.Description,
			}
		}
	}
	if payload := materialProcurementToResponse(material.Procurement); payload != nil {
		resp.Procurement = payload
	}
	if payload := materialInventoryToResponse(material.Inventory); payload != nil {
		resp.Inventory = payload
	}
	return resp
}

type adminProductRequest struct {
	ID                    string                         `json:"id"`
	SKU                   string                         `json:"sku"`
	Name                  string                         `json:"name"`
	Description           string                         `json:"description"`
	Shape                 string                         `json:"shape"`
	SizesMm               []int                          `json:"sizes_mm"`
	DefaultMaterialID     string                         `json:"default_material_id"`
	MaterialIDs           []string                       `json:"material_ids"`
	BasePrice             int64                          `json:"base_price"`
	Currency              string                         `json:"currency"`
	ImagePaths            []string                       `json:"image_paths"`
	IsPublished           bool                           `json:"is_published"`
	IsCustomizable        bool                           `json:"is_customizable"`
	InventoryStatus       string                         `json:"inventory_status"`
	CompatibleTemplateIDs []string                       `json:"compatible_template_ids"`
	LeadTimeDays          int                            `json:"lead_time_days"`
	PriceTiers            []adminProductPriceTierRequest `json:"price_tiers"`
	Variants              []adminProductVariantRequest   `json:"variants"`
	Inventory             *adminProductInventoryRequest  `json:"inventory"`
	CreatedAt             *string                        `json:"created_at"`
	UpdatedAt             *string                        `json:"updated_at"`
}

type adminProductPriceTierRequest struct {
	MinQuantity int   `json:"min_quantity"`
	UnitPrice   int64 `json:"unit_price"`
}

type adminProductVariantRequest struct {
	Name    string                             `json:"name"`
	Label   string                             `json:"label"`
	Options []adminProductVariantOptionRequest `json:"options"`
}

type adminProductVariantOptionRequest struct {
	Value        string `json:"value"`
	Label        string `json:"label"`
	PriceDelta   int64  `json:"price_delta"`
	ImagePath    string `json:"image_path"`
	IsDefault    bool   `json:"is_default"`
	Availability string `json:"availability"`
}

type adminProductInventoryRequest struct {
	InitialStock *int `json:"initial_stock"`
	SafetyStock  *int `json:"safety_stock"`
}

type adminProductResponse struct {
	ID                    string                         `json:"id"`
	SKU                   string                         `json:"sku"`
	Name                  string                         `json:"name"`
	Description           string                         `json:"description,omitempty"`
	Shape                 string                         `json:"shape,omitempty"`
	SizesMm               []int                          `json:"sizes_mm,omitempty"`
	DefaultMaterialID     string                         `json:"default_material_id,omitempty"`
	MaterialIDs           []string                       `json:"material_ids,omitempty"`
	BasePrice             int64                          `json:"base_price,omitempty"`
	Currency              string                         `json:"currency,omitempty"`
	ImagePaths            []string                       `json:"image_paths,omitempty"`
	IsPublished           bool                           `json:"is_published"`
	IsCustomizable        bool                           `json:"is_customizable"`
	InventoryStatus       string                         `json:"inventory_status,omitempty"`
	CompatibleTemplateIDs []string                       `json:"compatible_template_ids,omitempty"`
	LeadTimeDays          int                            `json:"lead_time_days,omitempty"`
	PriceTiers            []adminProductPriceTierPayload `json:"price_tiers,omitempty"`
	Variants              []adminProductVariantPayload   `json:"variants,omitempty"`
	Inventory             *adminProductInventoryPayload  `json:"inventory,omitempty"`
	CreatedAt             string                         `json:"created_at,omitempty"`
	UpdatedAt             string                         `json:"updated_at,omitempty"`
}

type adminProductPriceTierPayload struct {
	MinQuantity int   `json:"min_quantity"`
	UnitPrice   int64 `json:"unit_price"`
}

type adminProductVariantPayload struct {
	Name    string                             `json:"name"`
	Label   string                             `json:"label,omitempty"`
	Options []adminProductVariantOptionPayload `json:"options"`
}

type adminProductVariantOptionPayload struct {
	Value        string `json:"value"`
	Label        string `json:"label,omitempty"`
	PriceDelta   int64  `json:"price_delta,omitempty"`
	ImagePath    string `json:"image_path,omitempty"`
	IsDefault    bool   `json:"is_default,omitempty"`
	Availability string `json:"availability,omitempty"`
}

type adminProductInventoryPayload struct {
	InitialStock int `json:"initial_stock,omitempty"`
	SafetyStock  int `json:"safety_stock,omitempty"`
}

func decodeAdminProductRequest(r *http.Request, overrideID string) (services.Product, error) {
	limited := io.LimitReader(r.Body, maxCatalogRequestBody)
	defer r.Body.Close()
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	var req adminProductRequest
	if err := decoder.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			return services.Product{}, errors.New("request body required")
		}
		return services.Product{}, fmt.Errorf("invalid request body: %w", err)
	}

	product := services.Product{
		ProductSummary: services.ProductSummary{
			ID:                    strings.TrimSpace(req.ID),
			SKU:                   strings.TrimSpace(req.SKU),
			Name:                  req.Name,
			Description:           req.Description,
			Shape:                 req.Shape,
			SizesMm:               copyIntSlice(req.SizesMm),
			DefaultMaterialID:     strings.TrimSpace(req.DefaultMaterialID),
			MaterialIDs:           copyStringSlice(req.MaterialIDs),
			BasePrice:             req.BasePrice,
			Currency:              strings.ToUpper(strings.TrimSpace(req.Currency)),
			ImagePaths:            copyStringSlice(req.ImagePaths),
			IsPublished:           req.IsPublished,
			IsCustomizable:        req.IsCustomizable,
			InventoryStatus:       req.InventoryStatus,
			CompatibleTemplateIDs: copyStringSlice(req.CompatibleTemplateIDs),
			LeadTimeDays:          req.LeadTimeDays,
		},
	}
	if strings.TrimSpace(overrideID) != "" {
		product.ID = strings.TrimSpace(overrideID)
	}
	if req.CreatedAt != nil {
		created, err := parseTimePointer(req.CreatedAt)
		if err != nil {
			return services.Product{}, err
		}
		if !created.IsZero() {
			product.CreatedAt = created
		}
	}
	if req.UpdatedAt != nil {
		updated, err := parseTimePointer(req.UpdatedAt)
		if err != nil {
			return services.Product{}, err
		}
		if !updated.IsZero() {
			product.UpdatedAt = updated
		}
	}
	if len(req.PriceTiers) > 0 {
		product.PriceTiers = make([]services.ProductPriceTier, 0, len(req.PriceTiers))
		for _, tier := range req.PriceTiers {
			product.PriceTiers = append(product.PriceTiers, services.ProductPriceTier{
				MinQuantity: tier.MinQuantity,
				UnitPrice:   tier.UnitPrice,
			})
		}
	}
	if len(req.Variants) > 0 {
		product.Variants = make([]services.ProductVariant, 0, len(req.Variants))
		for _, variant := range req.Variants {
			options := make([]services.ProductVariantOption, 0, len(variant.Options))
			for _, option := range variant.Options {
				options = append(options, services.ProductVariantOption{
					Value:        option.Value,
					Label:        option.Label,
					PriceDelta:   option.PriceDelta,
					ImagePath:    option.ImagePath,
					IsDefault:    option.IsDefault,
					Availability: option.Availability,
				})
			}
			product.Variants = append(product.Variants, services.ProductVariant{
				Name:    variant.Name,
				Label:   variant.Label,
				Options: options,
			})
		}
	}
	if req.Inventory != nil {
		if req.Inventory.SafetyStock != nil {
			product.Inventory.SafetyStock = *req.Inventory.SafetyStock
		}
		if req.Inventory.InitialStock != nil {
			product.Inventory.InitialStock = *req.Inventory.InitialStock
		}
	}
	return product, nil
}

func newAdminProductResponse(product services.Product) adminProductResponse {
	resp := adminProductResponse{
		ID:                    product.ID,
		SKU:                   product.SKU,
		Name:                  product.Name,
		Description:           product.Description,
		Shape:                 product.Shape,
		SizesMm:               copyIntSlice(product.SizesMm),
		DefaultMaterialID:     product.DefaultMaterialID,
		MaterialIDs:           copyStringSlice(product.MaterialIDs),
		BasePrice:             product.BasePrice,
		Currency:              product.Currency,
		ImagePaths:            copyStringSlice(product.ImagePaths),
		IsPublished:           product.IsPublished,
		IsCustomizable:        product.IsCustomizable,
		InventoryStatus:       product.InventoryStatus,
		CompatibleTemplateIDs: copyStringSlice(product.CompatibleTemplateIDs),
		LeadTimeDays:          product.LeadTimeDays,
	}
	if len(product.PriceTiers) > 0 {
		resp.PriceTiers = make([]adminProductPriceTierPayload, 0, len(product.PriceTiers))
		for _, tier := range product.PriceTiers {
			resp.PriceTiers = append(resp.PriceTiers, adminProductPriceTierPayload{
				MinQuantity: tier.MinQuantity,
				UnitPrice:   tier.UnitPrice,
			})
		}
	}
	if len(product.Variants) > 0 {
		resp.Variants = make([]adminProductVariantPayload, 0, len(product.Variants))
		for _, variant := range product.Variants {
			options := make([]adminProductVariantOptionPayload, 0, len(variant.Options))
			for _, option := range variant.Options {
				options = append(options, adminProductVariantOptionPayload{
					Value:        option.Value,
					Label:        option.Label,
					PriceDelta:   option.PriceDelta,
					ImagePath:    option.ImagePath,
					IsDefault:    option.IsDefault,
					Availability: option.Availability,
				})
			}
			resp.Variants = append(resp.Variants, adminProductVariantPayload{
				Name:    variant.Name,
				Label:   variant.Label,
				Options: options,
			})
		}
	}
	if !product.CreatedAt.IsZero() {
		resp.CreatedAt = formatTimestamp(product.CreatedAt)
	}
	if !product.UpdatedAt.IsZero() {
		resp.UpdatedAt = formatTimestamp(product.UpdatedAt)
	}
	if product.Inventory.InitialStock > 0 || product.Inventory.SafetyStock > 0 {
		resp.Inventory = &adminProductInventoryPayload{
			InitialStock: product.Inventory.InitialStock,
			SafetyStock:  product.Inventory.SafetyStock,
		}
	}
	return resp
}

func materialProcurementToResponse(info services.MaterialProcurement) *adminMaterialProcurementResponse {
	if strings.TrimSpace(info.SupplierRef) == "" && strings.TrimSpace(info.SupplierName) == "" &&
		strings.TrimSpace(info.ContactEmail) == "" && strings.TrimSpace(info.ContactPhone) == "" &&
		info.LeadTimeDays == 0 && info.MinimumOrderQuantity == 0 && info.UnitCostCents == 0 &&
		strings.TrimSpace(info.Currency) == "" && strings.TrimSpace(info.Notes) == "" {
		return nil
	}
	return &adminMaterialProcurementResponse{
		SupplierRef:          info.SupplierRef,
		SupplierName:         info.SupplierName,
		ContactEmail:         info.ContactEmail,
		ContactPhone:         info.ContactPhone,
		LeadTimeDays:         info.LeadTimeDays,
		MinimumOrderQuantity: info.MinimumOrderQuantity,
		UnitCostCents:        info.UnitCostCents,
		Currency:             info.Currency,
		Notes:                info.Notes,
	}
}

func materialInventoryToResponse(info services.MaterialInventory) *adminMaterialInventoryResponse {
	if strings.TrimSpace(info.SKU) == "" && info.SafetyStock == 0 && info.ReorderPoint == 0 && info.ReorderQuantity == 0 && strings.TrimSpace(info.Warehouse) == "" {
		return nil
	}
	return &adminMaterialInventoryResponse{
		SKU:             info.SKU,
		SafetyStock:     info.SafetyStock,
		ReorderPoint:    info.ReorderPoint,
		ReorderQuantity: info.ReorderQuantity,
		Warehouse:       info.Warehouse,
	}
}

func isTemplateDraftEmpty(draft services.TemplateDraft) bool {
	return strings.TrimSpace(draft.Notes) == "" &&
		strings.TrimSpace(draft.PreviewImagePath) == "" &&
		strings.TrimSpace(draft.PreviewSVGPath) == "" &&
		len(draft.Metadata) == 0 &&
		draft.UpdatedAt.IsZero() &&
		strings.TrimSpace(draft.UpdatedBy) == ""
}
