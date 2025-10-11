package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

const (
	defaultTemplatePageSize = 24
	maxTemplatePageSize     = 100
)

// AssetURLResolver resolves storage paths to externally accessible URLs (e.g. CDN or signed links).
type AssetURLResolver interface {
	ResolveURL(ctx context.Context, path string) (string, error)
}

// AssetURLResolverFunc adapts a function to the AssetURLResolver interface.
type AssetURLResolverFunc func(ctx context.Context, path string) (string, error)

// ResolveURL implements AssetURLResolver.
func (fn AssetURLResolverFunc) ResolveURL(ctx context.Context, path string) (string, error) {
	if fn == nil {
		return path, nil
	}
	return fn(ctx, path)
}

// PublicHandlers exposes unauthenticated catalog endpoints.
type PublicHandlers struct {
	catalog         services.CatalogService
	previewResolver AssetURLResolver
	vectorResolver  AssetURLResolver
}

// PublicOption customises construction of PublicHandlers.
type PublicOption func(*PublicHandlers)

// WithPublicCatalogService injects the catalog service dependency.
func WithPublicCatalogService(svc services.CatalogService) PublicOption {
	return func(h *PublicHandlers) {
		h.catalog = svc
	}
}

// WithPublicPreviewResolver sets the resolver used for preview image URLs.
func WithPublicPreviewResolver(resolver AssetURLResolver) PublicOption {
	return func(h *PublicHandlers) {
		h.previewResolver = resolver
	}
}

// WithPublicVectorResolver sets the resolver used for SVG/vector URLs.
func WithPublicVectorResolver(resolver AssetURLResolver) PublicOption {
	return func(h *PublicHandlers) {
		h.vectorResolver = resolver
	}
}

// NewPublicHandlers constructs handlers for public catalog endpoints.
func NewPublicHandlers(opts ...PublicOption) *PublicHandlers {
	handler := &PublicHandlers{
		previewResolver: AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return path, nil
		}),
		vectorResolver: AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return path, nil
		}),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(handler)
		}
	}
	return handler
}

// Routes registers public catalog endpoints against the provided router.
func (h *PublicHandlers) Routes(r chi.Router) {
	if r == nil {
		return
	}
	r.Get("/templates", h.listTemplates)
	r.Get("/templates/{templateID}", h.getTemplate)
}

func (h *PublicHandlers) listTemplates(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	filter, err := parseTemplateListFilter(r)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	filter.PublishedOnly = true

	page, err := h.catalog.ListTemplates(r.Context(), filter)
	if err != nil {
		writeCatalogError(r.Context(), w, err)
		return
	}

	items := make([]templatePayload, 0, len(page.Items))
	for _, tpl := range page.Items {
		if !tpl.IsPublished {
			continue
		}
		previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, tpl.PreviewImagePath)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		items = append(items, templatePayload{
			ID:          tpl.ID,
			Name:        tpl.Name,
			Description: tpl.Description,
			Category:    tpl.Category,
			Style:       tpl.Style,
			Tags:        copyStringSlice(tpl.Tags),
			PreviewURL:  previewURL,
			Popularity:  tpl.Popularity,
			CreatedAt:   formatTimestamp(tpl.CreatedAt),
			UpdatedAt:   formatTimestamp(tpl.UpdatedAt),
		})
	}

	response := templateListResponse{
		Templates:     items,
		NextPageToken: page.NextPageToken,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicHandlers) getTemplate(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	templateID := strings.TrimSpace(chi.URLParam(r, "templateID"))
	if templateID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_template_id", "template id is required", http.StatusBadRequest))
		return
	}

	template, err := h.catalog.GetTemplate(r.Context(), templateID)
	if err != nil {
		writeCatalogError(r.Context(), w, err)
		return
	}
	if !template.IsPublished {
		httpx.WriteError(r.Context(), w, httpx.NewError("template_not_found", "template not found", http.StatusNotFound))
		return
	}

	previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, template.PreviewImagePath)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}
	svgURL, err := h.resolveAsset(r.Context(), h.vectorResolver, template.SVGPath)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}

	payload := templatePayload{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Category:    template.Category,
		Style:       template.Style,
		Tags:        copyStringSlice(template.Tags),
		PreviewURL:  previewURL,
		SVGURL:      svgURL,
		Popularity:  template.Popularity,
		CreatedAt:   formatTimestamp(template.CreatedAt),
		UpdatedAt:   formatTimestamp(template.UpdatedAt),
	}
	writeJSON(w, http.StatusOK, payload)
}

func (h *PublicHandlers) resolveAsset(ctx context.Context, resolver AssetURLResolver, path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	if isAbsoluteURL(path) {
		return path, nil
	}
	if resolver == nil {
		return path, nil
	}
	return resolver.ResolveURL(ctx, path)
}

func parseTemplateListFilter(r *http.Request) (services.TemplateFilter, error) {
	if r == nil {
		return services.TemplateFilter{}, errors.New("request cannot be nil")
	}
	values := r.URL.Query()

	filter := services.TemplateFilter{
		Pagination: services.Pagination{
			PageToken: strings.TrimSpace(values.Get("pageToken")),
		},
		SortBy:    domain.TemplateSortPopularity,
		SortOrder: domain.SortDesc,
	}

	if category := strings.TrimSpace(values.Get("category")); category != "" {
		filter.Category = &category
	}
	if style := strings.TrimSpace(values.Get("style")); style != "" {
		filter.Style = &style
	}
	filter.Tags = parseTagParameters(values)

	if pageSize, err := parsePageSize(values.Get("pageSize")); err != nil {
		return services.TemplateFilter{}, err
	} else {
		filter.Pagination.PageSize = pageSize
	}

	if sort := strings.TrimSpace(values.Get("sort")); sort != "" {
		sortField, order, err := parseSort(sort)
		if err != nil {
			return services.TemplateFilter{}, err
		}
		filter.SortBy = sortField
		filter.SortOrder = order
	}

	return filter, nil
}

func parsePageSize(raw string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultTemplatePageSize, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid pageSize: %w", err)
	}
	if value <= 0 {
		return 0, errors.New("pageSize must be greater than zero")
	}
	if value > maxTemplatePageSize {
		value = maxTemplatePageSize
	}
	return value, nil
}

func parseSort(raw string) (domain.TemplateSort, domain.SortOrder, error) {
	order := domain.SortDesc
	field := strings.TrimSpace(raw)
	if field == "" {
		return domain.TemplateSortPopularity, order, nil
	}

	switch field[0] {
	case '-':
		field = strings.TrimSpace(field[1:])
		order = domain.SortDesc
	case '+':
		field = strings.TrimSpace(field[1:])
		order = domain.SortAsc
	default:
		order = domain.SortDesc
	}

	switch strings.ToLower(field) {
	case "", "popularity":
		return domain.TemplateSortPopularity, order, nil
	case "createdat":
		return domain.TemplateSortCreatedAt, order, nil
	default:
		return "", "", fmt.Errorf("invalid sort value %q", raw)
	}
}

func parseTagParameters(values url.Values) []string {
	if values == nil {
		return nil
	}
	var raw []string
	raw = append(raw, values["tag"]...)
	raw = append(raw, values["tags"]...)

	if len(raw) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	var result []string
	for _, entry := range raw {
		for _, part := range strings.Split(entry, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func writeCatalogError(ctx context.Context, w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, services.ErrCatalogRepositoryMissing):
		httpx.WriteError(ctx, w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError("template_not_found", "template not found", http.StatusNotFound))
			return
		case repoErr.IsUnavailable():
			httpx.WriteError(ctx, w, httpx.NewError("catalog_unavailable", "catalog repository unavailable", http.StatusServiceUnavailable))
			return
		default:
			httpx.WriteError(ctx, w, httpx.NewError("catalog_error", err.Error(), http.StatusInternalServerError))
			return
		}
	}

	httpx.WriteError(ctx, w, httpx.NewError("catalog_error", err.Error(), http.StatusInternalServerError))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type templateListResponse struct {
	Templates     []templatePayload `json:"templates"`
	NextPageToken string            `json:"nextPageToken,omitempty"`
}

type templatePayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Style       string   `json:"style,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PreviewURL  string   `json:"previewUrl,omitempty"`
	SVGURL      string   `json:"svgUrl,omitempty"`
	Popularity  int      `json:"popularity,omitempty"`
	CreatedAt   string   `json:"createdAt,omitempty"`
	UpdatedAt   string   `json:"updatedAt,omitempty"`
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func copyStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func isAbsoluteURL(raw string) bool {
	if raw == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}
