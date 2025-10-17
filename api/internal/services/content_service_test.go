package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	domain "github.com/hanko-field/api/internal/domain"
	"github.com/hanko-field/api/internal/repositories"
)

func TestContentService_ListGuides_NormalizesFields(t *testing.T) {
	t.Helper()

	now := time.Date(2024, time.January, 5, 12, 30, 0, 0, time.FixedZone("JST", 9*3600))
	updated := now.Add(2 * time.Hour)

	stubRepo := &stubContentRepository{
		listResponse: domain.CursorPage[domain.ContentGuide]{
			Items: []domain.ContentGuide{
				{
					ID:          "guide_1",
					Slug:        " tea-ceremony ",
					Locale:      "EN_us",
					Category:    " Culture ",
					Title:       " Tea Ceremony ",
					Summary:     "  Learn the basics ",
					BodyHTML:    "<p>body</p>",
					HeroImage:   "images/hero.jpg",
					Tags:        []string{"Etiquette", " etiquette ", "Culture"},
					Status:      "Published",
					CreatedAt:   now,
					UpdatedAt:   updated,
					PublishedAt: now,
				},
			},
			NextPageToken: "next-token",
		},
	}

	service, err := NewContentService(ContentServiceDeps{
		Repository: stubRepo,
		Clock:      func() time.Time { return updated.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewContentService: %v", err)
	}

	requestLocale := "en-US"
	filter := ContentGuideFilter{
		Locale:        &requestLocale,
		PublishedOnly: true,
		Pagination: Pagination{
			PageSize:  10,
			PageToken: " token ",
		},
	}

	page, err := service.ListGuides(context.Background(), filter)
	if err != nil {
		t.Fatalf("ListGuides: %v", err)
	}

	if len(stubRepo.listFilters) != 1 {
		t.Fatalf("expected one filter call, got %d", len(stubRepo.listFilters))
	}
	captured := stubRepo.listFilters[0]
	if captured.Locale == nil || *captured.Locale != "en-us" {
		t.Fatalf("expected locale en-us, got %#v", captured.Locale)
	}
	if captured.FallbackLocale != defaultContentLocale {
		t.Fatalf("expected fallback locale %s, got %q", defaultContentLocale, captured.FallbackLocale)
	}
	if captured.OnlyPublished != true {
		t.Fatalf("expected OnlyPublished true")
	}
	if captured.Pagination.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", captured.Pagination.PageSize)
	}
	if captured.Pagination.PageToken != "token" {
		t.Fatalf("expected trimmed page token, got %q", captured.Pagination.PageToken)
	}

	if page.NextPageToken != "next-token" {
		t.Fatalf("unexpected next token %q", page.NextPageToken)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected one guide, got %d", len(page.Items))
	}
	guide := page.Items[0]
	if guide.Slug != "tea-ceremony" {
		t.Fatalf("expected slug trimmed, got %q", guide.Slug)
	}
	if guide.Locale != "en-us" {
		t.Fatalf("expected normalized locale, got %q", guide.Locale)
	}
	if guide.Category != "Culture" {
		t.Fatalf("expected category trimmed, got %q", guide.Category)
	}
	if guide.Title != "Tea Ceremony" {
		t.Fatalf("expected trimmed title, got %q", guide.Title)
	}
	if guide.Summary != "Learn the basics" {
		t.Fatalf("expected trimmed summary, got %q", guide.Summary)
	}
	if len(guide.Tags) != 2 {
		t.Fatalf("expected deduped tags, got %#v", guide.Tags)
	}
	if !guide.IsPublished {
		t.Fatalf("expected IsPublished true")
	}
	if guide.CreatedAt.Location() != time.UTC {
		t.Fatalf("expected createdAt in UTC, got %v", guide.CreatedAt.Location())
	}
	if guide.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected updatedAt in UTC, got %v", guide.UpdatedAt.Location())
	}
	if guide.PublishedAt.Location() != time.UTC {
		t.Fatalf("expected publishedAt in UTC, got %v", guide.PublishedAt.Location())
	}
}

func TestContentService_GetGuideBySlug_Fallback(t *testing.T) {
	t.Helper()

	stubRepo := &stubContentRepository{
		guidesBySlug: map[string]domain.ContentGuide{
			"ja|tea-ceremony": {
				ID:          "guide_1",
				Slug:        "tea-ceremony",
				Locale:      "ja",
				Title:       "Tea Ceremony",
				Status:      "published",
				UpdatedAt:   time.Now(),
				PublishedAt: time.Now(),
			},
		},
		getErr: stubRepoError{notFound: true},
	}

	service, err := NewContentService(ContentServiceDeps{
		Repository:    stubRepo,
		DefaultLocale: "ja",
	})
	if err != nil {
		t.Fatalf("NewContentService: %v", err)
	}

	guide, err := service.GetGuideBySlug(context.Background(), "tea-ceremony", "en-US")
	if err != nil {
		t.Fatalf("GetGuideBySlug: %v", err)
	}

	if len(stubRepo.getBySlugCalls) != 2 {
		t.Fatalf("expected fallback call, got %d", len(stubRepo.getBySlugCalls))
	}
	if stubRepo.getBySlugCalls[0].locale != "en-us" {
		t.Fatalf("expected first call en-us, got %q", stubRepo.getBySlugCalls[0].locale)
	}
	if stubRepo.getBySlugCalls[1].locale != "ja" {
		t.Fatalf("expected fallback locale ja, got %q", stubRepo.getBySlugCalls[1].locale)
	}

	if guide.Locale != "ja" {
		t.Fatalf("expected resolved locale ja, got %q", guide.Locale)
	}
	if !guide.IsPublished {
		t.Fatalf("expected published guide")
	}
}

func TestContentService_ListGuides_PageSizeValidation(t *testing.T) {
	t.Helper()

	stubRepo := &stubContentRepository{}
	service, err := NewContentService(ContentServiceDeps{
		Repository: stubRepo,
	})
	if err != nil {
		t.Fatalf("NewContentService: %v", err)
	}

	if _, err := service.ListGuides(context.Background(), ContentGuideFilter{}); err != nil {
		t.Fatalf("ListGuides default: %v", err)
	}
	if stubRepo.listFilters[0].Pagination.PageSize != defaultGuidePageSize {
		t.Fatalf("expected default page size %d got %d", defaultGuidePageSize, stubRepo.listFilters[0].Pagination.PageSize)
	}

	large := ContentGuideFilter{Pagination: Pagination{PageSize: 500}}
	if _, err := service.ListGuides(context.Background(), large); err != nil {
		t.Fatalf("ListGuides large: %v", err)
	}
	if stubRepo.listFilters[1].Pagination.PageSize != maxGuidePageSize {
		t.Fatalf("expected capped page size %d got %d", maxGuidePageSize, stubRepo.listFilters[1].Pagination.PageSize)
	}
}

func TestContentService_GetPage_FallbackAndNormalization(t *testing.T) {
	t.Helper()

	updated := time.Date(2024, time.March, 1, 15, 30, 0, 0, time.FixedZone("JST", 9*3600))

	defaultPage := domain.ContentPage{
		ID:          "page_default",
		Slug:        "about",
		Locale:      "",
		Title:       " About Us ",
		BodyHTML:    " <p>Hello</p> ",
		Status:      " Published ",
		IsPublished: false,
		UpdatedAt:   updated,
		SEO: map[string]string{
			"title":       " About ",
			"description": " Learn ",
			" ":           "ignored",
		},
	}

	stubRepo := &stubContentRepository{
		pages: map[string]domain.ContentPage{
			"ja|about": defaultPage,
		},
		pageErr: stubRepoError{notFound: true},
	}

	service, err := NewContentService(ContentServiceDeps{
		Repository:    stubRepo,
		DefaultLocale: "ja",
	})
	if err != nil {
		t.Fatalf("NewContentService: %v", err)
	}

	page, err := service.GetPage(context.Background(), " about ", "EN")
	if err != nil {
		t.Fatalf("GetPage: %v", err)
	}

	if len(stubRepo.getPageCalls) != 2 {
		t.Fatalf("expected fallback call, got %d", len(stubRepo.getPageCalls))
	}
	if stubRepo.getPageCalls[0].locale != "en" {
		t.Fatalf("expected first locale en, got %q", stubRepo.getPageCalls[0].locale)
	}
	if stubRepo.getPageCalls[1].locale != "ja" {
		t.Fatalf("expected fallback locale ja, got %q", stubRepo.getPageCalls[1].locale)
	}

	if page.Locale != "ja" {
		t.Fatalf("expected locale ja, got %q", page.Locale)
	}
	if page.Slug != "about" {
		t.Fatalf("expected trimmed slug, got %q", page.Slug)
	}
	if page.Title != "About Us" {
		t.Fatalf("expected trimmed title, got %q", page.Title)
	}
	if page.BodyHTML != "<p>Hello</p>" {
		t.Fatalf("expected trimmed body html, got %q", page.BodyHTML)
	}
	if !page.IsPublished {
		t.Fatalf("expected published page inferred from status")
	}
	if page.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected updated at UTC, got %v", page.UpdatedAt.Location())
	}
	if got := page.UpdatedAt; !got.Equal(updated.UTC()) {
		t.Fatalf("expected updated at %v got %v", updated.UTC(), got)
	}
	if len(page.SEO) != 2 {
		t.Fatalf("expected seo entries trimmed, got %#v", page.SEO)
	}
	if page.SEO["title"] != "About" {
		t.Fatalf("expected trimmed seo title, got %q", page.SEO["title"])
	}
	if page.SEO["description"] != "Learn" {
		t.Fatalf("expected trimmed seo description, got %q", page.SEO["description"])
	}
}

type stubContentRepository struct {
	listFilters    []repositories.ContentGuideFilter
	listResponse   domain.CursorPage[domain.ContentGuide]
	listErr        error
	guidesBySlug   map[string]domain.ContentGuide
	getErr         error
	getBySlugCalls []struct {
		slug   string
		locale string
	}
	pages        map[string]domain.ContentPage
	pageErr      error
	getPageCalls []struct {
		slug   string
		locale string
	}
}

func (s *stubContentRepository) ListGuides(_ context.Context, filter repositories.ContentGuideFilter) (domain.CursorPage[domain.ContentGuide], error) {
	s.listFilters = append(s.listFilters, filter)
	return s.listResponse, s.listErr
}

func (s *stubContentRepository) UpsertGuide(_ context.Context, guide domain.ContentGuide) (domain.ContentGuide, error) {
	if s.guidesBySlug == nil {
		s.guidesBySlug = make(map[string]domain.ContentGuide)
	}
	key := normalizeLocaleValue(guide.Locale) + "|" + guide.Slug
	s.guidesBySlug[key] = guide
	return guide, nil
}

func (s *stubContentRepository) DeleteGuide(context.Context, string) error {
	return nil
}

func (s *stubContentRepository) GetGuideBySlug(_ context.Context, slug string, locale string) (domain.ContentGuide, error) {
	if s.guidesBySlug == nil {
		s.guidesBySlug = make(map[string]domain.ContentGuide)
	}
	normalized := normalizeLocaleValue(locale)
	s.getBySlugCalls = append(s.getBySlugCalls, struct {
		slug   string
		locale string
	}{slug: strings.TrimSpace(slug), locale: normalized})
	if guide, ok := s.guidesBySlug[normalized+"|"+slug]; ok {
		return guide, nil
	}
	if s.getErr != nil {
		return domain.ContentGuide{}, s.getErr
	}
	return domain.ContentGuide{}, stubRepoError{notFound: true}
}

func (s *stubContentRepository) GetGuide(context.Context, string) (domain.ContentGuide, error) {
	return domain.ContentGuide{}, errors.New("not implemented")
}

func (s *stubContentRepository) GetPage(_ context.Context, slug string, locale string) (domain.ContentPage, error) {
	if s.pages == nil {
		s.pages = make(map[string]domain.ContentPage)
	}
	normalizedLocale := normalizeLocaleValue(locale)
	trimmedSlug := strings.TrimSpace(slug)
	s.getPageCalls = append(s.getPageCalls, struct {
		slug   string
		locale string
	}{
		slug:   trimmedSlug,
		locale: normalizedLocale,
	})
	if page, ok := s.pages[normalizedLocale+"|"+trimmedSlug]; ok {
		return page, nil
	}
	if s.pageErr != nil {
		return domain.ContentPage{}, s.pageErr
	}
	return domain.ContentPage{}, stubRepoError{notFound: true}
}

func (s *stubContentRepository) UpsertPage(_ context.Context, page domain.ContentPage) (domain.ContentPage, error) {
	if s.pages == nil {
		s.pages = make(map[string]domain.ContentPage)
	}
	key := normalizeLocaleValue(page.Locale) + "|" + strings.TrimSpace(page.Slug)
	s.pages[key] = page
	return page, nil
}

func (s *stubContentRepository) DeletePage(context.Context, string) error {
	return errors.New("not implemented")
}

type stubRepoError struct {
	notFound    bool
	unavailable bool
}

func (e stubRepoError) Error() string {
	return "content repository error"
}

func (e stubRepoError) IsNotFound() bool {
	return e.notFound
}

func (e stubRepoError) IsConflict() bool {
	return false
}

func (e stubRepoError) IsUnavailable() bool {
	return e.unavailable
}
