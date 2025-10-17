package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/platform/httpx"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

const (
	defaultTemplatePageSize   = 24
	maxTemplatePageSize       = 100
	defaultFontPageSize       = 50
	maxFontPageSize           = 100
	defaultMaterialPageSize   = 32
	maxMaterialPageSize       = 100
	defaultProductPageSize    = 24
	maxProductPageSize        = 100
	fontCacheControl          = "public, max-age=300"
	materialCacheControl      = "public, max-age=900"
	productCacheControl       = "public, max-age=300"
	defaultMaterialLocale     = "ja"
	defaultGuidePageSize      = 20
	maxGuidePageSize          = 60
	defaultGuideLocale        = "ja"
	guideCacheControl         = "public, max-age=900"
	priceDisplayModeInclusive = "tax_inclusive"
	priceDisplayModeExclusive = "tax_exclusive"
)

var guideHTMLPolicy = newGuideHTMLPolicy()

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
	catalog          services.CatalogService
	content          services.ContentService
	previewResolver  AssetURLResolver
	vectorResolver   AssetURLResolver
	priceDisplayMode string
}

// PublicOption customises construction of PublicHandlers.
type PublicOption func(*PublicHandlers)

// WithPublicCatalogService injects the catalog service dependency.
func WithPublicCatalogService(svc services.CatalogService) PublicOption {
	return func(h *PublicHandlers) {
		h.catalog = svc
	}
}

// WithPublicContentService injects the content service dependency.
func WithPublicContentService(svc services.ContentService) PublicOption {
	return func(h *PublicHandlers) {
		h.content = svc
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

// WithPublicPriceDisplayMode sets the price display mode surfaced to clients.
func WithPublicPriceDisplayMode(mode string) PublicOption {
	return func(h *PublicHandlers) {
		if h == nil {
			return
		}
		h.priceDisplayMode = normalizePriceDisplayMode(mode)
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
		priceDisplayMode: priceDisplayModeInclusive,
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
	r.Get("/fonts", h.listFonts)
	r.Get("/fonts/{fontID}", h.getFont)
	r.Get("/materials", h.listMaterials)
	r.Get("/materials/{materialID}", h.getMaterial)
	r.Get("/products", h.listProducts)
	r.Get("/products/{productID}", h.getProduct)
	r.Get("/content/guides", h.listGuides)
	r.Get("/content/guides/{slug}", h.getGuide)
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
		writeCatalogError(r.Context(), w, err, "template")
		return
	}

	items := make([]templatePayload, 0, len(page.Items))
	for _, tpl := range page.Items {
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
		writeCatalogError(r.Context(), w, err, "template")
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

func (h *PublicHandlers) listFonts(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	filter, err := parseFontListFilter(r)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	filter.PublishedOnly = true

	page, err := h.catalog.ListFonts(r.Context(), filter)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "font")
		return
	}

	items := make([]fontPayload, 0, len(page.Items))
	for _, font := range page.Items {
		previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, font.PreviewImagePath)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		items = append(items, fontPayload{
			ID:               font.ID,
			DisplayName:      font.DisplayName,
			Family:           font.Family,
			Scripts:          copyStringSlice(font.Scripts),
			PreviewURL:       previewURL,
			LetterSpacing:    font.LetterSpacing,
			IsPremium:        font.IsPremium,
			SupportedWeights: copyStringSlice(font.SupportedWeights),
			License: fontLicensePayload{
				Name: font.License.Name,
				URL:  strings.TrimSpace(font.License.URL),
			},
			CreatedAt: formatTimestamp(font.CreatedAt),
			UpdatedAt: formatTimestamp(font.UpdatedAt),
		})
	}

	w.Header().Set("Cache-Control", fontCacheControl)
	response := fontListResponse{
		Fonts:         items,
		NextPageToken: page.NextPageToken,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicHandlers) getFont(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	fontID := strings.TrimSpace(chi.URLParam(r, "fontID"))
	if fontID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_font_id", "font id is required", http.StatusBadRequest))
		return
	}

	font, err := h.catalog.GetFont(r.Context(), fontID)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "font")
		return
	}
	if !font.IsPublished {
		httpx.WriteError(r.Context(), w, httpx.NewError("font_not_found", "font not found", http.StatusNotFound))
		return
	}

	previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, font.PreviewImagePath)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}

	payload := fontPayload{
		ID:               font.ID,
		DisplayName:      font.DisplayName,
		Family:           font.Family,
		Scripts:          copyStringSlice(font.Scripts),
		PreviewURL:       previewURL,
		LetterSpacing:    font.LetterSpacing,
		IsPremium:        font.IsPremium,
		SupportedWeights: copyStringSlice(font.SupportedWeights),
		License: fontLicensePayload{
			Name: font.License.Name,
			URL:  strings.TrimSpace(font.License.URL),
		},
		CreatedAt: formatTimestamp(font.CreatedAt),
		UpdatedAt: formatTimestamp(font.UpdatedAt),
	}

	w.Header().Set("Cache-Control", fontCacheControl)
	writeJSON(w, http.StatusOK, payload)
}

func (h *PublicHandlers) listMaterials(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	filter, err := parseMaterialListFilter(r)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	filter.OnlyAvailable = true

	page, err := h.catalog.ListMaterials(r.Context(), filter)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "material")
		return
	}

	items := make([]materialPayload, 0, len(page.Items))
	for _, material := range page.Items {
		name, description, resolvedLocale := resolveMaterialLocalization(material, filter.Locale)
		previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, material.PreviewImagePath)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		items = append(items, materialPayload{
			ID:           material.ID,
			Name:         name,
			Description:  description,
			Category:     material.Category,
			Grain:        material.Grain,
			Color:        material.Color,
			IsAvailable:  material.IsAvailable,
			LeadTimeDays: material.LeadTimeDays,
			PreviewURL:   previewURL,
			Locale:       resolvedLocale,
			CreatedAt:    formatTimestamp(material.CreatedAt),
			UpdatedAt:    formatTimestamp(material.UpdatedAt),
		})
	}

	w.Header().Set("Cache-Control", materialCacheControl)
	response := materialListResponse{
		Materials:     items,
		NextPageToken: page.NextPageToken,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicHandlers) getMaterial(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	materialID := strings.TrimSpace(chi.URLParam(r, "materialID"))
	if materialID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_material_id", "material id is required", http.StatusBadRequest))
		return
	}

	locale := normalizeLocale(r.URL.Query().Get("lang"))
	if locale == "" {
		locale = defaultMaterialLocale
	}

	material, err := h.catalog.GetMaterial(r.Context(), materialID)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "material")
		return
	}

	name, description, resolvedLocale := resolveMaterialLocalization(material.MaterialSummary, locale)
	previewURL, err := h.resolveAsset(r.Context(), h.previewResolver, material.PreviewImagePath)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}

	photoURLs := make([]string, 0, len(material.Photos))
	for _, photo := range material.Photos {
		resolved, err := h.resolveAsset(r.Context(), h.previewResolver, photo)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		if resolved != "" {
			photoURLs = append(photoURLs, resolved)
		}
	}

	var sustainability *materialSustainabilityPayload
	if len(material.Sustainability.Certifications) > 0 || strings.TrimSpace(material.Sustainability.Notes) != "" {
		sustainability = &materialSustainabilityPayload{
			Certifications: copyStringSlice(material.Sustainability.Certifications),
			Notes:          strings.TrimSpace(material.Sustainability.Notes),
		}
	}

	payload := materialDetailPayload{
		materialPayload: materialPayload{
			ID:           material.ID,
			Name:         name,
			Description:  description,
			Category:     material.Category,
			Grain:        material.Grain,
			Color:        material.Color,
			IsAvailable:  material.IsAvailable,
			LeadTimeDays: material.LeadTimeDays,
			PreviewURL:   previewURL,
			Locale:       resolvedLocale,
			CreatedAt:    formatTimestamp(material.CreatedAt),
			UpdatedAt:    formatTimestamp(material.UpdatedAt),
		},
		Finish:         strings.TrimSpace(material.Finish),
		Hardness:       material.Hardness,
		Density:        material.Density,
		CareNotes:      strings.TrimSpace(material.CareNotes),
		Photos:         photoURLs,
		Sustainability: sustainability,
	}

	w.Header().Set("Cache-Control", materialCacheControl)
	writeJSON(w, http.StatusOK, payload)
}

func (h *PublicHandlers) listProducts(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	filter, err := parseProductListFilter(r)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}
	filter.PublishedOnly = true
	if filter.Pagination.PageSize == 0 {
		filter.Pagination.PageSize = defaultProductPageSize
	}

	page, err := h.catalog.ListProducts(r.Context(), filter)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "product")
		return
	}

	items := make([]productPayload, 0, len(page.Items))
	for _, product := range page.Items {
		payload, err := h.buildProductPayload(r.Context(), product)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		items = append(items, payload)
	}

	w.Header().Set("Cache-Control", productCacheControl)
	response := productListResponse{
		Products:      items,
		NextPageToken: page.NextPageToken,
		PriceDisplay:  h.currentPriceDisplayMode(),
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicHandlers) getProduct(w http.ResponseWriter, r *http.Request) {
	if h.catalog == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	productID := strings.TrimSpace(chi.URLParam(r, "productID"))
	if productID == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_product_id", "product id is required", http.StatusBadRequest))
		return
	}

	product, err := h.catalog.GetProduct(r.Context(), productID)
	if err != nil {
		writeCatalogError(r.Context(), w, err, "product")
		return
	}

	payload, err := h.buildProductDetailPayload(r.Context(), product)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}

	w.Header().Set("Cache-Control", productCacheControl)
	writeJSON(w, http.StatusOK, payload)
}

func (h *PublicHandlers) listGuides(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("content_unavailable", "content service is unavailable", http.StatusServiceUnavailable))
		return
	}

	filter, locale, err := parseGuideListFilter(r)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_request", err.Error(), http.StatusBadRequest))
		return
	}

	page, err := h.content.ListGuides(r.Context(), filter)
	if err != nil {
		writeContentError(r.Context(), w, err, "guide")
		return
	}

	ordered := make([]services.ContentGuide, len(page.Items))
	copy(ordered, page.Items)
	sort.SliceStable(ordered, func(i, j int) bool {
		return guideLess(ordered[i], ordered[j])
	})

	summaries := make([]guideSummaryPayload, 0, len(ordered))
	for _, guide := range ordered {
		summary, err := h.buildGuideSummaryPayload(r.Context(), guide)
		if err != nil {
			httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
			return
		}
		summaries = append(summaries, summary)
	}

	w.Header().Set("Cache-Control", guideCacheControl)
	etag := computeGuideListETag(ordered, locale)
	if etag != "" {
		w.Header().Set("ETag", etag)
		if matchesETag(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	response := guideListResponse{
		Guides:        summaries,
		NextPageToken: page.NextPageToken,
		Locale:        locale,
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *PublicHandlers) getGuide(w http.ResponseWriter, r *http.Request) {
	if h.content == nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("content_unavailable", "content service is unavailable", http.StatusServiceUnavailable))
		return
	}

	slug := strings.TrimSpace(chi.URLParam(r, "slug"))
	if slug == "" {
		httpx.WriteError(r.Context(), w, httpx.NewError("invalid_slug", "guide slug is required", http.StatusBadRequest))
		return
	}

	locale := normalizeLocale(r.URL.Query().Get("lang"))
	if locale == "" {
		locale = defaultGuideLocale
	}

	guide, err := h.content.GetGuideBySlug(r.Context(), slug, locale)
	if err != nil {
		writeContentError(r.Context(), w, err, "guide")
		return
	}

	detail, err := h.buildGuideDetailPayload(r.Context(), guide)
	if err != nil {
		httpx.WriteError(r.Context(), w, httpx.NewError("asset_resolution_failed", err.Error(), http.StatusInternalServerError))
		return
	}

	w.Header().Set("Cache-Control", guideCacheControl)
	etag := computeGuideETag(guide)
	if etag != "" {
		w.Header().Set("ETag", etag)
		if matchesETag(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}
	writeJSON(w, http.StatusOK, detail)
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

func (h *PublicHandlers) buildProductPayload(ctx context.Context, product services.ProductSummary) (productPayload, error) {
	imageURLs := make([]string, 0, len(product.ImagePaths))
	for _, path := range product.ImagePaths {
		resolved, err := h.resolveAsset(ctx, h.previewResolver, path)
		if err != nil {
			return productPayload{}, err
		}
		if strings.TrimSpace(resolved) != "" {
			imageURLs = append(imageURLs, resolved)
		}
	}

	preview := ""
	if len(imageURLs) > 0 {
		preview = imageURLs[0]
	}

	payload := productPayload{
		ID:                    strings.TrimSpace(product.ID),
		SKU:                   strings.TrimSpace(product.SKU),
		Name:                  strings.TrimSpace(product.Name),
		Description:           strings.TrimSpace(product.Description),
		Shape:                 strings.TrimSpace(product.Shape),
		SizesMm:               copyIntSlice(product.SizesMm),
		DefaultMaterialID:     strings.TrimSpace(product.DefaultMaterialID),
		MaterialIDs:           copyStringSlice(product.MaterialIDs),
		BasePrice:             product.BasePrice,
		Currency:              strings.TrimSpace(product.Currency),
		PreviewURL:            preview,
		ImageURLs:             imageURLs,
		IsCustomizable:        product.IsCustomizable,
		InventoryStatus:       strings.TrimSpace(product.InventoryStatus),
		CompatibleTemplateIDs: copyStringSlice(product.CompatibleTemplateIDs),
		LeadTimeDays:          product.LeadTimeDays,
		PriceDisplay:          h.currentPriceDisplayMode(),
		CreatedAt:             formatTimestamp(product.CreatedAt),
		UpdatedAt:             formatTimestamp(product.UpdatedAt),
	}

	return payload, nil
}

func (h *PublicHandlers) buildProductDetailPayload(ctx context.Context, product services.Product) (productDetailPayload, error) {
	base, err := h.buildProductPayload(ctx, product.ProductSummary)
	if err != nil {
		return productDetailPayload{}, err
	}

	tiers := make([]productPriceTierPayload, 0, len(product.PriceTiers))
	for _, tier := range product.PriceTiers {
		if tier.MinQuantity <= 0 && tier.UnitPrice == 0 {
			continue
		}
		tiers = append(tiers, productPriceTierPayload{
			MinQuantity: tier.MinQuantity,
			UnitPrice:   tier.UnitPrice,
			Currency:    base.Currency,
		})
	}

	return productDetailPayload{
		productPayload: base,
		PriceTiers:     tiers,
	}, nil
}

func (h *PublicHandlers) currentPriceDisplayMode() string {
	return normalizePriceDisplayMode(h.priceDisplayMode)
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

func parseFontListFilter(r *http.Request) (services.FontFilter, error) {
	if r == nil {
		return services.FontFilter{}, errors.New("request cannot be nil")
	}
	values := r.URL.Query()

	filter := services.FontFilter{
		Pagination: services.Pagination{
			PageToken: strings.TrimSpace(values.Get("pageToken")),
		},
	}

	if script := strings.TrimSpace(values.Get("script")); script != "" {
		normalized := strings.ToLower(script)
		filter.Script = &normalized
	}

	if premium, err := parseOptionalBoolParam("isPremium", values.Get("isPremium")); err != nil {
		return services.FontFilter{}, err
	} else {
		filter.IsPremium = premium
	}

	if pageSize, err := parseFontPageSize(values.Get("pageSize")); err != nil {
		return services.FontFilter{}, err
	} else {
		filter.Pagination.PageSize = pageSize
	}

	return filter, nil
}

func parseGuideListFilter(r *http.Request) (services.ContentGuideFilter, string, error) {
	if r == nil {
		return services.ContentGuideFilter{}, "", errors.New("request cannot be nil")
	}
	values := r.URL.Query()

	locale := normalizeLocale(values.Get("lang"))
	if locale == "" {
		locale = defaultGuideLocale
	}

	filter := services.ContentGuideFilter{
		PublishedOnly:  true,
		FallbackLocale: defaultGuideLocale,
		Pagination: services.Pagination{
			PageToken: strings.TrimSpace(values.Get("pageToken")),
		},
	}

	if locale != "" {
		localeCopy := locale
		filter.Locale = &localeCopy
	}

	if category := strings.TrimSpace(values.Get("category")); category != "" {
		normalized := strings.ToLower(category)
		filter.Category = &normalized
	}

	pageSize, err := parseGuidePageSize(values.Get("pageSize"))
	if err != nil {
		return services.ContentGuideFilter{}, "", err
	}
	filter.Pagination.PageSize = pageSize

	return filter, locale, nil
}

func parseMaterialListFilter(r *http.Request) (services.MaterialFilter, error) {
	if r == nil {
		return services.MaterialFilter{}, errors.New("request cannot be nil")
	}
	values := r.URL.Query()

	locale := normalizeLocale(values.Get("lang"))
	if locale == "" {
		locale = defaultMaterialLocale
	}

	filter := services.MaterialFilter{
		Locale: locale,
		Pagination: services.Pagination{
			PageToken: strings.TrimSpace(values.Get("pageToken")),
		},
	}

	if category := strings.TrimSpace(values.Get("category")); category != "" {
		normalized := strings.ToLower(category)
		filter.Category = &normalized
	}

	if pageSize, err := parseMaterialPageSize(values.Get("pageSize")); err != nil {
		return services.MaterialFilter{}, err
	} else {
		filter.Pagination.PageSize = pageSize
	}

	return filter, nil
}

func parseProductListFilter(r *http.Request) (services.ProductFilter, error) {
	if r == nil {
		return services.ProductFilter{}, errors.New("request cannot be nil")
	}
	values := r.URL.Query()

	filter := services.ProductFilter{
		Pagination: services.Pagination{
			PageToken: strings.TrimSpace(values.Get("pageToken")),
		},
	}

	if shape := strings.TrimSpace(values.Get("shape")); shape != "" {
		normalized := strings.ToLower(shape)
		filter.Shape = &normalized
	}

	if size := strings.TrimSpace(values.Get("size")); size != "" {
		normalized := strings.TrimSpace(size)
		if len(normalized) >= 2 {
			lower := strings.ToLower(normalized)
			if strings.HasSuffix(lower, "mm") {
				normalized = strings.TrimSpace(normalized[:len(normalized)-2])
			}
		}
		value, err := strconv.Atoi(normalized)
		if err != nil {
			return services.ProductFilter{}, fmt.Errorf("invalid size: %w", err)
		}
		if value <= 0 {
			return services.ProductFilter{}, errors.New("size must be greater than zero")
		}
		filter.SizeMm = &value
	}

	if material := strings.TrimSpace(values.Get("material")); material != "" {
		materialID := material
		filter.MaterialID = &materialID
	}

	if customizable, err := parseOptionalBoolParam("isCustomizable", values.Get("isCustomizable")); err != nil {
		return services.ProductFilter{}, err
	} else {
		filter.IsCustomizable = customizable
	}

	if pageSize, err := parseProductPageSize(values.Get("pageSize")); err != nil {
		return services.ProductFilter{}, err
	} else {
		filter.Pagination.PageSize = pageSize
	}

	return filter, nil
}

func parsePageSize(raw string) (int, error) {
	return parseLimitedPageSize(raw, defaultTemplatePageSize, maxTemplatePageSize)
}

func parseFontPageSize(raw string) (int, error) {
	return parseLimitedPageSize(raw, defaultFontPageSize, maxFontPageSize)
}

func parseMaterialPageSize(raw string) (int, error) {
	return parseLimitedPageSize(raw, defaultMaterialPageSize, maxMaterialPageSize)
}

func parseProductPageSize(raw string) (int, error) {
	return parseLimitedPageSize(raw, defaultProductPageSize, maxProductPageSize)
}

func parseGuidePageSize(raw string) (int, error) {
	return parseLimitedPageSize(raw, defaultGuidePageSize, maxGuidePageSize)
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

func parseLimitedPageSize(raw string, defaultSize, maxSize int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultSize, nil
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid pageSize: %w", err)
	}
	if value <= 0 {
		return 0, errors.New("pageSize must be greater than zero")
	}
	if value > maxSize {
		value = maxSize
	}
	return value, nil
}

func parseOptionalBoolParam(name string, raw string) (*bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	return &value, nil
}

func normalizeLocale(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.ReplaceAll(trimmed, "_", "-")
	return strings.ToLower(normalized)
}

func normalizePriceDisplayMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case priceDisplayModeExclusive:
		return priceDisplayModeExclusive
	case priceDisplayModeInclusive:
		return priceDisplayModeInclusive
	default:
		return priceDisplayModeInclusive
	}
}

func resolveMaterialLocalization(material services.MaterialSummary, requestedLocale string) (string, string, string) {
	baseName := strings.TrimSpace(material.Name)
	baseDescription := strings.TrimSpace(material.Description)
	defaultLocale := normalizeLocale(material.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = defaultMaterialLocale
	}

	requested := normalizeLocale(requestedLocale)
	if requested == "" {
		requested = defaultLocale
	}

	name := baseName
	description := baseDescription
	resolvedLocale := defaultLocale

	if translation, ok := findMaterialTranslation(material, requested); ok {
		name = fallbackNonEmpty(strings.TrimSpace(translation.Name), name)
		description = fallbackNonEmpty(strings.TrimSpace(translation.Description), description)
		if loc := normalizeLocale(translation.Locale); loc != "" {
			resolvedLocale = loc
		} else {
			resolvedLocale = requested
		}
		return name, description, resolvedLocale
	}

	if dash := strings.Index(requested, "-"); dash > 0 {
		base := requested[:dash]
		if translation, ok := findMaterialTranslation(material, base); ok {
			name = fallbackNonEmpty(strings.TrimSpace(translation.Name), name)
			description = fallbackNonEmpty(strings.TrimSpace(translation.Description), description)
			if loc := normalizeLocale(translation.Locale); loc != "" {
				resolvedLocale = loc
			} else {
				resolvedLocale = base
			}
			return name, description, resolvedLocale
		}
	}

	return name, description, resolvedLocale
}

func findMaterialTranslation(material services.MaterialSummary, locale string) (services.MaterialTranslation, bool) {
	target := normalizeLocale(locale)
	if target == "" || len(material.Translations) == 0 {
		return services.MaterialTranslation{}, false
	}
	for key, translation := range material.Translations {
		if normalizeLocale(key) == target {
			return translation, true
		}
		if normalizeLocale(translation.Locale) == target {
			return translation, true
		}
	}
	return services.MaterialTranslation{}, false
}

func fallbackNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(fallback)
}

func writeCatalogError(ctx context.Context, w http.ResponseWriter, err error, resource string) {
	if err == nil {
		return
	}
	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "resource"
	}
	codePrefix := strings.ToLower(resource)

	switch {
	case errors.Is(err, services.ErrCatalogRepositoryMissing):
		httpx.WriteError(ctx, w, httpx.NewError("catalog_unavailable", "catalog service is unavailable", http.StatusServiceUnavailable))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError(fmt.Sprintf("%s_not_found", codePrefix), fmt.Sprintf("%s not found", resource), http.StatusNotFound))
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

func writeContentError(ctx context.Context, w http.ResponseWriter, err error, resource string) {
	if err == nil {
		return
	}
	resource = strings.TrimSpace(resource)
	if resource == "" {
		resource = "resource"
	}

	switch {
	case errors.Is(err, services.ErrContentRepositoryMissing):
		httpx.WriteError(ctx, w, httpx.NewError("content_unavailable", "content service is unavailable", http.StatusServiceUnavailable))
		return
	}

	var repoErr repositories.RepositoryError
	if errors.As(err, &repoErr) {
		switch {
		case repoErr.IsNotFound():
			httpx.WriteError(ctx, w, httpx.NewError(fmt.Sprintf("%s_not_found", resource), fmt.Sprintf("%s not found", resource), http.StatusNotFound))
			return
		case repoErr.IsUnavailable():
			httpx.WriteError(ctx, w, httpx.NewError("content_unavailable", "content repository unavailable", http.StatusServiceUnavailable))
			return
		default:
			httpx.WriteError(ctx, w, httpx.NewError("content_error", err.Error(), http.StatusInternalServerError))
			return
		}
	}

	httpx.WriteError(ctx, w, httpx.NewError("content_error", err.Error(), http.StatusInternalServerError))
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type templateListResponse struct {
	Templates     []templatePayload `json:"templates"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

type templatePayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Category    string   `json:"category,omitempty"`
	Style       string   `json:"style,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PreviewURL  string   `json:"preview_url,omitempty"`
	SVGURL      string   `json:"svg_url,omitempty"`
	Popularity  int      `json:"popularity,omitempty"`
	CreatedAt   string   `json:"created_at,omitempty"`
	UpdatedAt   string   `json:"updated_at,omitempty"`
}

type fontListResponse struct {
	Fonts         []fontPayload `json:"fonts"`
	NextPageToken string        `json:"next_page_token,omitempty"`
}

type fontPayload struct {
	ID               string             `json:"id"`
	DisplayName      string             `json:"display_name"`
	Family           string             `json:"family"`
	Scripts          []string           `json:"scripts,omitempty"`
	PreviewURL       string             `json:"preview_url,omitempty"`
	LetterSpacing    float64            `json:"letter_spacing"`
	IsPremium        bool               `json:"is_premium"`
	SupportedWeights []string           `json:"supported_weights,omitempty"`
	License          fontLicensePayload `json:"license"`
	CreatedAt        string             `json:"created_at,omitempty"`
	UpdatedAt        string             `json:"updated_at,omitempty"`
}

type fontLicensePayload struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

type materialListResponse struct {
	Materials     []materialPayload `json:"materials"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

type materialPayload struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	Category     string `json:"category,omitempty"`
	Grain        string `json:"grain,omitempty"`
	Color        string `json:"color,omitempty"`
	IsAvailable  bool   `json:"is_available"`
	LeadTimeDays int    `json:"lead_time_days,omitempty"`
	PreviewURL   string `json:"preview_url,omitempty"`
	Locale       string `json:"locale,omitempty"`
	CreatedAt    string `json:"created_at,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type materialDetailPayload struct {
	materialPayload
	Finish         string                         `json:"finish,omitempty"`
	Hardness       float64                        `json:"hardness,omitempty"`
	Density        float64                        `json:"density,omitempty"`
	CareNotes      string                         `json:"care_notes,omitempty"`
	Photos         []string                       `json:"photos,omitempty"`
	Sustainability *materialSustainabilityPayload `json:"sustainability,omitempty"`
}

type materialSustainabilityPayload struct {
	Certifications []string `json:"certifications,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type productListResponse struct {
	Products      []productPayload `json:"products"`
	NextPageToken string           `json:"next_page_token,omitempty"`
	PriceDisplay  string           `json:"price_display,omitempty"`
}

type productPayload struct {
	ID                    string   `json:"id"`
	SKU                   string   `json:"sku,omitempty"`
	Name                  string   `json:"name"`
	Description           string   `json:"description,omitempty"`
	Shape                 string   `json:"shape,omitempty"`
	SizesMm               []int    `json:"sizes_mm,omitempty"`
	DefaultMaterialID     string   `json:"default_material_id,omitempty"`
	MaterialIDs           []string `json:"material_ids,omitempty"`
	BasePrice             int64    `json:"base_price,omitempty"`
	Currency              string   `json:"currency,omitempty"`
	PreviewURL            string   `json:"preview_url,omitempty"`
	ImageURLs             []string `json:"image_urls,omitempty"`
	IsCustomizable        bool     `json:"is_customizable"`
	InventoryStatus       string   `json:"inventory_status,omitempty"`
	CompatibleTemplateIDs []string `json:"compatible_template_ids,omitempty"`
	LeadTimeDays          int      `json:"lead_time_days,omitempty"`
	PriceDisplay          string   `json:"price_display,omitempty"`
	CreatedAt             string   `json:"created_at,omitempty"`
	UpdatedAt             string   `json:"updated_at,omitempty"`
}

type productDetailPayload struct {
	productPayload
	PriceTiers []productPriceTierPayload `json:"price_tiers,omitempty"`
}

type guideListResponse struct {
	Guides        []guideSummaryPayload `json:"guides"`
	NextPageToken string                `json:"next_page_token,omitempty"`
	Locale        string                `json:"locale,omitempty"`
}

type guideSummaryPayload struct {
	Slug         string   `json:"slug"`
	Locale       string   `json:"locale"`
	Category     string   `json:"category,omitempty"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary,omitempty"`
	HeroImageURL string   `json:"hero_image_url,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	IsPublished  bool     `json:"is_published"`
	PublishedAt  string   `json:"published_at,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
	CreatedAt    string   `json:"created_at,omitempty"`
}

type guideDetailPayload struct {
	guideSummaryPayload
	BodyHTML string `json:"body_html,omitempty"`
}

type productPriceTierPayload struct {
	MinQuantity int    `json:"min_quantity"`
	UnitPrice   int64  `json:"unit_price"`
	Currency    string `json:"currency,omitempty"`
}

func (h *PublicHandlers) buildGuideSummaryPayload(ctx context.Context, guide services.ContentGuide) (guideSummaryPayload, error) {
	heroURL, err := h.resolveAsset(ctx, h.previewResolver, guide.HeroImage)
	if err != nil {
		return guideSummaryPayload{}, err
	}

	locale := normalizeLocale(guide.Locale)
	if locale == "" {
		locale = defaultGuideLocale
	}

	return guideSummaryPayload{
		Slug:         strings.TrimSpace(guide.Slug),
		Locale:       locale,
		Category:     strings.TrimSpace(guide.Category),
		Title:        strings.TrimSpace(guide.Title),
		Summary:      strings.TrimSpace(guide.Summary),
		HeroImageURL: heroURL,
		Tags:         normalizeGuideTags(guide.Tags),
		IsPublished:  guide.IsPublished,
		PublishedAt:  formatTimestamp(guide.PublishedAt),
		UpdatedAt:    formatTimestamp(guide.UpdatedAt),
		CreatedAt:    formatTimestamp(guide.CreatedAt),
	}, nil
}

func (h *PublicHandlers) buildGuideDetailPayload(ctx context.Context, guide services.ContentGuide) (guideDetailPayload, error) {
	summary, err := h.buildGuideSummaryPayload(ctx, guide)
	if err != nil {
		return guideDetailPayload{}, err
	}
	return guideDetailPayload{
		guideSummaryPayload: summary,
		BodyHTML:            sanitizeGuideHTML(guide.BodyHTML),
	}, nil
}

func sanitizeGuideHTML(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	sanitized := strings.TrimSpace(guideHTMLPolicy.Sanitize(trimmed))
	return sanitized
}

func normalizeGuideTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	cleaned := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, trimmed)
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func computeGuideListETag(guides []services.ContentGuide, locale string) string {
	if len(guides) == 0 {
		return ""
	}

	type entry struct {
		slug      string
		locale    string
		updated   string
		published string
	}

	entries := make([]entry, 0, len(guides))
	for _, guide := range guides {
		slug, guideLocale, updated, published := guideOrderingFields(guide)
		entries = append(entries, entry{
			slug:      slug,
			locale:    guideLocale,
			updated:   updated,
			published: published,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].slug != entries[j].slug {
			return entries[i].slug < entries[j].slug
		}
		if entries[i].locale != entries[j].locale {
			return entries[i].locale < entries[j].locale
		}
		if entries[i].updated != entries[j].updated {
			return entries[i].updated < entries[j].updated
		}
		return entries[i].published < entries[j].published
	})

	hash := sha256.New()
	hash.Write([]byte(normalizeLocale(locale)))
	hash.Write([]byte("|"))
	for _, entry := range entries {
		hash.Write([]byte(entry.slug))
		hash.Write([]byte("|"))
		hash.Write([]byte(entry.locale))
		hash.Write([]byte("|"))
		hash.Write([]byte(entry.updated))
		hash.Write([]byte("|"))
		hash.Write([]byte(entry.published))
		hash.Write([]byte("|"))
	}
	return fmt.Sprintf("W/\"%x\"", hash.Sum(nil))
}

func computeGuideETag(guide services.ContentGuide) string {
	slug, locale, updated, published := guideOrderingFields(guide)
	hash := sha256.New()
	hash.Write([]byte(slug))
	hash.Write([]byte("|"))
	hash.Write([]byte(locale))
	hash.Write([]byte("|"))
	hash.Write([]byte(updated))
	hash.Write([]byte("|"))
	hash.Write([]byte(published))
	return fmt.Sprintf("W/\"%x\"", hash.Sum(nil))
}

func guideOrderingFields(guide services.ContentGuide) (slug string, locale string, updated string, published string) {
	slug = strings.TrimSpace(guide.Slug)
	if slug == "" {
		slug = guide.ID
	}
	locale = normalizeLocale(guide.Locale)
	updated = formatTimestamp(guide.UpdatedAt)
	published = formatTimestamp(guide.PublishedAt)
	return
}

func guideLess(a, b services.ContentGuide) bool {
	aslug, alocale, aupdated, apublished := guideOrderingFields(a)
	bslug, blocale, bupdated, bpublished := guideOrderingFields(b)
	if aslug != bslug {
		return aslug < bslug
	}
	if alocale != blocale {
		return alocale < blocale
	}
	if aupdated != bupdated {
		return aupdated < bupdated
	}
	return apublished < bpublished
}

func matchesETag(r *http.Request, etag string) bool {
	if etag == "" || r == nil {
		return false
	}
	raw := r.Header.Get("If-None-Match")
	if strings.TrimSpace(raw) == "" {
		return false
	}
	for _, candidate := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(candidate)
		if trimmed == "*" || trimmed == etag {
			return true
		}
	}
	return false
}

func newGuideHTMLPolicy() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowElements("figure", "figcaption")
	policy.AllowAttrs("class").OnElements("figure", "figcaption", "p", "span")
	policy.AllowAttrs("loading").OnElements("img")
	policy.RequireNoFollowOnLinks(true)
	return policy
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func copyStringSlice(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func copyIntSlice(in []int) []int {
	out := make([]int, len(in))
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
