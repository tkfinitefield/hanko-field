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

func TestPublicHandlers_GetPublicPromotion_Success(t *testing.T) {
	starts := time.Date(2024, time.March, 10, 9, 0, 0, 0, time.UTC)
	ends := starts.Add(24 * time.Hour)
	stub := &stubPromotionService{result: services.PromotionPublic{
		Code:              "SPRING10",
		IsAvailable:       true,
		StartsAt:          starts,
		EndsAt:            ends,
		DescriptionPublic: "Spring campaign",
		EligibleAudiences: []string{"new_customers", "vip"},
	}}
	handler := NewPublicHandlers(
		WithPublicPromotionService(stub),
		WithPublicPromotionRateLimit(5, time.Minute),
	)
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/promotions/SPRING10/public", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.Code)
	}

	var payload promotionPublicResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Code != "SPRING10" {
		t.Fatalf("expected code SPRING10 got %s", payload.Code)
	}
	if !payload.IsAvailable {
		t.Fatalf("expected promotion to be available")
	}
	if payload.Description != "Spring campaign" {
		t.Fatalf("unexpected description %q", payload.Description)
	}
	if len(payload.EligibleAudiences) != 2 {
		t.Fatalf("expected eligible audiences, got %v", payload.EligibleAudiences)
	}
	if stub.code != "SPRING10" {
		t.Fatalf("expected service to receive SPRING10 got %s", stub.code)
	}
}

func TestPublicHandlers_GetPublicPromotion_ServiceUnavailable(t *testing.T) {
	handler := NewPublicHandlers()
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/promotions/ANY/public", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 got %d", resp.Code)
	}
}

func TestPublicHandlers_GetPublicPromotion_InvalidCode(t *testing.T) {
	handler := NewPublicHandlers(
		WithPublicPromotionService(&stubPromotionService{result: services.PromotionPublic{Code: "INVALID"}}),
	)
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/promotions/%20/public", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", resp.Code)
	}
}

func TestPublicHandlers_GetPublicPromotion_RateLimited(t *testing.T) {
	stub := &stubPromotionService{result: services.PromotionPublic{Code: "LIMIT"}}
	handler := NewPublicHandlers(
		WithPublicPromotionService(stub),
		WithPublicPromotionRateLimit(1, time.Hour),
	)
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req1 := httptest.NewRequest(http.MethodGet, "/promotions/LIMIT/public", nil)
	resp1 := httptest.NewRecorder()
	router.ServeHTTP(resp1, req1)

	if resp1.Code != http.StatusOK {
		t.Fatalf("expected first call 200 got %d", resp1.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/promotions/LIMIT/public", nil)
	resp2 := httptest.NewRecorder()
	router.ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 got %d", resp2.Code)
	}
}

func TestPublicHandlers_GetPublicPromotion_NotFound(t *testing.T) {
	stub := &stubPromotionService{err: services.ErrPromotionNotFound}
	handler := NewPublicHandlers(
		WithPublicPromotionService(stub),
	)
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/promotions/MISSING/public", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", resp.Code)
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
					Slug:             "tensho-regular",
					DisplayName:      "Tensho Regular",
					Family:           "Tensho",
					Weight:           "regular",
					Scripts:          []string{"kanji", "kana"},
					PreviewImagePath: "fonts/font_001.png",
					LetterSpacing:    0.05,
					IsPremium:        true,
					IsPublished:      true,
					SupportedWeights: []string{"400", "700"},
					License: services.FontLicense{
						Name:          "Commercial",
						URL:           "https://example.com/license",
						AllowedUsages: []string{"app", "print"},
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
	if item.Slug != "tensho-regular" {
		t.Fatalf("expected slug tensho-regular got %s", item.Slug)
	}
	if item.Weight != "regular" {
		t.Fatalf("expected weight regular got %s", item.Weight)
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
	if len(item.License.AllowedUsages) != 2 || item.License.AllowedUsages[0] != "app" {
		t.Fatalf("expected allowed usages propagated got %#v", item.License.AllowedUsages)
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
			Slug:             "kana-regular",
			DisplayName:      "Kana Script",
			Family:           "Kana",
			Weight:           "regular",
			Scripts:          []string{"kana"},
			PreviewImagePath: "fonts/font_002.png",
			LetterSpacing:    0.1,
			IsPremium:        false,
			IsPublished:      true,
			SupportedWeights: []string{"400"},
			License: services.FontLicense{
				Name:          "Commercial",
				URL:           "https://example.com/license",
				AllowedUsages: []string{"svg"},
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
	if payload.Slug != "kana-regular" {
		t.Fatalf("expected slug kana-regular got %s", payload.Slug)
	}
	if payload.License.URL != "https://example.com/license" {
		t.Fatalf("expected license url got %s", payload.License.URL)
	}
	if len(payload.License.AllowedUsages) != 1 || payload.License.AllowedUsages[0] != "svg" {
		t.Fatalf("expected license usages got %#v", payload.License.AllowedUsages)
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

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != materialCacheControl {
		t.Fatalf("expected cache header %q got %q", materialCacheControl, cache)
	}

	var payload materialDetailPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.ID != "mat_hidden" {
		t.Fatalf("expected material id mat_hidden got %s", payload.ID)
	}
	if payload.IsAvailable {
		t.Fatalf("expected material to be unavailable")
	}
}

func TestPublicHandlers_ListProducts(t *testing.T) {
	createdAt := time.Date(2024, time.August, 1, 10, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.August, 2, 10, 0, 0, 0, time.UTC)
	stubService := &stubCatalogService{
		productListResp: domain.CursorPage[services.ProductSummary]{
			Items: []services.ProductSummary{
				{
					ID:                    "prod_round",
					SKU:                   "SKU-001",
					Name:                  "Round Hanko",
					Description:           "Classic round seal",
					Shape:                 "round",
					SizesMm:               []int{45, 60},
					DefaultMaterialID:     "mat_wood",
					MaterialIDs:           []string{"mat_wood", "mat_titanium"},
					BasePrice:             5500,
					Currency:              "JPY",
					ImagePaths:            []string{"products/round.png", "products/round_alt.png"},
					IsCustomizable:        true,
					InventoryStatus:       "in_stock",
					CompatibleTemplateIDs: []string{"tpl_classic"},
					LeadTimeDays:          5,
					CreatedAt:             createdAt,
					UpdatedAt:             updatedAt,
				},
			},
			NextPageToken: "next-product",
		},
	}

	handler := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
		WithPublicPriceDisplayMode(priceDisplayModeExclusive),
	)

	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/products", nil)
	values := req.URL.Query()
	values.Set("shape", "Round")
	values.Set("size", "45 mm")
	values.Set("material", " mat_wood ")
	values.Set("isCustomizable", "true")
	values.Set("pageSize", "12")
	values.Set("pageToken", " token ")
	req.URL.RawQuery = values.Encode()

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != productCacheControl {
		t.Fatalf("expected cache control %q got %q", productCacheControl, cache)
	}

	var payload productListResponse
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.PriceDisplay != priceDisplayModeExclusive {
		t.Fatalf("expected price display %q got %q", priceDisplayModeExclusive, payload.PriceDisplay)
	}
	if payload.NextPageToken != "next-product" {
		t.Fatalf("expected next page token next-product got %s", payload.NextPageToken)
	}
	if len(payload.Products) != 1 {
		t.Fatalf("expected 1 product got %d", len(payload.Products))
	}

	product := payload.Products[0]
	if product.ID != "prod_round" {
		t.Fatalf("expected product id prod_round got %s", product.ID)
	}
	if product.PriceDisplay != priceDisplayModeExclusive {
		t.Fatalf("expected product price display %q got %q", priceDisplayModeExclusive, product.PriceDisplay)
	}
	if product.PreviewURL != "https://cdn.example.com/products/round.png" {
		t.Fatalf("expected resolved preview url got %s", product.PreviewURL)
	}
	if len(product.ImageURLs) != 2 || product.ImageURLs[1] != "https://cdn.example.com/products/round_alt.png" {
		t.Fatalf("expected resolved image urls got %#v", product.ImageURLs)
	}
	if product.BasePrice != 5500 || product.Currency != "JPY" {
		t.Fatalf("expected price 5500 JPY got %d %s", product.BasePrice, product.Currency)
	}
	if product.CreatedAt != createdAt.Format(time.RFC3339) {
		t.Fatalf("expected created_at %s got %s", createdAt.Format(time.RFC3339), product.CreatedAt)
	}
	if product.UpdatedAt != updatedAt.Format(time.RFC3339) {
		t.Fatalf("expected updated_at %s got %s", updatedAt.Format(time.RFC3339), product.UpdatedAt)
	}

	if stubService.productListFilter.Shape == nil || *stubService.productListFilter.Shape != "round" {
		t.Fatalf("expected shape filter round got %#v", stubService.productListFilter.Shape)
	}
	if stubService.productListFilter.SizeMm == nil || *stubService.productListFilter.SizeMm != 45 {
		t.Fatalf("expected size filter 45 got %#v", stubService.productListFilter.SizeMm)
	}
	if stubService.productListFilter.MaterialID == nil || *stubService.productListFilter.MaterialID != "mat_wood" {
		t.Fatalf("expected material filter mat_wood got %#v", stubService.productListFilter.MaterialID)
	}
	if stubService.productListFilter.IsCustomizable == nil || !*stubService.productListFilter.IsCustomizable {
		t.Fatalf("expected customizable filter true got %#v", stubService.productListFilter.IsCustomizable)
	}
	if !stubService.productListFilter.PublishedOnly {
		t.Fatalf("expected published-only filter true got %v", stubService.productListFilter.PublishedOnly)
	}
	if stubService.productListFilter.Pagination.PageSize != 12 {
		t.Fatalf("expected page size 12 got %d", stubService.productListFilter.Pagination.PageSize)
	}
	if stubService.productListFilter.Pagination.PageToken != "token" {
		t.Fatalf("expected trimmed page token got %q", stubService.productListFilter.Pagination.PageToken)
	}
}

func TestPublicHandlers_ListProducts_InvalidSize(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{}))

	req := httptest.NewRequest(http.MethodGet, "/products?size=abc", nil)
	rec := httptest.NewRecorder()

	handler.listProducts(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestPublicHandlers_GetProduct(t *testing.T) {
	createdAt := time.Date(2024, time.September, 1, 11, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, time.September, 2, 11, 0, 0, 0, time.UTC)
	stubService := &stubCatalogService{
		productGetProd: services.Product{
			ProductSummary: services.ProductSummary{
				ID:                    "prod_square",
				SKU:                   "SKU-900",
				Name:                  "Square Hanko",
				Description:           "Modern square seal",
				Shape:                 "square",
				SizesMm:               []int{30},
				DefaultMaterialID:     "mat_titanium",
				MaterialIDs:           []string{"mat_titanium"},
				BasePrice:             7800,
				Currency:              "JPY",
				ImagePaths:            []string{"products/square.png"},
				IsCustomizable:        false,
				InventoryStatus:       "made_to_order",
				CompatibleTemplateIDs: []string{"tpl_modern"},
				LeadTimeDays:          7,
				CreatedAt:             createdAt,
				UpdatedAt:             updatedAt,
			},
			PriceTiers: []services.ProductPriceTier{
				{MinQuantity: 1, UnitPrice: 7800},
				{MinQuantity: 10, UnitPrice: 7300},
			},
		},
	}

	handler := NewPublicHandlers(
		WithPublicCatalogService(stubService),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/products/prod_square", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if cache := rec.Result().Header.Get("Cache-Control"); cache != productCacheControl {
		t.Fatalf("expected cache control %q got %q", productCacheControl, cache)
	}

	var payload productDetailPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.ID != "prod_square" {
		t.Fatalf("expected id prod_square got %s", payload.ID)
	}
	if payload.PreviewURL != "https://cdn.example.com/products/square.png" {
		t.Fatalf("expected preview url got %s", payload.PreviewURL)
	}
	if payload.InventoryStatus != "made_to_order" {
		t.Fatalf("expected inventory status made_to_order got %s", payload.InventoryStatus)
	}
	if payload.PriceDisplay != priceDisplayModeInclusive {
		t.Fatalf("expected price display default inclusive got %s", payload.PriceDisplay)
	}
	if len(payload.PriceTiers) != 2 {
		t.Fatalf("expected 2 price tiers got %d", len(payload.PriceTiers))
	}
	if payload.PriceTiers[1].MinQuantity != 10 || payload.PriceTiers[1].UnitPrice != 7300 {
		t.Fatalf("unexpected price tier payload %#v", payload.PriceTiers[1])
	}
	if payload.PriceTiers[1].Currency != "JPY" {
		t.Fatalf("expected price tier currency JPY got %s", payload.PriceTiers[1].Currency)
	}
	if stubService.productGetID != "prod_square" {
		t.Fatalf("expected service to receive trimmed product id got %s", stubService.productGetID)
	}
}

func TestPublicHandlers_GetProduct_NotFound(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{
		productGetErr: newRepositoryError(true, false, false),
	}))

	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/products/missing", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 got %d", rec.Code)
	}
}

func TestPublicHandlers_GetProduct_InvalidID(t *testing.T) {
	handler := NewPublicHandlers(WithPublicCatalogService(&stubCatalogService{}))

	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/products/%20", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 got %d", rec.Code)
	}
}

func TestPublicHandlers_ListGuides(t *testing.T) {
	stub := &stubContentService{
		listResponse: domain.CursorPage[services.ContentGuide]{
			Items: []services.ContentGuide{
				{
					ID:          "guide_1",
					Slug:        "tea-ceremony",
					Locale:      "en-US",
					Category:    "Culture",
					Title:       "Tea Ceremony",
					Summary:     " Learn the basics ",
					HeroImage:   "images/hero.jpg",
					Tags:        []string{"Etiquette", " etiquette"},
					Status:      "published",
					IsPublished: true,
					PublishedAt: time.Date(2024, time.January, 2, 0, 0, 0, 0, time.UTC),
					UpdatedAt:   time.Date(2024, time.January, 3, 0, 0, 0, 0, time.UTC),
					CreatedAt:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			NextPageToken: "next",
		},
	}

	handler := NewPublicHandlers(
		WithPublicContentService(stub),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + strings.TrimPrefix(path, "/"), nil
		})),
	)

	values := url.Values{}
	values.Set("lang", "EN_us")
	values.Set("category", "CULTURE")
	values.Set("pageSize", "10")
	values.Set("pageToken", " token ")
	req := httptest.NewRequest(http.MethodGet, "/content/guides?"+values.Encode(), nil)
	resp := httptest.NewRecorder()

	handler.listGuides(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.Code)
	}

	if resp.Header().Get("Cache-Control") != guideCacheControl {
		t.Fatalf("expected Cache-Control %s", guideCacheControl)
	}
	if resp.Header().Get("ETag") == "" {
		t.Fatalf("expected ETag header")
	}

	var payload guideListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.Locale != "en-us" {
		t.Fatalf("expected locale en-us got %s", payload.Locale)
	}
	if payload.NextPageToken != "next" {
		t.Fatalf("expected next token got %s", payload.NextPageToken)
	}
	if len(payload.Guides) != 1 {
		t.Fatalf("expected 1 guide got %d", len(payload.Guides))
	}
	guide := payload.Guides[0]
	if guide.HeroImageURL != "https://cdn.example.com/images/hero.jpg" {
		t.Fatalf("expected resolved hero url got %s", guide.HeroImageURL)
	}
	if guide.Summary != "Learn the basics" {
		t.Fatalf("expected trimmed summary got %q", guide.Summary)
	}
	if len(guide.Tags) != 1 || guide.Tags[0] != "Etiquette" {
		t.Fatalf("expected deduped tags got %#v", guide.Tags)
	}

	if stub.listFilter.Locale == nil || *stub.listFilter.Locale != "en-us" {
		t.Fatalf("expected locale filter en-us got %#v", stub.listFilter.Locale)
	}
	if stub.listFilter.Category == nil || *stub.listFilter.Category != "culture" {
		t.Fatalf("expected normalized category got %#v", stub.listFilter.Category)
	}
	if stub.listFilter.FallbackLocale != defaultGuideLocale {
		t.Fatalf("expected fallback locale %s got %q", defaultGuideLocale, stub.listFilter.FallbackLocale)
	}
	if stub.listFilter.Pagination.PageSize != 10 {
		t.Fatalf("expected page size 10 got %d", stub.listFilter.Pagination.PageSize)
	}
	if stub.listFilter.Pagination.PageToken != "token" {
		t.Fatalf("expected trimmed token got %q", stub.listFilter.Pagination.PageToken)
	}
}

func TestPublicHandlers_GetGuide_SanitizesHTML(t *testing.T) {
	guide := services.ContentGuide{
		ID:          "guide_1",
		Slug:        "tea-ceremony",
		Locale:      "ja",
		Title:       "Tea Ceremony",
		BodyHTML:    "<p>Safe</p><script>alert('x')</script>",
		HeroImage:   "images/hero.jpg",
		Status:      "published",
		IsPublished: true,
		UpdatedAt:   time.Date(2024, time.February, 10, 0, 0, 0, 0, time.UTC),
		PublishedAt: time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
	}
	stub := &stubContentService{detailGuide: guide}

	publicHandlers := NewPublicHandlers(
		WithPublicContentService(stub),
		WithPublicPreviewResolver(AssetURLResolverFunc(func(_ context.Context, path string) (string, error) {
			return "https://cdn.example.com/" + path, nil
		})),
	)

	router := chi.NewRouter()
	router.Route("/", publicHandlers.Routes)

	req := httptest.NewRequest(http.MethodGet, "/content/guides/tea-ceremony?lang=en", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}

	var payload guideDetailPayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if strings.Contains(payload.BodyHTML, "script") {
		t.Fatalf("expected sanitized body, got %q", payload.BodyHTML)
	}
	if payload.BodyHTML != "<p>Safe</p>" {
		t.Fatalf("expected preserved paragraph got %q", payload.BodyHTML)
	}
	if payload.HeroImageURL != "https://cdn.example.com/images/hero.jpg" {
		t.Fatalf("expected resolved hero url got %s", payload.HeroImageURL)
	}
	if stub.getSlug != "tea-ceremony" || stub.getLocale != "en" {
		t.Fatalf("expected service called with slug/locale got %s/%s", stub.getSlug, stub.getLocale)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("expected ETag header")
	}
}

func TestPublicHandlers_GetGuide_NotModified(t *testing.T) {
	guide := services.ContentGuide{
		ID:          "guide_1",
		Slug:        "tea-ceremony",
		Locale:      "ja",
		Status:      "published",
		IsPublished: true,
		UpdatedAt:   time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC),
	}
	stub := &stubContentService{detailGuide: guide}

	handler := NewPublicHandlers(WithPublicContentService(stub))
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	etag := computeGuideETag(guide)
	req := httptest.NewRequest(http.MethodGet, "/content/guides/tea-ceremony", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("expected 304 got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body on 304")
	}
	if rec.Header().Get("ETag") != etag {
		t.Fatalf("expected ETag to be preserved")
	}
}

func TestPublicHandlers_GetPage_SanitizesHTMLAndSetsHeaders(t *testing.T) {
	t.Helper()

	updated := time.Date(2024, time.March, 10, 9, 0, 0, 0, time.FixedZone("JST", 9*3600))
	stub := &stubContentService{
		pageDetail: services.ContentPage{
			ID:       "page_1",
			Slug:     "about",
			Locale:   "JA",
			Title:    " About ",
			BodyHTML: " <p>Hello</p><script>alert('x')</script> ",
			SEO: map[string]string{
				"title":       " About ",
				"description": " Learn ",
				" ":           "ignored",
			},
			Status:      "published",
			IsPublished: true,
			UpdatedAt:   updated,
		},
	}

	handler := NewPublicHandlers(WithPublicContentService(stub))

	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/content/pages/about?lang=en", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rec.Code)
	}
	if stub.pageSlug != "about" {
		t.Fatalf("expected slug about got %q", stub.pageSlug)
	}
	if stub.pageLocale != "en" {
		t.Fatalf("expected locale en got %q", stub.pageLocale)
	}
	if rec.Header().Get("Cache-Control") != pageCacheControl {
		t.Fatalf("expected Cache-Control %s", pageCacheControl)
	}
	if rec.Header().Get("ETag") == "" {
		t.Fatalf("expected ETag header")
	}

	var payload contentPagePayload
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Slug != "about" {
		t.Fatalf("expected slug about got %q", payload.Slug)
	}
	expectedLocale := normalizeLocale(stub.pageDetail.Locale)
	if payload.Locale != expectedLocale {
		t.Fatalf("expected locale %s got %q", expectedLocale, payload.Locale)
	}
	if payload.Title != "About" {
		t.Fatalf("expected trimmed title got %q", payload.Title)
	}
	if payload.BodyHTML != "<p>Hello</p>" {
		t.Fatalf("expected sanitized body got %q", payload.BodyHTML)
	}
	if strings.Contains(payload.BodyHTML, "script") {
		t.Fatalf("expected no script tag got %q", payload.BodyHTML)
	}
	if !payload.IsPublished {
		t.Fatalf("expected published flag true")
	}
	if payload.UpdatedAt != updated.UTC().Format(time.RFC3339) {
		t.Fatalf("expected updated at %s got %s", updated.UTC().Format(time.RFC3339), payload.UpdatedAt)
	}
	if len(payload.SEO) != 2 {
		t.Fatalf("expected trimmed seo map got %#v", payload.SEO)
	}
	if payload.SEO["title"] != "About" {
		t.Fatalf("expected trimmed seo title got %q", payload.SEO["title"])
	}
	if payload.SEO["description"] != "Learn" {
		t.Fatalf("expected trimmed seo description got %q", payload.SEO["description"])
	}
}

func TestPublicHandlers_GetPage_ReturnsNotModifiedWhenETagMatches(t *testing.T) {
	t.Helper()

	page := services.ContentPage{
		ID:          "page_1",
		Slug:        "about",
		Locale:      "ja",
		Title:       "About",
		BodyHTML:    "<p>Hello</p>",
		Status:      "published",
		IsPublished: true,
		UpdatedAt:   time.Date(2024, time.March, 10, 0, 0, 0, 0, time.UTC),
	}
	stub := &stubContentService{pageDetail: page}

	handler := NewPublicHandlers(WithPublicContentService(stub))
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	etag := computePageETag(page, "about")

	req := httptest.NewRequest(http.MethodGet, "/content/pages/about", nil)
	req.Header.Set("If-None-Match", etag)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotModified {
		t.Fatalf("expected 304 got %d", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("expected empty body on 304")
	}
	if rec.Header().Get("ETag") != etag {
		t.Fatalf("expected ETag to persist got %q", rec.Header().Get("ETag"))
	}
	if stub.pageSlug != "about" {
		t.Fatalf("expected slug recorded got %q", stub.pageSlug)
	}
}

func TestPublicHandlers_GetPage_ReturnsNotFoundForUnpublished(t *testing.T) {
	t.Helper()

	stub := &stubContentService{
		pageDetail: services.ContentPage{
			ID:          "page_1",
			Slug:        "about",
			Locale:      "ja",
			Title:       "About",
			Status:      "draft",
			IsPublished: false,
		},
	}

	handler := NewPublicHandlers(WithPublicContentService(stub))
	router := chi.NewRouter()
	router.Route("/", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/content/pages/about", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rec.Code)
	}
}

type stubContentService struct {
	listFilter   services.ContentGuideFilter
	listResponse domain.CursorPage[services.ContentGuide]
	listErr      error

	detailGuide services.ContentGuide
	detailErr   error
	getSlug     string
	getLocale   string

	pageDetail services.ContentPage
	pageErr    error
	pageSlug   string
	pageLocale string
}

func (s *stubContentService) ListGuides(_ context.Context, filter services.ContentGuideFilter) (domain.CursorPage[services.ContentGuide], error) {
	s.listFilter = filter
	return s.listResponse, s.listErr
}

func (s *stubContentService) GetGuideBySlug(_ context.Context, slug string, locale string) (services.ContentGuide, error) {
	s.getSlug = slug
	s.getLocale = locale
	if s.detailErr != nil {
		return services.ContentGuide{}, s.detailErr
	}
	return s.detailGuide, nil
}

func (s *stubContentService) GetGuide(context.Context, string) (services.ContentGuide, error) {
	return services.ContentGuide{}, errors.New("not implemented")
}

func (s *stubContentService) UpsertGuide(context.Context, services.UpsertContentGuideCommand) (services.ContentGuide, error) {
	return services.ContentGuide{}, errors.New("not implemented")
}

func (s *stubContentService) DeleteGuide(context.Context, string) error {
	return nil
}

func (s *stubContentService) GetPage(_ context.Context, slug string, locale string) (services.ContentPage, error) {
	s.pageSlug = slug
	s.pageLocale = locale
	if s.pageErr != nil {
		return services.ContentPage{}, s.pageErr
	}
	return s.pageDetail, nil
}

func (s *stubContentService) UpsertPage(context.Context, services.UpsertContentPageCommand) (services.ContentPage, error) {
	return services.ContentPage{}, errors.New("not implemented")
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
	productListFilter  services.ProductFilter
	productListResp    domain.CursorPage[services.ProductSummary]
	productListErr     error

	getID          string
	getTemplate    domain.Template
	getErr         error
	fontGetID      string
	fontGetFont    services.Font
	fontGetErr     error
	materialGetID  string
	materialGetMat services.Material
	materialGetErr error
	productGetID   string
	productGetProd services.Product
	productGetErr  error

	adminUpsertCmd          services.UpsertTemplateCommand
	adminUpsertResp         services.Template
	adminUpsertErr          error
	adminDeleteCmd          services.DeleteTemplateCommand
	adminDeleteErr          error
	adminFontUpsertCmd      services.UpsertFontCommand
	adminFontUpsertResp     services.FontSummary
	adminFontUpsertErr      error
	adminFontDeleteID       string
	adminFontDeleteErr      error
	adminMaterialUpsertCmd  services.UpsertMaterialCommand
	adminMaterialUpsertResp services.MaterialSummary
	adminMaterialUpsertErr  error
	adminMaterialDeleteCmd  services.DeleteMaterialCommand
	adminMaterialDeleteErr  error
	adminProductUpsertCmd   services.UpsertProductCommand
	adminProductUpsertResp  services.Product
	adminProductUpsertErr   error
	adminProductDeleteCmd   services.DeleteProductCommand
	adminProductDeleteErr   error
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

func (s *stubCatalogService) UpsertTemplate(_ context.Context, cmd services.UpsertTemplateCommand) (services.Template, error) {
	s.adminUpsertCmd = cmd
	if s.adminUpsertErr != nil {
		return services.Template{}, s.adminUpsertErr
	}
	if s.adminUpsertResp.ID != "" {
		return s.adminUpsertResp, nil
	}
	return cmd.Template, nil
}

func (s *stubCatalogService) DeleteTemplate(_ context.Context, cmd services.DeleteTemplateCommand) error {
	s.adminDeleteCmd = cmd
	return s.adminDeleteErr
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

func (s *stubCatalogService) UpsertFont(_ context.Context, cmd services.UpsertFontCommand) (services.FontSummary, error) {
	s.adminFontUpsertCmd = cmd
	if s.adminFontUpsertErr != nil {
		return services.FontSummary{}, s.adminFontUpsertErr
	}
	if s.adminFontUpsertResp.ID != "" {
		return s.adminFontUpsertResp, nil
	}
	return cmd.Font, nil
}

func (s *stubCatalogService) DeleteFont(_ context.Context, fontID string) error {
	s.adminFontDeleteID = fontID
	return s.adminFontDeleteErr
}

func (s *stubCatalogService) ListMaterials(_ context.Context, filter services.MaterialFilter) (domain.CursorPage[services.MaterialSummary], error) {
	s.materialListFilter = filter
	return s.materialListResp, s.materialListErr
}

func (s *stubCatalogService) GetMaterial(_ context.Context, materialID string) (services.Material, error) {
	s.materialGetID = materialID
	if s.materialGetErr != nil {
		return services.Material{}, s.materialGetErr
	}
	return s.materialGetMat, nil
}

func (s *stubCatalogService) UpsertMaterial(_ context.Context, cmd services.UpsertMaterialCommand) (services.MaterialSummary, error) {
	s.adminMaterialUpsertCmd = cmd
	if s.adminMaterialUpsertErr != nil {
		return services.MaterialSummary{}, s.adminMaterialUpsertErr
	}
	if s.adminMaterialUpsertResp.ID != "" {
		return s.adminMaterialUpsertResp, nil
	}
	return cmd.Material, nil
}

func (s *stubCatalogService) DeleteMaterial(_ context.Context, cmd services.DeleteMaterialCommand) error {
	s.adminMaterialDeleteCmd = cmd
	return s.adminMaterialDeleteErr
}

func (s *stubCatalogService) ListProducts(_ context.Context, filter services.ProductFilter) (domain.CursorPage[services.ProductSummary], error) {
	s.productListFilter = filter
	return s.productListResp, s.productListErr
}

func (s *stubCatalogService) GetProduct(_ context.Context, productID string) (services.Product, error) {
	s.productGetID = productID
	if s.productGetErr != nil {
		return services.Product{}, s.productGetErr
	}
	return s.productGetProd, nil
}

func (s *stubCatalogService) UpsertProduct(_ context.Context, cmd services.UpsertProductCommand) (services.Product, error) {
	s.adminProductUpsertCmd = cmd
	if s.adminProductUpsertErr != nil {
		return services.Product{}, s.adminProductUpsertErr
	}
	if s.adminProductUpsertResp.ID != "" {
		return s.adminProductUpsertResp, nil
	}
	return cmd.Product, nil
}

func (s *stubCatalogService) DeleteProduct(_ context.Context, cmd services.DeleteProductCommand) error {
	s.adminProductDeleteCmd = cmd
	return s.adminProductDeleteErr
}

type stubPromotionService struct {
	result services.PromotionPublic
	err    error
	code   string
}

func (s *stubPromotionService) GetPublicPromotion(_ context.Context, code string) (services.PromotionPublic, error) {
	s.code = code
	if s.err != nil {
		return services.PromotionPublic{}, s.err
	}
	res := s.result
	if strings.TrimSpace(res.Code) == "" {
		res.Code = code
	}
	return res, nil
}

func (s *stubPromotionService) ValidatePromotion(context.Context, services.ValidatePromotionCommand) (services.PromotionValidationResult, error) {
	return services.PromotionValidationResult{}, errors.New("not implemented")
}

func (s *stubPromotionService) ListPromotions(context.Context, services.PromotionListFilter) (domain.CursorPage[services.Promotion], error) {
	return domain.CursorPage[services.Promotion]{}, errors.New("not implemented")
}

func (s *stubPromotionService) CreatePromotion(context.Context, services.UpsertPromotionCommand) (services.Promotion, error) {
	return services.Promotion{}, errors.New("not implemented")
}

func (s *stubPromotionService) UpdatePromotion(context.Context, services.UpsertPromotionCommand) (services.Promotion, error) {
	return services.Promotion{}, errors.New("not implemented")
}

func (s *stubPromotionService) DeletePromotion(context.Context, string) error {
	return errors.New("not implemented")
}

func (s *stubPromotionService) ListPromotionUsage(context.Context, services.PromotionUsageFilter) (domain.CursorPage[services.PromotionUsage], error) {
	return domain.CursorPage[services.PromotionUsage]{}, errors.New("not implemented")
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
