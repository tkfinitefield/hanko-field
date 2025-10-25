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

const maxTemplateRequestBody = 256 * 1024

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

func decodeAdminTemplateRequest(r *http.Request, overrideID string) (services.Template, error) {
	limited := io.LimitReader(r.Body, maxTemplateRequestBody)
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

func isTemplateDraftEmpty(draft services.TemplateDraft) bool {
	return strings.TrimSpace(draft.Notes) == "" &&
		strings.TrimSpace(draft.PreviewImagePath) == "" &&
		strings.TrimSpace(draft.PreviewSVGPath) == "" &&
		len(draft.Metadata) == 0 &&
		draft.UpdatedAt.IsZero() &&
		strings.TrimSpace(draft.UpdatedBy) == ""
}
