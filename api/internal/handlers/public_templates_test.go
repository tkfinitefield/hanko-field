package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
	"github.com/hanko-field/api/internal/services"
)

func TestPublicHandlers_ListTemplates(t *testing.T) {
	stubService := &stubCatalogService{
		listResponse: domain.CursorPage[domain.TemplateSummary]{
			Items: []domain.TemplateSummary{
				{
					ID:               "tpl_001",
					Name:             "Classic Round",
					Description:      "A timeless round template",
					Category:         "round",
					Style:            "classic",
					Tags:             []string{"modern", "round"},
					PreviewImagePath: "previews/tpl_001.png",
					Popularity:       42,
					IsPublished:      true,
					CreatedAt:        time.Date(2024, time.January, 10, 9, 0, 0, 0, time.UTC),
					UpdatedAt:        time.Date(2024, time.January, 12, 9, 0, 0, 0, time.UTC),
				},
			},
			NextPageToken: "next-token",
		},
	}

	handler := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	values := url.Values{}
	values.Set("category", "  round ")
	values.Set("style", "  classic ")
	values.Add("tag", "modern")
	values.Add("tag", "modern")
	values.Add("tags", "modern,featured")
	values.Set("pageSize", "15")
	values.Set("sort", "-createdAt")

	req := httptest.NewRequest(http.MethodGet, "/templates?"+values.Encode(), nil)
	w := httptest.NewRecorder()

	handler.listTemplates(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 got %d", resp.StatusCode)
	}

	var payload templateListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.NextPageToken != "next-token" {
		t.Fatalf("expected next token %q got %q", "next-token", payload.NextPageToken)
	}
	if len(payload.Templates) != 1 {
		t.Fatalf("expected 1 template got %d", len(payload.Templates))
	}
	item := payload.Templates[0]
	if item.ID != "tpl_001" {
		t.Fatalf("expected template id tpl_001 got %s", item.ID)
	}
	if item.PreviewURL != "https://cdn.example.com/previews/tpl_001.png" {
		t.Fatalf("expected resolved preview url, got %s", item.PreviewURL)
	}
	if item.CreatedAt == "" || item.UpdatedAt == "" {
		t.Fatalf("expected timestamps to be present")
	}

	filter := stubService.listFilter
	if filter.Category == nil || *filter.Category != "round" {
		t.Fatalf("expected category filter round got %#v", filter.Category)
	}
	if filter.Style == nil || *filter.Style != "classic" {
		t.Fatalf("expected style filter classic got %#v", filter.Style)
	}
	if filter.SortBy != domain.TemplateSortCreatedAt {
		t.Fatalf("expected sort field createdAt got %v", filter.SortBy)
	}
	if filter.SortOrder != domain.SortDesc {
		t.Fatalf("expected sort order desc got %v", filter.SortOrder)
	}
	if !filter.PublishedOnly {
		t.Fatalf("expected published flag to be true")
	}
	if filter.Pagination.PageSize != 15 {
		t.Fatalf("expected page size 15 got %d", filter.Pagination.PageSize)
	}
	if len(filter.Tags) != 2 {
		t.Fatalf("expected deduped tags got %#v", filter.Tags)
	}
}

func TestPublicHandlers_ListTemplates_InvalidSort(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{}))

	req := httptest.NewRequest(http.MethodGet, "/templates?sort=unknown", nil)
	resp := httptest.NewRecorder()

	handler.listTemplates(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", resp.Code)
	}
}

func TestPublicHandlers_GetTemplate(t *testing.T) {
	template := domain.Template{
		TemplateSummary: domain.TemplateSummary{
			ID:               "tpl_002",
			Name:             "Art Deco",
			Description:      "Deco inspired",
			Category:         "square",
			Style:            "art-deco",
			Tags:             []string{"deco"},
			PreviewImagePath: "previews/tpl_002.png",
			IsPublished:      true,
			Popularity:       7,
			CreatedAt:        time.Date(2024, time.January, 15, 8, 0, 0, 0, time.UTC),
			UpdatedAt:        time.Date(2024, time.January, 16, 8, 0, 0, 0, time.UTC),
		},
		SVGPath: "vectors/tpl_002.svg",
	}

	stubService := &stubCatalogService{getTemplate: template}

	publicHandlers := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn/" + path, nil
		})),
		WithPublicVectorResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn-vectors/" + path, nil
		})),
	)

	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/templates/tpl_002", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	var payload templatePayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.SVGURL != "https://cdn-vectors/vectors/tpl_002.svg" {
		t.Fatalf("expected svg url got %s", payload.SVGURL)
	}
	if stubService.getID != "tpl_002" {
		t.Fatalf("expected service to receive id tpl_002 got %s", stubService.getID)
	}
}

func TestPublicHandlers_GetTemplate_NotFound(t *testing.T) {
	stubService := &stubCatalogService{
		getErr: newRepositoryError(true, false, false),
	}
	publicHandlers := NewPublicHandlers(WithPublicCatalogService(stubService))
	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/templates/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
}

func TestPublicHandlers_ListFonts(t *testing.T) {
	createdAt := time.Date(2024, time.March, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.March, 2, 0, 0, 0, 0, time.UTC)
	stubService := &stubCatalogService{
		fontListResp: domain.CursorPage[services.FontSummary]{
			Items: []services.FontSummary{
				{
					ID:               "font_001",
					DisplayName:      "Tensho Regular",
					Family:           "Tensho",
					Scripts:          []string{"kanji", "kana"},
					PreviewImagePath: "fonts/font_001.png",
					LetterSpacing:    0.05,
					IsPremium:        true,
					IsPublished:      true,
					SupportedWeights: []string{"400", "700"},
					License: services.FontLicense{
						Name: "Commercial",
						URL:  "https://example.com/license",
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			},
			NextPageToken: "next-font",
		},
	}

	handler := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	values := url.Values{}
	values.Set("script", "KANJI")
	values.Set("isPremium", "true")
	values.Set("pageSize", "120")
	values.Set("pageToken", " next ")

	req := httptest.NewRequest(http.MethodGet, "/fonts?"+values.Encode(), nil)
	w := httptest.NewRecorder()

	handler.listFonts(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 got %d", resp.StatusCode)
	}
	if cache := resp.Header.Get("Cache-Control"); cache != fontCacheControl {
		t.Fatalf("expected cache control %q got %q", fontCacheControl, cache)
	}

	var payload fontListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if payload.NextPageToken != "next-font" {
		t.Fatalf("expected next token next-font got %q", payload.NextPageToken)
	}
	if len(payload.Fonts) != 1 {
		t.Fatalf("expected 1 font got %d", len(payload.Fonts))
	}
	item := payload.Fonts[0]
	if item.ID != "font_001" {
		t.Fatalf("expected font id font_001 got %s", item.ID)
	}
	if item.PreviewURL != "https://cdn.example.com/fonts/font_001.png" {
		t.Fatalf("expected resolved preview url got %s", item.PreviewURL)
	}
	if item.LetterSpacing != 0.05 {
		t.Fatalf("expected letter spacing 0.05 got %v", item.LetterSpacing)
	}
	if item.CreatedAt == "" || item.UpdatedAt == "" {
		t.Fatalf("expected timestamps to be present")
	}
	if item.License.URL != "https://example.com/license" {
		t.Fatalf("expected license url preserved got %s", item.License.URL)
	}

	if stubService.fontListFilter.Script == nil || *stubService.fontListFilter.Script != "kanji" {
		t.Fatalf("expected script filter kanji got %#v", stubService.fontListFilter.Script)
	}
	if stubService.fontListFilter.IsPremium == nil || !*stubService.fontListFilter.IsPremium {
		t.Fatalf("expected isPremium filter true got %#v", stubService.fontListFilter.IsPremium)
	}
	if stubService.fontListFilter.Pagination.PageSize != maxFontPageSize {
		t.Fatalf("expected page size capped to %d got %d", maxFontPageSize, stubService.fontListFilter.Pagination.PageSize)
	}
	if stubService.fontListFilter.Pagination.PageToken != "next" {
		t.Fatalf("expected trimmed page token got %q", stubService.fontListFilter.Pagination.PageToken)
	}
	if !stubService.fontListFilter.PublishedOnly {
		t.Fatalf("expected published flag to be true")
	}
}

func TestPublicHandlers_ListFonts_InvalidPremium(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{}))
	req := httptest.NewRequest(http.MethodGet, "/fonts?isPremium=maybe", nil)
	rec := httptest.NewRecorder()

	handler.listFonts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestPublicHandlers_GetFont(t *testing.T) {
	font := services.Font{
		FontSummary: services.FontSummary{
			ID:               "font_002",
			DisplayName:      "Kana Script",
			Family:           "Kana",
			Scripts:          []string{"kana"},
			PreviewImagePath: "fonts/font_002.png",
			LetterSpacing:    0.1,
			IsPremium:        false,
			IsPublished:      true,
			SupportedWeights: []string{"400"},
			License: services.FontLicense{
				Name: "Commercial",
				URL:  "https://example.com/license",
			},
			CreatedAt: time.Date(2024, time.April, 3, 0, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, time.April, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	stubService := &stubCatalogService{fontGetFont: font}
	publicHandlers := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn/" + path, nil
		})),
	)

	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/fonts/font_002", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != fontCacheControl {
		t.Fatalf("expected cache control %q got %q", fontCacheControl, cache)
	}

	var payload fontPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.PreviewURL != "https://cdn/fonts/font_002.png" {
		t.Fatalf("expected resolved preview url got %s", payload.PreviewURL)
	}
	if stubService.fontGetID != "font_002" {
		t.Fatalf("expected service to receive trimmed id font_002 got %s", stubService.fontGetID)
	}
}

func TestPublicHandlers_GetFont_Unpublished(t *testing.T) {
	stubService := &stubCatalogService{
		fontGetFont: services.Font{
			FontSummary: services.FontSummary{
				ID:          "font_003",
				DisplayName: "Hidden Font",
				IsPublished: false,
			},
		},
	}
	publicHandlers := NewPublicHandlers(WithPublicCatalogService(stubService))
	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/fonts/font_003", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != "" {
		t.Fatalf("expected cache control to be empty on error, got %q", cache)
	}
}

func TestPublicHandlers_GetFont_NotFound(t *testing.T) {
	stubService := &stubCatalogService{
		fontGetErr: newRepositoryError(true, false, false),
	}
	publicHandlers := NewPublicHandlers(WithPublicCatalogService(stubService))
	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/fonts/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
}

func TestPublicHandlers_ListMaterials(t *testing.T) {
	createdAt := time.Date(2024, time.May, 1, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.May, 2, 12, 0, 0, 0, time.UTC)
	stubService := &stubCatalogService{
		materialListResp: domain.CursorPage[services.MaterialSummary]{
			Items: []services.MaterialSummary{
				{
					ID:               "mat_wood",
					Name:             "柘植",
					Description:      "和風の木材",
					Category:         "wood",
					Grain:            "fine",
					Color:            "#aa7733",
					IsAvailable:      true,
					LeadTimeDays:     3,
					PreviewImagePath: "materials/mat_wood.png",
					DefaultLocale:    "ja",
					Translations: map[string]services.MaterialTranslation{
						"en": {
							Locale:      "en",
							Name:        "Boxwood",
							Description: "Traditional Japanese hardwood",
						},
					},
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
			},
			NextPageToken: "next-material",
		},
	}

	handler := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	values := url.Values{}
	values.Set("lang", "en-US")
	values.Set("category", "  Wood ")
	values.Set("pageToken", " nxt ")
	values.Set("pageSize", "200")

	req := httptest.NewRequest(http.MethodGet, "/materials?"+values.Encode(), nil)
	rec := httptest.NewRecorder()

	handler.listMaterials(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != materialCacheControl {
		t.Fatalf("expected cache header %q got %q", materialCacheControl, cache)
	}

	var payload materialListResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.NextPageToken != "next-material" {
		t.Fatalf("expected next token next-material got %q", payload.NextPageToken)
	}
	if len(payload.Materials) != 1 {
		t.Fatalf("expected 1 material got %d", len(payload.Materials))
	}
	item := payload.Materials[0]
	if item.ID != "mat_wood" {
		t.Fatalf("expected material id mat_wood got %s", item.ID)
	}
	if item.Name != "Boxwood" {
		t.Fatalf("expected translated name Boxwood got %s", item.Name)
	}
	if item.Description != "Traditional Japanese hardwood" {
		t.Fatalf("expected translated description got %s", item.Description)
	}
	if item.Locale != "en" {
		t.Fatalf("expected resolved locale en got %s", item.Locale)
	}
	if item.PreviewURL != "https://cdn.example.com/materials/mat_wood.png" {
		t.Fatalf("expected resolved preview url got %s", item.PreviewURL)
	}
	if item.CreatedAt == "" || item.UpdatedAt == "" {
		t.Fatalf("expected timestamps to be present")
	}

	filter := stubService.materialListFilter
	if filter.Locale != "en-us" {
		t.Fatalf("expected locale filter en-us got %s", filter.Locale)
	}
	if filter.Category == nil || *filter.Category != "wood" {
		t.Fatalf("expected category filter wood got %#v", filter.Category)
	}
	if !filter.OnlyAvailable {
		t.Fatalf("expected OnlyAvailable flag true")
	}
	if filter.Pagination.PageSize != maxMaterialPageSize {
		t.Fatalf("expected page size capped to %d got %d", maxMaterialPageSize, filter.Pagination.PageSize)
	}
	if filter.Pagination.PageToken != "nxt" {
		t.Fatalf("expected trimmed page token nxt got %q", filter.Pagination.PageToken)
	}
}

func TestPublicHandlers_ListMaterials_InvalidPageSize(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{}))
	req := httptest.NewRequest(http.MethodGet, "/materials?pageSize=0", nil)
	rec := httptest.NewRecorder()

	handler.listMaterials(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestPublicHandlers_GetMaterial(t *testing.T) {
	createdAt := time.Date(2024, time.June, 1, 8, 30, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.June, 2, 8, 30, 0, 0, time.UTC)
	stubService := &stubCatalogService{
		materialGetMat: services.Material{
			MaterialSummary: services.MaterialSummary{
				ID:               "mat_titanium",
				Name:             "チタン",
				Description:      "高耐久の素材",
				Category:         "metal",
				Grain:            "smooth",
				Color:            "#cccccc",
				IsAvailable:      true,
				LeadTimeDays:     5,
				PreviewImagePath: "materials/titanium.png",
				DefaultLocale:    "ja",
				Translations: map[string]services.MaterialTranslation{
					"en": {
						Locale:      "en",
						Name:        "Titanium",
						Description: "Durable metal",
					},
				},
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Finish:    "matte",
			Hardness:  9.5,
			Density:   4.5,
			CareNotes: "Wipe with dry cloth.",
			Sustainability: services.MaterialSustainability{
				Certifications: []string{"ISO9001"},
				Notes:          "Recyclable",
			},
			Photos: []string{"materials/titanium_detail.png"},
		},
	}

	publicHandlers := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/materials/mat_titanium?lang=en", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != materialCacheControl {
		t.Fatalf("expected cache-control %q got %q", materialCacheControl, cache)
	}

	var payload materialDetailPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Name != "Titanium" {
		t.Fatalf("expected translated name Titanium got %s", payload.Name)
	}
	if payload.Description != "Durable metal" {
		t.Fatalf("expected translated description got %s", payload.Description)
	}
	if payload.Locale != "en" {
		t.Fatalf("expected resolved locale en got %s", payload.Locale)
	}
	if payload.PreviewURL != "https://cdn.example.com/materials/titanium.png" {
		t.Fatalf("expected resolved preview url got %s", payload.PreviewURL)
	}
	if payload.Finish != "matte" {
		t.Fatalf("expected finish matte got %s", payload.Finish)
	}
	if len(payload.Photos) != 1 || payload.Photos[0] != "https://cdn.example.com/materials/titanium_detail.png" {
		t.Fatalf("expected resolved photo url got %#v", payload.Photos)
	}
	if payload.Sustainability == nil || len(payload.Sustainability.Certifications) != 1 {
		t.Fatalf("expected sustainability payload got %#v", payload.Sustainability)
	}
	if stubService.materialGetID != "mat_titanium" {
		t.Fatalf("expected service to receive trimmed id mat_titanium got %s", stubService.materialGetID)
	}
	if stubService.materialGetLocale != "en" {
		t.Fatalf("expected locale en got %s", stubService.materialGetLocale)
	}
}

func TestPublicHandlers_GetMaterial_NotAvailable(t *testing.T) {
	stubService := &stubCatalogService{
		materialGetMat: services.Material{
			MaterialSummary: services.MaterialSummary{
				ID:          "mat_hidden",
				Name:        "非公開素材",
				IsAvailable: false,
			},
		},
	}
	publicHandlers := NewPublicHandlers(WithPublicCatalogService(stubService))
	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/materials/mat_hidden", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != "" {
		t.Fatalf("expected no cache header on error got %q", cache)
	}
}

type stubCatalogService struct {
	listFilter         services.TemplateFilter
	listResponse       domain.CursorPage[domain.TemplateSummary]
	listErr            error
	fontListFilter     services.FontFilter
	fontListResp       domain.CursorPage[services.FontSummary]
	fontListErr        error
	materialListFilter services.MaterialFilter
	materialListResp   domain.CursorPage[services.MaterialSummary]
	materialListErr    error

	getID             string
	getTemplate       domain.Template
	getErr            error
	fontGetID         string
	fontGetFont       services.Font
	fontGetErr        error
	materialGetID     string
	materialGetLocale string
	materialGetMat    services.Material
	materialGetErr    error
}

func (s *stubCatalogService) ListTemplates(_ context.Context, filter services.TemplateFilter) (domain.CursorPage[domain.TemplateSummary], error) {
	s.listFilter = filter
	return s.listResponse, s.listErr
}

func (s *stubCatalogService) GetTemplate(_ context.Context, templateID string) (services.Template, error) {
	s.getID = templateID
	if s.getErr != nil {
		return services.Template{}, s.getErr
	}
	return services.Template(s.getTemplate), nil
}

func (s *stubCatalogService) UpsertTemplate(context.Context, services.UpsertTemplateCommand) (services.Template, error) {
	return services.Template{}, errors.New("not implemented")
}

func (s *stubCatalogService) DeleteTemplate(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogService) ListFonts(_ context.Context, filter services.FontFilter) (domain.CursorPage[services.FontSummary], error) {
	s.fontListFilter = filter
	return s.fontListResp, s.fontListErr
}

func (s *stubCatalogService) GetFont(_ context.Context, fontID string) (services.Font, error) {
	s.fontGetID = fontID
	if s.fontGetErr != nil {
		return services.Font{}, s.fontGetErr
	}
	return s.fontGetFont, nil
}

func (s *stubCatalogService) UpsertFont(context.Context, services.UpsertFontCommand) (services.FontSummary, error) {
	return services.FontSummary{}, errors.New("not implemented")
}

func (s *stubCatalogService) DeleteFont(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogService) ListMaterials(_ context.Context, filter services.MaterialFilter) (domain.CursorPage[services.MaterialSummary], error) {
	s.materialListFilter = filter
	return s.materialListResp, s.materialListErr
}

func (s *stubCatalogService) GetMaterial(_ context.Context, materialID string, locale string) (services.Material, error) {
	s.materialGetID = materialID
	s.materialGetLocale = locale
	if s.materialGetErr != nil {
		return services.Material{}, s.materialGetErr
	}
	return s.materialGetMat, nil
}

func (s *stubCatalogService) UpsertMaterial(context.Context, services.UpsertMaterialCommand) (services.MaterialSummary, error) {
	return services.MaterialSummary{}, errors.New("not implemented")
}

func (s *stubCatalogService) DeleteMaterial(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubCatalogService) ListProducts(context.Context, services.ProductFilter) (domain.CursorPage[services.ProductSummary], error) {
	return domain.CursorPage[services.ProductSummary]{}, errors.New("not implemented")
}

func (s *stubCatalogService) UpsertProduct(context.Context, services.UpsertProductCommand) (services.ProductSummary, error) {
	return services.ProductSummary{}, errors.New("not implemented")
}

func (s *stubCatalogService) DeleteProduct(context.Context, string) error {
	return errors.New("not implemented")
}

type stubRepoError struct {
	notFound    bool
	conflict    bool
	unavailable bool
}

func newRepositoryError(notFound, conflict, unavailable bool) repositories.RepositoryError {
	return &stubRepoError{notFound: notFound, conflict: conflict, unavailable: unavailable}
}

func (e *stubRepoError) Error() string {
	return "stub repository error"
}

func (e *stubRepoError) IsNotFound() bool    { return e.notFound }
func (e *stubRepoError) IsConflict() bool    { return e.conflict }
func (e *stubRepoError) IsUnavailable() bool { return e.unavailable }
